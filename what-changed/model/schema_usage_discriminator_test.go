// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"strings"
	"testing"
)

func TestRequiredDiscriminator(t *testing.T) {
	const pets = `
  /pets:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Pet'}`
	const widgetRequestWithDiscriminator = `      requestBody:
        content:
          application/json:
            schema: {type: object, discriminator: DISCRIMINATOR}
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Widget'}`
	petSchemas := func(mapping string) string {
		return `
    Pet:
      oneOf: [{$ref: '#/components/schemas/Cat'}, {$ref: '#/components/schemas/Dog'}]
      discriminator: {propertyName: kind, mapping: ` + mapping + `}
    Cat: {type: object, properties: {kind: {type: string}}}
    Dog: {type: object, properties: {kind: {type: string}}}`
	}
	const localMapping = "{cat: '#/components/schemas/Cat', dog: Dog}"
	build := func(op, mapping, required string) string {
		spec := usageSpec(op, usageWidget(required)+petSchemas(mapping))
		return strings.Replace(spec, "\ncomponents:", pets+"\ncomponents:", 1)
	}
	for _, tc := range []struct {
		name, op, mapping string
		breaking          bool
	}{
		{"unrelated response only discriminator", usageResponseOnly, localMapping, false},
		{"unrelated discriminator without a mapping", usageResponseOnly, "{}", false},
		{"widget also used by a request", usageShared, localMapping, true},
		// Only the response discriminator can select Widget, by name, so only that response uses it.
		{"widget only reachable through the discriminator", usageUnused, localMapping, false},
		{"mapping outside the document", usageResponseOnly, "{cat: 'https://example.com/cat.yaml#/Cat'}", true},
		{"mapping to a file", usageResponseOnly, "{cat: 'cat.yaml'}", true},
		{"mapping to a schema that does not exist", usageResponseOnly, "{ghost: Ghost}", true},
		// A mapping value that is not a schema name is a relative URI of another document.
		{"mapping to a relative path", usageResponseOnly, "{dog: 'Dog/properties/kind'}", true},
		{
			"request discriminator selecting widget by name",
			strings.Replace(widgetRequestWithDiscriminator, "DISCRIMINATOR", "{propertyName: kind}", 1), localMapping, true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requireEntryBreaking(t, build(tc.op, tc.mapping, "name"), build(tc.op, tc.mapping, "name, id"), tc.breaking)
		})
	}
}

func TestRequiredDiscriminatorTargetOnlyUsedByResponse(t *testing.T) {
	// Dog is only reachable through the discriminator that a response uses, by name or through its mapping.
	const spec = `openapi: 3.1.0
info: {title: Direction, version: 1.0.0}
paths:
  /pets:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Pet'}
components:
  schemas:
    Pet:
      type: object
      discriminator: DISCRIMINATOR
    Dog: {type: object, required: [kind], properties: {kind: {type: string}, name: {type: string}}}`
	for _, tc := range []struct {
		name, discriminator string
		breaking            bool
	}{
		{"mapping by reference", "{propertyName: kind, mapping: {dog: '#/components/schemas/Dog'}}", false},
		{"mapping by name", "{propertyName: kind, mapping: {dog: Dog}}", false},
		{"default mapping by reference", "{propertyName: kind, defaultMapping: '#/components/schemas/Dog'}", false},
		{"default mapping by name", "{propertyName: kind, defaultMapping: Dog}", false},
		{"mapping and default mapping", "{propertyName: kind, mapping: {dog: Dog}, defaultMapping: Dog}", false},
		{"default mapping outside the document", "{propertyName: kind, mapping: {dog: Dog}, defaultMapping: 'other.yaml'}", true},
		{"default mapping to a schema that does not exist", "{propertyName: kind, mapping: {dog: Dog}, defaultMapping: Ghost}", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			old := strings.Replace(spec, "DISCRIMINATOR", tc.discriminator, 1)
			requireEntryBreaking(t, old, strings.Replace(old, "required: [kind]", "required: [kind, name]", 1), tc.breaking)
		})
	}
}

