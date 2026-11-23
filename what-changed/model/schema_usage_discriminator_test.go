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
		// A components section that is not schemas has nothing to follow.
		spec := usageSpec(op, usageWidget(required)+petSchemas(mapping)+"\n  responses:\n    Ok: {description: ok}")
		return strings.Replace(spec, "\ncomponents:", pets+"\ncomponents:", 1)
	}
	for _, tc := range []struct {
		name, op, mapping string
		breaking          bool
	}{
		{"unrelated response only discriminator", usageResponseOnly, localMapping, false},
		{"unrelated discriminator without a mapping", usageResponseOnly, "{}", false},
		{"widget also used by a request", usageShared, localMapping, true},
		{"widget not used at all", usageUnused, localMapping, true},
		{"mapping outside the document", usageResponseOnly, "{cat: 'https://example.com/cat.yaml#/Cat'}", true},
		{"mapping to a file", usageResponseOnly, "{cat: 'cat.yaml'}", true},
		{"mapping to a schema that does not exist", usageResponseOnly, "{ghost: Ghost}", true},
		{
			"request discriminator selecting widget by name",
			strings.Replace(widgetRequestWithDiscriminator, "DISCRIMINATOR", "{propertyName: kind}", 1), localMapping, true,
		},
		{
			"request discriminator mapping to widget",
			strings.Replace(widgetRequestWithDiscriminator, "DISCRIMINATOR", "{propertyName: kind, mapping: {w: '#/components/schemas/Widget'}}", 1), localMapping, true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requireEntryBreaking(t, build(tc.op, tc.mapping, "name"), build(tc.op, tc.mapping, "name, id"), tc.breaking)
		})
	}
}

func TestRequiredDiscriminatorTargetOnlyUsedByResponse(t *testing.T) {
	// Dog is only reachable through the discriminator that a response uses.
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
