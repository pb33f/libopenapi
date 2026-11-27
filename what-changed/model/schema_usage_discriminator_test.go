// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
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
            schema: SCHEMA
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
		// Pet only lists Cat and Dog, so its discriminator cannot select Widget, which nothing else uses.
		{"widget that the union does not list", usageUnused, localMapping, true},
		{"mapping outside the document", usageResponseOnly, "{cat: 'https://example.com/cat.yaml#/Cat'}", true},
		{"mapping to a file", usageResponseOnly, "{cat: 'cat.yaml'}", true},
		{"mapping to a schema that does not exist", usageResponseOnly, "{ghost: Ghost}", true},
		// A mapping value that is not a schema name is a relative URI of another document.
		{"mapping to a relative path", usageResponseOnly, "{dog: 'Dog/properties/kind'}", true},
		{
			"request discriminator next to a oneOf that does not list widget",
			strings.Replace(widgetRequestWithDiscriminator, "SCHEMA", "{oneOf: [{$ref: '#/components/schemas/Cat'}, {$ref: '#/components/schemas/Dog'}], discriminator: {propertyName: kind}}", 1), localMapping, false,
		},
		{
			"request discriminator next to an anyOf that does not list widget",
			strings.Replace(widgetRequestWithDiscriminator, "SCHEMA", "{anyOf: [{$ref: '#/components/schemas/Cat'}, {$ref: '#/components/schemas/Dog'}], discriminator: {propertyName: kind}}", 1), localMapping, false,
		},
		{
			"request discriminator on a parent that widget does not build on",
			strings.Replace(widgetRequestWithDiscriminator, "SCHEMA", "{type: object, discriminator: {propertyName: kind}}", 1), localMapping, false,
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

func TestRequiredDiscriminatorTargetThatIsNotASchema(t *testing.T) {
	// Dog is only reachable through the discriminator that a response uses. A pointer to a value that cannot be a
	// schema cannot be followed, so the usage is unknown. A boolean schema can be a target.
	const spec = `openapi: VERSION
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
    Dog: {type: object, required: [kind], properties: {kind: {type: string}, name: {type: string}}}
    Anything: true`
	for _, tc := range []struct {
		name, version, discriminator string
		breaking                     bool
	}{
		{"mapping to a string", "3.1.0", "{propertyName: kind, mapping: {dog: Dog, title: '#/info/title'}}", true},
		{"default mapping to a string", "3.2.0", "{propertyName: kind, mapping: {dog: Dog}, defaultMapping: '#/info/title'}", true},
		{"mapping to a boolean schema", "3.1.0", "{propertyName: kind, mapping: {dog: Dog, any: Anything}}", false},
		{"default mapping to a boolean schema", "3.2.0", "{propertyName: kind, mapping: {dog: Dog}, defaultMapping: '#/components/schemas/Anything'}", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			old := strings.NewReplacer("VERSION", tc.version, "DISCRIMINATOR", tc.discriminator).Replace(spec)
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
	// Widget is only used by the /widgets response, and builds on Pet with allOf. Each case uses Pet, which holds a
	// discriminator, from a request or from a place whose direction is not known, so Pet can select Widget by name
	// from there and the required entry must stay breaking.
	const pet = "{$ref: '#/components/schemas/Pet'}"
	const petSchema = "\n    Pet: {type: object, properties: {kind: {type: string}}, discriminator: {propertyName: kind}}"
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
		{"under not", pets("{not: " + pet + "}"), petSchema, ""},
		{"under if", pets("{if: " + pet + ", then: {type: object}}"), petSchema, ""},
		{"holder with unevaluated properties", pets(pet), "\n    Pet: {type: object, unevaluatedProperties: false, properties: {kind: {type: string}}, discriminator: {propertyName: kind}}", ""},
		{"in a callback", callback, petSchema, ""},
		{"in a webhook", "", petSchema, webhook},
		// Walking a child of the response parent Base finds Pet under not.
		{"in a child of another parent", pets("{$ref: '#/components/schemas/Base'}"), petSchema + `
    Base: {type: object, properties: {kind: {type: string}}, discriminator: {propertyName: kind}}
    Kid: {allOf: [{$ref: '#/components/schemas/Base'}], properties: {odd: {not: ` + pet + `}}}`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			build := func(required string) string {
				spec := usageSpec(usageResponseOnly, usageChild(required, "allOf: ["+pet+"]")+tc.schemas)
				return strings.Replace(spec, "\ncomponents:", tc.paths+"\ncomponents:", 1) + tc.webhooks
			}
			requireEntryBreaking(t, build("name"), build("name, id"), true)
		})
	}
}

func TestRequiredDiscriminatorParent(t *testing.T) {
	// A request sends Base, which holds a discriminator, and the /widgets response uses Widget. Base can select any
	// component schema that builds on it with allOf by name, so a request can send that schema as well.
	const bases = `
  /bases:
    post:
      requestBody:
        content:
          application/json:
            schema: {$ref: '#/components/schemas/Base'}
      responses:
        '204': {description: ok}`
	const base = `
    Base: {type: object, properties: {kind: {type: string}}, discriminator: {propertyName: kind}}`
	for _, tc := range []struct {
		name, builds, schemas string
		breaking              bool
	}{
		{"widget builds on base", "allOf: [{$ref: '#/components/schemas/Base'}]", "", true},
		{"widget builds on base through another child", "allOf: [{$ref: '#/components/schemas/Mid'}]", "\n    Mid: {allOf: [{$ref: '#/components/schemas/Base'}]}", true},
		{"widget builds on base through a reference", "allOf: [{$ref: '#/components/schemas/Alias'}]", "\n    Alias: {$ref: '#/components/schemas/Base'}", true},
		{"widget builds on base in a nested allOf", "allOf: [{allOf: [{$ref: '#/components/schemas/Base'}]}]", "", true},
		{"widget does not build on base", "", "", false},
		{"schema that builds on itself", "", "\n    Loop: {allOf: [{$ref: '#/components/schemas/Loop'}, {$ref: '#/components/schemas/Base'}]}", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			build := func(required string) string {
				spec := usageSpec(usageResponseOnly, usageChild(required, tc.builds)+base+tc.schemas)
				return strings.Replace(spec, "\ncomponents:", bases+"\ncomponents:", 1)
			}
			requireEntryBreaking(t, build("name"), build("name, id"), tc.breaking)
		})
	}
}

