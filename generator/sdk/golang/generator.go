// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package gosdk

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"go/format"
	"go/token"
	"sort"
	"strconv"
	"strings"
	"text/template"

	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	highv3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	modelgen "github.com/pb33f/libopenapi/generator/golang"
	"github.com/pb33f/libopenapi/generator/sdk"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

//go:embed templates/client.tmpl
var clientTemplateSource string

//go:embed templates/resources.tmpl
var resourcesTemplateSource string

//go:embed templates/workflows.tmpl
var workflowsTemplateSource string

var templateFunctions = template.FuncMap{
	"quote":   strconv.Quote,
	"comment": commentLine,
}

var clientTemplate = template.Must(template.New("client").Funcs(templateFunctions).Parse(clientTemplateSource))
var resourcesTemplate = template.Must(template.New("resources").Funcs(templateFunctions).Parse(resourcesTemplateSource))
var workflowsTemplate = template.Must(template.New("workflows").Funcs(templateFunctions).Parse(workflowsTemplateSource))

// Generate prepares an OpenAPI document and emits a Go SDK without workflows.
// Call sdk.Prepare, sdk.PrepareArazzo, and GenerateContract when generating
// finite Arazzo workflow helpers.
func Generate(document *highv3.Document, options Options) (*sdk.Result, error) {
	if len(options.Workflows) > 0 {
		return nil, errors.New("sdk/golang: Generate does not accept workflows; prepare the contract and call GenerateContract")
	}
	contract, err := sdk.Prepare(document, options.Prepare)
	if err != nil {
		return nil, err
	}
	return GenerateContract(contract, options)
}

// GenerateContract emits a Go SDK from a prepared contract.
func GenerateContract(contract *sdk.Contract, options Options) (*sdk.Result, error) {
	if contract == nil {
		return nil, errors.New("sdk/golang: contract is required")
	}
	if len(contract.Operations) == 0 {
		return nil, errors.New("sdk/golang: contract contains no selected operations")
	}
	packageName := options.packageName()
	if !token.IsIdentifier(packageName) || token.Lookup(packageName).IsKeyword() {
		return nil, fmt.Errorf("sdk/golang: invalid package name %q", packageName)
	}
	var sdkEmitter *emitter
	modelOptions := append([]modelgen.Option(nil), options.Models...)
	modelOptions = append(modelOptions,
		modelgen.WithPackageName(packageName),
		modelgen.WithGeneratedComment(true),
		modelgen.WithOptionalNullableAsDoublePointer(true),
		modelgen.WithTypeNameResolver(func(name string) string { return sdkEmitter.typeNames[name] }),
	)
	modelsGenerator := modelgen.NewGenerator(modelOptions...)
	sdkEmitter = newEmitter(contract, packageName, options.Workflows, modelsGenerator)
	view, err := sdkEmitter.prepareView()
	if err != nil {
		return nil, err
	}
	models, err := modelsGenerator.RenderSchemas(sdkEmitter.schemas)
	if err != nil {
		return nil, fmt.Errorf("sdk/golang: generate models: %w", err)
	}
	if err := resolveWorkflowFieldTypes(models.Types, view.Workflows); err != nil {
		return nil, err
	}
	client, err := render(clientTemplate, view)
	if err != nil {
		return nil, err
	}
	resources, err := render(resourcesTemplate, view)
	if err != nil {
		return nil, err
	}
	result := &sdk.Result{Files: []sdk.File{
		{Path: "models.gen.go", Content: models.Source},
		{Path: "client.gen.go", Content: client},
		{Path: "resources.gen.go", Content: resources},
	}}
	if len(view.Workflows) > 0 {
		workflows, err := render(workflowsTemplate, view)
		if err != nil {
			return nil, err
		}
		result.Files = append(result.Files, sdk.File{Path: "workflows.gen.go", Content: workflows})
	}
	for _, diagnostic := range models.Diagnostics {
		result.Diagnostics = append(result.Diagnostics, sdk.Diagnostic{
			Code: diagnostic.Code, Path: diagnostic.Path, Message: diagnostic.Message,
		})
	}
	return result, nil
}

func render(tmpl *template.Template, value any) ([]byte, error) {
	var output bytes.Buffer
	if err := tmpl.Execute(&output, value); err != nil {
		return nil, fmt.Errorf("sdk/golang: execute %s template: %w", tmpl.Name(), err)
	}
	formatted, err := format.Source(output.Bytes())
	if err != nil {
		return nil, fmt.Errorf("sdk/golang: format %s: %w\n%s", tmpl.Name(), err, output.Bytes())
	}
	return formatted, nil
}

