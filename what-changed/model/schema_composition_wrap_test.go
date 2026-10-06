// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

const wrapObject = `type: object
required: [id]
properties:
  id:
    type: string`

func wrapDocument(schema string) string {
	return "openapi: 3.1.0\ninfo: {title: Composition, version: 1.0.0}\npaths: {}\ncomponents:\n  schemas:\n    Entry:\n      " + strings.ReplaceAll(schema, "\n", "\n      ")
}

func wrapAlternative(keyword, object, alternative string) string {
	return keyword + ":\n  - " + strings.ReplaceAll(object, "\n", "\n    ") + "\n  - " + strings.ReplaceAll(alternative, "\n", "\n    ")
}

func TestCompareSchemas_ObjectCompositionSafety(t *testing.T) {
	cases := []struct {
		name, old, next string
	}{
		{"overlapping oneOf", wrapObject, wrapAlternative("oneOf", wrapObject, "type: object")},
		{"wrapper assertion", wrapObject, wrapAlternative("anyOf", wrapObject, "type: 'null'") + "\nmaxProperties: 0"},
		{"wrapper unknown assertion", wrapObject, wrapAlternative("anyOf", wrapObject, "type: 'null'") + "\nx-validation: restricted"},
		{"dropped string type", strings.Replace(wrapObject, "type: object", "type: [object, string]", 1), wrapAlternative("anyOf", wrapObject, "type: 'null'")},
		{"old nullable", wrapObject + "\nnullable: true", wrapAlternative("anyOf", wrapObject, "type: array")},
		{"nullable oneOf overlap", wrapObject, wrapAlternative("oneOf", wrapObject+"\nnullable: true", "type: 'null'")},
		{"duplicate null oneOf", strings.Replace(wrapObject, "type: object", "type: [object, 'null']", 1), wrapAlternative("oneOf", wrapObject, "type: 'null'") + "\n  - type: 'null'"},
		{"null branch with constraint", strings.Replace(wrapObject, "type: object", "type: [object, 'null']", 1), wrapAlternative("anyOf", wrapObject, "type: 'null'\nnot: {}")},
		{"ambiguous object branches", wrapObject, wrapAlternative("anyOf", wrapObject, wrapObject)},
		{"removed property", wrapObject, wrapAlternative("anyOf", "type: object", "type: array")},
		{"added requirement", wrapObject, wrapAlternative("anyOf", strings.Replace(wrapObject, "required: [id]", "required: [id, name]", 1), "type: array")},
		{"narrowed property", wrapObject + "\n    maxLength: 20", wrapAlternative("anyOf", wrapObject+"\n    maxLength: 3", "type: array")},
		{"narrowed object", wrapObject + "\nmaxProperties: 20", wrapAlternative("anyOf", wrapObject+"\nmaxProperties: 1", "type: array")},
		{"reverse direction", wrapAlternative("anyOf", wrapObject, "type: array"), wrapObject},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			left, right := test_BuildDoc(wrapDocument(tc.old), wrapDocument(tc.next))
			l := left.Components.Value.FindSchema("Entry").Value
			r := right.Components.Value.FindSchema("Entry").Value
			changes := CompareSchemas(l, r)
			require.NotNil(t, changes)
			assert.Positive(t, changes.TotalBreakingChanges())
		})
	}
}

func TestCompareSchemas_ObjectCompositionPreservesChangesAndNodes(t *testing.T) {
	for _, keyword := range []string{"oneOf", "anyOf"} {
		t.Run(keyword, func(t *testing.T) {
			old := wrapObject + "\nxml: {name: before}\nexamples: [{id: old}]"
			next := wrapObject + "\nxml: {name: after}\nexamples: [{id: old}, {id: new}]"
			left, right := test_BuildDoc(wrapDocument(old), wrapDocument(wrapAlternative(keyword, next, "type: 'null'")))
			l := left.Components.Value.FindSchema("Entry").Value
			r := right.Components.Value.FindSchema("Entry").Value
			beforeL, err := yaml.Marshal(l.GetValueNode())
			require.NoError(t, err)
			beforeR, err := yaml.Marshal(r.GetValueNode())
			require.NoError(t, err)
			changes := CompareSchemas(l, r)
			require.NotNil(t, changes)
			require.NotNil(t, changes.XMLChanges)
			var exampleChange bool
			for _, change := range changes.GetAllChanges() {
				if change.Property == "examples" {
					exampleChange = true
				}
			}
			assert.True(t, exampleChange)
			afterL, err := yaml.Marshal(l.GetValueNode())
			require.NoError(t, err)
			afterR, err := yaml.Marshal(r.GetValueNode())
			require.NoError(t, err)
			assert.Equal(t, beforeL, afterL)
			assert.Equal(t, beforeR, afterR)
			assert.Equal(t, changes.TotalChanges(), CompareSchemas(l, r).TotalChanges())
		})
	}
}