func TestRequiredDiscriminatorNextToAUnionSelectsNoChild(t *testing.T) {
	// A request sends Base, and Widget builds on it with allOf. Next to a oneOf or an anyOf, the discriminator can
	// only select the listed Cat, so Widget stays only used by the /widgets response.
	const bases = `
  /bases:
    post:
      requestBody:
        content:
          application/json:
            schema: {$ref: '#/components/schemas/Base'}
      responses:
        '204': {description: ok}`
	for _, union := range []string{"oneOf", "anyOf"} {
		t.Run(union, func(t *testing.T) {
			schemas := `
    Base: {` + union + `: [{$ref: '#/components/schemas/Cat'}], discriminator: {propertyName: kind}}
    Cat: {type: object, properties: {kind: {type: string}}}`
			build := func(required string) string {
				spec := usageSpec(usageResponseOnly, usageChild(required, "allOf: [{$ref: '#/components/schemas/Base'}]")+schemas)
				return strings.Replace(spec, "\ncomponents:", bases+"\ncomponents:", 1)
			}
			requireEntryBreaking(t, build("name"), build("name, id"), false)
		})
	}
}

func TestRequiredDiscriminatorChildOfAResponseParent(t *testing.T) {
	// Nothing refers to Widget, but it builds on Pet, which only a response uses, so only that response can select it.
	const pets = `
  /pets:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Pet'}`
	const pet = `
    Pet: {type: object, properties: {kind: {type: string}}, discriminator: {propertyName: kind}}`
	build := func(required string) string {
		spec := usageSpec(usageUnused, usageChild(required, "allOf: [{$ref: '#/components/schemas/Pet'}]")+pet)
		return strings.Replace(spec, "\ncomponents:", pets+"\ncomponents:", 1)
	}
	requireEntryBreaking(t, build("name"), build("name, id"), false)
}