type emitter struct {
	contract            *sdk.Contract
	packageName         string
	componentSchemas    *orderedmap.Map[string, *highbase.SchemaProxy]
	schemas             *orderedmap.Map[string, *highbase.SchemaProxy]
	typeNames           map[string]string
	names               *modelgen.NameRegistry
	collectedComponents map[string]struct{}
	reachableComponents map[string]struct{}
	visitedSchemas      map[*highbase.SchemaProxy]struct{}
	collectErr          error
	workflows           []*sdk.Workflow
	models              *modelgen.Generator
}

func newEmitter(contract *sdk.Contract, packageName string, workflows []*sdk.Workflow, models *modelgen.Generator) *emitter {
	reserved := []string{
		"APIError", "APIKey", "BasicAuth", "BearerToken", "Client", "Credential", "CredentialFunc",
		"DefaultServer", "File", "HTTPDoer", "MutualTLS", "NewClient", "Null", "NullableValue",
		"Option", "RequestOption", "Response", "ResponseDecodeError", "SecurityScheme",
		"WithCredential", "WithDefaultHeader", "WithHTTPClient", "WithMaxResponseBody", "WithRequestHeader", "WithTimeout",
	}
	names := modelgen.NewNameRegistry(reserved...)
	components := contract.Schemas
	if components == nil {
		components = orderedmap.New[string, *highbase.SchemaProxy]()
	}
	emitter := &emitter{
		contract: contract, packageName: packageName, componentSchemas: components,
		schemas: orderedmap.New[string, *highbase.SchemaProxy](), typeNames: make(map[string]string),
		names: names, collectedComponents: make(map[string]struct{}),
		reachableComponents: make(map[string]struct{}), visitedSchemas: make(map[*highbase.SchemaProxy]struct{}),
		workflows: workflows, models: models,
	}
	emitter.findReachableComponents()
	for name := range components.FromOldest() {
		if _, reachable := emitter.reachableComponents[name]; reachable {
			emitter.typeNames[name] = emitter.names.Claim(modelgen.PublicName(name), "Model")
		}
	}
	return emitter
}

func (e *emitter) findReachableComponents() {
	visited := make(map[*highbase.SchemaProxy]struct{})
	var walk func(*highbase.SchemaProxy)
	walk = func(proxy *highbase.SchemaProxy) {
		if proxy == nil {
			return
		}
		if _, seen := visited[proxy]; seen {
			return
		}
		visited[proxy] = struct{}{}
		if proxy.IsReference() {
			name, component := componentSchemaRefName(proxy.GetReference())
			if !component {
				return
			}
			e.reachableComponents[name] = struct{}{}
			if target, ok := e.componentSchemas.Get(name); ok {
				walk(target)
			}
			return
		}
		schema := proxy.Schema()
		if schema == nil {
			return
		}
		for _, child := range schema.AllOf {
			walk(child)
		}
		for _, child := range schema.OneOf {
			walk(child)
		}
		for _, child := range schema.AnyOf {
			walk(child)
		}
		if schema.Items != nil && schema.Items.IsA() {
			walk(schema.Items.A)
		}
		if schema.AdditionalProperties != nil && schema.AdditionalProperties.IsA() {
			walk(schema.AdditionalProperties.A)
		}
		for _, children := range []*orderedmap.Map[string, *highbase.SchemaProxy]{schema.Properties, schema.PatternProperties} {
			if children != nil {
				for _, child := range children.FromOldest() {
					walk(child)
				}
			}
		}
	}
	for _, operation := range e.contract.Operations {
		if operation == nil {
			continue
		}
		for _, parameter := range operation.Parameters {
			if parameter != nil {
				walk(parameter.Schema)
			}
		}
		if operation.RequestBody != nil && operation.RequestBody.Content != nil {
			for _, mediaType := range operation.RequestBody.Content.FromOldest() {
				if mediaType != nil {
					walk(mediaType.Schema)
				}
			}
		}
		for _, response := range operation.Responses {
			if response != nil && response.Content != nil {
				for _, mediaType := range response.Content.FromOldest() {
					if mediaType != nil {
						walk(mediaType.Schema)
					}
				}
			}
		}
	}
}

func componentSchemaRefName(ref string) (string, bool) {
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(ref, prefix) || len(ref) == len(prefix) || strings.Contains(ref[len(prefix):], "/") {
		return "", false
	}
	return modelgen.RefName(ref), true
}

