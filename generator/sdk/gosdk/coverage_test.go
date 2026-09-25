// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package gosdk

import (
	"errors"
	"strings"
	"testing"
	"text/template"

	"github.com/pb33f/go-yaml"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	highv3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	modelgen "github.com/pb33f/libopenapi/generator/golang"
	"github.com/pb33f/libopenapi/generator/sdk"
	"github.com/pb33f/libopenapi/orderedmap"
)

func TestGenerateContractRejectsInvalidTopLevelInputs(t *testing.T) {
	tests := []struct {
		name     string
		contract *sdk.Contract
		options  Options
		want     string
	}{
		{name: "nil contract", want: "contract is required"},
		{name: "no operations", contract: &sdk.Contract{}, want: "no selected operations"},
		{name: "invalid package", contract: contractWith(operationWithSuccess("get")), options: Options{PackageName: "not-valid"}, want: "invalid package name"},
		{name: "keyword package", contract: contractWith(operationWithSuccess("get")), options: Options{PackageName: "type"}, want: "invalid package name"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := GenerateContract(test.contract, test.options)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error containing %q, got %v", test.want, err)
			}
		})
	}
}

func TestGenerateRejectsPrebuiltWorkflows(t *testing.T) {
	_, err := Generate(nil, Options{Workflows: []*sdk.Workflow{{ID: "run"}}})
	if err == nil || !strings.Contains(err.Error(), "GenerateContract") {
		t.Fatalf("expected workflow API guidance, got %v", err)
	}
}

func TestGenerateContractRejectsCustomMappedParameterScalar(t *testing.T) {
	operation := operationWithSuccess("search")
	operation.Parameters = []*sdk.Parameter{{
		Name: "createdAfter", In: "query", Style: "form",
		Schema: highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}, Format: "date-time"}),
	}}
	_, err := GenerateContract(contractWith(operation), Options{
		Models: []modelgen.Option{modelgen.WithFormatMapping("date-time", "time.Time", "time")},
	})
	if err == nil || !strings.Contains(err.Error(), "not supported for parameter serialization") {
		t.Fatalf("expected custom scalar parameter rejection, got %v", err)
	}
}

func TestUnreachableComponentDoesNotRenameInlineSchema(t *testing.T) {
	components := orderedmap.New[string, *highbase.SchemaProxy]()
	components.Set("CreateRequest", schema("string", ""))
	operation := operationWithSuccess("create")
	operation.RequestBody = jsonBody(highbase.CreateSchemaProxy(&highbase.Schema{
		Type: []string{"object"}, Properties: schemaMap("value", schema("string", "")),
	}))
	contract := contractWith(operation)
	contract.Schemas = components
	result, err := GenerateContract(contract, Options{})
	if err != nil {
		t.Fatal(err)
	}
	models := generatedFile(t, result.Files, "models.gen.go")
	if !strings.Contains(models, "type CreateRequest struct") || strings.Contains(models, "type CreateRequestModel struct") {
		t.Fatalf("unreachable component changed inline schema naming:\n%s", models)
	}
}

func TestRenderReportsTemplateFailures(t *testing.T) {
	tests := []struct {
		name   string
		source string
		value  any
		want   string
	}{
		{name: "execute", source: "package p\nvar _ = {{.Missing}}", value: struct{}{}, want: "execute broken template"},
		{name: "format", source: "package", want: "format broken:"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tmpl := template.Must(template.New("broken").Parse(test.source))
			_, err := render(tmpl, test.value)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error containing %q, got %v", test.want, err)
			}
		})
	}
}

func TestPrepareViewRejectsContractCollisionsAndSecurityErrors(t *testing.T) {
	base := operationWithSuccess("first")
	base.Resource, base.Name = "Things", "Get"
	duplicate := operationWithSuccess("second")
	duplicate.Resource, duplicate.Name = "Things", "Get"

	tests := []struct {
		name     string
		contract *sdk.Contract
		want     string
	}{
		{
			name: "undefined security scheme",
			contract: contractWith(&sdk.Operation{
				ID: "secure", Method: "GET", Path: "/secure", Resource: "Secure", Name: "Get",
				Responses: successResponses(), Security: []*sdk.SecurityRequirement{{Schemes: []*sdk.RequiredSecurityScheme{{Name: "missing"}}}},
			}),
			want: "undefined security scheme",
		},
		{name: "method collision", contract: contractWith(base, duplicate), want: "both map to Things.Get"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			emitter := newTestEmitter(test.contract, "client", nil)
			_, err := emitter.prepareView()
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error containing %q, got %v", test.want, err)
			}
		})
	}
}

func TestGenerateRejectsNilDocument(t *testing.T) {
	if _, err := Generate(nil, Options{}); err == nil {
		t.Fatal("expected nil document preparation error")
	}
}

func TestPrepareViewSkipsNilEntriesAndSurfacesCollectionErrors(t *testing.T) {
	operation := operationWithSuccess("get")
	operation.Security = []*sdk.SecurityRequirement{nil, {Schemes: []*sdk.RequiredSecurityScheme{nil}}}
	contract := contractWith(nil, operation)
	contract.SecuritySchemes = []*sdk.SecurityScheme{nil, {Name: "unused", Type: "apiKey"}}
	emitter := newTestEmitter(contract, "client", nil)
	emitter.collectErr = errors.New("collection failed")
	_, err := emitter.prepareView()
	if err == nil || err.Error() != "collection failed" {
		t.Fatalf("expected collection failure, got %v", err)
	}
}

func TestPrepareViewReportsWorkflowCompositionErrors(t *testing.T) {
	emitter := newTestEmitter(contractWith(operationWithSuccess("get")), "client", []*sdk.Workflow{{ID: "broken"}})
	_, err := emitter.prepareView()
	if err == nil || !strings.Contains(err.Error(), "workflow and operation are required") {
		t.Fatalf("expected workflow composition error, got %v", err)
	}
}