func BenchmarkPreservedObjectBranch(b *testing.B) {
	for _, schema := range []struct{ name, value string }{
		{"ordinary", wrapObject},
		{"wrapped", wrapAlternative("oneOf", wrapObject, "type: 'null'")},
	} {
		b.Run(schema.name, func(b *testing.B) {
			left, right := test_BuildDoc(wrapDocument(wrapObject), wrapDocument(schema.value))
			l := left.Components.Value.FindSchema("Entry").Value
			r := right.Components.Value.FindSchema("Entry").Value
			ls, rs := l.Schema(), r.Schema()
			preservedObjectBranch(l, r, ls, rs)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				preservedObjectBranch(l, r, ls, rs)
			}
		})
	}
}

func TestPreservedObjectBranch_UnresolvedExternalBranch(t *testing.T) {
	left, _ := test_BuildDoc(wrapDocument(wrapObject), wrapDocument(wrapObject))
	info, err := datamodel.ExtractSpecInfo([]byte(wrapDocument("anyOf:\n  - $ref: './unresolved.yaml#/Thing'")))
	require.NoError(t, err)
	config := datamodel.NewDocumentConfiguration()
	config.SkipExternalRefResolution = true
	right, err := v3.CreateDocumentFromConfig(info, config)
	require.NoError(t, err)
	l := left.Components.Value.FindSchema("Entry").Value
	r := right.Components.Value.FindSchema("Entry").Value
	require.NotNil(t, r.Schema())
	require.Nil(t, r.Schema().AnyOf.Value[0].Value.Schema())
	assert.Nil(t, preservedObjectBranch(l, r, l.Schema(), r.Schema()))
}

func TestCompareSchemas_ObjectCompositionReportsWidening(t *testing.T) {
	for _, keyword := range []string{"oneOf", "anyOf"} {
		left, right := test_BuildDoc(wrapDocument(wrapObject), wrapDocument(wrapAlternative(keyword, wrapObject, "type: 'null'")))
		changes := CompareSchemas(left.Components.Value.FindSchema("Entry").Value, right.Components.Value.FindSchema("Entry").Value)
		require.NotNil(t, changes)
		require.Len(t, changes.GetAllChanges(), 1)
		assert.Equal(t, ObjectAdded, changes.GetAllChanges()[0].ChangeType)
		assert.Equal(t, keyword, changes.GetAllChanges()[0].Property)
		assert.Zero(t, changes.TotalBreakingChanges())
	}
}

func TestCompareSchemas_ObjectCompositionNullAssertions(t *testing.T) {
	for _, keyword := range []string{"oneOf", "anyOf"} {
		for _, tc := range []struct {
			name, assertion string
			widening        bool
		}{
			{"enum excludes null", "enum: [{id: ok}]", true},
			{"const excludes null", "const: {id: ok}", true},
			{"enum includes null", "enum: [{id: ok}, null]", false},
			{"const is null", "const: null", false},
			{"conditional excludes null", "if: {type: 'null'}\nthen: false", true},
			{"unknown constraint", "customConstraint: true", true},
		} {
			t.Run(keyword+"/"+tc.name, func(t *testing.T) {
				object := wrapObject + "\n" + tc.assertion
				old := strings.Replace(object, "type: object", "type: [object, 'null']", 1)
				left, right := test_BuildDoc(wrapDocument(old), wrapDocument(wrapAlternative(keyword, object, "type: 'null'")))
				changes := CompareSchemas(left.Components.Value.FindSchema("Entry").Value, right.Components.Value.FindSchema("Entry").Value)
				assert.Equal(t, tc.widening, changes.TotalChanges() > 0)
				assert.Zero(t, changes.TotalBreakingChanges())
			})
		}
	}
}