func (e *emitter) prepareView() (*clientView, error) {
	view := &clientView{PackageName: e.packageName}
	if len(e.contract.Servers) > 0 {
		view.DefaultServer = e.contract.Servers[0]
	}
	knownSchemes := make(map[string]struct{}, len(e.contract.SecuritySchemes))
	for _, scheme := range e.contract.SecuritySchemes {
		if scheme == nil {
			continue
		}
		view.SecuritySchemes = append(view.SecuritySchemes, securitySchemeView{
			Name: scheme.Name, Type: scheme.Type, In: scheme.In,
			ParameterName: scheme.ParameterName, Scheme: scheme.Scheme,
		})
		knownSchemes[scheme.Name] = struct{}{}
	}
	resourceIndexes := make(map[string]int)
	resourceMethods := make(map[string]map[string]string)
	for _, operation := range e.contract.Operations {
		if operation == nil {
			continue
		}
		if view.DefaultServer == "" && len(operation.Servers) > 0 {
			view.DefaultServer = operation.Servers[0]
		}
		if len(operation.Servers) > 0 && view.DefaultServer != "" && operation.Servers[0] != view.DefaultServer {
			return nil, fmt.Errorf("sdk/golang: operation %q uses unsupported operation-specific server %q", operation.ID, operation.Servers[0])
		}
		for _, alternative := range operation.Security {
			if alternative == nil {
				continue
			}
			for _, scheme := range alternative.Schemes {
				if scheme == nil {
					continue
				}
				if _, ok := knownSchemes[scheme.Name]; !ok {
					return nil, fmt.Errorf("sdk/golang: operation %q references undefined security scheme %q", operation.ID, scheme.Name)
				}
			}
		}
		resourceName := modelgen.PublicName(operation.Resource)
		resourceIndex, exists := resourceIndexes[resourceName]
		if !exists {
			resourceIndex = len(view.Resources)
			resourceIndexes[resourceName] = resourceIndex
			resourceMethods[resourceName] = make(map[string]string)
			view.Resources = append(view.Resources, resourceView{Name: resourceName, TypeName: e.names.Claim(resourceName+"Resource", "Resource")})
		}
		methodName := modelgen.PublicName(operation.Name)
		if prior, exists := resourceMethods[resourceName][methodName]; exists {
			return nil, fmt.Errorf("sdk/golang: operations %q and %q both map to %s.%s", prior, operation.ID, resourceName, methodName)
		}
		resourceMethods[resourceName][methodName] = operation.ID
		operationView, err := e.prepareOperation(operation, methodName)
		if err != nil {
			return nil, err
		}
		operationView.ResourceName = resourceName
		if operationView.HasQueryParams {
			view.HasQueryParams = true
		}
		view.Resources[resourceIndex].Operations = append(view.Resources[resourceIndex].Operations, operationView)
	}
	operationViews := make(map[string]*operationView, len(e.contract.Operations))
	for resourceIndex := range view.Resources {
		for operationIndex := range view.Resources[resourceIndex].Operations {
			operation := &view.Resources[resourceIndex].Operations[operationIndex]
			operationViews[operation.ID] = operation
		}
	}
	for _, workflow := range e.workflows {
		prepared, err := e.prepareWorkflow(workflow, operationViews)
		if err != nil {
			return nil, err
		}
		view.Workflows = append(view.Workflows, prepared)
	}
	if e.collectErr != nil {
		return nil, e.collectErr
	}
	return view, nil
}