func TestPrepareOperationRejectsUnsupportedParametersAndBodies(t *testing.T) {
	stringSchema := schema("string", "")
	objectSchema := schema("object", "")
	mediaWithoutSchema := orderedmap.New[string, *highv3.MediaType]()
	mediaWithoutSchema.Set("application/json", &highv3.MediaType{})
	unsupportedMedia := orderedmap.New[string, *highv3.MediaType]()
	unsupportedMedia.Set("text/plain", &highv3.MediaType{Schema: stringSchema})

	tests := []struct {
		name      string
		parameter *sdk.Parameter
		body      *sdk.RequestBody
		want      string
	}{
		{name: "missing parameter schema", parameter: &sdk.Parameter{Name: "id", In: "query", Style: "form"}, want: "has no schema"},
		{name: "object parameter", parameter: &sdk.Parameter{Name: "filter", In: "query", Style: "form", Schema: objectSchema}, want: "object or tuple"},
		{name: "cookie", parameter: &sdk.Parameter{Name: "session", In: "cookie", Style: "form", Schema: stringSchema}, want: "cookie serialization"},
		{name: "unknown location", parameter: &sdk.Parameter{Name: "id", In: "matrix", Style: "simple", Schema: stringSchema}, want: "unsupported location"},
		{name: "allow reserved", parameter: &sdk.Parameter{Name: "q", In: "query", Style: "form", AllowReserved: true, Schema: stringSchema}, want: "allowReserved"},
		{name: "unsupported style", parameter: &sdk.Parameter{Name: "q", In: "query", Style: "spaceDelimited", Schema: stringSchema}, want: "unsupported query style"},
		{name: "body field collision", parameter: &sdk.Parameter{Name: "body", In: "query", Style: "form", Schema: stringSchema}, body: jsonBody(stringSchema), want: "collides with request body"},
		{name: "unsupported body media", body: &sdk.RequestBody{Content: unsupportedMedia}, want: "unsupported media types"},
		{name: "body media without schema", body: &sdk.RequestBody{Content: mediaWithoutSchema}, want: "request body has no schema"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			operation := operationWithSuccess("probe")
			if test.parameter != nil {
				operation.Parameters = []*sdk.Parameter{test.parameter}
			}
			operation.RequestBody = test.body
			emitter := newTestEmitter(contractWith(operation), "client", nil)
			_, err := emitter.prepareOperation(operation, "Probe")
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error containing %q, got %v", test.want, err)
			}
		})
	}
}

func TestPrepareOperationCoversOptionalAndSchemaFailureBranches(t *testing.T) {
	operation := operationWithSuccess("probe")
	operation.Parameters = []*sdk.Parameter{
		nil,
		{Name: "user-id", In: "header", Style: "simple", Schema: schema("string", "")},
	}
	operation.RequestBody = &sdk.RequestBody{Required: false, Content: jsonContent(schema("string", ""))}
	emitter := newTestEmitter(contractWith(operation), "client", nil)
	view, err := emitter.prepareOperation(operation, "Probe")
	if err != nil || view.Parameters[0].Type != "*string" || view.Body.FieldType != "*string" {
		t.Fatalf("unexpected optional types: %#v, %v", view, err)
	}

	duplicate := operationWithSuccess("duplicate")
	duplicate.Parameters = []*sdk.Parameter{
		{Name: "user-id", In: "query", Style: "form", Schema: schema("string", "")},
		{Name: "user_id", In: "query", Style: "form", Schema: schema("string", "")},
	}
	if _, err := newTestEmitter(contractWith(duplicate), "client", nil).prepareOperation(duplicate, "Duplicate"); err == nil || !strings.Contains(err.Error(), "same Go field name") {
		t.Fatalf("expected field collision, got %v", err)
	}

	for _, test := range []struct {
		name string
		set  func(*sdk.Operation)
		want string
	}{
		{name: "body", set: func(operation *sdk.Operation) {
			operation.RequestBody = jsonBody(&highbase.SchemaProxy{})
		}, want: "schema could not be resolved"},
		{name: "response", set: func(operation *sdk.Operation) {
			operation.Responses = []*sdk.Response{{Status: "200", Content: jsonContent(&highbase.SchemaProxy{})}}
		}, want: "schema could not be resolved"},
		{name: "parameter reference", set: func(operation *sdk.Operation) {
			operation.Parameters = []*sdk.Parameter{{
				Name: "id", In: "query", Style: "form",
				Schema: highbase.CreateSchemaProxyRefWithSchema("#/components/schemas/%", &highbase.Schema{Type: []string{"string"}}),
			}}
		}, want: "not defined in components"},
	} {
		t.Run(test.name, func(t *testing.T) {
			operation := operationWithSuccess("broken")
			test.set(operation)
			_, err := newTestEmitter(contractWith(operation), "client", nil).prepareOperation(operation, "Broken")
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error containing %q, got %v", test.want, err)
			}
		})
	}
}

func TestPrepareOperationCoversScalarArrayAndInlineTypes(t *testing.T) {
	types := []struct {
		name   string
		schema *highbase.SchemaProxy
		want   string
	}{
		{name: "string", schema: schema("string", ""), want: "string"},
		{name: "int32", schema: schema("integer", "int32"), want: "int32"},
		{name: "int64", schema: schema("integer", "int64"), want: "int64"},
		{name: "integer", schema: schema("integer", ""), want: "int"},
		{name: "float", schema: schema("number", "float"), want: "float32"},
		{name: "number", schema: schema("number", ""), want: "float64"},
		{name: "boolean", schema: schema("boolean", ""), want: "bool"},
		{name: "untyped array", schema: schema("array", ""), want: "[]any"},
		{name: "typed array", schema: arraySchema(schema("string", "")), want: "[]string"},
		{name: "unknown", schema: highbase.CreateSchemaProxy(&highbase.Schema{}), want: "any"},
	}
	for _, test := range types {
		t.Run(test.name, func(t *testing.T) {
			emitter := newTestEmitter(&sdk.Contract{}, "client", nil)
			got, err := emitter.schemaType(test.schema, "Payload")
			if err != nil || got != test.want {
				t.Fatalf("schemaType() = %q, %v; want %q", got, err, test.want)
			}
		})
	}

	properties := orderedmap.New[string, *highbase.SchemaProxy]()
	properties.Set("value", schema("string", ""))
	object := highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"object"}, Properties: properties})
	union := highbase.CreateSchemaProxy(&highbase.Schema{OneOf: []*highbase.SchemaProxy{schema("string", ""), schema("integer", "")}})
	emitter := newTestEmitter(&sdk.Contract{}, "client", nil)
	if got, err := emitter.schemaType(object, "Payload"); err != nil || got != "Payload" {
		t.Fatalf("object type = %q, %v", got, err)
	}
	if got, err := emitter.schemaType(union, "Choice"); err != nil || got != "ChoiceUnion" {
		t.Fatalf("union type = %q, %v", got, err)
	}
}

