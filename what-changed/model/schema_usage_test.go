// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

// usageSpec builds a document whose operation and components are given as YAML
// fragments indented for their place in the document.
func usageSpec(operation, schemas string) string {
	return `openapi: 3.1.0
info: {title: Direction, version: 1.0.0}
paths:
  /widgets:
    post:
` + operation + `
components:
  schemas:
` + schemas
}

const (
	usageRequestOnly = `      requestBody:
        content:
          application/json:
            schema: {$ref: '#/components/schemas/Widget'}
      responses:
        '204': {description: ok}`
	usageResponseOnly = `      responses:
        '200':
          description: ok
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Widget'}`
	usageShared = `      requestBody:
        content:
          application/json:
            schema: {$ref: '#/components/schemas/Widget'}
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Widget'}`
	usageCallback = `      responses:
        '204': {description: ok}
      callbacks:
        done:
          '{$request.body#/url}':
            post:
              requestBody:
                content:
                  application/json:
                    schema: {$ref: '#/components/schemas/Widget'}
              responses:
                '204': {description: ok}`
	usageUnused = `      responses:
        '204': {description: ok}`
)

func usageWidget(required string) string {
	return `    Widget:
      type: object
      required: [` + required + `]
      properties:
        name: {type: string}
        id: {type: string}`
}

func compareUsageDocuments(t *testing.T, left, right string) *DocumentChanges {
	t.Helper()
	lInfo, err := datamodel.ExtractSpecInfo([]byte(left))
	require.NoError(t, err)
	rInfo, err := datamodel.ExtractSpecInfo([]byte(right))
	require.NoError(t, err)
	lDoc, err := v3.CreateDocumentFromConfig(lInfo, datamodel.NewDocumentConfiguration())
	require.NoError(t, err)
	rDoc, err := v3.CreateDocumentFromConfig(rInfo, datamodel.NewDocumentConfiguration())
	require.NoError(t, err)
	return CompareDocuments(lDoc, rDoc)
}

func TestCompareDocuments_SchemaRequiredDirection(t *testing.T) {
	cases := []struct {
		name, operation string
		old, next       string
		breaking        bool
	}{
		{"added in response schema", usageResponseOnly, "name", "name, id", false},
		{"removed in response schema", usageResponseOnly, "name, id", "name", true},
		{"added in request schema", usageRequestOnly, "name", "name, id", true},
		{"removed in request schema", usageRequestOnly, "name, id", "name", false},
		{"added in shared schema", usageShared, "name", "name, id", true},
		{"removed in shared schema", usageShared, "name, id", "name", true},
		{"added in callback schema", usageCallback, "name", "name, id", true},
		{"added in unused schema", usageUnused, "name", "name, id", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			changes := compareUsageDocuments(t,
				usageSpec(tc.operation, usageWidget(tc.old)),
				usageSpec(tc.operation, usageWidget(tc.next)))
			require.NotNil(t, changes)
			assert.Equal(t, 1, changes.TotalChanges())
			assert.Equal(t, tc.breaking, changes.TotalBreakingChanges() > 0)
		})
	}
}

func TestCompareDocuments_SchemaRequiredDirectionUnionOfDocuments(t *testing.T) {
	// The old document also sends Widget in a request, so the new required entry
	// can still break a client of the old contract.
	changes := compareUsageDocuments(t,
		usageSpec(usageShared, usageWidget("name")),
		usageSpec(usageResponseOnly, usageWidget("name, id")))
	require.NotNil(t, changes)
	assert.Positive(t, changes.TotalBreakingChanges())
}