func (e *emitter) prepareOperation(operation *sdk.Operation, methodName string) (operationView, error) {
	view := operationView{
		ID: operation.ID, MethodName: methodName,
		ParamsType: e.names.Claim(modelgen.PublicName(operation.ID)+"Params", "Params"),
		Summary:    operation.Summary,
		HTTPMethod: operation.Method, Path: operation.Path,
	}
	fieldNames := make(map[string]string)
	for _, parameter := range operation.Parameters {
		if parameter == nil {
			continue
		}
		if parameter.Schema == nil {
			return view, fmt.Errorf("sdk/golang: operation %q parameter %q has no schema", operation.ID, parameter.Name)
		}
		if !parameterSchemaSupported(parameter.Schema) {
			return view, fmt.Errorf("sdk/golang: operation %q parameter %q uses unsupported object or tuple serialization", operation.ID, parameter.Name)
		}
		if parameter.In == "cookie" {
			return view, fmt.Errorf("sdk/golang: operation %q parameter %q uses unsupported cookie serialization", operation.ID, parameter.Name)
		}
		if parameter.In != "path" && parameter.In != "query" && parameter.In != "header" {
			return view, fmt.Errorf("sdk/golang: operation %q parameter %q has unsupported location %q", operation.ID, parameter.Name, parameter.In)
		}
		if parameter.AllowReserved {
			return view, fmt.Errorf("sdk/golang: operation %q parameter %q uses unsupported allowReserved serialization", operation.ID, parameter.Name)
		}
		if !supportedStyle(parameter.In, parameter.Style) {
			return view, fmt.Errorf("sdk/golang: operation %q parameter %q uses unsupported %s style %q", operation.ID, parameter.Name, parameter.In, parameter.Style)
		}
		typeName, err := e.schemaType(parameter.Schema, operation.ID+modelgen.PublicName(parameter.Name)+"Parameter")
		if err != nil {
			return view, fmt.Errorf("sdk/golang: operation %q parameter %q: %w", operation.ID, parameter.Name, err)
		}
		if !parameter.Required {
			typeName = "*" + typeName
		}
		fieldName := modelgen.PublicName(parameter.Name)
		if prior, ok := fieldNames[fieldName]; ok {
			return view, fmt.Errorf("sdk/golang: operation %q parameters %q and %q have the same Go field name %q", operation.ID, prior, parameter.Name, fieldName)
		}
		fieldNames[fieldName] = parameter.Name
		encoder, array, err := e.parameterEncoder(parameter.Schema)
		if err != nil {
			return view, fmt.Errorf("sdk/golang: operation %q parameter %q: %w", operation.ID, parameter.Name, err)
		}
		view.Parameters = append(view.Parameters, parameterView{
			Name: parameter.Name, FieldName: fieldName, Type: typeName, In: parameter.In,
			Required: parameter.Required, Explode: parameter.Explode, Description: parameter.Description,
			Encoder: encoder, Array: array,
		})
		if parameter.In == "query" {
			view.HasQueryParams = true
		}
	}
	if operation.RequestBody != nil {
		if prior, exists := fieldNames["Body"]; exists {
			return view, fmt.Errorf("sdk/golang: operation %q parameter %q collides with request body field Body", operation.ID, prior)
		}
		mediaType, schema, err := jsonMedia(operation.RequestBody.Content)
		if err != nil {
			return view, fmt.Errorf("sdk/golang: operation %q request body: %w", operation.ID, err)
		}
		if schema == nil {
			return view, fmt.Errorf("sdk/golang: operation %q request body has no schema", operation.ID)
		}
		typeName, err := e.schemaType(schema, operation.ID+"Request")
		if err != nil {
			return view, fmt.Errorf("sdk/golang: operation %q request body: %w", operation.ID, err)
		}
		fieldType := typeName
		if !operation.RequestBody.Required {
			fieldType = "*" + fieldType
		}
		view.Body = &bodyView{Type: typeName, FieldType: fieldType, Required: operation.RequestBody.Required, ContentType: mediaType, Description: operation.RequestBody.Description}
	}
	view.HasParameters = len(view.Parameters) > 0 || view.Body != nil
	responseType, successes, errorResponses, err := e.prepareResponses(operation)
	if err != nil {
		return view, err
	}
	view.ResponseType = responseType
	view.SuccessStatuses = successes
	view.SuccessCondition, err = statusCondition(successes)
	if err != nil {
		return view, fmt.Errorf("sdk/golang: operation %q success responses: %w", operation.ID, err)
	}
	for index := range errorResponses {
		errorResponses[index].Condition, err = statusCondition([]string{errorResponses[index].Status})
		if err != nil {
			return view, fmt.Errorf("sdk/golang: operation %q error response %s: %w", operation.ID, errorResponses[index].Status, err)
		}
	}
	view.ErrorResponses = errorResponses
	for _, alternative := range operation.Security {
		if alternative == nil {
			view.Security = append(view.Security, nil)
			continue
		}
		prepared := make([]securityRequirementView, 0, len(alternative.Schemes))
		for _, scheme := range alternative.Schemes {
			if scheme != nil {
				prepared = append(prepared, securityRequirementView{Name: scheme.Name, Scopes: append([]string(nil), scheme.Scopes...)})
			}
		}
		view.Security = append(view.Security, prepared)
	}
	return view, nil
}