func TestPrepareResponsesContracts(t *testing.T) {
	emitter := newTestEmitter(&sdk.Contract{}, "client", nil)
	tests := []struct {
		name      string
		responses []*sdk.Response
		want      string
	}{
		{name: "no success", responses: []*sdk.Response{{Status: "400"}}, want: "no declared 2xx"},
		{name: "incompatible success bodies", responses: []*sdk.Response{
			{Status: "200", Content: jsonContent(schema("string", ""))},
			{Status: "201", Content: jsonContent(schema("integer", ""))},
		}, want: "incompatible success response types"},
		{name: "unsupported response media", responses: []*sdk.Response{{Status: "200", Content: media("text/plain", schema("string", ""))}}, want: "unsupported media types"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			operation := &sdk.Operation{ID: "read", Responses: test.responses}
			_, _, _, _, err := emitter.prepareResponses(operation)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error containing %q, got %v", test.want, err)
			}
		})
	}

	responseType, successes, decodedSuccesses, failures, err := emitter.prepareResponses(&sdk.Operation{ID: "read", Responses: []*sdk.Response{
		nil,
		{Status: "200", Content: jsonContent(schema("string", ""))},
		{Status: "default", Content: jsonContent(schema("object", ""))},
	}})
	if err != nil || responseType != "string" || len(successes) != 1 || len(decodedSuccesses) != 1 || decodedSuccesses[0] != "200" || len(failures) != 1 || failures[0].Status != "default" {
		t.Fatalf("unexpected prepared responses: %q %#v %#v %#v %v", responseType, successes, decodedSuccesses, failures, err)
	}
	_, successes, _, _, err = emitter.prepareResponses(&sdk.Operation{ID: "wildcard", Responses: []*sdk.Response{{Status: "2xx"}}})
	if err != nil || len(successes) != 1 || successes[0] != "2XX" {
		t.Fatalf("lowercase wildcard was not normalized: %#v, %v", successes, err)
	}
}

func TestDecodeStatusConditionHonorsExactBodylessOverride(t *testing.T) {
	tests := []struct {
		name      string
		decoded   []string
		successes []string
		want      string
		wantError bool
	}{
		{name: "exact override", decoded: []string{"2XX"}, successes: []string{"2XX", "204"}, want: "(response.StatusCode/100 == 2) && response.StatusCode != 204"},
		{name: "decoded exact", decoded: []string{"200"}, successes: []string{"200", "204"}, want: "response.StatusCode == 200"},
		{name: "nonoverlapping exact", decoded: []string{"2XX"}, successes: []string{"2XX", "304"}, want: "response.StatusCode/100 == 2"},
		{name: "irrelevant malformed status", decoded: []string{"2XX"}, successes: []string{"2XX", "bad"}, want: "response.StatusCode/100 == 2"},
		{name: "invalid decoded status", decoded: []string{"bad"}, wantError: true},
		{name: "invalid overlapping exact", decoded: []string{"2XX"}, successes: []string{"2A4"}, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			condition, err := decodeStatusCondition(test.decoded, test.successes)
			if test.wantError {
				if err == nil {
					t.Fatalf("condition = %q, error = nil", condition)
				}
				return
			}
			if err != nil || condition != test.want {
				t.Fatalf("condition = %q, error = %v, want %q", condition, err, test.want)
			}
		})
	}
}