func TestCompareDocuments_SchemaRequiredDirectionInlineAndNested(t *testing.T) {
	const operation = `      parameters:
        - name: filter
          in: query
          schema: {$ref: '#/components/schemas/Filter'}
      requestBody:
        $ref: '#/components/requestBodies/WidgetInput'
      responses:
        '200':
          $ref: '#/components/responses/WidgetOutput'
        '201':
          description: ok
          headers:
            X-Page: {schema: {$ref: '#/components/schemas/Page'}}
          content:
            application/json:
              schema:
                type: object
                required: [%s]
                properties:
                  name: {type: string}
                  id: {type: string}`
	const schemas = `    Filter: {type: object, required: [%s], properties: {name: {type: string}, id: {type: string}}}
    Page: {type: object, required: [%s], properties: {name: {type: string}, id: {type: string}}}
    Input: {type: object, properties: {part: {$ref: '#/components/schemas/Part'}}}
    Output: {type: object, properties: {part: {$ref: '#/components/schemas/Part'}, detail: {$ref: '#/components/schemas/Detail'}}}
    Part: {type: object, required: [%s], properties: {name: {type: string}, id: {type: string}}}
    Detail:
      type: object
      required: [%s]
      properties: {name: {type: string}, id: {type: string}}
  requestBodies:
    WidgetInput:
      content:
        application/json:
          schema: {$ref: '#/components/schemas/Input'}
  responses:
    WidgetOutput:
      description: ok
      content:
        application/json:
          schema: {$ref: '#/components/schemas/Output'}`
	build := func(inline, filter, page, part, detail string) string {
		return usageSpec(fillRequired(operation, inline), fillRequired(schemas, filter, page, part, detail))
	}
	cases := []struct {
		name          string
		old, next     [5]string
		breakingCount int
	}{
		{"inline response schema", [5]string{"name", "name", "name", "name", "name"}, [5]string{"name, id", "name", "name", "name", "name"}, 0},
		{"parameter schema", [5]string{"name", "name", "name", "name", "name"}, [5]string{"name", "name, id", "name", "name", "name"}, 1},
		{"response header schema", [5]string{"name", "name", "name", "name", "name"}, [5]string{"name", "name", "name, id", "name", "name"}, 0},
		{"schema reached from request and response", [5]string{"name", "name", "name", "name", "name"}, [5]string{"name", "name", "name", "name, id", "name"}, 1},
		{"schema reached through a response component", [5]string{"name", "name", "name", "name", "name"}, [5]string{"name", "name", "name", "name", "name, id"}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			changes := compareUsageDocuments(t,
				build(tc.old[0], tc.old[1], tc.old[2], tc.old[3], tc.old[4]),
				build(tc.next[0], tc.next[1], tc.next[2], tc.next[3], tc.next[4]))
			require.NotNil(t, changes)
			assert.Equal(t, 1, changes.TotalChanges())
			assert.Equal(t, tc.breakingCount, changes.TotalBreakingChanges())
		})
	}
}

// fillRequired places each value in the next %s of the template, in order.
func fillRequired(template string, values ...string) string {
	for _, v := range values {
		template = strings.Replace(template, "%s", v, 1)
	}
	return template
}

func TestCompareDocuments_SchemaRequiredDirectionUnknownUsage(t *testing.T) {
	// A $dynamicRef hides where schemas are used, so the configured rule applies.
	const operation = usageResponseOnly + `
      x-meta: {$dynamicRef: '#meta'}`
	changes := compareUsageDocuments(t,
		usageSpec(operation, usageWidget("name")),
		usageSpec(operation, usageWidget("name, id")))
	require.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
}

func TestCompareDocuments_SchemaRequiredDirectionForgetsUsage(t *testing.T) {
	compareUsageDocuments(t,
		usageSpec(usageResponseOnly, usageWidget("name")),
		usageSpec(usageResponseOnly, usageWidget("name, id")))
	schemaUsageMu.RLock()
	defer schemaUsageMu.RUnlock()
	assert.Empty(t, schemaUsageByIndex)
}

func TestCompareSchemas_RequiredWithoutDocumentUsage(t *testing.T) {
	// CompareSchemas on its own has no document, so it keeps the configured rule.
	l := buildSchemaProxyFromYAML(t, "type: object\nrequired: [name]")
	r := buildSchemaProxyFromYAML(t, "type: object\nrequired: [name, id]")
	changes := CompareSchemas(l, r)
	require.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
}

func TestRequiredChangeBreaking_RespectsConfiguredRule(t *testing.T) {
	defer ResetActiveBreakingRulesConfig()
	SetActiveBreakingRulesConfig(&BreakingRulesConfig{
		Schema: &SchemaRules{Required: rule(false, false, false)},
	})
	added, removed := requiredChangeBreaking(nil, nil)
	assert.False(t, added)
	assert.False(t, removed)

	SetActiveBreakingRulesConfig(&BreakingRulesConfig{
		Schema: &SchemaRules{Required: rule(true, false, true)},
	})
	added, removed = requiredChangeBreaking(nil, nil)
	assert.True(t, added)
	assert.True(t, removed)
}

func usageRoot(t *testing.T, raw string) *yaml.Node {
	t.Helper()
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(raw), &node))
	return &node
}

func usageAt(t *testing.T, usage *schemaUsage, root *yaml.Node, ref string) schemaDirection {
	t.Helper()
	require.NotNil(t, usage)
	w := &usageWalker{root: root.Content[0]}
	target := w.resolve(ref)
	require.NotNil(t, target, ref)
	return usage.nodes[target]
}