func (e *emitter) prepareWorkflow(workflow *sdk.Workflow, operations map[string]*operationView) (workflowView, error) {
	if workflow == nil || workflow.Operation == nil {
		return workflowView{}, errors.New("sdk/golang: workflow and operation are required")
	}
	operation, ok := operations[workflow.Operation.ID]
	if !ok {
		return workflowView{}, fmt.Errorf("sdk/golang: workflow %q operation %q is not generated", workflow.ID, workflow.Operation.ID)
	}
	view := workflowView{
		ID: workflow.ID, MethodName: e.names.Claim(modelgen.PublicName(workflow.ID), "Workflow"),
		InputType:    e.names.Claim(modelgen.PublicName(workflow.ID)+"Input", "Input"),
		Summary:      workflow.Summary,
		ResourceName: operation.ResourceName, OperationMethod: operation.MethodName,
		OperationParams: operation.ParamsType, HasParameters: operation.HasParameters,
		ResponseType: operation.ResponseType,
	}
	inputSchema, inputShape, err := workflowInputSchema(workflow.Inputs)
	if err != nil {
		return workflowView{}, fmt.Errorf("sdk/golang: workflow %q inputs: %w", workflow.ID, err)
	}
	inputKey := "__workflow_" + view.InputType
	e.schemas.Set(inputKey, inputSchema)
	e.typeNames[inputKey] = view.InputType
	e.collectSchema(inputSchema)
	view.SuccessCondition, err = statusCondition(workflow.SuccessStatuses)
	if err != nil {
		return workflowView{}, fmt.Errorf("sdk/golang: workflow %q success responses: %w", workflow.ID, err)
	}
	operationSuccess := make(map[string]struct{}, len(operation.SuccessStatuses))
	for _, status := range operation.SuccessStatuses {
		operationSuccess[status] = struct{}{}
	}
	for _, status := range workflow.SuccessStatuses {
		if _, ok := operationSuccess[status]; !ok {
			return workflowView{}, fmt.Errorf("sdk/golang: workflow %q accepts status %s but operation %q does not", workflow.ID, status, workflow.Operation.ID)
		}
	}
	declaredInputs := make(map[string]struct{}, inputShape.Properties.Len())
	for name := range inputShape.Properties.FromOldest() {
		declaredInputs[name] = struct{}{}
	}
	operationParameters := make(map[string]parameterView, len(operation.Parameters))
	for _, parameter := range operation.Parameters {
		operationParameters[parameter.In+"\x00"+parameter.Name] = parameter
	}
	for _, binding := range workflow.ParameterBindings {
		if binding == nil {
			continue
		}
		parameter, exists := operationParameters[binding.In+"\x00"+binding.Name]
		if !exists {
			return workflowView{}, fmt.Errorf("sdk/golang: workflow %q parameter %s %q is not generated", workflow.ID, binding.In, binding.Name)
		}
		if _, exists := declaredInputs[binding.Input]; !exists {
			return workflowView{}, fmt.Errorf("sdk/golang: workflow %q parameter input %q is not declared by workflow inputs", workflow.ID, binding.Input)
		}
		view.ParameterFields = append(view.ParameterFields, workflowAssignmentView{
			Target: parameter.FieldName, Input: binding.Input, ExpectedType: parameter.Type,
		})
	}
	if len(workflow.PayloadBindings) > 0 {
		if operation.Body == nil || !operation.Body.Required {
			return workflowView{}, fmt.Errorf("sdk/golang: workflow %q requires a declared required JSON request body", workflow.ID)
		}
		view.BodyType = operation.Body.Type
		for _, binding := range workflow.PayloadBindings {
			if binding == nil {
				continue
			}
			if strings.Contains(binding.Input, ".") {
				return workflowView{}, fmt.Errorf("sdk/golang: workflow %q payload input %q is nested; only direct workflow inputs are supported", workflow.ID, binding.Input)
			}
			if _, exists := declaredInputs[binding.Input]; !exists {
				return workflowView{}, fmt.Errorf("sdk/golang: workflow %q payload input %q is not declared by workflow inputs", workflow.ID, binding.Input)
			}
			view.BodyFields = append(view.BodyFields, workflowAssignmentView{
				Input: binding.Input, ExpectedModel: operation.Body.Type, ExpectedField: binding.Property,
			})
		}
	}
	return view, nil
}

func workflowInputSchema(node *yaml.Node) (*highbase.SchemaProxy, *highbase.Schema, error) {
	if node == nil {
		return nil, nil, errors.New("an object JSON Schema is required")
	}
	wrapper := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Value: "schema"}, node,
	}}
	schemaDocument := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{node}}
	lowSchema, err := lowbase.ExtractSchema(context.Background(), wrapper, index.NewSpecIndex(schemaDocument))
	if err != nil {
		return nil, nil, fmt.Errorf("parse JSON Schema: %w", err)
	}
	proxy := highbase.NewSchemaProxy(lowSchema)
	schema, err := proxy.BuildSchema()
	if err != nil {
		return nil, nil, fmt.Errorf("build JSON Schema: %w", err)
	}
	if schema == nil || len(schema.Type) != 1 || schema.Type[0] != "object" || schema.Properties == nil {
		return nil, nil, errors.New("inputs must be an object JSON Schema with declared properties")
	}
	return proxy, schema, nil
}