func TestPrepareWorkflowContracts(t *testing.T) {
	operation := operationView{
		ID: "create", ResourceName: "Things", MethodName: "Create", ParamsType: "CreateParams",
		SuccessStatuses: []string{"201"}, ResponseType: "Thing", HasParameters: true,
		Parameters: []parameterView{{Name: "tenantId", FieldName: "TenantID", Type: "string", In: "path", Required: true}},
		Body:       &bodyView{Type: "CreateThing", Required: true},
	}
	emitter := newTestEmitter(&sdk.Contract{}, "client", nil)
	tests := []struct {
		name       string
		workflow   *sdk.Workflow
		operations map[string]*operationView
		want       string
	}{
		{name: "nil workflow", want: "workflow and operation are required"},
		{name: "nil operation", workflow: &sdk.Workflow{ID: "run"}, want: "workflow and operation are required"},
		{name: "operation absent", workflow: &sdk.Workflow{ID: "run", Operation: &sdk.Operation{ID: "missing"}}, want: "is not generated"},
		{name: "invalid success", workflow: &sdk.Workflow{ID: "run", Operation: &sdk.Operation{ID: "create"}, Inputs: workflowInputs(t, `type: object
properties: {tenantId: {type: string}}`), SuccessStatuses: []string{"202"}}, operations: map[string]*operationView{"create": &operation}, want: "accepts status 202"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := emitter.prepareWorkflow(test.workflow, test.operations)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error containing %q, got %v", test.want, err)
			}
		})
	}

	for _, body := range []*bodyView{nil, {Type: "CreateThing", Required: false}} {
		withoutRequiredBody := operation
		withoutRequiredBody.Body = body
		_, err := emitter.prepareWorkflow(&sdk.Workflow{
			ID: "body", Operation: &sdk.Operation{ID: "create"}, Inputs: workflowInputs(t, `type: object
properties: {tenantId: {type: string}, name: {type: string}}`), SuccessStatuses: []string{"201"},
			PayloadBindings: []*sdk.PayloadBinding{{Property: "name", Input: "name"}},
		}, map[string]*operationView{"create": &withoutRequiredBody})
		if err == nil || !strings.Contains(err.Error(), "required JSON request body") {
			t.Fatalf("expected required-body error, got %v", err)
		}
	}

	view, err := emitter.prepareWorkflow(&sdk.Workflow{
		ID: "createWorkflow", Summary: "Create one", Description: "Creates a thing.",
		Operation: &sdk.Operation{ID: "create"}, Inputs: workflowInputs(t, `type: object
required: [tenantId, thingName]
properties: {tenantId: {type: string}, thingName: {type: string}}`), SuccessStatuses: []string{"201"},
		ParameterBindings: []*sdk.ParameterBinding{{Name: "tenantId", In: "path", Input: "tenantId"}},
		PayloadBindings:   []*sdk.PayloadBinding{{Property: "name", Input: "thingName"}},
	}, map[string]*operationView{"create": &operation})
	if err != nil || view.MethodName == "" || len(view.ParameterFields) != 1 || len(view.BodyFields) != 1 {
		t.Fatalf("unexpected workflow view: %#v, %v", view, err)
	}
	_, err = emitter.prepareWorkflow(&sdk.Workflow{
		ID: "nested", Operation: &sdk.Operation{ID: "create"}, Inputs: workflowInputs(t, `type: object
properties: {tenantId: {type: string}, thing: {type: object}}`), SuccessStatuses: []string{"201"},
		PayloadBindings: []*sdk.PayloadBinding{{Property: "name", Input: "thing.name"}},
	}, map[string]*operationView{"create": &operation})
	if err == nil || !strings.Contains(err.Error(), "nested") {
		t.Fatalf("expected nested input rejection, got %v", err)
	}
}

func TestResolveWorkflowFieldTypeContracts(t *testing.T) {
	if err := resolveWorkflowFieldTypes(nil, nil); err != nil {
		t.Fatalf("empty workflows: %v", err)
	}

	types := []*modelgen.GeneratedType{
		{Name: "RunInput", Kind: modelgen.KindObject, Fields: []modelgen.GeneratedField{{Name: "Value", Source: "value", Type: "string"}}},
		{Name: "Body", Kind: modelgen.KindObject, Fields: []modelgen.GeneratedField{{Name: "Value", Source: "value", Type: "string"}}},
	}
	workflows := []workflowView{{ID: "run", InputType: "RunInput",
		ParameterFields: []workflowAssignmentView{{Target: "Value", Input: "value", ExpectedType: "string"}},
		BodyFields:      []workflowAssignmentView{{Input: "value", ExpectedModel: "Body", ExpectedField: "value"}},
	}}
	err := resolveWorkflowFieldTypes(types, workflows)
	if err != nil || workflows[0].ParameterFields[0].Source != "Value" || workflows[0].BodyFields[0].Target != "Value" {
		t.Fatalf("unexpected resolved workflow assignments: %#v, %v", workflows, err)
	}
}

func TestGenerateContractReportsUnresolvableWorkflowPayloadField(t *testing.T) {
	properties := schemaMap("actual", schema("string", ""))
	bodySchema := highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"object"}, Properties: properties})
	operation := operationWithSuccess("create")
	operation.RequestBody = jsonBody(bodySchema)
	workflow := &sdk.Workflow{
		ID: "run", Operation: operation, Inputs: workflowInputs(t, `type: object
required: [value]
properties: {value: {type: string}}`), SuccessStatuses: []string{"204"},
		PayloadBindings: []*sdk.PayloadBinding{{Property: "missing", Input: "value"}},
	}
	_, err := GenerateContract(contractWith(operation), Options{Workflows: []*sdk.Workflow{workflow}})
	if err == nil || !strings.Contains(err.Error(), "payload property \"missing\" is not present") {
		t.Fatalf("expected workflow payload field error, got %v", err)
	}
}