// An inline object hoisted into a reusable component and referenced via `oneOf: [$ref, null]` is an
// identical, non-breaking restructure: the referenced schema is byte-identical to the old inline
// object and the null branch preserves nullability. The naive diff used to report the object's
// properties/required as removed (breaking) because it never resolved the $ref.
func TestCompareSchemas_InlineObjectWrappedInOneOfRefIsNotBreaking(t *testing.T) {
	low.ClearHashCache()
	left := `openapi: 3.1.0
components:
  schemas:
    Widget:
      type: object
      properties:
        config:
          type:
          - object
          - 'null'
          required:
          - mode
          - items
          properties:
            mode:
              type: integer
              enum: [1, 2]
            items:
              type: array
              items:
                type: object
                required: [kind]
                properties:
                  kind:
                    type: string`

	right := `openapi: 3.1.0
components:
  schemas:
    Widget:
      type: object
      properties:
        config:
          oneOf:
          - $ref: '#/components/schemas/Config'
          - type: 'null'
    Config:
      type: object
      required:
      - mode
      - items
      properties:
        mode:
          type: integer
          enum: [1, 2]
        items:
          type: array
          items:
            type: object
            required: [kind]
            properties:
              kind:
                type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)
	lSchemaProxy := leftDoc.Components.Value.FindSchema("Widget").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Widget").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	// The restructure is equivalent, so there are no changes at all — and definitely no breaking ones.
	assert.Nil(t, changes)
	assert.Equal(t, 0, changes.TotalChanges())
	assert.Equal(t, 0, changes.TotalBreakingChanges())
}

// Widening an object into `anyOf: [<same object>, <new alternative>]` still accepts every value the
// object accepted before, so it is not breaking. The naive diff used to report the object's
// properties/required as removed because the top-level node changed from an object to an anyOf.
func TestCompareSchemas_ObjectWidenedIntoAnyOfIsNotBreaking(t *testing.T) {
	low.ClearHashCache()
	left := `openapi: 3.1.0
components:
  schemas:
    Entry:
      type: object
      required: [kind]
      properties:
        kind:
          type: string
          enum: [alpha, beta]`

	right := `openapi: 3.1.0
components:
  schemas:
    Entry:
      anyOf:
      - type: object
        required: [kind]
        properties:
          kind:
            type: string
            enum: [alpha, beta]
      - type: array
        items:
          type: object
          required: [kind]
          properties:
            kind:
              type: string
              enum: [alpha, beta]`

	leftDoc, rightDoc := test_BuildDoc(left, right)
	lSchemaProxy := leftDoc.Components.Value.FindSchema("Entry").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Entry").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.Equal(t, 0, changes.TotalBreakingChanges())
}

// The combined case: hoisted into a $ref AND the referenced schema was widened (an enum gained a
// member). The wrapper itself is non-breaking, and the genuine inner change must still be surfaced
// (as a non-breaking enum addition) rather than hidden.
func TestCompareSchemas_WrappedRefWithInnerWideningSurfacesNonBreakingChange(t *testing.T) {
	low.ClearHashCache()
	left := `openapi: 3.1.0
components:
  schemas:
    Widget:
      type: object
      properties:
        config:
          type:
          - object
          - 'null'
          required:
          - mode
          properties:
            mode:
              type: integer
              enum: [1, 2]`

	right := `openapi: 3.1.0
components:
  schemas:
    Widget:
      type: object
      properties:
        config:
          oneOf:
          - $ref: '#/components/schemas/Config'
          - type: 'null'
    Config:
      type: object
      required:
      - mode
      properties:
        mode:
          type: integer
          enum: [1, 2, 3]`

	leftDoc, rightDoc := test_BuildDoc(left, right)
	lSchemaProxy := leftDoc.Components.Value.FindSchema("Widget").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Widget").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	// The enum gaining a member is surfaced as a change, but it is not breaking.
	assert.Greater(t, changes.TotalChanges(), 0)
	assert.Equal(t, 0, changes.TotalBreakingChanges())
}

// Guard: the reverse direction (a composition collapsing to a single object, dropping an
// alternative) is a real narrowing and must stay breaking — the fix must not fire here.
func TestCompareSchemas_AnyOfUnwrappedToObjectStaysBreaking(t *testing.T) {
	low.ClearHashCache()
	left := `openapi: 3.1.0
components:
  schemas:
    Entry:
      anyOf:
      - type: object
        required: [kind]
        properties:
          kind:
            type: string
      - type: array
        items:
          type: object`

	right := `openapi: 3.1.0
components:
  schemas:
    Entry:
      type: object
      required: [kind]
      properties:
        kind:
          type: string`

	leftDoc, rightDoc := test_BuildDoc(left, right)
	lSchemaProxy := leftDoc.Components.Value.FindSchema("Entry").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Entry").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Greater(t, changes.TotalBreakingChanges(), 0)
}

// Guard: an object replaced by an anyOf whose branches do NOT preserve the original object's
// properties is a genuine change (the original property is gone) and must stay breaking.
func TestCompareSchemas_ObjectReplacedByNonPreservingAnyOfStaysBreaking(t *testing.T) {
	low.ClearHashCache()
	left := `openapi: 3.1.0
components:
  schemas:
    Entry:
      type: object
      required: [kind]
      properties:
        kind:
          type: string`

	right := `openapi: 3.1.0
components:
  schemas:
    Entry:
      anyOf:
      - type: object
        required: [label]
        properties:
          label:
            type: string
      - type: array
        items:
          type: object`

	leftDoc, rightDoc := test_BuildDoc(left, right)
	lSchemaProxy := leftDoc.Components.Value.FindSchema("Entry").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Entry").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Greater(t, changes.TotalBreakingChanges(), 0)
}

// Guard: dropping nullability while hoisting into a ref is a real narrowing (null was accepted, now
// it isn't). The fix must not fire because the composition has no null-accepting branch.
func TestCompareSchemas_WrapDroppingNullabilityStaysBreaking(t *testing.T) {
	low.ClearHashCache()
	left := `openapi: 3.1.0
components:
  schemas:
    Widget:
      type: object
      properties:
        config:
          type:
          - object
          - 'null'
          required:
          - mode
          properties:
            mode:
              type: integer`

	right := `openapi: 3.1.0
components:
  schemas:
    Widget:
      type: object
      properties:
        config:
          oneOf:
          - $ref: '#/components/schemas/Config'
    Config:
      type: object
      required:
      - mode
      properties:
        mode:
          type: integer`

	leftDoc, rightDoc := test_BuildDoc(left, right)
	lSchemaProxy := leftDoc.Components.Value.FindSchema("Widget").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Widget").Value

	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Greater(t, changes.TotalBreakingChanges(), 0)
}

// Guard: a composition branch that is a circular $ref must not be followed (it would recurse). The
// preservation shortcut bails out, so the comparison falls back to the normal diff without panicking.
func TestCompareSchemas_WrapWithCircularRefBranchIsHandled(t *testing.T) {
	low.ClearHashCache()
	left := `openapi: 3.1.0
components:
  schemas:
    Widget:
      type: object
      properties:
        config:
          type: object
          required: [mode]
          properties:
            mode:
              type: integer`

	right := `openapi: 3.1.0
components:
  schemas:
    Widget:
      type: object
      properties:
        config:
          oneOf:
          - $ref: '#/components/schemas/Node'
          - type: 'null'
    Node:
      type: object
      required: [mode]
      properties:
        mode:
          type: integer
        child:
          $ref: '#/components/schemas/Node'`

	leftDoc, rightDoc := test_BuildDoc(left, right)
	lSchemaProxy := leftDoc.Components.Value.FindSchema("Widget").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Widget").Value

	assert.NotPanics(t, func() {
		CompareSchemas(lSchemaProxy, rSchemaProxy)
	})
}