func resolveWorkflowFieldTypes(types []*modelgen.GeneratedType, workflows []workflowView) error {
	if len(workflows) == 0 {
		return nil
	}
	typesByName := make(map[string]*modelgen.GeneratedType, len(types))
	for _, generatedType := range types {
		if generatedType != nil {
			typesByName[generatedType.Name] = generatedType
		}
	}
	for workflowIndex := range workflows {
		workflow := &workflows[workflowIndex]
		inputType := typesByName[workflow.InputType]
		if inputType == nil || inputType.Kind != modelgen.KindObject {
			return fmt.Errorf("sdk/golang: workflow %q inputs %q are not a struct model", workflow.ID, workflow.InputType)
		}
		for assignmentIndex := range workflow.ParameterFields {
			assignment := &workflow.ParameterFields[assignmentIndex]
			inputField, ok := generatedField(typesByName, inputType, assignment.Input, make(map[string]struct{}))
			if !ok {
				return fmt.Errorf("sdk/golang: workflow %q input %q is not present on %s", workflow.ID, assignment.Input, workflow.InputType)
			}
			if inputField.Type != assignment.ExpectedType {
				return fmt.Errorf("sdk/golang: workflow %q input %q has type %s, operation parameter requires %s", workflow.ID, assignment.Input, inputField.Type, assignment.ExpectedType)
			}
			assignment.Source = inputField.Name
		}
		for assignmentIndex := range workflow.BodyFields {
			assignment := &workflow.BodyFields[assignmentIndex]
			inputField, ok := generatedField(typesByName, inputType, assignment.Input, make(map[string]struct{}))
			if !ok {
				return fmt.Errorf("sdk/golang: workflow %q input %q is not present on %s", workflow.ID, assignment.Input, workflow.InputType)
			}
			bodyType := typesByName[assignment.ExpectedModel]
			bodyField, ok := generatedField(typesByName, bodyType, assignment.ExpectedField, make(map[string]struct{}))
			if !ok {
				return fmt.Errorf("sdk/golang: workflow %q payload property %q is not present on %s", workflow.ID, assignment.ExpectedField, assignment.ExpectedModel)
			}
			if inputField.Type != bodyField.Type {
				return fmt.Errorf("sdk/golang: workflow %q input %q has type %s, payload property %q requires %s", workflow.ID, assignment.Input, inputField.Type, assignment.ExpectedField, bodyField.Type)
			}
			assignment.Source = inputField.Name
			assignment.Target = bodyField.Name
		}
	}
	return nil
}

func generatedField(types map[string]*modelgen.GeneratedType, generatedType *modelgen.GeneratedType, source string, visited map[string]struct{}) (modelgen.GeneratedField, bool) {
	if generatedType == nil {
		return modelgen.GeneratedField{}, false
	}
	if _, seen := visited[generatedType.Name]; seen {
		return modelgen.GeneratedField{}, false
	}
	visited[generatedType.Name] = struct{}{}
	for _, field := range generatedType.Fields {
		if field.Source == source {
			return field, true
		}
	}
	for _, embedded := range generatedType.Embedded {
		if field, ok := generatedField(types, types[embedded], source, visited); ok {
			return field, true
		}
	}
	return modelgen.GeneratedField{}, false
}

func (e *emitter) prepareResponses(operation *sdk.Operation) (string, []string, []errorResponseView, error) {
	responseType := "struct{}"
	var selectedType string
	var successes []string
	var errorResponses []errorResponseView
	for _, response := range operation.Responses {
		if response == nil {
			continue
		}
		status := normalizeStatus(response.Status)
		isSuccess := statusIsSuccess(status)
		_, schema, err := jsonMedia(response.Content)
		if err != nil {
			return "", nil, nil, fmt.Errorf("sdk/golang: operation %q response %s: %w", operation.ID, status, err)
		}
		typeName := ""
		if schema != nil {
			typeName, err = e.schemaType(schema, operation.ID+statusName(status)+"Response")
			if err != nil {
				return "", nil, nil, fmt.Errorf("sdk/golang: operation %q response %s: %w", operation.ID, status, err)
			}
		}
		if isSuccess {
			successes = append(successes, status)
			if typeName != "" {
				if selectedType != "" && selectedType != typeName {
					return "", nil, nil, fmt.Errorf("sdk/golang: operation %q has incompatible success response types %s and %s", operation.ID, selectedType, typeName)
				}
				selectedType = typeName
			}
			continue
		}
		errorResponses = append(errorResponses, errorResponseView{Status: status, Type: typeName})
	}
	if len(successes) == 0 {
		return "", nil, nil, fmt.Errorf("sdk/golang: operation %q has no declared 2xx response", operation.ID)
	}
	if selectedType != "" {
		responseType = selectedType
	}
	return responseType, successes, errorResponses, nil
}