func TestRequiredDiscriminatorChildNextToAReference(t *testing.T) {
	// Nothing refers to Widget, and Alias refers to it next to an allOf with Pet, which only a response uses. With
	// TransformSiblingRefs the schema model reads Alias as both, so only that response can select it. Without it, the
	// model reads Alias as Widget alone, so Alias does not build on Pet and Widget stays unused.
	const pets = `
  /pets:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Pet'}`
	const schemas = `
    Pet: {type: object, properties: {kind: {type: string}}, discriminator: {propertyName: kind}}
    Alias: {$ref: '#/components/schemas/Widget', allOf: [{$ref: '#/components/schemas/Pet'}]}`
	for _, transform := range []bool{true, false} {
		t.Run(fmt.Sprintf("TransformSiblingRefs %v", transform), func(t *testing.T) {
			build := func(required string) *v3.Document {
				spec := usageSpec(usageUnused, usageWidget(required)+schemas)
				info, err := datamodel.ExtractSpecInfo([]byte(strings.Replace(spec, "\ncomponents:", pets+"\ncomponents:", 1)))
				require.NoError(t, err)
				config := datamodel.NewDocumentConfiguration()
				config.TransformSiblingRefs = transform
				doc, err := v3.CreateDocumentFromConfig(info, config)
				require.NoError(t, err)
				return doc
			}
			changes := CompareDocuments(build("name"), build("name, id"))
			require.NotNil(t, changes)
			for _, ch := range changes.GetAllChanges() {
				if ch.Property == v3.RequiredLabel {
					assert.Equal(t, !transform, ch.Breaking)
					return
				}
			}
			t.Fatal("missing required change")
		})
	}
}