func TestRequiredDiscriminatorMappingToAnInlineSchema(t *testing.T) {
	// The request discriminator maps to the inline response schema, so a request can use it too.
	const spec = `openapi: 3.1.0
info: {title: Direction, version: 1.0.0}
paths:
  /widgets:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              discriminator:
                propertyName: kind
                mapping: {w: '#/paths/~1widgets/post/responses/200/content/application~1json/schema'}
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema: {type: object, required: [name], properties: {name: {type: string}, id: {type: string}}}`
	requireEntryBreaking(t, spec, strings.Replace(spec, "required: [name]", "required: [name, id]", 1), true)
}

func TestRequiredDiscriminatorWithoutAKnownDirection(t *testing.T) {
	// Widget is only used by the /widgets response. Each case adds a discriminator that can select Widget by name
	// from a request, or from a place whose direction is not known, so the required entry must stay breaking.
	const pet = "{type: object, properties: {kind: {type: string}}, discriminator: {propertyName: kind}}"
	pets := func(schema string) string {
		return `
  /pets:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema: ` + schema
	}
	const callback = `
  /pets:
    post:
      responses:
        '204': {description: ok}
      callbacks:
        done:
          '{$request.body#/url}':
            post:
              requestBody:
                content:
                  application/json:
                    schema: ` + pet + `
              responses:
                '204': {description: ok}`
	const webhook = `
webhooks:
  incoming:
    post:
      requestBody:
        content:
          application/json:
            schema: ` + pet + `
      responses:
        '204': {description: ok}`
	for _, tc := range []struct{ name, paths, schemas, webhooks string }{
		{"under not", pets("{not: " + pet + "}"), "", ""},
		{"under if", pets("{if: " + pet + ", then: {type: object}}"), "", ""},
		{"holder with unevaluated properties", pets("{type: object, unevaluatedProperties: false, properties: {kind: {type: string}}, discriminator: {propertyName: kind}}"), "", ""},
		{"in a callback", callback, "", ""},
		{"in a webhook", "", "", webhook},
		// Walking a schema that nothing refers to finds a second discriminator, which adds a direction.
		{"in a schema that nothing refers to", pets(pet), "\n    Odd: {not: " + pet + "}", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			build := func(required string) string {
				spec := usageSpec(usageResponseOnly, usageWidget(required)+tc.schemas)
				return strings.Replace(spec, "\ncomponents:", tc.paths+"\ncomponents:", 1) + tc.webhooks
			}
			requireEntryBreaking(t, build("name"), build("name, id"), true)
		})
	}
}

func TestRequiredDiscriminatorSelectsASchemaThatNothingRefersTo(t *testing.T) {
	// Nothing refers to Widget, but the request discriminator can select it by name, and the response uses a schema
	// inside it. A request can therefore send that schema as well.
	const spec = `openapi: 3.1.0
info: {title: Direction, version: 1.0.0}
paths:
  /pets:
    post:
      requestBody:
        content:
          application/json:
            schema: {$ref: '#/components/schemas/Pet'}
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Widget/properties/part'}
components:
  schemas:
    Pet: {type: object, required: [kind], properties: {kind: {type: string}}, discriminator: {propertyName: kind}}
    Widget:
      allOf: [{$ref: '#/components/schemas/Pet'}]
      properties:
        part: {type: object, required: [name], properties: {name: {type: string}, id: {type: string}}}`
	requireEntryBreaking(t, spec, strings.Replace(spec, "required: [name]", "required: [name, id]", 1), true)
}

func TestRequiredDiscriminatorRemovalFromARequest(t *testing.T) {
	// Widget is only used by a request, but a response discriminator can select it by name, so removing a required
	// entry, which is only safe for a request, must stay breaking.
	const pets = `
  /pets:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema: {type: object, properties: {kind: {type: string}}, discriminator: {propertyName: kind}}`
	build := func(required string) string {
		return strings.Replace(usageSpec(usageRequestOnly, usageWidget(required)), "\ncomponents:", pets+"\ncomponents:", 1)
	}
	requireEntryBreaking(t, build("name, id"), build("name"), true)
}