func TestBuildSchemaUsage_Directions(t *testing.T) {
	root := usageRoot(t, `openapi: 3.2.0
paths:
  /a:
    $ref: '#/components/pathItems/Shared'
  /b:
    parameters:
      - {name: id, in: path, schema: {$ref: '#/components/schemas/PathParam'}}
    summary: path level fields do not change the result
    get:
      responses:
        '200': {content: {application/json: {schema: {$ref: '#/components/schemas/Out'}}}}
      x-extra: {schema: {$ref: '#/components/schemas/Extra'}}
    additionalOperations:
      COPY:
        requestBody: {content: {application/json: {schema: {$ref: '#/components/schemas/In'}}}}
    servers: ~
  /c: not a path item
  /d:
    get: not an operation
  /e:
    get: {$ref: '#/x-operations/read'}
    put: {$ref: '#/paths/~1e/put'}
    post: {$ref: '#/x-operations/ping'}
x-operations:
  ping: {$ref: '#/x-operations/pong'}
  pong: {$ref: '#/x-operations/ping'}
  read:
    responses:
      '200': {content: {application/json: {schema: {$ref: '#/components/schemas/ByOperationRef'}}}}
webhooks:
  ping:
    post:
      requestBody: {content: {application/json: {schema: {$ref: '#/components/schemas/Hook'}}}}
components:
  pathItems:
    Shared:
      $ref: '#/components/pathItems/Loop'
    Loop:
      $ref: '#/components/pathItems/Shared'
      put:
        requestBody: {content: {application/json: {schema: {$ref: '#/components/schemas/Shared%7EIn'}}}}
  schemas:
    PathParam: {type: string}
    Out:
      type: object
      properties:
        pet: {$ref: '#/components/schemas/Pet'}
        tags: {type: array, items: [{$ref: '#/components/schemas/Out/properties/pet'}]}
    Pet:
      discriminator:
        propertyName: kind
        mapping: {cat: Cat, dog: '#/components/schemas/Dog'}
      x-alias: &anchor {type: object}
      x-again: *anchor
    Cat: {type: object}
    Dog: {type: object}
    In: {type: object}
    Shared~In: {type: object}
    Extra: {type: object}
    Hook: {type: object}
    Unused: {type: object}
    ByOperationRef: {type: object}
`)
	usage := buildSchemaUsage(root)
	assert.Equal(t, directionRequest, usageAt(t, usage, root, "#/components/schemas/PathParam"))
	assert.Equal(t, directionResponse, usageAt(t, usage, root, "#/components/schemas/Out"))
	assert.Equal(t, directionResponse, usageAt(t, usage, root, "#/components/schemas/Cat"))
	assert.Equal(t, directionResponse, usageAt(t, usage, root, "#/components/schemas/Dog"))
	assert.Equal(t, directionRequest, usageAt(t, usage, root, "#/components/schemas/In"))
	assert.Equal(t, directionRequest, usageAt(t, usage, root, "#/components/schemas/Shared~0In"))
	assert.Equal(t, directionBoth, usageAt(t, usage, root, "#/components/schemas/Extra"))
	assert.Equal(t, directionBoth, usageAt(t, usage, root, "#/components/schemas/Hook"))
	assert.Equal(t, schemaDirection(0), usageAt(t, usage, root, "#/components/schemas/Unused"))
	assert.Equal(t, directionResponse, usageAt(t, usage, root, "#/components/schemas/ByOperationRef"))
	assert.Equal(t, directionResponse, usageAt(t, usage, root, "#/components/schemas/Out/properties/tags/items/0"))
}

func TestBuildSchemaUsage_GivesUp(t *testing.T) {
	const spec = `openapi: 3.1.0
paths:
  /a:
    get:
      responses:
        '200': {content: {application/json: {schema: %s}}}
components:
  schemas:
    Widget: {type: object}
    Base:
      discriminator: {propertyName: kind, mapping: {gone: Missing}}
`
	for name, schema := range map[string]string{
		"external reference":       `{$ref: 'other.yaml#/Widget'}`,
		"missing reference":        `{$ref: '#/components/schemas/Missing'}`,
		"bad escape":               `{$ref: '#/components/schemas/%zz'}`,
		"bad array index":          `{$ref: '#/components/schemas/Widget/allOf/x'}`,
		"array index out of range": `{allOf: [{$ref: '#/paths/~1a/get/responses/200/content/application~1json/schema/allOf/9'}]}`,
		"dynamic reference":        `{$dynamicRef: '#node'}`,
		"unresolved mapping":       `{$ref: '#/components/schemas/Base'}`,
	} {
		t.Run(name, func(t *testing.T) {
			assert.Nil(t, buildSchemaUsage(usageRoot(t, strings.Replace(spec, "%s", schema, 1))))
		})
	}
	t.Run("external operation reference", func(t *testing.T) {
		assert.Nil(t, buildSchemaUsage(usageRoot(t, "paths:\n  /a:\n    get: {$ref: 'operations/read.yaml'}\n")))
	})
	t.Run("missing path item reference", func(t *testing.T) {
		assert.Nil(t, buildSchemaUsage(usageRoot(t, "paths:\n  /a: {$ref: '#/components/pathItems/Missing'}\n")))
	})
}