func TestRequiredDiscriminatorKeywordsNextToAReference(t *testing.T) {
	// Nothing else refers to Widget, so requiring a property of it is only not breaking when the response can select
	// it. The schema model reads the keywords next to a $ref only with TransformSiblingRefs, and only where the schema
	// is written: when it reaches a schema through a $ref, or an alias in a list, it follows the $ref of that schema and
	// drops the rest. A discriminator anywhere in the keywords it drops makes the usage unknown, as before this PR.
	const pets = `
  /pets:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Pet'}`
	const pet = "\n    Pet: {type: object, properties: {kind: {type: string}}, discriminator: {propertyName: kind}}"
	const other = "\n    Other: {type: object}"
	holder := strings.Replace(usageResponseOnly, "Widget", "Holder", 1)
	for _, tc := range []struct {
		name, operation, schemas, paths string
		// breaking and known are the results with the transform on and then off.
		breaking, known [2]bool
	}{
		{"discriminator written next to a reference",
			strings.Replace(usageResponseOnly, "{$ref: '#/components/schemas/Widget'}", "{$ref: '#/components/schemas/Other', discriminator: {propertyName: kind, mapping: {w: Widget}}}", 1),
			other, "", [2]bool{false, true}, [2]bool{true, false}},
		{"discriminator next to a reference, reached through a reference", holder,
			other + "\n    Holder: {$ref: '#/components/schemas/Other', discriminator: {propertyName: kind, mapping: {w: Widget}}}",
			"", [2]bool{true, true}, [2]bool{false, false}},
		{"discriminator in the properties next to a reference, reached through a reference", holder,
			other + "\n    Holder: {$ref: '#/components/schemas/Other', properties: {p: {discriminator: {propertyName: kind, mapping: {w: Widget}}}}}",
			"", [2]bool{true, true}, [2]bool{false, false}},
		{"discriminator in an allOf written next to a reference",
			strings.Replace(usageResponseOnly, "{$ref: '#/components/schemas/Widget'}", "{$ref: '#/components/schemas/Other', allOf: [{discriminator: {propertyName: kind, mapping: {w: Widget}}}]}", 1),
			other, "", [2]bool{false, true}, [2]bool{true, false}},
		{"parent discriminator next to a reference, reached through a reference", holder,
			other + "\n    Holder: {$ref: '#/components/schemas/Other', discriminator: {propertyName: kind}}" +
				"\n    Child: {allOf: [{$ref: '#/components/schemas/Holder'}, {$ref: '#/components/schemas/Widget'}]}",
			"", [2]bool{true, true}, [2]bool{false, false}},
		{"allOf next to a reference, reached through a reference from a child", usageUnused,
			pet + other + "\n    Child: {allOf: [{$ref: '#/components/schemas/Mid'}, {$ref: '#/components/schemas/Widget'}]}" +
				"\n    Mid: {$ref: '#/components/schemas/Other', allOf: [{$ref: '#/components/schemas/Pet'}]}",
			pets, [2]bool{true, true}, [2]bool{false, true}},
		// The sibling transform skips an alias in a list, so the model reads it as the target of its $ref there, but
		// not in every other place.
		{"discriminator next to a reference in an alias",
			"      x-holder: &holder {$ref: '#/components/schemas/Other', discriminator: {propertyName: kind, mapping: {w: Widget}}}\n" +
				strings.Replace(usageResponseOnly, "{$ref: '#/components/schemas/Widget'}", "{anyOf: [*holder]}", 1),
			other, "", [2]bool{true, true}, [2]bool{false, false}},
		{"discriminator in the properties next to a reference in an alias",
			"      x-holder: &holder {$ref: '#/components/schemas/Other', properties: {p: {discriminator: {propertyName: kind, mapping: {w: Widget}}}}}\n" +
				strings.Replace(usageResponseOnly, "{$ref: '#/components/schemas/Widget'}", "{anyOf: [*holder]}", 1),
			other, "", [2]bool{true, true}, [2]bool{false, false}},
		{"allOf next to a reference in an alias", usageUnused,
			pet + other + "\n    Mid: &mid {$ref: '#/components/schemas/Other', allOf: [{$ref: '#/components/schemas/Pet'}]}" +
				"\n    Child: {allOf: [*mid, {$ref: '#/components/schemas/Widget'}]}",
			pets, [2]bool{true, true}, [2]bool{false, true}},
		// The schemas a discriminator selects are read as written, as the model reads a component schema, so the
		// keywords next to their $ref count with the transform. Keywords the model drops inside them make the usage
		// unknown.
		{"discriminator next to a reference in a mapping target",
			strings.Replace(usageResponseOnly, "{$ref: '#/components/schemas/Widget'}", "{type: object, discriminator: {propertyName: kind, mapping: {d: Dog}}}", 1),
			other + "\n    Dog: {$ref: '#/components/schemas/Other', discriminator: {propertyName: kind, mapping: {w: Widget}}}",
			"", [2]bool{false, true}, [2]bool{true, false}},
		{"properties next to a reference in a mapping target",
			strings.Replace(usageResponseOnly, "{$ref: '#/components/schemas/Widget'}", "{type: object, discriminator: {propertyName: kind, mapping: {d: Dog}}}", 1),
			other + "\n    Dog: {$ref: '#/components/schemas/Other', properties: {w: {$ref: '#/components/schemas/Widget'}}}",
			"", [2]bool{false, true}, [2]bool{true, false}},
		{"properties next to a reference, reached through a reference from a mapping target",
			strings.Replace(usageResponseOnly, "{$ref: '#/components/schemas/Widget'}", "{type: object, discriminator: {propertyName: kind, mapping: {d: Dog}}}", 1),
			other + "\n    Dog: {$ref: '#/components/schemas/Holder'}" +
				"\n    Holder: {$ref: '#/components/schemas/Other', properties: {w: {$ref: '#/components/schemas/Widget'}}}",
			"", [2]bool{true, true}, [2]bool{false, false}},
		// Next to an anyOf, the discriminator of the mapping target selects no children, so only its mapping counts.
		{"discriminator next to a reference and an anyOf in a mapping target",
			strings.Replace(usageResponseOnly, "{$ref: '#/components/schemas/Widget'}", "{type: object, discriminator: {propertyName: kind, mapping: {d: Dog}}}", 1),
			other + "\n    Dog: {$ref: '#/components/schemas/Other', anyOf: [{type: object}], discriminator: {propertyName: kind, mapping: {w: Widget}}}",
			"", [2]bool{false, true}, [2]bool{true, false}},
		{"discriminator next to a reference in a child", usageUnused,
			pet + "\n    Cat: {$ref: '#/components/schemas/Pet', discriminator: {propertyName: kind, mapping: {w: Widget}}}",
			pets, [2]bool{false, true}, [2]bool{true, false}},
		{"properties next to a reference in a child", usageUnused,
			pet + "\n    Cat: {$ref: '#/components/schemas/Pet', properties: {w: {$ref: '#/components/schemas/Widget'}}}",
			pets, [2]bool{false, true}, [2]bool{true, false}},
	} {
		for i, transform := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s, TransformSiblingRefs %v", tc.name, transform), func(t *testing.T) {
				build := func(required string) *v3.Document {
					spec := usageSpec(tc.operation, usageWidget(required)+tc.schemas)
					if tc.paths != "" {
						spec = strings.Replace(spec, "\ncomponents:", tc.paths+"\ncomponents:", 1)
					}
					info, err := datamodel.ExtractSpecInfo([]byte(spec))
					require.NoError(t, err)
					config := datamodel.NewDocumentConfiguration()
					config.TransformSiblingRefs = transform
					doc, err := v3.CreateDocumentFromConfig(info, config)
					require.NoError(t, err)
					return doc
				}
				left := build("name")
				assert.Equal(t, tc.known[i], buildSchemaUsage(left.Index) != nil)
				changes := CompareDocuments(left, build("name, id"))
				require.NotNil(t, changes)
				for _, ch := range changes.GetAllChanges() {
					if ch.Property == v3.RequiredLabel {
						assert.Equal(t, tc.breaking[i], ch.Breaking)
						return
					}
				}
				t.Fatal("missing required change")
			})
		}
	}
}

