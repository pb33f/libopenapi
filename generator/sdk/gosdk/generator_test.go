// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package gosdk

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	highv3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	modelgen "github.com/pb33f/libopenapi/generator/golang"
	"github.com/pb33f/libopenapi/generator/sdk"
	"github.com/pb33f/libopenapi/orderedmap"
)

func TestGeneratedSDKExecutesResourceOperations(t *testing.T) {
	model := buildFixture(t)
	contract, err := sdk.Prepare(&model.Model, sdk.PrepareOptions{
		Names: map[string]sdk.OperationName{
			"createJob":       {Resource: "Jobs", Method: "Create"},
			"inspectSegments": {Resource: "Segments", Method: "Inspect"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	arazzoDocument, err := libopenapi.NewArazzoDocument([]byte(startJobArazzo))
	if err != nil {
		t.Fatal(err)
	}
	workflows, err := sdk.PrepareArazzo(arazzoDocument, contract, "startJob")
	if err != nil {
		t.Fatal(err)
	}
	options := Options{
		PackageName: "example",
		Workflows:   workflows,
	}
	first, err := GenerateContract(contract, options)
	if err != nil {
		t.Fatal(err)
	}
	second, err := GenerateContract(contract, options)
	if err != nil {
		t.Fatal(err)
	}
	if !equalFiles(first.Files, second.Files) {
		t.Fatal("generation is not deterministic")
	}
	clientSource := generatedFile(t, first.Files, "client.gen.go")
	resourcesSource := generatedFile(t, first.Files, "resources.gen.go")
	for _, obsolete := range []string{`"reflect"`, "reflect.", "statusMatches(", "append([]byte(nil), payload...)"} {
		if strings.Contains(clientSource+resourcesSource, obsolete) {
			t.Fatalf("generated source contains obsolete runtime machinery %q", obsolete)
		}
	}
	for _, expected := range []string{
		"buffer.Grow(length + bytes.MinRead)",
		"target := request",
		"var securityWidgetsList = [][]securityRequirement",
		"response.StatusCode == 201",
		`input.Body, "application/json"`,
	} {
		if !strings.Contains(clientSource+resourcesSource, expected) {
			t.Fatalf("generated source does not contain %q", expected)
		}
	}
	if strings.Contains(resourcesSource, "var body any") {
		t.Fatal("generated required request bodies use an unnecessary interface variable")
	}
	directory := t.TempDir()
	for _, file := range first.Files {
		if err := os.WriteFile(filepath.Join(directory, file.Path), file.Content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(directory, "go.mod"), []byte("module generated.example/sdk\n\ngo 1.25.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "client_test.go"), []byte(generatedRuntimeTest), 0o600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "test", "-race", "./...")
	command.Dir = directory
	command.Env = append(os.Environ(), "GOWORK=off")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("generated SDK did not compile and execute: %v\n%s", err, output)
	}
}

func TestGenerateRejectsUnsupportedSelectedWireBehavior(t *testing.T) {
	document, err := libopenapi.NewDocument([]byte(unsupportedFixture))
	if err != nil {
		t.Fatal(err)
	}
	model, err := document.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	_, err = Generate(&model.Model, Options{Prepare: sdk.PrepareOptions{Operations: []string{"upload"}}})
	if err == nil || !strings.Contains(err.Error(), "unsupported media types multipart/form-data") {
		t.Fatalf("expected source operation failure, got %v", err)
	}
}

func TestGeneratePreservesModelDiagnosticFields(t *testing.T) {
	document, err := libopenapi.NewDocument([]byte(diagnosticFixture))
	if err != nil {
		t.Fatal(err)
	}
	model, err := document.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	result, err := Generate(&model.Model, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Diagnostics) != 1 {
		t.Fatalf("expected one diagnostic, got %#v", result.Diagnostics)
	}
	diagnostic := result.Diagnostics[0]
	if diagnostic.Code != modelgen.DiagnosticValidationKeyword || diagnostic.Path == "" || diagnostic.Message == "" || strings.Contains(diagnostic.Message, diagnostic.Code) {
		t.Fatalf("diagnostic fields were not preserved: %#v", diagnostic)
	}
}

func TestGenerateReservesComponentNamesBeforeCollectingInlineSchemas(t *testing.T) {
	for _, fixture := range []string{schemaCollisionInlineFirst, schemaCollisionComponentFirst} {
		document, err := libopenapi.NewDocument([]byte(fixture))
		if err != nil {
			t.Fatal(err)
		}
		model, err := document.BuildV3Model()
		if err != nil {
			t.Fatal(err)
		}
		result, err := Generate(&model.Model, Options{PackageName: "collision"})
		if err != nil {
			t.Fatal(err)
		}
		models := string(result.Files[0].Content)
		if !strings.Contains(models, "type FooRequest struct") || !strings.Contains(models, "ComponentValue string") ||
			!strings.Contains(models, "type FooRequest2 struct") || !strings.Contains(models, "InlineValue string") {
			t.Fatalf("component and inline schemas were not generated distinctly:\n%s", models)
		}
	}
}

func TestGeneratedSDKCompilesWithFixedIdentifierComponentNames(t *testing.T) {
	components := orderedmap.New[string, *highbase.SchemaProxy]()
	properties := orderedmap.New[string, *highbase.SchemaProxy]()
	for _, name := range []string{"APIKey", "NewClient", "Null", "WithTimeout"} {
		components.Set(name, schema("string", ""))
		properties.Set(strings.ToLower(name), highbase.CreateSchemaProxyRef("#/components/schemas/"+name))
	}
	components.Set("Envelope", highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"object"}, Properties: properties}))
	operation := operationWithSuccess("read")
	operation.Responses = []*sdk.Response{{Status: "200", Content: jsonContent(highbase.CreateSchemaProxyRef("#/components/schemas/Envelope"))}}
	contract := contractWith(operation)
	contract.Schemas = components
	result, err := GenerateContract(contract, Options{PackageName: "conflicts"})
	if err != nil {
		t.Fatal(err)
	}
	compileGeneratedFiles(t, result.Files)
}

func TestGeneratedWorkflowAssignsPromotedAllOfFields(t *testing.T) {
	components := orderedmap.New[string, *highbase.SchemaProxy]()
	components.Set("BaseRequest", highbase.CreateSchemaProxy(&highbase.Schema{
		Type: []string{"object"}, Required: []string{"name"}, Properties: schemaMap("name", schema("string", "")),
	}))
	components.Set("ExtendedRequest", highbase.CreateSchemaProxy(&highbase.Schema{
		AllOf: []*highbase.SchemaProxy{highbase.CreateSchemaProxyRef("#/components/schemas/BaseRequest")},
	}))
	operation := operationWithSuccess("create")
	operation.RequestBody = jsonBody(highbase.CreateSchemaProxyRef("#/components/schemas/ExtendedRequest"))
	contract := contractWith(operation)
	contract.Schemas = components
	workflow := &sdk.Workflow{
		ID: "createWorkflow", Operation: operation, Inputs: workflowInputs(t, `type: object
required: [name]
properties: {name: {type: string}}`), SuccessStatuses: []string{"204"},
		PayloadBindings: []*sdk.PayloadBinding{{Property: "name", Input: "name"}},
	}
	result, err := GenerateContract(contract, Options{PackageName: "promoted", Workflows: []*sdk.Workflow{workflow}})
	if err != nil {
		t.Fatal(err)
	}
	workflowSource := generatedFile(t, result.Files, "workflows.gen.go")
	if !strings.Contains(workflowSource, "body.Name = input.Name") {
		t.Fatalf("promoted field was not assigned after body construction:\n%s", workflowSource)
	}
	compileGeneratedFiles(t, result.Files)
}

func compileGeneratedFiles(t *testing.T, files []sdk.File) {
	t.Helper()
	directory := t.TempDir()
	for _, file := range files {
		if err := os.WriteFile(filepath.Join(directory, file.Path), file.Content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(directory, "go.mod"), []byte("module generated.example/compile\n\ngo 1.25.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "test", "./...")
	command.Dir = directory
	command.Env = append(os.Environ(), "GOWORK=off")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("generated SDK did not compile: %v\n%s", err, output)
	}
}

func TestGenerateRejectsMixedEffectiveServers(t *testing.T) {
	for _, paths := range []string{
		"  /ordinary:\n    get: {operationId: ordinary, responses: {\"204\": {description: done}}}\n  /special:\n    get:\n      operationId: special\n      servers: [{url: https://special.example.test/v1}]\n      responses: {\"204\": {description: done}}",
		"  /special:\n    get:\n      operationId: special\n      servers: [{url: https://special.example.test/v1}]\n      responses: {\"204\": {description: done}}\n  /ordinary:\n    get: {operationId: ordinary, responses: {\"204\": {description: done}}}",
	} {
		fixture := "openapi: 3.1.0\ninfo: {title: Servers, version: 1.0.0}\npaths:\n" + paths + "\n"
		document, err := libopenapi.NewDocument([]byte(fixture))
		if err != nil {
			t.Fatal(err)
		}
		model, err := document.BuildV3Model()
		if err != nil {
			t.Fatal(err)
		}
		_, err = Generate(&model.Model, Options{})
		if err == nil || !strings.Contains(err.Error(), "operation-specific server") {
			t.Fatalf("expected incompatible server error, got %v", err)
		}
	}
}

func TestGenerateUsesLoneOperationServerAsDefault(t *testing.T) {
	document, err := libopenapi.NewDocument([]byte(`openapi: 3.1.0
info: {title: Servers, version: 1.0.0}
paths:
  /special:
    get:
      operationId: special
      servers: [{url: https://special.example.test/v1}]
      responses: {"204": {description: done}}
`))
	if err != nil {
		t.Fatal(err)
	}
	model, err := document.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	result, err := Generate(&model.Model, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(result.Files[1].Content), `const DefaultServer = "https://special.example.test/v1"`) {
		t.Fatalf("operation server did not become the default:\n%s", result.Files[1].Content)
	}
}

func TestGenerateResolvesServerVariableDefaults(t *testing.T) {
	document, err := libopenapi.NewDocument([]byte(`openapi: 3.1.0
info: {title: Servers, version: 1.0.0}
servers:
  - url: https://{region}.example.test/{version}
    variables:
      region: {default: us}
      version: {default: v1}
paths:
  /health:
    get: {operationId: health, responses: {"204": {description: done}}}
`))
	if err != nil {
		t.Fatal(err)
	}
	model, err := document.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	result, err := Generate(&model.Model, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(result.Files[1].Content), `const DefaultServer = "https://us.example.test/v1"`) {
		t.Fatalf("server defaults were not resolved:\n%s", result.Files[1].Content)
	}
}

func TestGenerateRejectsMixedAndObjectParameterTypes(t *testing.T) {
	for _, schema := range []string{
		`{type: [string, object]}`,
		`{type: [array, object], items: {type: string}}`,
		`{type: array, items: {type: [string, object]}}`,
	} {
		fixture := fmt.Sprintf(`openapi: 3.1.0
info: {title: Parameters, version: 1.0.0}
paths:
  /search:
    get:
      operationId: search
      parameters:
        - {name: filter, in: query, schema: %s}
      responses: {"204": {description: done}}
`, schema)
		document, err := libopenapi.NewDocument([]byte(fixture))
		if err != nil {
			t.Fatal(err)
		}
		model, err := document.BuildV3Model()
		if err != nil {
			t.Fatal(err)
		}
		_, err = Generate(&model.Model, Options{})
		if err == nil || !strings.Contains(err.Error(), "unsupported object or tuple serialization") {
			t.Fatalf("expected parameter rejection for %s, got %v", schema, err)
		}
	}
}

func buildFixture(t *testing.T) *libopenapi.DocumentModel[highv3.Document] {
	t.Helper()
	document, err := libopenapi.NewDocument([]byte(sdkFixture))
	if err != nil {
		t.Fatal(err)
	}
	model, err := document.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func equalFiles(left, right []sdk.File) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i].Path != right[i].Path || !bytes.Equal(left[i].Content, right[i].Content) {
			return false
		}
	}
	return true
}

func generatedFile(t *testing.T, files []sdk.File, path string) string {
	t.Helper()
	for _, file := range files {
		if file.Path == path {
			return string(file.Content)
		}
	}
	t.Fatalf("generated file %s was not found", path)
	return ""
}

const sdkFixture = `openapi: 3.1.0
info: {title: Example, version: 1.0.0}
servers: [{url: https://api.example.test}]
security:
  - bearerAuth: []
    tenantKey: []
  - serviceKey: []
  - oauth: [widgets:read]
paths:
  /tenants/{tenantId}/widgets:
    parameters:
      - &tenantId
        name: tenantId
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
        - name: labels
          in: query
          explode: true
          schema: {type: array, items: {type: string}}
        - name: codes
          in: query
          explode: false
          schema: {type: array, items: {type: integer}}
        - name: active
          in: query
          schema: {type: boolean}
        - name: ratio
          in: query
          schema: {type: number, format: float}
        - name: priority
          in: header
          schema: {type: integer, format: int32}
      responses:
        "2XX":
          description: widgets
          content:
            application/json: {schema: {$ref: "#/components/schemas/WidgetPage"}}
        "204": {description: no widgets}
        "400":
          description: bad request
          content:
            application/json: {schema: {$ref: "#/components/schemas/Problem"}}
  /tenants/{tenantId}/widgets/{widgetId}:
    get:
      operationId: getWidget
      tags: [Widgets]
      parameters:
        - *tenantId
        - name: widgetId
          in: path
          required: true
          schema: {type: string}
      responses:
        "200":
          description: widget
          content:
            application/json: {schema: {$ref: "#/components/schemas/Widget"}}
  /tenants/{tenantId}/jobs:
    post:
      operationId: createJob
      tags: [Jobs]
      parameters: [*tenantId]
      requestBody:
        required: true
        content:
          application/json: {schema: {$ref: "#/components/schemas/JobStart"}}
      responses:
        "201":
          description: created
          content:
            application/json: {schema: {$ref: "#/components/schemas/Job"}}
  /tenants/{tenantId}/segments/{parts}:
    get:
      operationId: inspectSegments
      tags: [Segments]
      parameters:
        - *tenantId
        - name: parts
          in: path
          required: true
          style: simple
          schema: {type: array, items: {type: string}}
      responses:
        "204": {description: inspected}
  /tenants/{tenantId}/settings/{settingId}:
    patch:
      operationId: patchSetting
      tags: [Settings]
      parameters:
        - *tenantId
        - name: settingId
          in: path
          required: true
          schema: {type: string}
      requestBody:
        required: true
        content:
          application/json: {schema: {$ref: "#/components/schemas/SettingPatch"}}
      responses:
        "200":
          description: setting
          content:
            application/json: {schema: {$ref: "#/components/schemas/Setting"}}
components:
  securitySchemes:
    bearerAuth: {type: http, scheme: bearer}
    tenantKey: {type: apiKey, in: header, name: X-Tenant-Key}
    serviceKey: {type: apiKey, in: query, name: api_key}
    oauth:
      type: oauth2
      flows:
        clientCredentials:
          tokenUrl: https://auth.example.test/token
          scopes: {widgets:read: Read widgets}
  schemas:
    Widget:
      type: object
      required: [id]
      properties: {id: {type: string}}
    WidgetPage:
      type: object
      required: [items]
      properties:
        items: {type: array, items: {$ref: "#/components/schemas/Widget"}}
    Problem:
      type: object
      properties: {detail: {type: string}}
    JobStart:
      type: object
      required: [taskId, idempotencyKey]
      properties:
        taskId: {type: string}
        idempotencyKey: {type: string}
        inputs: {type: object, additionalProperties: true}
    Job:
      type: object
      required: [id]
      properties: {id: {type: string}}
    SettingPatch:
      type: object
      properties:
        note: {type: [string, "null"]}
        enabled: {type: boolean}
    Setting:
      type: object
      required: [id]
      properties: {id: {type: string}}
`

const unsupportedFixture = `openapi: 3.1.0
info: {title: Unsupported, version: 1.0.0}
paths:
  /upload:
    post:
      operationId: upload
      requestBody:
        content:
          multipart/form-data:
            schema: {type: object}
      responses:
        "204": {description: done}
`

const diagnosticFixture = `openapi: 3.1.0
info: {title: Diagnostic, version: 1.0.0}
paths:
  /widgets:
    get:
      operationId: listWidgets
      responses:
        "200":
          description: widgets
          content:
            application/json:
              schema:
                type: array
                items: {$ref: "#/components/schemas/Widget"}
components:
  schemas:
    Widget:
      type: string
      minLength: 1
`

const schemaCollisionInlineFirst = `openapi: 3.1.0
info: {title: Collision, version: 1.0.0}
paths:
  /inline:
    post:
      operationId: foo
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [inlineValue]
              properties: {inlineValue: {type: string}}
      responses: {"204": {description: done}}
  /component:
    get:
      operationId: getComponent
      responses:
        "200":
          description: component
          content:
            application/json: {schema: {$ref: "#/components/schemas/fooRequest"}}
components:
  schemas:
    fooRequest:
      type: object
      required: [componentValue]
      properties: {componentValue: {type: string}}
`

const schemaCollisionComponentFirst = `openapi: 3.1.0
info: {title: Collision, version: 1.0.0}
paths:
  /component:
    get:
      operationId: getComponent
      responses:
        "200":
          description: component
          content:
            application/json: {schema: {$ref: "#/components/schemas/fooRequest"}}
  /inline:
    post:
      operationId: foo
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [inlineValue]
              properties: {inlineValue: {type: string}}
      responses: {"204": {description: done}}
components:
  schemas:
    fooRequest:
      type: object
      required: [componentValue]
      properties: {componentValue: {type: string}}
`

const startJobArazzo = `arazzo: 1.0.1
info: {title: Start job, version: 1.0.0}
sourceDescriptions:
  - {name: api, url: https://example.test/openapi.yaml, type: openapi}
workflows:
  - workflowId: startJob
    summary: Start exactly one durable job.
    inputs:
      type: object
      required: [tenant, taskId, idempotencyKey]
      properties:
        tenant: {type: string}
        taskId: {type: string}
        idempotencyKey: {type: string}
        inputs: {type: object, additionalProperties: true}
    steps:
      - stepId: startRun
        operationId: createJob
        parameters:
          - {name: tenantId, in: path, value: $inputs.tenant}
        requestBody:
          contentType: application/json
          payload:
            taskId: $inputs.taskId
            idempotencyKey: $inputs.idempotencyKey
            inputs: $inputs.inputs
        successCriteria:
          - condition: $statusCode == 201
`

const generatedRuntimeTest = `package example

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestGeneratedClient(t *testing.T) {
	var mu sync.Mutex
	var patches []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.EscapedPath(), "/v1/") {
			http.Error(w, r.URL.EscapedPath(), http.StatusBadRequest)
			return
		}
		personal := r.Header.Get("Authorization") == "Bearer token" && r.Header.Get("X-Tenant-Key") == "tenant"
		oauth := r.Header.Get("Authorization") == "Bearer scoped"
		service := r.URL.Query().Get("api_key") == "service"
		if !personal && !service && !oauth {
			http.Error(w, "missing credentials", http.StatusUnauthorized)
			return
		}
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.EscapedPath(), "/widgets"):
			if requestID := r.Header.Get("X-Request-ID"); requestID != "" { w.Header().Set("X-Request-ID", requestID) }
			if r.URL.Query().Get("limit") == "2" {
				if got := r.URL.Query()["labels"]; len(got) != 2 || got[0] != "red" || got[1] != "blue" { http.Error(w, fmt.Sprint(got), http.StatusBadRequest); return }
				if r.URL.Query().Get("codes") != "3,5" || r.URL.Query().Get("active") != "false" || r.URL.Query().Get("ratio") != "1.5" || r.Header.Get("priority") != "7" {
					http.Error(w, r.URL.RawQuery+" priority="+r.Header.Get("priority"), http.StatusBadRequest)
					return
				}
			}
			if r.URL.Query().Get("limit") == "99" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, ` + "`" + `{"detail":"bad limit"}` + "`" + `)
				return
			}
			if r.URL.Query().Get("limit") == "98" {
				w.Header().Set("X-Error", "malformed")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, "{")
				return
			}
			if r.URL.Query().Get("limit") == "97" {
				w.Header().Set("X-Correlation-ID", "broken-success")
				_, _ = io.WriteString(w, "{")
				return
			}
			if r.URL.Query().Get("limit") == "96" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			if r.URL.Query().Get("limit") == "95" {
				w.Header().Set("X-Correlation-ID", "empty-success")
				return
			}
			w.Header().Set("X-Page", "one")
			_, _ = io.WriteString(w, ` + "`" + `{"items":[{"id":"a-1"}]}` + "`" + `)
		case r.Method == http.MethodGet && strings.Contains(r.URL.EscapedPath(), "/widgets/"):
			if !strings.HasSuffix(r.URL.EscapedPath(), "/widgets/widget%2F1") {
				http.Error(w, r.URL.EscapedPath(), http.StatusBadRequest)
				return
			}
			_, _ = io.WriteString(w, ` + "`" + `{"id":"widget/1"}` + "`" + `)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/jobs"):
			if !strings.HasSuffix(r.URL.EscapedPath(), "/tenants/tenant-1/jobs") {
				http.Error(w, r.URL.EscapedPath(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, ` + "`" + `{"id":"run-1"}` + "`" + `)
		case r.Method == http.MethodGet && strings.Contains(r.URL.EscapedPath(), "/segments/"):
			if !strings.HasSuffix(r.URL.EscapedPath(), "/segments/alpha%2Fone,beta%20two") {
				http.Error(w, r.URL.EscapedPath(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "/settings/"):
			body, _ := io.ReadAll(r.Body)
			mu.Lock()
			patches = append(patches, string(body))
			mu.Unlock()
			_, _ = io.WriteString(w, ` + "`" + `{"id":"setting-1"}` + "`" + `)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL+"/v1",
		WithCredential("bearerAuth", BearerToken("token")),
		WithCredential("tenantKey", APIKey("tenant")),
	)
	if err != nil { t.Fatal(err) }
	incomplete, err := NewClient(server.URL+"/v1", WithCredential("bearerAuth", BearerToken("token")))
	if err != nil { t.Fatal(err) }
	if _, err := incomplete.Widgets.List(context.Background(), &ListWidgetsParams{TenantID: "tenant-1"}); err == nil || !strings.Contains(err.Error(), "no complete credential alternative") {
		t.Fatalf("expected incomplete AND credentials to fail, got %v", err)
	}
	serviceClient, err := NewClient(server.URL+"/v1", WithCredential("serviceKey", APIKey("service")))
	if err != nil { t.Fatal(err) }
	if _, err := serviceClient.Widgets.List(context.Background(), &ListWidgetsParams{TenantID: "tenant-1"}); err != nil {
		t.Fatalf("OR credential alternative failed: %v", err)
	}
	missingScopeClient, err := NewClient(server.URL+"/v1", WithCredential("oauth", BearerToken("scoped")))
	if err != nil { t.Fatal(err) }
	if _, err := missingScopeClient.Widgets.List(context.Background(), &ListWidgetsParams{TenantID: "tenant-1"}); err == nil || !strings.Contains(err.Error(), "widgets:read") {
		t.Fatalf("expected missing OAuth scope failure, got %v", err)
	}
	scopedClient, err := NewClient(server.URL+"/v1", WithCredential("oauth", BearerToken("scoped", "widgets:read")))
	if err != nil { t.Fatal(err) }
	if _, err := scopedClient.Widgets.List(context.Background(), &ListWidgetsParams{TenantID: "tenant-1"}); err != nil {
		t.Fatalf("scoped OAuth credential failed: %v", err)
	}
	limit := 2
	labels := []string{"red", "blue"}
	codes := []int{3, 5}
	active := false
	ratio := float32(1.5)
	priority := int32(7)
	page, err := client.Widgets.List(context.Background(), &ListWidgetsParams{
		TenantID: "tenant-1", Limit: &limit, Labels: &labels, Codes: &codes,
		Active: &active, Ratio: &ratio, Priority: &priority,
	})
	if err != nil { t.Fatal(err) }
	if page.Value.Items[0].ID != "a-1" || page.Header.Get("X-Page") != "one" { t.Fatalf("unexpected page: %#v", page) }
	widget, err := client.Widgets.GetWidget(context.Background(), &GetWidgetParams{TenantID: "tenant-1", WidgetID: "widget/1"})
	if err != nil || widget.Value.ID != "widget/1" { t.Fatalf("unexpected widget: %#v %v", widget, err) }
	segments, err := client.Segments.Inspect(context.Background(), &InspectSegmentsParams{TenantID: "tenant-1", Parts: []string{"alpha/one", "beta two"}})
	if err != nil || segments.StatusCode != http.StatusNoContent { t.Fatalf("unexpected segments response: %#v %v", segments, err) }
	run, err := client.StartJob(context.Background(), &StartJobInput{Tenant: "tenant-1", TaskID: "task-1", IdempotencyKey: "i-1"})
	if err != nil || run.StatusCode != http.StatusCreated || run.Value.ID != "run-1" { t.Fatalf("unexpected run: %#v %v", run, err) }
	_, err = client.Settings.PatchSetting(context.Background(), &PatchSettingParams{TenantID: "tenant-1", SettingID: "setting-1", Body: SettingPatch{}})
	if err != nil { t.Fatal(err) }
	_, err = client.Settings.PatchSetting(context.Background(), &PatchSettingParams{TenantID: "tenant-1", SettingID: "setting-1", Body: SettingPatch{Note: Null[string]()}})
	if err != nil { t.Fatal(err) }
	disabled := false
	_, err = client.Settings.PatchSetting(context.Background(), &PatchSettingParams{TenantID: "tenant-1", SettingID: "setting-1", Body: SettingPatch{Enabled: &disabled}})
	if err != nil { t.Fatal(err) }
	mu.Lock()
	gotPatches := append([]string(nil), patches...)
	mu.Unlock()
	wantPatches := []string{"{}", ` + "`" + `{"note":null}` + "`" + `, ` + "`" + `{"enabled":false}` + "`" + `}
	if len(gotPatches) != len(wantPatches) { t.Fatalf("patches: %#v", gotPatches) }
	for i := range wantPatches { if gotPatches[i] != wantPatches[i] { t.Fatalf("patch %d: got %s want %s", i, gotPatches[i], wantPatches[i]) } }
	bad := 99
	_, err = client.Widgets.List(context.Background(), &ListWidgetsParams{TenantID: "tenant-1", Limit: &bad})
	var apiError *APIError
	if !errors.As(err, &apiError) { t.Fatalf("expected APIError, got %v", err) }
	problem, ok := apiError.Value.(*Problem)
	if !ok || problem.Detail == nil || *problem.Detail != "bad limit" { t.Fatalf("unexpected typed error: %#v", apiError.Value) }
	malformed := 98
	_, err = client.Widgets.List(context.Background(), &ListWidgetsParams{TenantID: "tenant-1", Limit: &malformed})
	if !errors.As(err, &apiError) || apiError.StatusCode != http.StatusBadRequest || apiError.Header.Get("X-Error") != "malformed" || string(apiError.Body) != "{" || errors.Unwrap(apiError) == nil {
		t.Fatalf("malformed declared error lost response context: %#v %v", apiError, err)
	}
	malformedSuccess := 97
	_, err = client.Widgets.List(context.Background(), &ListWidgetsParams{TenantID: "tenant-1", Limit: &malformedSuccess})
	var decodeError *ResponseDecodeError
	if !errors.As(err, &decodeError) || decodeError.StatusCode != http.StatusOK || decodeError.Header.Get("X-Correlation-ID") != "broken-success" || string(decodeError.Body) != "{" || errors.Unwrap(decodeError) == nil {
		t.Fatalf("malformed success lost response context: %#v %v", decodeError, err)
	}
	bodylessSuccess := 96
	response, err := client.Widgets.List(context.Background(), &ListWidgetsParams{TenantID: "tenant-1", Limit: &bodylessSuccess})
	if err != nil || response.StatusCode != http.StatusNoContent {
		t.Fatalf("bodyless declared success failed: %#v %v", response, err)
	}
	emptySuccess := 95
	_, err = client.Widgets.List(context.Background(), &ListWidgetsParams{TenantID: "tenant-1", Limit: &emptySuccess})
	if !errors.As(err, &decodeError) || decodeError.StatusCode != http.StatusOK || decodeError.Header.Get("X-Correlation-ID") != "empty-success" || len(decodeError.Body) != 0 || errors.Unwrap(decodeError) == nil {
		t.Fatalf("empty success was not rejected with response context: %#v %v", decodeError, err)
	}

	for _, status := range []int{http.StatusTemporaryRedirect, http.StatusPermanentRedirect} {
		var followed atomic.Int32
		target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			followed.Add(1)
			_, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
		}))
		redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, target.URL+"/captured", status)
		}))
		redirectClient, clientErr := NewClient(redirect.URL+"/v1", WithCredential("serviceKey", APIKey("service")))
		if clientErr != nil { t.Fatal(clientErr) }
		_, requestErr := redirectClient.StartJob(context.Background(), &StartJobInput{Tenant: "tenant-1", TaskID: "task-1", IdempotencyKey: "i-1"})
		redirect.Close()
		target.Close()
		if requestErr == nil { t.Fatalf("expected %d redirect response to remain an API error", status) }
		if followed.Load() != 0 { t.Fatalf("cross-origin %d redirect replayed the request body", status) }
	}

	var wait sync.WaitGroup
	errorsSeen := make(chan error, 16)
	for index := 0; index < 16; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			requestID := fmt.Sprintf("request-%d", index)
			response, err := client.Widgets.List(context.Background(), &ListWidgetsParams{TenantID: "tenant-1"}, WithRequestHeader("X-Request-ID", requestID))
			if err != nil { errorsSeen <- err; return }
			if response.Header.Get("X-Request-ID") != requestID { errorsSeen <- fmt.Errorf("header crossed requests: got %q want %q", response.Header.Get("X-Request-ID"), requestID) }
		}(index)
	}
	wait.Wait()
	close(errorsSeen)
	for err := range errorsSeen { t.Error(err) }
}

type doerFunc func(*http.Request) (*http.Response, error)

func (do doerFunc) Do(request *http.Request) (*http.Response, error) { return do(request) }

func unavailableCredential() Credential {
	return CredentialFunc(func(context.Context, SecurityScheme, []string, *http.Request) error {
		return errors.New("tenant key unavailable")
	})
}

// A credential that fails part-way through one alternative must not leave its
// partial writes on the request the fallback alternative sends.
func TestFailedAlternativeDoesNotLeakIntoFallback(t *testing.T) {
	var authorization, query string
	recorder := doerFunc(func(request *http.Request) (*http.Response, error) {
		authorization, query = request.Header.Get("Authorization"), request.URL.RawQuery
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("{}"))}, nil
	})
	client, err := NewClient("https://api.example.test/v1", WithHTTPClient(recorder),
		WithCredential("bearerAuth", BearerToken("token")),
		WithCredential("tenantKey", unavailableCredential()),
		WithCredential("serviceKey", APIKey("service")),
	)
	if err != nil { t.Fatal(err) }
	if _, err := client.Widgets.List(context.Background(), &ListWidgetsParams{TenantID: "tenant-1"}); err != nil {
		t.Fatalf("fallback credential alternative failed: %v", err)
	}
	if authorization != "" || query != "api_key=service" {
		t.Fatalf("failed alternative leaked into the fallback request: Authorization=%q query=%q", authorization, query)
	}

	withoutFallback, err := NewClient("https://api.example.test/v1", WithHTTPClient(recorder),
		WithCredential("bearerAuth", BearerToken("token")),
		WithCredential("tenantKey", unavailableCredential()),
	)
	if err != nil { t.Fatal(err) }
	if _, err := withoutFallback.Widgets.List(context.Background(), &ListWidgetsParams{TenantID: "tenant-1"}); err == nil || !strings.Contains(err.Error(), "tenant key unavailable") {
		t.Fatalf("expected the credential failure to surface, got %v", err)
	}
}

// Content-Length is a preallocation hint and never a bound: the limit holds and
// the body arrives whole whether the length is declared, absent, or wrong.
func TestResponseBodyReadPaths(t *testing.T) {
	payload := "{\"items\":[]}"
	for _, test := range []struct {
		name          string
		contentLength int64
		body          string
		wantErr       string
	}{
		{name: "declared length", contentLength: int64(len(payload)), body: payload},
		{name: "unknown length", contentLength: -1, body: payload},
		{name: "understated length", contentLength: 2, body: payload},
		{name: "declared length over the limit", contentLength: 65, body: strings.Repeat(" ", 65), wantErr: "exceeds 64 bytes"},
		{name: "unknown length over the limit", contentLength: -1, body: strings.Repeat(" ", 65), wantErr: "exceeds 64 bytes"},
	} {
		t.Run(test.name, func(t *testing.T) {
			client, err := NewClient("https://api.example.test/v1", WithMaxResponseBody(64),
				WithCredential("serviceKey", APIKey("service")),
				WithHTTPClient(doerFunc(func(*http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: http.StatusOK, ContentLength: test.contentLength, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(test.body))}, nil
				})),
			)
			if err != nil { t.Fatal(err) }
			page, err := client.Widgets.List(context.Background(), &ListWidgetsParams{TenantID: "tenant-1"})
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) { t.Fatalf("expected %q, got %v", test.wantErr, err) }
				return
			}
			if err != nil || page.Value.Items == nil { t.Fatalf("body did not arrive whole: %#v %v", page, err) }
		})
	}
}
`
