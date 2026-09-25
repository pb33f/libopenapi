// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package sdk

import (
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi"
	higharazzo "github.com/pb33f/libopenapi/datamodel/high/arazzo"
)

func TestPrepareArazzoFiniteOperation(t *testing.T) {
	document, err := libopenapi.NewDocument([]byte(prepareFixture))
	if err != nil {
		t.Fatal(err)
	}
	model, err := document.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	contract, err := Prepare(&model.Model, PrepareOptions{Operations: []string{"listWidgets"}})
	if err != nil {
		t.Fatal(err)
	}
	arazzoDocument, err := libopenapi.NewArazzoDocument([]byte(`arazzo: 1.0.1
info: {title: list, version: 1.0.0}
sourceDescriptions:
  - {name: api, url: https://example.test/openapi.yaml, type: openapi}
workflows:
  - workflowId: listWorkflow
    inputs:
      type: object
      required: [tenantId]
      properties: {tenantId: {type: string}}
    steps:
      - stepId: list
        operationId: listWidgets
        parameters:
          - {name: tenantId, in: path, value: $inputs.tenantId}
        successCriteria:
          - condition: $statusCode == 200
`))
	if err != nil {
		t.Fatal(err)
	}
	workflows, err := PrepareArazzo(arazzoDocument, contract, "listWorkflow")
	if err != nil {
		t.Fatal(err)
	}
	if len(workflows) != 1 || workflows[0].Operation.ID != "listWidgets" || workflows[0].SuccessStatuses[0] != "200" {
		t.Fatalf("unexpected workflow: %#v", workflows)
	}
}

func TestPrepareArazzoRejectsMultipleSteps(t *testing.T) {
	document, err := libopenapi.NewArazzoDocument([]byte(`arazzo: 1.0.1
info: {title: invalid, version: 1.0.0}
sourceDescriptions: [{name: api, url: https://example.test/openapi.yaml, type: openapi}]
workflows:
  - workflowId: invalid
    steps:
      - {stepId: one, operationId: listWidgets}
      - {stepId: two, operationId: listWidgets}
`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = PrepareArazzo(document, &Contract{})
	if err == nil {
		t.Fatal("expected unsupported multi-step workflow error")
	}
}

func TestPrepareArazzoRejectsContradictoryStatusCriteria(t *testing.T) {
	document, err := libopenapi.NewDocument([]byte(prepareFixture))
	if err != nil {
		t.Fatal(err)
	}
	model, err := document.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	contract, err := Prepare(&model.Model, PrepareOptions{Operations: []string{"listWidgets"}})
	if err != nil {
		t.Fatal(err)
	}
	arazzoDocument, err := libopenapi.NewArazzoDocument([]byte(`arazzo: 1.0.1
info: {title: contradictory, version: 1.0.0}
sourceDescriptions: [{name: api, url: https://example.test/openapi.yaml, type: openapi}]
workflows:
  - workflowId: contradictory
    inputs:
      type: object
      required: [tenantId]
      properties: {tenantId: {type: string}}
    steps:
      - stepId: list
        operationId: listWidgets
        parameters:
          - {name: tenantId, in: path, value: $inputs.tenantId}
        successCriteria:
          - condition: $statusCode == 200
          - condition: $statusCode == 202
`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = PrepareArazzo(arazzoDocument, contract)
	if err == nil || !strings.Contains(err.Error(), "cannot both match") {
		t.Fatalf("expected contradictory status criteria failure, got %v", err)
	}
}

func TestPrepareParameterBindingsFollowArazzoStep(t *testing.T) {
	operation := &Operation{Parameters: []*Parameter{
		{Name: "tenantId", In: "path", Required: true},
		{Name: "limit", In: "query"},
	}}
	scalar := func(value string) *yaml.Node { return &yaml.Node{Kind: yaml.ScalarNode, Value: value} }
	valid := []*higharazzo.Parameter{{Name: "tenantId", In: "path", Value: scalar("$inputs.tenant")}}
	bindings, err := prepareParameterBindings(valid, operation)
	if err != nil || len(bindings) != 1 || bindings[0].Input != "tenant" {
		t.Fatalf("unexpected parameter bindings: %#v, %v", bindings, err)
	}
	tests := []struct {
		name       string
		parameters []*higharazzo.Parameter
		message    string
	}{
		{name: "missing required", message: "is not bound"},
		{name: "unknown target", parameters: []*higharazzo.Parameter{{Name: "other", In: "path", Value: scalar("$inputs.tenant")}}, message: "is not declared"},
		{name: "duplicate target", parameters: append(valid, valid[0]), message: "more than once"},
		{name: "nested input", parameters: []*higharazzo.Parameter{{Name: "tenantId", In: "path", Value: scalar("$inputs.tenant.id")}}, message: "unsupported step parameter expression"},
		{name: "reusable", parameters: []*higharazzo.Parameter{{Reference: "$components.parameters.tenant"}}, message: "direct named parameter bindings"},
		{name: "structured value", parameters: []*higharazzo.Parameter{{Name: "tenantId", In: "path", Value: &yaml.Node{Kind: yaml.MappingNode}}}, message: "must bind directly"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := prepareParameterBindings(test.parameters, operation)
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("expected %q, got %v", test.message, err)
			}
		})
	}
}

func TestPrepareArazzoValidatesRequestBodyContract(t *testing.T) {
	document, err := libopenapi.NewDocument([]byte(workflowBodyOpenAPI))
	if err != nil {
		t.Fatal(err)
	}
	model, err := document.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	contract, err := Prepare(&model.Model, PrepareOptions{})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		body    string
		message string
	}{
		{name: "content type", body: "contentType: application/xml\n          payload:\n            id: $inputs.id", message: `contentType "application/xml" is not declared`},
		{name: "required property", body: "contentType: application/json\n          payload:\n            note: $inputs.note", message: `does not bind required OpenAPI property "id"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			arazzo, parseErr := libopenapi.NewArazzoDocument([]byte("arazzo: 1.0.1\ninfo: {title: body, version: 1.0.0}\nsourceDescriptions: [{name: api, url: https://example.test/openapi.yaml, type: openapi}]\nworkflows:\n  - workflowId: create\n    steps:\n      - stepId: create\n        operationId: createThing\n        requestBody:\n          " + test.body + "\n        successCriteria:\n          - condition: $statusCode == 201\n"))
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			_, prepareErr := PrepareArazzo(arazzo, contract)
			if prepareErr == nil || !strings.Contains(prepareErr.Error(), test.message) {
				t.Fatalf("expected %q, got %v", test.message, prepareErr)
			}
		})
	}
}

const workflowBodyOpenAPI = `openapi: 3.1.0
info: {title: Workflow body, version: 1.0.0}
paths:
  /things:
    post:
      operationId: createThing
      requestBody:
        required: true
        content:
          application/json:
            schema:
              allOf:
                - $ref: "#/components/schemas/ThingIdentity"
                - type: object
                  properties:
                    note: {type: string}
      responses:
        "201": {description: created}
components:
  schemas:
    ThingIdentity:
      type: object
      required: [id]
      properties:
        id: {type: string}
`