func TestRequiredDiscriminatorElsewhereWithKeywordsTheModelDrops(t *testing.T) {
	// The /widgets response reaches Widget only through the properties next to the $ref of Holder, which the model
	// drops, and /bases holds a discriminator. Before this PR the discriminator made the usage unknown, so following
	// keywords the model drops did not matter. When both happen, the usage stays unknown.
	const bases = `
  /bases:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema: {type: object, discriminator: {propertyName: kind%s}}`
	const other = "\n    Other: {type: object}"
	holder := strings.Replace(usageResponseOnly, "Widget", "Holder", 1)
	for _, tc := range []struct {
		name, mapping, schemas string
		breaking, known        bool
	}{
		{"discriminator elsewhere", "", "\n    Holder: {$ref: '#/components/schemas/Other', properties: {w: {$ref: '#/components/schemas/Widget'}}}", true, false},
		{"discriminator that also selects the schema", ", mapping: {h: Holder}", "\n    Holder: {$ref: '#/components/schemas/Other', properties: {w: {$ref: '#/components/schemas/Widget'}}}", true, false},
		{"no keywords next to a reference", "", "\n    Holder: {type: object, properties: {w: {$ref: '#/components/schemas/Widget'}}}", false, true},
	} {
		for _, transform := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s, TransformSiblingRefs %v", tc.name, transform), func(t *testing.T) {
				build := func(required string) *v3.Document {
					spec := usageSpec(holder, usageWidget(required)+other+tc.schemas)
					spec = strings.Replace(spec, "\ncomponents:", fmt.Sprintf(bases, tc.mapping)+"\ncomponents:", 1)
					info, err := datamodel.ExtractSpecInfo([]byte(spec))
					require.NoError(t, err)
					config := datamodel.NewDocumentConfiguration()
					config.TransformSiblingRefs = transform
					doc, err := v3.CreateDocumentFromConfig(info, config)
					require.NoError(t, err)
					return doc
				}
				left := build("name")
				assert.Equal(t, tc.known, buildSchemaUsage(left.Index) != nil)
				changes := CompareDocuments(left, build("name, id"))
				require.NotNil(t, changes)
				for _, ch := range changes.GetAllChanges() {
					if ch.Property == v3.RequiredLabel {
						assert.Equal(t, tc.breaking, ch.Breaking)
						return
					}
				}
				t.Fatal("missing required change")
			})
		}
	}
}