func TestSchemaReferencesAndTraversal(t *testing.T) {
	components := orderedmap.New[string, *highbase.SchemaProxy]()
	childNames := []string{
		"AllOfChild", "OneOfChild", "AnyOfChild", "PrefixChild", "ContainsChild",
		"IfChild", "ElseChild", "ThenChild", "PropertyNamesChild", "UnevaluatedItemsChild",
		"NotChild", "ContentChild", "ItemsChild", "AdditionalPropertiesChild",
		"UnevaluatedPropertiesChild", "PropertyChild", "PatternChild", "DependentChild", "DefinitionChild",
	}
	for _, name := range childNames {
		components.Set(name, schema("string", ""))
	}
	ref := func(name string) *highbase.SchemaProxy {
		return highbase.CreateSchemaProxyRef("#/components/schemas/" + name)
	}
	components.Set("Container", highbase.CreateSchemaProxy(&highbase.Schema{
		AllOf:                 []*highbase.SchemaProxy{ref("AllOfChild")},
		OneOf:                 []*highbase.SchemaProxy{ref("OneOfChild")},
		AnyOf:                 []*highbase.SchemaProxy{ref("AnyOfChild")},
		PrefixItems:           []*highbase.SchemaProxy{ref("PrefixChild")},
		Contains:              ref("ContainsChild"),
		If:                    ref("IfChild"),
		Else:                  ref("ElseChild"),
		Then:                  ref("ThenChild"),
		PropertyNames:         ref("PropertyNamesChild"),
		UnevaluatedItems:      ref("UnevaluatedItemsChild"),
		Not:                   ref("NotChild"),
		ContentSchema:         ref("ContentChild"),
		Items:                 &highbase.DynamicValue[*highbase.SchemaProxy, bool]{A: ref("ItemsChild")},
		AdditionalProperties:  &highbase.DynamicValue[*highbase.SchemaProxy, bool]{A: ref("AdditionalPropertiesChild")},
		UnevaluatedProperties: &highbase.DynamicValue[*highbase.SchemaProxy, bool]{A: ref("UnevaluatedPropertiesChild")},
		Properties:            schemaMap("property", ref("PropertyChild")),
		PatternProperties:     schemaMap("pattern", ref("PatternChild")),
		DependentSchemas:      schemaMap("dependent", ref("DependentChild")),
		Defs:                  schemaMap("definition", ref("DefinitionChild")),
	}))
	emitter := newTestEmitter(&sdk.Contract{Schemas: components}, "client", nil)
	name, err := emitter.schemaType(highbase.CreateSchemaProxyRef("#/components/schemas/Container"), "ignored")
	if err != nil || name != "Container" {
		t.Fatalf("component type = %q, %v", name, err)
	}
	// A second visit exercises cycle/duplicate protection rather than duplicating models.
	emitter.collectSchema(highbase.CreateSchemaProxyRef("#/components/schemas/Container"))
	container, _ := components.Get("Container")
	emitter.collectSchema(container)
	exactShapeChildren := []string{"AllOfChild", "OneOfChild", "AnyOfChild", "ItemsChild", "AdditionalPropertiesChild", "PropertyChild", "PatternChild"}
	if emitter.collectErr != nil || emitter.schemas.Len() != len(exactShapeChildren)+1 {
		t.Fatalf("unexpected traversal result: schemas=%d err=%v", emitter.schemas.Len(), emitter.collectErr)
	}
	for _, name := range exactShapeChildren {
		if _, ok := emitter.schemas.Get(name); !ok {
			t.Errorf("referenced component %q was not collected", name)
		}
	}

	if _, err := emitter.schemaType(nil, "missing"); err == nil || !strings.Contains(err.Error(), "schema is required") {
		t.Fatalf("expected nil schema error, got %v", err)
	}
	if _, err := emitter.schemaType(&highbase.SchemaProxy{}, "broken"); err == nil || !strings.Contains(err.Error(), "could not be resolved") {
		t.Fatalf("expected unresolved schema error, got %v", err)
	}
	if _, err := emitter.schemaType(highbase.CreateSchemaProxyRef("#/components/schemas/Missing"), "missing"); err == nil || !strings.Contains(err.Error(), "not defined") {
		t.Fatalf("expected missing reference error, got %v", err)
	}
}

func TestSchemaReferenceAndCollectionErrors(t *testing.T) {
	invalidReference := highbase.CreateSchemaProxyRef("#/components/responses/Problem")
	emitter := newTestEmitter(&sdk.Contract{}, "client", nil)
	if _, err := emitter.schemaType(invalidReference, "invalid"); err == nil || !strings.Contains(err.Error(), "not a components/schemas reference") {
		t.Fatalf("expected non-schema reference error, got %v", err)
	}
	array := arraySchema(&highbase.SchemaProxy{})
	if _, err := emitter.schemaType(array, "items"); err == nil || !strings.Contains(err.Error(), "schema could not be resolved") {
		t.Fatalf("expected invalid array item error, got %v", err)
	}

	invalidCollector := newTestEmitter(&sdk.Contract{}, "client", nil)
	invalidCollector.collectSchema(invalidReference)
	if invalidCollector.collectErr == nil || !strings.Contains(invalidCollector.collectErr.Error(), "not a components/schemas reference") {
		t.Fatalf("expected non-schema collected reference error, got %v", invalidCollector.collectErr)
	}

	missingCollector := newTestEmitter(&sdk.Contract{}, "client", nil)
	missingCollector.collectSchema(highbase.CreateSchemaProxyRef("#/components/schemas/Missing"))
	if missingCollector.collectErr == nil || !strings.Contains(missingCollector.collectErr.Error(), "not defined") {
		t.Fatalf("expected missing collected component error, got %v", missingCollector.collectErr)
	}

	unresolvedCollector := newTestEmitter(&sdk.Contract{}, "client", nil)
	unresolvedCollector.collectSchema(&highbase.SchemaProxy{})
	if unresolvedCollector.collectErr != nil {
		t.Fatalf("unresolved inline schema should be ignored for later model validation: %v", unresolvedCollector.collectErr)
	}
}

func TestReferenceKindsUseCorrectResolutionPath(t *testing.T) {
	components := orderedmap.New[string, *highbase.SchemaProxy]()
	components.Set("Tenant/Record~V2", schema("string", ""))
	emitter := newTestEmitter(&sdk.Contract{Schemas: components}, "client", nil)
	name, err := emitter.schemaType(highbase.CreateSchemaProxyRef("#/components/schemas/Tenant~1Record~0V2"), "ignored")
	if err != nil || name != "TenantRecordV2" {
		t.Fatalf("JSON pointer component resolution = %q, %v", name, err)
	}

	operation := operationWithSuccess("read")
	operation.Responses = []*sdk.Response{{
		Status: "200", Content: jsonContent(highbase.CreateSchemaProxyRef("../shared.yaml#/components/schemas/Record")),
	}}
	_, err = GenerateContract(contractWith(operation), Options{Models: []modelgen.Option{
		modelgen.WithExternalRefTypeResolver(func(ref string) string {
			if ref == "../shared.yaml#/components/schemas/Record" {
				return "SharedRecord"
			}
			return ""
		}),
	}})
	if err == nil || !strings.Contains(err.Error(), "external schema reference") {
		t.Fatalf("expected external reference rejection, got %v", err)
	}
}

func TestParameterSchemaSupportEdges(t *testing.T) {
	if parameterSchemaSupported(nil) || parameterSchemaSupported(&highbase.SchemaProxy{}) {
		t.Fatal("nil and unresolved parameter schemas must be rejected")
	}
	if parameterSchemaSupported(highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string", "integer"}})) {
		t.Fatal("multiple concrete types must be rejected")
	}
	if parameterSchemaSupported(arraySchema(highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"object"}}))) {
		t.Fatal("arrays of objects must be rejected")
	}
	if !parameterSchemaSupported(highbase.CreateSchemaProxy(&highbase.Schema{Enum: []*yaml.Node{{Value: "one"}}})) {
		t.Fatal("enum-only scalar schema must be supported")
	}
	if !parameterSchemaSupported(highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"null", "string"}})) {
		t.Fatal("nullable scalar schema must be supported")
	}
}

