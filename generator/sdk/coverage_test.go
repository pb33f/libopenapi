// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package sdk

import (
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	higharazzo "github.com/pb33f/libopenapi/datamodel/high/arazzo"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	highv3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

func TestPrepareInputAndSelectionErrors(t *testing.T) {
	t.Run("nil document", func(t *testing.T) {
		assertErrorContains(t, prepareContract(nil, PrepareOptions{}), "OpenAPI document is required")
	})
	t.Run("blank selection", func(t *testing.T) {
		assertErrorContains(t, prepareContract(&highv3.Document{}, PrepareOptions{Operations: []string{" "}}), "operationId cannot be empty")
	})
	t.Run("selected operation without paths", func(t *testing.T) {
		assertErrorContains(t, prepareContract(&highv3.Document{}, PrepareOptions{Operations: []string{"missing"}}), "has no paths")
	})
	t.Run("no paths is valid without selection", func(t *testing.T) {
		contract, err := Prepare(&highv3.Document{}, PrepareOptions{})
		if err != nil || contract.Schemas == nil {
			t.Fatalf("unexpected empty contract: %#v, %v", contract, err)
		}
	})
	t.Run("unknown operation", func(t *testing.T) {
		assertErrorContains(t, prepareYAMLContract(t, minimalOpenAPI, PrepareOptions{Operations: []string{"missing"}}), `operationId "missing" was not found`)
	})
	t.Run("duplicate operation", func(t *testing.T) {
		assertErrorContains(t, prepareYAMLContract(t, `openapi: 3.1.0
info: {title: duplicate, version: 1.0.0}
paths:
  /a: {get: {operationId: repeated, responses: {'200': {description: ok}}}}
  /b: {post: {operationId: repeated, responses: {'200': {description: ok}}}}
`, PrepareOptions{}), `duplicate operationId "repeated"`)
	})
}

func TestPrepareFullContractAndOverrides(t *testing.T) {
	contract, err := prepareYAML(t, `openapi: 3.1.0
info: {title: Rich API, version: 1.0.0}
servers:
  - url: https://{region}.example.test/{version}
    variables:
      region: {default: us}
      version: {default: v2}
  - {url: ''}
security:
  - oauth: [read, write]
paths:
  /things/{id}:
    servers: [{url: https://path.example.test}]
    parameters:
      - {name: id, in: path, required: true, schema: {type: string}}
      - {name: replace, in: query, schema: {type: string}}
      - null
    get:
      operationId: getThings
      tags: ['']
      servers: [{url: https://operation.example.test}]
      parameters:
        - {name: replace, in: query, required: true, style: spaceDelimited, explode: false, allowReserved: true, schema: {type: string}}
        - {name: cookie, in: cookie, schema: {type: string}}
      requestBody:
        required: false
        description: optional body
        content: {application/json: {schema: {type: object}}}
      responses:
        '200': {description: ok}
        '404': null
        default: {description: fallback}
components:
  schemas: {Thing: {type: object}}
  securitySchemes:
    oauth: {type: oauth2, flows: {clientCredentials: {tokenUrl: https://example.test/token, scopes: {read: Read, write: Write}}}}
`, PrepareOptions{Names: map[string]OperationName{"getThings": {Resource: "Inventory", Method: "fetch"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(contract.Servers) != 1 || contract.Servers[0] != "https://us.example.test/v2" {
		t.Fatalf("document metadata was not prepared: %#v", contract)
	}
	if contract.Schemas.Len() != 1 || len(contract.SecuritySchemes) != 1 {
		t.Fatalf("components were not retained: %#v", contract)
	}
	op := contract.Operations[0]
	if op.Resource != "Inventory" || op.Name != "fetch" || len(op.Servers) != 1 || op.Servers[0] != "https://operation.example.test" {
		t.Fatalf("operation overrides or server precedence failed: %#v", op)
	}
	byName := make(map[string]*Parameter, len(op.Parameters))
	for _, parameter := range op.Parameters {
		byName[parameter.Name] = parameter
	}
	if len(op.Parameters) != 4 || !byName["replace"].Required || byName["replace"].Style != "spaceDelimited" || byName["replace"].Explode || !byName["replace"].AllowReserved || byName["cookie"].Style != "form" || !byName["cookie"].Explode {
		t.Fatalf("parameter merge/defaults failed: %#v", op.Parameters)
	}
	if op.RequestBody == nil || op.RequestBody.Required || len(op.Responses) != 3 || op.Responses[2].Status != "default" {
		t.Fatalf("body/responses were not prepared: %#v %#v", op.RequestBody, op.Responses)
	}
	if len(op.Security) != 1 || len(op.Security[0].Schemes) != 1 || strings.Join(op.Security[0].Schemes[0].Scopes, ",") != "read,write" {
		t.Fatalf("security scopes were lost: %#v", op.Security)
	}
}

func TestPrepareServerPrecedenceAndInvalidServers(t *testing.T) {
	t.Run("document servers are shared", func(t *testing.T) {
		contract, err := prepareYAML(t, `openapi: 3.1.0
info: {title: server, version: 1.0.0}
servers: [{url: https://document.example.test}]
paths:
  /a: {get: {operationId: a, responses: {'200': {description: ok}}}}
  /b: {get: {operationId: b, responses: {'200': {description: ok}}}}
`, PrepareOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if &contract.Servers[0] != &contract.Operations[0].Servers[0] || &contract.Servers[0] != &contract.Operations[1].Servers[0] {
			t.Fatal("inherited document server slice was copied")
		}
	})
	t.Run("path server", func(t *testing.T) {
		contract, err := prepareYAML(t, `openapi: 3.1.0
info: {title: server, version: 1.0.0}
servers: [{url: https://document.example.test}]
paths:
  /x:
    servers: [{url: https://path.example.test}]
    get: {operationId: x, responses: {'200': {description: ok}}}
`, PrepareOptions{})
		if err != nil || contract.Operations[0].Servers[0] != "https://path.example.test" {
			t.Fatalf("path server not selected: %#v, %v", contract, err)
		}
	})
	t.Run("default relative server", func(t *testing.T) {
		contract, err := prepareYAML(t, minimalOpenAPI, PrepareOptions{})
		if err != nil || contract.Operations[0].Servers[0] != "/" {
			t.Fatalf("default server not selected: %#v, %v", contract, err)
		}
	})
	t.Run("nil variable", func(t *testing.T) {
		variables := orderedmap.New[string, *highv3.ServerVariable]()
		variables.Set("region", nil)
		_, err := resolveServerURL(&highv3.Server{URL: "https://{region}.example.test", Variables: variables})
		assertErrorContains(t, err, `variable "region" has no definition`)
	})
	t.Run("malformed operation server", func(t *testing.T) {
		assertErrorContains(t, prepareYAMLContract(t, `openapi: 3.1.0
info: {title: server, version: 1.0.0}
paths:
  /x:
    get:
      operationId: x
      servers: [{url: 'https://{missing}.example.test'}]
      responses: {'200': {description: ok}}
`, PrepareOptions{}), `prepare server`)
	})
}

func TestPrepareHelperEdgeCases(t *testing.T) {
	required, explode := true, false
	parameters := mergeParameters(
		[]*highv3.Parameter{nil, {Name: "q", In: "query"}, {Name: "p", In: "path", Required: &required}},
		[]*highv3.Parameter{{Name: "q", In: "query", Required: &required, Explode: &explode}},
	)
	if len(parameters) != 2 || !parameters[0].Required || parameters[0].Explode || parameters[1].Style != "simple" || parameters[1].Explode {
		t.Fatalf("unexpected parameters: %#v", parameters)
	}
	if prepareRequestBody(nil) != nil || prepareResponses(nil) != nil {
		t.Fatal("nil body or responses should remain nil")
	}
	responses := &highv3.Responses{Codes: orderedmap.New[string, *highv3.Response]()}
	responses.Codes.Set("204", nil)
	if got := prepareResponses(responses); len(got) != 0 {
		t.Fatalf("nil responses should be skipped: %#v", got)
	}
	if got := prepareSecurity([]*highbase.SecurityRequirement{nil}); len(got) != 1 || len(got[0].Schemes) != 0 {
		t.Fatalf("nil security alternative should be preserved: %#v", got)
	}
	if got := methodSeed("things", "Things"); got != "things" {
		t.Fatalf("same-length resource should not produce empty name: %q", got)
	}
	if plural, singular := methodSeed("getGoals", "Goals"), methodSeed("getGoal", "Goals"); plural != "get" || singular != "getGoal" {
		t.Fatalf("plural and singular operations collided: %q, %q", plural, singular)
	}
	optionalPath := prepareParameter(&highv3.Parameter{Name: "id", In: "path"})
	if !optionalPath.Required {
		t.Fatal("path parameter must be required even when the source omits required")
	}
}

func TestPrepareSkipsNilComponentsPathsAndUnnamedOperations(t *testing.T) {
	securitySchemes := orderedmap.New[string, *highv3.SecurityScheme]()
	securitySchemes.Set("missing", nil)
	pathItems := orderedmap.New[string, *highv3.PathItem]()
	pathItems.Set("/hidden", nil)
	pathItems.Set("/unnamed", &highv3.PathItem{Get: &highv3.Operation{OperationId: " "}})
	document := &highv3.Document{
		Components: &highv3.Components{SecuritySchemes: securitySchemes},
		Paths:      &highv3.Paths{PathItems: pathItems},
	}
	contract, err := Prepare(document, PrepareOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(contract.SecuritySchemes) != 0 || len(contract.Operations) != 0 {
		t.Fatalf("nil components and unnamed operations should be skipped: %#v", contract)
	}
}

func TestPrepareArazzoTopLevelErrorsAndSelection(t *testing.T) {
	assertErrorContains(t, prepareArazzo(nil, &Contract{}), "Arazzo document is required")
	assertErrorContains(t, prepareArazzo(&higharazzo.Arazzo{}, nil), "OpenAPI contract is required")
	assertErrorContains(t, prepareArazzo(&higharazzo.Arazzo{}, &Contract{}, " "), "workflowId cannot be empty")
	assertErrorContains(t, prepareArazzo(&higharazzo.Arazzo{}, &Contract{}, "missing"), `workflowId "missing" was not found`)

	document := &higharazzo.Arazzo{Workflows: []*higharazzo.Workflow{nil, {WorkflowId: "ignored"}}}
	assertErrorContains(t, prepareArazzo(document, &Contract{}, "ignored"), "exactly one operation step")
	workflows, err := PrepareArazzo(&higharazzo.Arazzo{Workflows: []*higharazzo.Workflow{nil}}, &Contract{})
	if err != nil || len(workflows) != 0 {
		t.Fatalf("nil workflow should be ignored: %#v, %v", workflows, err)
	}
	selected := &higharazzo.Workflow{WorkflowId: "selected", Steps: []*higharazzo.Step{{
		OperationId: "op", SuccessCriteria: []*higharazzo.Criterion{{Condition: "$statusCode == 200"}},
	}}}
	workflows, err = PrepareArazzo(
		&higharazzo.Arazzo{Workflows: []*higharazzo.Workflow{{WorkflowId: "ignored"}, selected}},
		&Contract{Operations: []*Operation{{ID: "op"}}},
		"selected",
	)
	if err != nil || len(workflows) != 1 || workflows[0].ID != "selected" {
		t.Fatalf("workflow selection failed: %#v, %v", workflows, err)
	}
}

func TestPrepareWorkflowRejectsUnsupportedShapes(t *testing.T) {
	operation := &Operation{ID: "op"}
	operations := map[string]*Operation{"op": operation}
	validStep := func() *higharazzo.Step {
		return &higharazzo.Step{OperationId: "op", SuccessCriteria: []*higharazzo.Criterion{{Condition: "$statusCode == 200"}}}
	}
	tests := []struct {
		name string
		edit func(*higharazzo.Workflow)
		want string
	}{
		{"missing id", func(w *higharazzo.Workflow) { w.WorkflowId = "" }, "workflowId is required"},
		{"workflow outputs", func(w *higharazzo.Workflow) {
			w.Outputs = orderedmap.New[string, *higharazzo.OutputValue]()
		}, "workflow-level dependencies"},
		{"workflow dependency", func(w *higharazzo.Workflow) { w.DependsOn = []string{"x"} }, "workflow-level dependencies"},
		{"no steps", func(w *higharazzo.Workflow) { w.Steps = nil }, "exactly one operation step"},
		{"nil step", func(w *higharazzo.Workflow) { w.Steps = []*higharazzo.Step{nil} }, "exactly one operation step"},
		{"operation path", func(w *higharazzo.Workflow) { w.Steps[0].OperationPath = "/x" }, "exactly one operationId"},
		{"workflow reference", func(w *higharazzo.Workflow) { w.Steps[0].WorkflowId = "other" }, "exactly one operationId"},
		{"missing operation id", func(w *higharazzo.Workflow) { w.Steps[0].OperationId = "" }, "exactly one operationId"},
		{"step outputs", func(w *higharazzo.Workflow) {
			w.Steps[0].Outputs = orderedmap.New[string, *higharazzo.OutputValue]()
		}, "step actions and outputs"},
		{"step parameter", func(w *higharazzo.Workflow) { w.Steps[0].Parameters = []*higharazzo.Parameter{{}} }, "direct named parameter bindings"},
		{"unknown operation", func(w *higharazzo.Workflow) { w.Steps[0].OperationId = "missing" }, `operationId "missing"`},
		{"unexpected body", func(w *higharazzo.Workflow) { w.Steps[0].RequestBody = &higharazzo.RequestBody{} }, "does not"},
		{"required body missing", func(w *higharazzo.Workflow) { operation.RequestBody = &RequestBody{Required: true} }, "required OpenAPI request body"},
		{"no criteria", func(w *higharazzo.Workflow) { w.Steps[0].SuccessCriteria = nil }, "criterion is required"},
		{"nil criterion", func(w *higharazzo.Workflow) { w.Steps[0].SuccessCriteria = []*higharazzo.Criterion{nil} }, "only simple"},
		{"criterion context", func(w *higharazzo.Workflow) { w.Steps[0].SuccessCriteria[0].Context = "$response.body" }, "only simple"},
		{"criterion type", func(w *higharazzo.Workflow) { w.Steps[0].SuccessCriteria[0].Type = "regex" }, "only simple"},
		{"criterion expression", func(w *higharazzo.Workflow) { w.Steps[0].SuccessCriteria[0].Condition = "$statusCode > 199" }, "unsupported success criterion"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			operation.RequestBody = nil
			workflow := &higharazzo.Workflow{WorkflowId: "workflow", Steps: []*higharazzo.Step{validStep()}}
			test.edit(workflow)
			_, err := prepareWorkflow(workflow, operations)
			assertErrorContains(t, err, test.want)
		})
	}
}

func TestPrepareWorkflowAcceptsRepeatedEquivalentStatusCriteria(t *testing.T) {
	workflow := &higharazzo.Workflow{WorkflowId: "workflow", Steps: []*higharazzo.Step{{
		OperationId: "op",
		SuccessCriteria: []*higharazzo.Criterion{
			{Condition: "$statusCode == 200"},
			{Condition: "$statusCode == 200"},
		},
	}}}
	prepared, err := prepareWorkflow(workflow, map[string]*Operation{"op": {ID: "op"}})
	if err != nil || len(prepared.SuccessStatuses) != 1 || prepared.SuccessStatuses[0] != "200" {
		t.Fatalf("equivalent criteria should collapse to one status: %#v, %v", prepared, err)
	}
}

func TestPrepareWorkflowRequestBodyErrorsAndSuccess(t *testing.T) {
	contract, err := prepareYAML(t, workflowBodyOpenAPI, PrepareOptions{})
	if err != nil {
		t.Fatal(err)
	}
	operation := contract.Operations[0]
	workflowWithBody := func(body *higharazzo.RequestBody) *higharazzo.Workflow {
		return &higharazzo.Workflow{WorkflowId: "create", Steps: []*higharazzo.Step{{OperationId: "createThing", RequestBody: body, SuccessCriteria: []*higharazzo.Criterion{{Condition: "$statusCode == 201"}}}}}
	}
	tests := []struct {
		name string
		body *higharazzo.RequestBody
		want string
	}{
		{"replacement", &higharazzo.RequestBody{Replacements: []*higharazzo.PayloadReplacement{{}}}, "replacements"},
		{"empty content type", &higharazzo.RequestBody{}, "contentType is required"},
		{"missing media", &higharazzo.RequestBody{ContentType: "application/xml"}, "is not declared"},
		{"non-object payload", &higharazzo.RequestBody{ContentType: "application/json", Payload: &yaml.Node{Kind: yaml.SequenceNode}}, "must be an object"},
		{"odd mapping", &higharazzo.RequestBody{ContentType: "application/json", Payload: &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{{Kind: yaml.ScalarNode, Value: "id"}}}}, "must be an object"},
		{"non-scalar property", &higharazzo.RequestBody{ContentType: "application/json", Payload: mappingNode(&yaml.Node{Kind: yaml.MappingNode}, scalarNode("$inputs.id"))}, "properties must map directly"},
		{"bad expression", &higharazzo.RequestBody{ContentType: "application/json", Payload: mappingNode(scalarNode("id"), scalarNode("literal"))}, "unsupported payload expression"},
		{"empty input", &higharazzo.RequestBody{ContentType: "application/json", Payload: mappingNode(scalarNode("id"), scalarNode("$inputs."))}, "unsupported payload expression"},
		{"unknown property", &higharazzo.RequestBody{ContentType: "application/json", Payload: mappingNode(scalarNode("other"), scalarNode("$inputs.other"))}, `property "other" is not declared`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := prepareWorkflow(workflowWithBody(test.body), map[string]*Operation{"createThing": operation})
			assertErrorContains(t, err, test.want)
		})
	}
	payload := mappingNode(scalarNode("id"), scalarNode("$inputs.id"), scalarNode("note"), scalarNode("$inputs.note"))
	workflow, err := prepareWorkflow(workflowWithBody(&higharazzo.RequestBody{ContentType: "application/json", Payload: payload}), map[string]*Operation{"createThing": operation})
	if err != nil {
		t.Fatal(err)
	}
	if len(workflow.PayloadBindings) != 2 || len(workflow.SuccessStatuses) != 1 || workflow.SuccessStatuses[0] != "201" {
		t.Fatalf("workflow was not fully prepared: %#v", workflow)
	}
	nestedPayload := mappingNode(scalarNode("id"), scalarNode("$inputs.thing.id"))
	workflow, err = prepareWorkflow(workflowWithBody(&higharazzo.RequestBody{ContentType: "application/json", Payload: nestedPayload}), map[string]*Operation{"createThing": operation})
	if err != nil || workflow.PayloadBindings[0].Input != "thing.id" {
		t.Fatalf("runtime expression parser result was not preserved: %#v, %v", workflow, err)
	}
}

func TestPrepareWorkflowPreservesDeclaredInputs(t *testing.T) {
	inputs := mappingNode(
		scalarNode("type"), scalarNode("object"),
		scalarNode("properties"), mappingNode(scalarNode("id"), mappingNode(scalarNode("type"), scalarNode("string"))),
	)
	workflow, err := prepareWorkflow(&higharazzo.Workflow{
		WorkflowId: "declaredInputs",
		Inputs:     inputs,
		Steps: []*higharazzo.Step{{
			OperationId: "op", SuccessCriteria: []*higharazzo.Criterion{{Condition: "$statusCode == 200"}},
		}},
	}, map[string]*Operation{"op": {ID: "op"}})
	if err != nil {
		t.Fatal(err)
	}
	if workflow.Inputs != inputs {
		t.Fatal("prepared workflow did not borrow the declared inputs schema")
	}
}

func TestValidatePayloadSchemaErrors(t *testing.T) {
	assertErrorContains(t, validatePayloadBindings(nil, nil), "schema is required")
	assertErrorContains(t, collectPayloadShape(nil, map[string]struct{}{}, map[string]struct{}{}, map[*highbase.SchemaProxy]struct{}{}), "schema is required")
	assertErrorContains(t, collectPayloadShape(&highbase.SchemaProxy{}, map[string]struct{}{}, map[string]struct{}{}, map[*highbase.SchemaProxy]struct{}{}), "could not be resolved")
	if bindings, err := preparePayloadBindings(nil); err != nil || bindings != nil {
		t.Fatalf("nil payload should produce no bindings: %#v, %v", bindings, err)
	}

	contract, err := prepareYAML(t, `openapi: 3.1.0
info: {title: schemas, version: 1.0.0}
paths: {}
components:
  schemas:
    Union: {oneOf: [{type: string}, {type: integer}]}
    Choice: {anyOf: [{type: string}, {type: integer}]}
`, PrepareOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Union", "Choice"} {
		assertErrorContains(t, validatePayloadBindings(nil, contract.Schemas.GetOrZero(name)), "not supported")
	}

	bodyContract, err := prepareYAML(t, workflowBodyOpenAPI, PrepareOptions{})
	if err != nil {
		t.Fatal(err)
	}
	bodySchema := bodyContract.Operations[0].RequestBody.Content.GetOrZero("application/json").Schema
	assertErrorContains(t, validatePayloadBindings([]*PayloadBinding{nil}, bodySchema), `does not bind required OpenAPI property "id"`)
	bodySchema.Schema().AllOf = append(bodySchema.Schema().AllOf, nil)
	assertErrorContains(t, validatePayloadBindings(nil, bodySchema), "schema is required")

	proxy := contract.Schemas.GetOrZero("Union")
	properties, required := map[string]struct{}{}, map[string]struct{}{}
	visited := map[*highbase.SchemaProxy]struct{}{proxy: {}}
	if err := collectPayloadShape(proxy, properties, required, visited); err != nil {
		t.Fatalf("already visited schema should be ignored: %v", err)
	}
	if err := validatePayloadBindings([]*PayloadBinding{nil}, contract.Schemas.GetOrZero("Choice")); err == nil {
		t.Fatal("choice schema should still be rejected after nil binding is ignored")
	}
}

func TestPrepareWorkflowRejectsUnresolvableMediaSchema(t *testing.T) {
	content := orderedmap.New[string, *highv3.MediaType]()
	content.Set("application/json", &highv3.MediaType{})
	operation := &Operation{ID: "op", RequestBody: &RequestBody{Content: content}}
	workflow := &higharazzo.Workflow{WorkflowId: "workflow", Steps: []*higharazzo.Step{{
		OperationId:     "op",
		RequestBody:     &higharazzo.RequestBody{ContentType: "application/json"},
		SuccessCriteria: []*higharazzo.Criterion{{Condition: "$statusCode == 200"}},
	}}}
	_, err := prepareWorkflow(workflow, map[string]*Operation{"op": operation})
	assertErrorContains(t, err, "has no resolvable schema")
}

func prepareYAML(t *testing.T, source string, options PrepareOptions) (*Contract, error) {
	t.Helper()
	document, err := libopenapi.NewDocument([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	model, err := document.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	return Prepare(&model.Model, options)
}

func prepareYAMLContract(t *testing.T, source string, options PrepareOptions) error {
	t.Helper()
	_, err := prepareYAML(t, source, options)
	return err
}

func prepareContract(document *highv3.Document, options PrepareOptions) error {
	_, err := Prepare(document, options)
	return err
}

func prepareArazzo(document *higharazzo.Arazzo, contract *Contract, ids ...string) error {
	_, err := PrepareArazzo(document, contract, ids...)
	return err
}

func assertErrorContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got %v", want, err)
	}
}

func scalarNode(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Value: value}
}

func mappingNode(nodes ...*yaml.Node) *yaml.Node {
	return &yaml.Node{Kind: yaml.MappingNode, Content: nodes}
}

const minimalOpenAPI = `openapi: 3.1.0
info: {title: minimal, version: 1.0.0}
paths:
  /health:
    get:
      operationId: health
      responses: {'204': {description: ok}}
`