func TestRequiredDiscriminatorElsewhereNextToTheReferenceOfAResponseOrPathItem(t *testing.T) {
	// The model reads a response or path item with a $ref as the one it refers to, and drops the rest. Only the content
	// next to that $ref reaches Widget, and /bases holds a discriminator, so the usage stays unknown, as before.
	const content = "content: {application/json: {schema: {$ref: '#/components/schemas/Widget'}}}"
	const bases = "\n  /bases:\n    get:\n      responses:\n        '200': {description: ok, content: {application/json: {schema: {type: object, discriminator: {propertyName: kind}}}}}"
	for _, tc := range []struct {
		name, path      string
		breaking, known bool
	}{
		{"response", "    get:\n      responses:\n        '200': {$ref: '#/components/responses/Empty', " + content + "}", true, false},
		{"path item", "    $ref: '#/components/pathItems/Empty'\n    get:\n      responses:\n        '200': {description: ok, " + content + "}", true, false},
		{"no reference", "    get:\n      responses:\n        '200': {description: ok, " + content + "}", false, true},
	} {
		for _, transform := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s, TransformSiblingRefs %v", tc.name, transform), func(t *testing.T) {
				build := func(required string) *v3.Document {
					spec := "openapi: 3.1.0\ninfo: {title: Direction, version: 1.0.0}\npaths:\n  /widgets:\n" + tc.path + bases +
						"\ncomponents:\n  responses:\n    Empty: {description: ok}\n  pathItems:\n    Empty: {}\n  schemas:\n" + usageWidget(required)
					info, err := datamodel.ExtractSpecInfo([]byte(spec))
					require.NoError(t, err)
					config := datamodel.NewDocumentConfiguration()
					config.TransformSiblingRefs = transform
					doc, err := v3.CreateDocumentFromConfig(info, config)
					require.NoError(t, err)
					return doc
				}
				left := build("name")
				assert.Equal(t, tc.known, buildSchemaUsage(left.Index) != nil)
				changes := CompareDocuments(left, build("name, id"))
				require.NotNil(t, changes)
				for _, ch := range changes.GetAllChanges() {
					if ch.Property == v3.RequiredLabel {
						assert.Equal(t, tc.breaking, ch.Breaking)
						return
					}
				}
				t.Fatal("missing required change")
			})
		}
	}
}

func TestSchemaUsageDiscriminatorWithAReferenceThatIsAnAlias(t *testing.T) {
	// The model does not resolve a $ref whose value is an alias, while the walk follows it, as before. With a
	// discriminator, the usage is then unknown, as before this PR.
	const widgets = "\n  /widgets: {get: {responses: {'200': {description: ok, content: {application/json: {schema: {$ref: *widget}}}}}}}"
	const bases = "\n  /bases: {post: {requestBody: {content: {application/json: {schema: {$ref: '#/components/schemas/Base'}}}}}}"
	for _, tc := range []struct {
		name, paths, schemas string
		known                bool
	}{
		{"discriminator elsewhere", widgets + bases, "", false},
		{"no discriminator", widgets, "", true},
		{"child that builds on the parent through it", bases, "\n    Child: {allOf: [{$ref: *base}]}", false},
	} {
		var root yaml.Node
		require.NoError(t, yaml.Unmarshal([]byte(`x-refs: {widget: &widget '#/components/schemas/Widget', base: &base '#/components/schemas/Base'}
paths:`+tc.paths+`
components:
  schemas:
    Widget: {type: object}
    Base: {type: object, discriminator: {propertyName: kind}}`+tc.schemas), &root))
		idx := index.NewSpecIndexWithConfig(&root, index.CreateClosedAPIIndexConfig())
		assert.Equal(t, tc.known, buildSchemaUsage(idx) != nil, tc.name)
	}
}

func TestSchemaUsageDiscriminatorParentWithUnresolvedReferences(t *testing.T) {
	// The schema behind a reference that cannot be followed could build on Base, so Base could select Remote.
	for _, tc := range []struct {
		ref   string
		known bool
	}{
		{"#/components/schemas/Base", true},
		{"other.yaml#/Base", false},
		{"#/components/schemas/Missing", false},
	} {
		var root yaml.Node
		require.NoError(t, yaml.Unmarshal([]byte(`paths:
  /bases: {post: {requestBody: {content: {application/json: {schema: {$ref: '#/components/schemas/Base'}}}}}}
components:
  schemas:
    Base: {type: object, discriminator: {propertyName: kind}}
    Remote: {allOf: [{$ref: '`+tc.ref+`'}]}`), &root))
		idx := index.NewSpecIndexWithConfig(&root, index.CreateClosedAPIIndexConfig())
		assert.Equal(t, tc.known, buildSchemaUsage(idx) != nil, tc.ref)
	}
}