func TestParameterEncodersAndStatusConditions(t *testing.T) {
	encoderTests := []struct {
		name    string
		schema  *highbase.SchemaProxy
		encoder string
		array   bool
		wantErr string
	}{
		{name: "nil", wantErr: "schema is required"},
		{name: "unresolved", schema: &highbase.SchemaProxy{}, wantErr: "could not be resolved"},
		{name: "string", schema: schema("string", ""), encoder: "encodeString"},
		{name: "integer", schema: schema("integer", "int64"), encoder: "encodeInteger"},
		{name: "float", schema: schema("number", "float"), encoder: "encodeFloat32"},
		{name: "double", schema: schema("number", "double"), encoder: "encodeFloat64"},
		{name: "boolean", schema: schema("boolean", ""), encoder: "encodeBoolean"},
		{name: "array", schema: arraySchema(schema("string", "")), encoder: "encodeString", array: true},
		{name: "untyped array", schema: schema("array", ""), wantErr: "item schema is required"},
		{name: "enum", schema: highbase.CreateSchemaProxy(&highbase.Schema{Enum: []*yaml.Node{{Value: "one"}}}), encoder: "encodeAny"},
		{name: "unsupported", schema: schema("object", ""), wantErr: "unsupported parameter type"},
	}
	for _, test := range encoderTests {
		t.Run(test.name, func(t *testing.T) {
			encoder, array, err := newTestEmitter(&sdk.Contract{}, "client", nil).parameterEncoder(test.schema)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("expected %q, got %v", test.wantErr, err)
				}
				return
			}
			if err != nil || encoder != test.encoder || array != test.array {
				t.Fatalf("parameterEncoder() = %q, %t, %v", encoder, array, err)
			}
		})
	}
	mappedEmitter := newEmitter(&sdk.Contract{}, "client", nil, modelgen.NewGenerator(modelgen.WithFormatMapping("date-time", "time.Time", "time")))
	if _, _, err := mappedEmitter.parameterEncoder(schema("string", "date-time")); err == nil || !strings.Contains(err.Error(), "not supported for parameter serialization") {
		t.Fatalf("expected mapped string rejection, got %v", err)
	}

	conditionTests := []struct {
		statuses []string
		want     string
		wantErr  bool
	}{
		{statuses: nil, want: "false"},
		{statuses: []string{"201", "2xx"}, want: "response.StatusCode == 201 || response.StatusCode/100 == 2"},
		{statuses: []string{"default"}, want: "true"},
		{statuses: []string{"099"}, wantErr: true},
		{statuses: []string{"600"}, wantErr: true},
		{statuses: []string{"2XY"}, wantErr: true},
		{statuses: []string{"20"}, wantErr: true},
	}
	for _, test := range conditionTests {
		condition, err := statusCondition(test.statuses)
		if test.wantErr {
			if err == nil {
				t.Fatalf("statusCondition(%v) unexpectedly succeeded", test.statuses)
			}
			continue
		}
		if err != nil || condition != test.want {
			t.Fatalf("statusCondition(%v) = %q, %v", test.statuses, condition, err)
		}
	}
}

func TestHelpersCoverEdgeContracts(t *testing.T) {
	nilDefinition := orderedmap.New[string, *highv3.MediaType]()
	nilDefinition.Set("application/json", nil)
	if _, _, err := jsonMedia(nilDefinition); err == nil || !strings.Contains(err.Error(), "has no definition") {
		t.Fatalf("expected nil media definition error, got %v", err)
	}
	if mediaType, schemaProxy, err := jsonMedia(media("application/problem+json", schema("string", ""))); err != nil || mediaType != "application/problem+json" || schemaProxy == nil {
		t.Fatalf("unexpected suffix JSON result: %q %#v %v", mediaType, schemaProxy, err)
	}
	if mediaType, schemaProxy, err := jsonMedia(nil); err != nil || mediaType != "" || schemaProxy != nil {
		t.Fatalf("unexpected empty media result: %q %#v %v", mediaType, schemaProxy, err)
	}
	for _, test := range []struct {
		location, style string
		want            bool
	}{
		{"path", "simple", true}, {"header", "simple", true}, {"query", "form", true}, {"cookie", "form", false},
	} {
		if got := supportedStyle(test.location, test.style); got != test.want {
			t.Fatalf("supportedStyle(%q, %q) = %t", test.location, test.style, got)
		}
	}
	if statusName("default") != "Default" || statusName("2xx") != "Status2XX" {
		t.Fatal("unexpected status names")
	}
	if commentLine("  first line\nsecond */ line") != "first line" || commentLine(" \n") != "" {
		t.Fatal("unexpected comment sanitization")
	}
	if commentLine("unsafe */ suffix") != "unsafe * / suffix" {
		t.Fatal("comment terminator was not sanitized")
	}
	registry := modelgen.NewNameRegistry("Value", "Thing", "ThingModel", "Thing__2")
	if got := registry.Claim("", ""); got != "Value__2" {
		t.Fatalf("empty preferred claim = %q", got)
	}
	if got := registry.Claim("Thing", "Model"); got != "Thing__3" {
		t.Fatalf("collision claim = %q", got)
	}
}