func TestBuildSchemaUsage_DocumentShapes(t *testing.T) {
	assert.Nil(t, buildSchemaUsage(nil))
	assert.Nil(t, buildSchemaUsage(usageRoot(t, "just a string")))
	assert.Nil(t, buildSchemaUsage(&yaml.Node{Kind: yaml.DocumentNode}))

	usage := buildSchemaUsage(usageRoot(t, "openapi: 3.1.0\npaths: not a map\nwebhooks: [nope]\n"))
	require.NotNil(t, usage)
	assert.Empty(t, usage.nodes)

	// discriminator values that are not mappings, and mappings with no mapping key, change nothing.
	usage = buildSchemaUsage(usageRoot(t, `paths:
  /a:
    get:
      responses:
        '200': {content: {application/json: {schema: {discriminator: kind, x: {discriminator: {propertyName: kind}}}}}}
`))
	require.NotNil(t, usage)
	assert.NotEmpty(t, usage.nodes)

	// a pointer through an alias resolves to the aliased node.
	root := usageRoot(t, "a: &shared {b: {type: object}}\nc: *shared\n")
	w := &usageWalker{root: root.Content[0]}
	assert.NotNil(t, w.resolve("#/c/b"))
	assert.Nil(t, w.resolve("#/a/b/type/x"))
	assert.Nil(t, w.resolve("#"))
}

func TestSchemaPairDirection(t *testing.T) {
	direction, known := schemaPairDirection(nil, nil)
	assert.False(t, known)
	assert.Zero(t, direction)

	_, known = schemaPairDirection(&base.SchemaProxy{}, nil)
	assert.False(t, known, "a proxy with no node has no usage")

	unregistered := buildSchemaProxyFromYAML(t, "type: object")
	_, known = schemaPairDirection(nil, unregistered)
	assert.False(t, known, "a proxy whose index has no usage record has no known direction")

	// Build a document, register its usage, and look up its schemas directly.
	raw := usageSpec(usageResponseOnly, usageWidget("name"))
	info, err := datamodel.ExtractSpecInfo([]byte(raw))
	require.NoError(t, err)
	doc, err := v3.CreateDocumentFromConfig(info, datamodel.NewDocumentConfiguration())
	require.NoError(t, err)
	forget := trackSchemaUsage(nil, doc.Index)
	defer forget()
	widget := doc.Components.Value.FindSchema("Widget").Value
	direction, known = schemaPairDirection(widget, nil)
	assert.True(t, known)
	assert.Equal(t, directionResponse, direction)
	direction, known = schemaPairDirection(widget, unregistered)
	assert.False(t, known)
	assert.Zero(t, direction)
}

func TestTrackSchemaUsage_SkipsUnknownDocuments(t *testing.T) {
	idx := index.NewSpecIndexWithConfig(usageRoot(t, "paths:\n  /a: {$ref: 'other.yaml'}\n"), index.CreateClosedAPIIndexConfig())
	forget := trackSchemaUsage(idx)
	schemaUsageMu.RLock()
	_, stored := schemaUsageByIndex[idx]
	schemaUsageMu.RUnlock()
	assert.False(t, stored)
	forget()
}

func TestRequiredChangeBreaking_DirectionFromProxies(t *testing.T) {
	low.ClearHashCache()
	raw := usageSpec(usageRequestOnly, usageWidget("name"))
	info, err := datamodel.ExtractSpecInfo([]byte(raw))
	require.NoError(t, err)
	doc, err := v3.CreateDocumentFromConfig(info, datamodel.NewDocumentConfiguration())
	require.NoError(t, err)
	defer trackSchemaUsage(doc.Index)()
	widget := doc.Components.Value.FindSchema("Widget").Value
	added, removed := requiredChangeBreaking(widget, widget)
	assert.True(t, added)
	assert.False(t, removed)
}

func TestTrackSchemaUsage_SharedDocument(t *testing.T) {
	// Two comparisons that share a document keep its usage until both finish.
	info, err := datamodel.ExtractSpecInfo([]byte(usageSpec(usageResponseOnly, usageWidget("name"))))
	require.NoError(t, err)
	doc, err := v3.CreateDocumentFromConfig(info, datamodel.NewDocumentConfiguration())
	require.NoError(t, err)
	widget := doc.Components.Value.FindSchema("Widget").Value

	first := trackSchemaUsage(doc.Index)
	second := trackSchemaUsage(doc.Index, doc.Index)
	first()
	_, known := schemaPairDirection(widget, nil)
	assert.True(t, known)
	second()
	_, known = schemaPairDirection(widget, nil)
	assert.False(t, known)
}
