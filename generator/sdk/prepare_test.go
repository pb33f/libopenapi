// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package sdk

import (
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
)

func TestPrepareInheritanceSelectionAndSecurity(t *testing.T) {
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
	if len(contract.Operations) != 1 {
		t.Fatalf("got %d operations", len(contract.Operations))
	}
	operation := contract.Operations[0]
	if operation.Resource != "Widgets" || operation.Name != "list" {
		t.Fatalf("unexpected public name: %s.%s", operation.Resource, operation.Name)
	}
	if len(operation.Parameters) != 2 || operation.Parameters[0].Name != "tenantId" || operation.Parameters[1].Name != "limit" {
		t.Fatalf("unexpected effective parameters: %#v", operation.Parameters)
	}
	if len(operation.Security) != 1 || len(operation.Security[0].Schemes) != 2 || operation.Security[0].Schemes[0].Name != "bearerAuth" {
		t.Fatalf("security AND requirement was not preserved: %#v", operation.Security)
	}
}

func TestPrepareExplicitEmptySecurityOverridesDocument(t *testing.T) {
	document, err := libopenapi.NewDocument([]byte(prepareFixture))
	if err != nil {
		t.Fatal(err)
	}
	model, err := document.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	contract, err := Prepare(&model.Model, PrepareOptions{Operations: []string{"health"}})
	if err != nil {
		t.Fatal(err)
	}
	if contract.Operations[0].Security == nil || len(contract.Operations[0].Security) != 0 {
		t.Fatalf("explicit empty security was not preserved: %#v", contract.Operations[0].Security)
	}
}

func TestPrepareRejectsUndeclaredServerVariable(t *testing.T) {
	document, err := libopenapi.NewDocument([]byte(`openapi: 3.1.0
info: {title: Server, version: 1.0.0}
servers: [{url: "https://{region}.example.test"}]
paths: {}
`))
	if err != nil {
		t.Fatal(err)
	}
	model, err := document.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	_, err = Prepare(&model.Model, PrepareOptions{})
	if err == nil || !strings.Contains(err.Error(), "undeclared or malformed variable") {
		t.Fatalf("expected undeclared server variable failure, got %v", err)
	}
}

const prepareFixture = `openapi: 3.1.0
info: {title: Example, version: 1.0.0}
servers: [{url: https://api.example.test}]
security:
  - bearerAuth: []
    tenantKey: []
paths:
  /tenants/{tenantId}/widgets:
    parameters:
      - name: tenantId
        in: path
        required: true
        schema: {type: string}
    get:
      operationId: listWidgets
      tags: [Widgets]
      parameters:
        - name: limit
          in: query
          schema: {type: integer}
      responses:
        "200": {description: ok}
  /health:
    get:
      operationId: health
      security: []
      responses:
        "204": {description: ok}
components:
  securitySchemes:
    bearerAuth: {type: http, scheme: bearer}
    tenantKey: {type: apiKey, in: header, name: X-Tenant-Key}
`