func TestGenerateContractSurfacesPipelineErrors(t *testing.T) {
	operation := operationWithSuccess("read")
	operation.Responses = []*sdk.Response{{
		Status:  "200",
		Content: jsonContent(highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}, Format: "broken"})),
	}}
	_, err := GenerateContract(contractWith(operation), Options{Models: []modelgen.Option{
		modelgen.WithFormatMapping("broken", "not valid", ""),
	}})
	if err == nil || !strings.Contains(err.Error(), "generate models") {
		t.Fatalf("expected model generation error, got %v", err)
	}

	assertRenderFailure := func(t *testing.T, target **template.Template, source string, options Options) {
		t.Helper()
		original := *target
		*target = template.Must(template.New("broken").Parse(source))
		t.Cleanup(func() { *target = original })
		_, err := GenerateContract(contractWith(operationWithSuccess("read")), options)
		if err == nil || !strings.Contains(err.Error(), "broken") {
			t.Fatalf("expected embedded template failure, got %v", err)
		}
	}
	t.Run("client", func(t *testing.T) {
		assertRenderFailure(t, &clientTemplate, "package p\nvar _ = {{.Missing}}", Options{})
	})
	t.Run("resources", func(t *testing.T) {
		assertRenderFailure(t, &resourcesTemplate, "package", Options{})
	})
	t.Run("workflows", func(t *testing.T) {
		workflow := &sdk.Workflow{
			ID: "run", Operation: operationWithSuccess("read"),
			Inputs:          workflowInputs(t, "type: object\nproperties: {value: {type: string}}"),
			SuccessStatuses: []string{"204"},
		}
		assertRenderFailure(t, &workflowsTemplate, "package", Options{Workflows: []*sdk.Workflow{workflow}})
	})
}

func TestPrepareOperationRejectsInvalidResponseStatuses(t *testing.T) {
	for _, test := range []struct {
		name      string
		responses []*sdk.Response
		want      string
	}{
		{name: "success", responses: []*sdk.Response{{Status: "2a0"}}, want: "success responses"},
		{name: "JSON success", responses: []*sdk.Response{{Status: "2a0", Content: jsonContent(schema("string", ""))}}, want: "JSON success responses"},
		{name: "error", responses: []*sdk.Response{{Status: "204"}, {Status: "oops"}}, want: "error response"},
	} {
		t.Run(test.name, func(t *testing.T) {
			operation := operationWithSuccess("read")
			operation.Responses = test.responses
			_, err := newTestEmitter(contractWith(operation), "client", nil).prepareOperation(operation, "Things")
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q error, got %v", test.want, err)
			}
		})
	}
}

func TestPrepareWorkflowValidationBranches(t *testing.T) {
	operation := operationView{
		ID: "create", ResourceName: "Things", MethodName: "Create", ParamsType: "CreateParams",
		SuccessStatuses: []string{"201"}, ResponseType: "Thing", HasParameters: true,
		Parameters: []parameterView{{Name: "tenantId", FieldName: "TenantID", Type: "string", In: "path", Required: true}},
		Body:       &bodyView{Type: "CreateThing", Required: true},
	}
	validInputs := func(t *testing.T) *yaml.Node {
		return workflowInputs(t, "type: object\nproperties: {tenant: {type: string}, name: {type: string}}")
	}
	tests := []struct {
		name   string
		inputs func(*testing.T) *yaml.Node
		set    func(*sdk.Workflow)
		want   string
	}{
		{name: "missing inputs", want: "object JSON Schema"},
		{name: "scalar inputs", inputs: func(t *testing.T) *yaml.Node { return workflowInputs(t, "type: string") }, want: "object JSON Schema"},
		{name: "invalid success", inputs: validInputs, set: func(workflow *sdk.Workflow) { workflow.SuccessStatuses = []string{"2999"} }, want: "success responses"},
		{name: "unknown parameter", inputs: validInputs, set: func(workflow *sdk.Workflow) {
			workflow.ParameterBindings = []*sdk.ParameterBinding{{Name: "missing", In: "path", Input: "tenant"}}
		}, want: "is not generated"},
		{name: "undeclared parameter input", inputs: validInputs, set: func(workflow *sdk.Workflow) {
			workflow.ParameterBindings = []*sdk.ParameterBinding{{Name: "tenantId", In: "path", Input: "missing"}}
		}, want: "not declared"},
		{name: "undeclared payload input", inputs: validInputs, set: func(workflow *sdk.Workflow) {
			workflow.PayloadBindings = []*sdk.PayloadBinding{{Property: "name", Input: "missing"}}
		}, want: "not declared"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workflow := &sdk.Workflow{ID: "run", Operation: &sdk.Operation{ID: "create"}, SuccessStatuses: []string{"201"}}
			if test.inputs != nil {
				workflow.Inputs = test.inputs(t)
			}
			if test.set != nil {
				test.set(workflow)
			}
			_, err := newTestEmitter(&sdk.Contract{}, "client", nil).prepareWorkflow(workflow, map[string]*operationView{"create": &operation})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q error, got %v", test.want, err)
			}
		})
	}

	workflow := &sdk.Workflow{
		ID: "skipNil", Operation: &sdk.Operation{ID: "create"}, Inputs: validInputs(t), SuccessStatuses: []string{"201"},
		ParameterBindings: []*sdk.ParameterBinding{nil, {Name: "tenantId", In: "path", Input: "tenant"}},
		PayloadBindings:   []*sdk.PayloadBinding{nil, {Property: "name", Input: "name"}},
	}
	view, err := newTestEmitter(&sdk.Contract{}, "client", nil).prepareWorkflow(workflow, map[string]*operationView{"create": &operation})
	if err != nil || len(view.ParameterFields) != 1 || len(view.BodyFields) != 1 {
		t.Fatalf("nil bindings were not ignored: %#v, %v", view, err)
	}
}

func TestWorkflowInputSchemaReportsReferenceFailures(t *testing.T) {
	for _, test := range []struct {
		name   string
		schema string
		want   string
	}{
		{name: "root reference", schema: "$ref: '#/components/schemas/Missing'", want: "parse JSON Schema"},
		{name: "child reference", schema: "type: object\nproperties:\n  value:\n    $ref: '#/components/schemas/Missing'", want: "build JSON Schema"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := workflowInputSchema(workflowInputs(t, test.schema))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q error, got %v", test.want, err)
			}
		})
	}
	proxy, schema, err := workflowInputSchema(workflowInputs(t, `type: object
$defs:
  Identifier: {type: string}
properties:
  value: {$ref: '#/$defs/Identifier'}
`))
	if err != nil || proxy == nil || schema == nil || schema.Properties == nil || schema.Properties.GetOrZero("value") == nil {
		t.Fatalf("valid local workflow input reference was not resolved: %#v %#v %v", proxy, schema, err)
	}
}