func TestSchemaUsageDiscriminatorStopsOnceUnknown(t *testing.T) {
	// The first mapping value makes the usage unknown, so the usage is dropped and nothing else is resolved: not the
	// other mapping values, the default mapping, the children of Pet, or the schemas Cat and Dog build on.
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(`components:
  schemas:
    Pet:
      type: object
      discriminator:
        propertyName: kind
        mapping: {remote: 'https://example.com/cat.yaml', cat: Cat, dog: '#/components/schemas/Dog'}
        defaultMapping: Cat
    Remote: {allOf: [{$ref: 'other.yaml#/Pet'}]}
    Cat: {allOf: [{$ref: '#/components/schemas/Pet'}]}
    Dog: {allOf: [{$ref: '#/components/schemas/Pet'}]}`), &root))
	idx := index.NewSpecIndexWithConfig(&root, index.CreateClosedAPIIndexConfig())
	schemas := usageField(usageField(root.Content[0], v3.ComponentsLabel), v3.SchemasLabel)
	pet := usageField(schemas, "Pet")

	w := usageWalker{index: idx, targets: make(map[string]*yaml.Node), schemas: schemas}
	w.discriminator(pet, usageField(pet, v3.DiscriminatorLabel), responseSchema)
	assert.True(t, w.unknown)
	assert.Empty(t, w.targets)
	assert.Nil(t, w.children)

	// Remote makes the usage unknown while looking for the children, so the references of Cat and Dog are not resolved.
	w = usageWalker{index: idx, targets: make(map[string]*yaml.Node), schemas: schemas}
	w.allOfChildren(pet)
	assert.True(t, w.unknown)
	assert.Empty(t, w.targets)
}

func TestSchemaUsageDiscriminatorChildrenOfAChain(t *testing.T) {
	// Each schema builds on the one before it, so every schema in the chain is a child of the first one. Storing each
	// direct link once keeps the index linear in the size of the chain, instead of one entry per ancestor and child.
	const length = 256
	var spec strings.Builder
	spec.WriteString("components:\n  schemas:\n    S0: {type: object, discriminator: {propertyName: kind}}\n")
	for i := 1; i < length; i++ {
		fmt.Fprintf(&spec, "    S%d: {allOf: [{$ref: '#/components/schemas/S%d'}]}\n", i, i-1)
	}
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(spec.String()), &root))
	idx := index.NewSpecIndexWithConfig(&root, index.CreateClosedAPIIndexConfig())
	schemas := usageField(usageField(root.Content[0], v3.ComponentsLabel), v3.SchemasLabel)

	w := usageWalker{index: idx, targets: make(map[string]*yaml.Node), schemas: schemas}
	children := w.allOfChildren(usageField(schemas, "S0"))
	assert.False(t, w.unknown)
	// The children are returned in the order they are declared.
	if assert.Len(t, children, length) {
		for i, child := range children {
			assert.Same(t, usageField(schemas, fmt.Sprintf("S%d", i)), child)
		}
	}
	entries := 0
	for _, children := range w.children {
		entries += len(children)
	}
	// One link from each schema to the item of its allOf, and one from that item to the schema it refers to.
	assert.Equal(t, 2*(length-1), entries)
}

// usageChild is Widget with the keyword that makes it build on another schema, or without one when builds is empty.
func usageChild(required, builds string) string {
	if builds == "" {
		return usageWidget(required)
	}
	return strings.Replace(usageWidget(required), "    Widget:\n", "    Widget:\n      "+builds+"\n", 1)
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
	// Widget is only used by a request, but it builds on Pet, so the response discriminator on Pet can select it by
	// name. Removing a required entry, which is only safe for a request, must stay breaking.
	const pets = `
  /pets:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Pet'}`
	const pet = `
    Pet: {type: object, properties: {kind: {type: string}}, discriminator: {propertyName: kind}}`
	build := func(required string) string {
		spec := usageSpec(usageRequestOnly, usageChild(required, "allOf: [{$ref: '#/components/schemas/Pet'}]")+pet)
		return strings.Replace(spec, "\ncomponents:", pets+"\ncomponents:", 1)
	}
	requireEntryBreaking(t, build("name, id"), build("name"), true)
}