func (e *emitter) schemaType(proxy *highbase.SchemaProxy, hint string) (string, error) {
	if proxy == nil {
		return "", errors.New("schema is required")
	}
	if proxy.IsReference() {
		ref := proxy.GetReference()
		if name, component := componentSchemaRefName(ref); component {
			if err := e.ensureComponent(name); err != nil {
				return "", err
			}
			return e.typeNames[name], nil
		}
		if strings.HasPrefix(ref, "#/") {
			return "", fmt.Errorf("schema reference %q is not a components/schemas reference", ref)
		}
		return "", fmt.Errorf("external schema reference %q is not supported by SDK generation", ref)
	}
	schema := proxy.Schema()
	if schema == nil {
		return "", errors.New("schema could not be resolved")
	}
	types := make([]string, 0, len(schema.Type))
	for _, schemaType := range schema.Type {
		if schemaType != "null" {
			types = append(types, schemaType)
		}
	}
	if len(types) == 1 {
		switch types[0] {
		case "string":
			if mapped := e.models.ScalarType(schema.Format, "string"); mapped != "string" {
				return e.addInlineSchema(hint, proxy), nil
			}
			return "string", nil
		case "integer":
			if schema.Format == "int32" {
				return "int32", nil
			}
			if schema.Format == "int64" {
				return "int64", nil
			}
			return "int", nil
		case "number":
			if schema.Format == "float" {
				return "float32", nil
			}
			return "float64", nil
		case "boolean":
			return "bool", nil
		case "array":
			if schema.Items == nil || !schema.Items.IsA() {
				return "[]any", nil
			}
			itemType, err := e.schemaType(schema.Items.A, hint+"Item")
			if err != nil {
				return "", err
			}
			return "[]" + itemType, nil
		case "object":
			return e.addInlineSchema(hint, proxy), nil
		}
	}
	if schema.Properties != nil || len(schema.AllOf) > 0 || len(schema.OneOf) > 0 || len(schema.AnyOf) > 0 {
		name := e.addInlineSchema(hint, proxy)
		if len(schema.OneOf) > 0 || len(schema.AnyOf) > 0 {
			name += "Union"
		}
		return name, nil
	}
	return "any", nil
}

func (e *emitter) addInlineSchema(hint string, proxy *highbase.SchemaProxy) string {
	key := hint
	for suffix := 2; ; suffix++ {
		_, generated := e.schemas.Get(key)
		_, component := e.reachableComponents[key]
		if !generated && !component {
			break
		}
		key = fmt.Sprintf("%s%d", hint, suffix)
	}
	e.schemas.Set(key, proxy)
	typeName := e.names.Claim(modelgen.PublicName(key), "Model")
	e.typeNames[key] = typeName
	e.collectSchema(proxy)
	return typeName
}

func (e *emitter) ensureComponent(name string) error {
	if _, collected := e.collectedComponents[name]; collected {
		return nil
	}
	proxy, ok := e.componentSchemas.Get(name)
	if !ok || proxy == nil {
		return fmt.Errorf("schema reference %q is not defined in components", name)
	}
	e.collectedComponents[name] = struct{}{}
	if e.typeNames[name] == "" {
		e.typeNames[name] = e.names.Claim(modelgen.PublicName(name), "Model")
	}
	e.schemas.Set(name, proxy)
	e.collectSchema(proxy)
	return nil
}

func (e *emitter) collectSchema(proxy *highbase.SchemaProxy) {
	if proxy == nil {
		return
	}
	if _, visited := e.visitedSchemas[proxy]; visited {
		return
	}
	e.visitedSchemas[proxy] = struct{}{}
	if proxy.IsReference() {
		ref := proxy.GetReference()
		if name, component := componentSchemaRefName(ref); component {
			if err := e.ensureComponent(name); err != nil && e.collectErr == nil {
				e.collectErr = fmt.Errorf("sdk/golang: %w", err)
			}
		} else if e.collectErr == nil {
			if strings.HasPrefix(ref, "#/") {
				e.collectErr = fmt.Errorf("sdk/golang: schema reference %q is not a components/schemas reference", ref)
			} else {
				e.collectErr = fmt.Errorf("sdk/golang: external schema reference %q is not supported by SDK generation", ref)
			}
		}
		return
	}
	schema := proxy.Schema()
	if schema == nil {
		return
	}
	visit := e.collectSchema
	for _, child := range schema.AllOf {
		visit(child)
	}
	for _, child := range schema.OneOf {
		visit(child)
	}
	for _, child := range schema.AnyOf {
		visit(child)
	}
	if schema.Items != nil && schema.Items.IsA() {
		visit(schema.Items.A)
	}
	if schema.AdditionalProperties != nil && schema.AdditionalProperties.IsA() {
		visit(schema.AdditionalProperties.A)
	}
	for _, children := range []*orderedmap.Map[string, *highbase.SchemaProxy]{schema.Properties, schema.PatternProperties} {
		if children == nil {
			continue
		}
		for _, child := range children.FromOldest() {
			visit(child)
		}
	}
}