func TestResolveWorkflowFieldTypeErrors(t *testing.T) {
	types := []*modelgen.GeneratedType{
		nil,
		{Name: "Input", Kind: modelgen.KindObject, Fields: []modelgen.GeneratedField{{Name: "Text", Source: "text", Type: "string"}}},
		{Name: "Body", Kind: modelgen.KindObject, Fields: []modelgen.GeneratedField{{Name: "Count", Source: "count", Type: "int"}}},
		{Name: "Alias", Kind: modelgen.KindObject, Embedded: []string{"Alias"}},
	}
	tests := []struct {
		name     string
		workflow workflowView
		want     string
	}{
		{name: "missing input model", workflow: workflowView{ID: "run", InputType: "Missing"}, want: "not a struct model"},
		{name: "missing parameter input", workflow: workflowView{ID: "run", InputType: "Input", ParameterFields: []workflowAssignmentView{{Input: "missing"}}}, want: "is not present"},
		{name: "parameter type mismatch", workflow: workflowView{ID: "run", InputType: "Input", ParameterFields: []workflowAssignmentView{{Input: "text", ExpectedType: "int"}}}, want: "operation parameter requires"},
		{name: "missing payload input", workflow: workflowView{ID: "run", InputType: "Input", BodyFields: []workflowAssignmentView{{Input: "missing", ExpectedModel: "Body", ExpectedField: "count"}}}, want: "is not present"},
		{name: "payload type mismatch", workflow: workflowView{ID: "run", InputType: "Input", BodyFields: []workflowAssignmentView{{Input: "text", ExpectedModel: "Body", ExpectedField: "count"}}}, want: "payload property"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := resolveWorkflowFieldTypes(types, []workflowView{test.workflow})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q error, got %v", test.want, err)
			}
		})
	}
	if _, ok := generatedField(map[string]*modelgen.GeneratedType{"Alias": types[3]}, types[3], "missing", make(map[string]struct{})); ok {
		t.Fatal("cyclic embedded type unexpectedly resolved a field")
	}
	if _, ok := generatedField(nil, nil, "missing", make(map[string]struct{})); ok {
		t.Fatal("nil generated type unexpectedly resolved a field")
	}
}

func TestReachabilityAndCollectionBranches(t *testing.T) {
	components := orderedmap.New[string, *highbase.SchemaProxy]()
	for _, name := range []string{"One", "Any", "Additional"} {
		components.Set(name, schema("string", ""))
	}
	root := highbase.CreateSchemaProxy(&highbase.Schema{
		OneOf:                []*highbase.SchemaProxy{highbase.CreateSchemaProxyRef("#/components/schemas/One")},
		AnyOf:                []*highbase.SchemaProxy{highbase.CreateSchemaProxyRef("#/components/schemas/Any")},
		AdditionalProperties: &highbase.DynamicValue[*highbase.SchemaProxy, bool]{A: highbase.CreateSchemaProxyRef("#/components/schemas/Additional")},
	})
	operation := operationWithSuccess("read")
	operation.Responses = []*sdk.Response{{Status: "200", Content: jsonContent(root)}}
	emitter := newTestEmitter(&sdk.Contract{Operations: []*sdk.Operation{operation}, Schemas: components}, "client", nil)
	for _, name := range []string{"One", "Any", "Additional"} {
		if _, ok := emitter.reachableComponents[name]; !ok {
			t.Fatalf("component %q was not discovered", name)
		}
	}
	emitter.collectSchema(nil)
	emitter.collectSchema(highbase.CreateSchemaProxyRef("../shared.yaml#/components/schemas/Record"))
	if emitter.collectErr == nil || !strings.Contains(emitter.collectErr.Error(), "external schema reference") {
		t.Fatalf("expected external reference collection error, got %v", emitter.collectErr)
	}
}

func schema(schemaType, format string) *highbase.SchemaProxy {
	return highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{schemaType}, Format: format})
}

func arraySchema(items *highbase.SchemaProxy) *highbase.SchemaProxy {
	return highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"array"}, Items: &highbase.DynamicValue[*highbase.SchemaProxy, bool]{A: items}})
}

func schemaMap(name string, proxy *highbase.SchemaProxy) *orderedmap.Map[string, *highbase.SchemaProxy] {
	result := orderedmap.New[string, *highbase.SchemaProxy]()
	result.Set(name, proxy)
	return result
}

func media(mediaType string, proxy *highbase.SchemaProxy) *orderedmap.Map[string, *highv3.MediaType] {
	content := orderedmap.New[string, *highv3.MediaType]()
	content.Set(mediaType, &highv3.MediaType{Schema: proxy})
	return content
}

func jsonContent(proxy *highbase.SchemaProxy) *orderedmap.Map[string, *highv3.MediaType] {
	return media("application/json", proxy)
}

func jsonBody(proxy *highbase.SchemaProxy) *sdk.RequestBody {
	return &sdk.RequestBody{Required: true, Content: jsonContent(proxy)}
}

func successResponses() []*sdk.Response {
	return []*sdk.Response{{Status: "204"}}
}

func operationWithSuccess(id string) *sdk.Operation {
	return &sdk.Operation{ID: id, Method: "GET", Path: "/" + id, Resource: "Things", Name: id, Responses: successResponses()}
}

func contractWith(operations ...*sdk.Operation) *sdk.Contract {
	return &sdk.Contract{Operations: operations, Schemas: orderedmap.New[string, *highbase.SchemaProxy]()}
}

func newTestEmitter(contract *sdk.Contract, packageName string, workflows []*sdk.Workflow) *emitter {
	return newEmitter(contract, packageName, workflows, modelgen.NewGenerator())
}

func workflowInputs(t *testing.T, source string) *yaml.Node {
	t.Helper()
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(source), &node); err != nil {
		t.Fatal(err)
	}
	return node.Content[0]
}