func jsonMedia(content *orderedmap.Map[string, *highv3.MediaType]) (string, *highbase.SchemaProxy, error) {
	if content == nil || content.Len() == 0 {
		return "", nil, nil
	}
	for mediaType, value := range content.FromOldest() {
		if mediaType == "application/json" || strings.HasSuffix(mediaType, "+json") {
			if value == nil {
				return "", nil, fmt.Errorf("media type %q has no definition", mediaType)
			}
			return mediaType, value.Schema, nil
		}
	}
	keys := make([]string, 0, content.Len())
	for mediaType := range content.FromOldest() {
		keys = append(keys, mediaType)
	}
	sort.Strings(keys)
	return "", nil, fmt.Errorf("unsupported media types %s", strings.Join(keys, ", "))
}

func supportedStyle(location, style string) bool {
	switch location {
	case "path", "header":
		return style == "simple"
	case "query":
		return style == "form"
	default:
		return false
	}
}

func parameterSchemaSupported(proxy *highbase.SchemaProxy) bool {
	if proxy == nil {
		return false
	}
	schema := proxy.Schema()
	if schema == nil {
		return false
	}
	var nonNullType string
	for _, schemaType := range schema.Type {
		switch schemaType {
		case "null":
			continue
		case "string", "integer", "number", "boolean", "array":
			if nonNullType != "" {
				return false
			}
			nonNullType = schemaType
		default:
			return false
		}
	}
	if nonNullType == "array" {
		return schema.Items != nil && schema.Items.IsA() && parameterSchemaSupported(schema.Items.A)
	}
	return nonNullType != "" || len(schema.Enum) > 0
}

func (e *emitter) parameterEncoder(proxy *highbase.SchemaProxy) (string, bool, error) {
	if proxy == nil {
		return "", false, errors.New("schema is required")
	}
	schema := proxy.Schema()
	if schema == nil {
		return "", false, errors.New("schema could not be resolved")
	}
	var nonNullType string
	for _, schemaType := range schema.Type {
		if schemaType != "null" {
			nonNullType = schemaType
			break
		}
	}
	switch nonNullType {
	case "string":
		if mapped := e.models.ScalarType(schema.Format, "string"); mapped != "string" {
			return "", false, fmt.Errorf("custom scalar format %q maps to %s and is not supported for parameter serialization", schema.Format, mapped)
		}
		return "encodeString", false, nil
	case "integer":
		return "encodeInteger", false, nil
	case "number":
		if schema.Format == "float" {
			return "encodeFloat32", false, nil
		}
		return "encodeFloat64", false, nil
	case "boolean":
		return "encodeBoolean", false, nil
	case "array":
		if schema.Items == nil || !schema.Items.IsA() {
			return "", false, errors.New("array parameter item schema is required")
		}
		encoder, _, err := e.parameterEncoder(schema.Items.A)
		return encoder, true, err
	default:
		if len(schema.Enum) > 0 {
			return "encodeAny", false, nil
		}
		return "", false, fmt.Errorf("unsupported parameter type %q", nonNullType)
	}
}

func statusCondition(statuses []string) (string, error) {
	conditions := make([]string, 0, len(statuses))
	for _, status := range statuses {
		canonical := strings.ToUpper(status)
		switch {
		case canonical == "DEFAULT":
			conditions = append(conditions, "true")
		case len(canonical) == 3 && canonical[1:] == "XX" && canonical[0] >= '1' && canonical[0] <= '5':
			conditions = append(conditions, fmt.Sprintf("response.StatusCode/100 == %c", canonical[0]))
		case len(canonical) == 3:
			code, err := strconv.Atoi(canonical)
			if err != nil || code < 100 || code > 599 {
				return "", fmt.Errorf("unsupported response status %q", status)
			}
			conditions = append(conditions, fmt.Sprintf("response.StatusCode == %d", code))
		default:
			return "", fmt.Errorf("unsupported response status %q", status)
		}
	}
	if len(conditions) == 0 {
		return "false", nil
	}
	return strings.Join(conditions, " || "), nil
}

func statusIsSuccess(status string) bool {
	return len(status) == 3 && status[0] == '2'
}

func normalizeStatus(status string) string {
	if len(status) == 3 && strings.EqualFold(status[1:], "xx") {
		return strings.ToUpper(status)
	}
	return status
}

func statusName(status string) string {
	if status == "default" {
		return "Default"
	}
	return "Status" + strings.ToUpper(status)
}

func commentLine(value string) string {
	value = strings.TrimSpace(strings.Split(value, "\n")[0])
	if value == "" {
		return ""
	}
	return strings.ReplaceAll(value, "*/", "* /")
}
