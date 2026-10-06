// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package overlay

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	highoverlay "github.com/pb33f/libopenapi/datamodel/high/overlay"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestApply_Overlay12MergeRules(t *testing.T) {
	cases := []struct {
		name, target, action, want string
		err                        error
	}{
		{"primitive type", "value: old", "target: $.value\nupdate: true", "value: true", nil},
		{"null primitive", "value: 42", "target: $.value\nupdate: null", "value: null", nil},
		{"string primitive", "value: false", "target: $.value\nupdate: 'false'", "value: 'false'", nil},
		{"array object append", "value: [old]", "target: $.value\nupdate: {name: new}", "value: [old, {name: new}]", nil},
		{"array primitive append", "value: [old]", "target: $.value\nupdate: 7", "value: [old, 7]", nil},
		{"array null append", "value: []", "target: $.value\nupdate: null", "value: [null]", nil},
		{"array concatenate", "value: [old]", "target: $.value\nupdate: [new, next]", "value: [old, new, next]", nil},
		{"object merge", "value: {keep: yes, nested: {tags: [old], flag: false}}", "target: $.value\nupdate: {nested: {tags: [new], flag: 7}, added: null}", "value: {keep: yes, nested: {tags: [old, new], flag: 7}, added: null}", nil},
		{"object rejects primitive", "value: {}", "target: $.value\nupdate: 7", "", ErrIncompatibleUpdate},
		{"primitive rejects object", "value: old", "target: $.value\nupdate: {}", "", ErrIncompatibleUpdate},
		{"nested array rejects primitive", "value: {tags: [old]}", "target: $.value\nupdate: {tags: new}", "", ErrIncompatibleUpdate},
		{"nested object rejects array", "value: {schema: {type: string}}", "target: $.value\nupdate: {schema: []}", "", ErrIncompatibleUpdate},
		{"mixed targets", "value: [old, {}]", "target: $.value[*]\nupdate: {}", "", ErrIncompatibleUpdate},
		{"mixed primitives", "value: [old, 7, null, false]", "target: $.value[*]\nupdate: true", "value: [true, true, true, true]", nil},
		{"missing target succeeds", "value: old", "target: $.missing\ncopy: $.alsoMissing", "value: old", nil},
		{"remove ignores invalid copy and update", "value: old\nkeep: true", "target: $.value\ncopy: invalid[\nupdate: {}\nremove: true", "keep: true", nil},
		{"copy primitive to array", "source: 7\nvalue: []", "target: $.value\ncopy: $.source", "source: 7\nvalue: [7]", nil},
		{"copy changes primitive type", "source: false\nvalue: old", "target: $.value\ncopy: $.source", "source: false\nvalue: false", nil},
		{"copy nested mismatch", "source: {tags: new}\nvalue: {tags: [old]}", "target: $.value\ncopy: $.source", "", ErrCopyTypeMismatch},
		{"copy self snapshot", "a: [one]\nb: []", "target: $.*\ncopy: $.a", "a: [one, one]\nb: [one]", nil},
		{"copy descendant snapshot", "value: {items: [one], child: {items: [two]}}", "target: $.value.child\ncopy: $.value", "value: {items: [one], child: {items: [two, one], child: {items: [two]}}}", nil},
		{"remove repeated selection", "value: [one, two]", "target: $.value[0,0]\nremove: true", "value: [two]", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ov := parseOverlay(t, "overlay: 1.2.0\ninfo: {title: Rules, version: '1'}\nactions:\n  - "+strings.ReplaceAll(tc.action, "\n", "\n    "))
			result, err := Apply([]byte(tc.target), ov)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				require.Nil(t, result)
				return
			}
			require.NoError(t, err)
			assert.YAMLEq(t, tc.want, string(result.Bytes))
		})
	}
}

func TestApply_ReusableActions(t *testing.T) {
	ov := parseOverlay(t, `overlay: 1.2.7
$self: https://example.com/overlays/errors.yaml
extends: ../openapi.yaml
info: {title: Shared errors, version: '1'}
components:
  actions:
    'error/~default':
      description: Shared error responses
      fields:
        description: Add error response
        remove: true
        update:
          '404': {description: Not Found}
actions:
  - $ref: '#/components/actions/error~1~0default'
    target: $.paths.*.*.responses
    description: ''
    remove: false
  - target: $.paths['/items'].get.responses['404'].description
    update: Missing item
`)
	target := `openapi: 3.2.0
info: {title: Items, version: '1'}
paths:
  /items:
    get:
      responses:
        '200': {description: OK}
  /other:
    delete:
      responses:
        '204': {description: No Content}
`
	before, err := ov.Render()
	require.NoError(t, err)
	for range 2 {
		result, err := Apply([]byte(target), ov)
		require.NoError(t, err)
		assert.YAMLEq(t, `openapi: 3.2.0
info: {title: Items, version: '1'}
paths:
  /items:
    get:
      responses:
        '200': {description: OK}
        '404': {description: Missing item}
  /other:
    delete:
      responses:
        '204': {description: No Content}
        '404': {description: Not Found}
`, string(result.Bytes))
		rendered, err := ov.Render()
		require.NoError(t, err)
		assert.Equal(t, before, rendered)
		ov = parseOverlay(t, string(rendered))
	}
	effective, err := resolveAction(ov, ov.Actions[0])
	require.NoError(t, err)
	assert.Empty(t, effective.Description)
	assert.False(t, effective.Remove)
}

func TestApply_ReusableOverrides(t *testing.T) {
	for _, action := range []string{
		"update: {title: Local}",
		"copy: ''\nupdate: {title: Local}",
	} {
		ov := parseOverlay(t, "overlay: 1.2.0\ninfo: {title: Overrides, version: '1'}\ncomponents:\n  actions:\n    shared:\n      fields:\n        update: {title: Shared, description: Shared}\nactions:\n  - $ref: '#/components/actions/shared'\n    target: $.info\n    "+strings.ReplaceAll(action, "\n", "\n    "))
		result, err := Apply([]byte("info: {title: Original}"), ov)
		require.NoError(t, err)
		assert.YAMLEq(t, "info: {title: Local}", string(result.Bytes))
	}
	// Constructed models can override a reusable removal without changing the bool API.
	actions := orderedmap.New[string, *highoverlay.ReusableAction]()
	actions.Set("shared", &highoverlay.ReusableAction{Fields: &highoverlay.Action{Remove: true}})
	action := &highoverlay.Action{Ref: "#/components/actions/shared", Target: "$.info"}
	action.SetRemove(false)
	ov := &highoverlay.Overlay{Overlay: "1.2.0", Info: &highoverlay.Info{Title: "Constructed", Version: "1"}, Components: &highoverlay.Components{Actions: actions}, Actions: []*highoverlay.Action{action}}
	result, err := Apply([]byte("info: {title: Original}"), ov)
	require.NoError(t, err)
	assert.YAMLEq(t, "info: {title: Original}", string(result.Bytes))
}

func TestApply_InvalidReusableActions(t *testing.T) {
	cases := []struct {
		name, component, action string
		err                     error
	}{
		{"missing component", "{}", "{$ref: '#/components/actions/absent', target: '$'}", ErrInvalidActionReference},
		{"external ref", "{}", "{$ref: 'other.yaml#/components/actions/a', target: '$'}", ErrInvalidActionReference},
		{"wrong prefix", "{}", "{$ref: '#/actions/a', target: '$'}", ErrInvalidActionReference},
		{"bad escape", "{}", "{$ref: '#/components/actions/a~2', target: '$'}", ErrInvalidActionReference},
		{"bad percent", "{}", "{$ref: '#/components/actions/%zz', target: '$'}", ErrInvalidActionReference},
		{"nested pointer", "{}", "{$ref: '#/components/actions/a/fields', target: '$'}", ErrInvalidActionReference},
		{"empty ref", "{}", "{$ref: '', target: '$'}", ErrInvalidActionReference},
		{"missing target", "{a: {fields: {update: {}}}}", "{$ref: '#/components/actions/a'}", ErrMissingTarget},
		{"forbidden target", "{a: {fields: {target: '', update: {}}}}", "{target: '$'}", ErrInvalidReusableAction},
		{"nested reference", "{a: {fields: {$ref: '#/components/actions/a'}}}", "{target: '$'}", ErrInvalidReusableAction},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ov := parseOverlay(t, "overlay: 1.2.0\ninfo: {title: Invalid, version: '1'}\ncomponents:\n  actions: "+tc.component+"\nactions: ["+tc.action+"]")
			_, err := Apply([]byte("info: {}"), ov)
			require.ErrorIs(t, err, tc.err)
		})
	}
}

func TestReusableKey(t *testing.T) {
	for ref, want := range map[string]string{"#/components/actions/": "", "#/components/actions/a%20b": "a b", "#/components/actions/a~01": "a~1", "#/components/actions/a%7E1b": "a/b", "#/components/actions/a%23b": "a#b"} {
		got, err := reusableKey(ref)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
	for _, ref := range []string{"#/components/actions/a~", "#/components/actions/a b", "#/components/actions/a#b"} {
		_, err := reusableKey(ref)
		require.ErrorIs(t, err, ErrInvalidActionReference)
	}
}

func BenchmarkOverlayMergeWide(b *testing.B) {
	for _, width := range []int{20, 200, 2000} {
		b.Run(fmt.Sprint(width), func(b *testing.B) {
			target := &yaml.Node{Kind: yaml.MappingNode}
			update := &yaml.Node{Kind: yaml.MappingNode}
			for i := range width {
				target.Content = append(target.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprint(i)}, &yaml.Node{Kind: yaml.ScalarNode, Value: "old"})
				update.Content = append(update.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprint(i)}, &yaml.Node{Kind: yaml.ScalarNode, Value: "new"})
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if err := mergeNode(target, update); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestApply_SameDocumentRelativeReference(t *testing.T) {
	for _, ref := range []string{"overlay.yaml#/components/actions/a", "./overlay.yaml#/components/actions/a", "../overlays/overlay.yaml#/components/actions/a"} {
		ov := parseOverlay(t, "overlay: 1.2.0\n$self: https://example.com/overlays/overlay.yaml\ninfo: {title: Relative references, version: '1'}\ncomponents: {actions: {a: {fields: {update: {title: Changed}}}}}\nactions:\n  - $ref: '"+ref+"'\n    target: $.info")
		result, err := Apply([]byte("info: {title: Original}"), ov)
		require.NoError(t, err)
		assert.YAMLEq(t, "info: {title: Changed}", string(result.Bytes))
	}
	for _, ref := range []string{"other.yaml#/components/actions/a", "https://example.com/overlays/overlay.yaml#/components/actions/a", "overlay.yaml?other#/components/actions/a", "%zz#/components/actions/a"} {
		_, err := actionReferenceKey(&highoverlay.Overlay{Self: "https://example.com/overlays/overlay.yaml"}, ref)
		require.ErrorIs(t, err, ErrInvalidActionReference)
	}
}

func TestApply_ValidationBoundaries(t *testing.T) {
	valid := func() *highoverlay.Overlay {
		return &highoverlay.Overlay{Overlay: "1.2.0", Info: &highoverlay.Info{Title: "Valid", Version: "1"}, Actions: []*highoverlay.Action{{Target: "$"}}}
	}
	for _, version := range []string{"1.2.8", "1.1.3", "1.0.99"} {
		ov := valid()
		ov.Overlay = version
		_, err := Apply([]byte("info: {}"), ov)
		require.NoError(t, err)
	}
	for _, version := range []string{"1.3.0", "2.0.0", "1.2", "1.2.bad"} {
		ov := valid()
		ov.Overlay = version
		_, err := Apply([]byte("info: {}"), ov)
		require.ErrorIs(t, err, ErrUnsupportedVersion)
	}
	ov := parseOverlay(t, "overlay: 1.2.0\ninfo: {version: '1'}\nactions: [{target: '$'}]")
	_, err := Apply([]byte("info: {}"), ov)
	require.ErrorIs(t, err, ErrInvalidInfo)
	ov = valid()
	ov.Self = "api.yaml#fragment"
	_, err = Apply([]byte("info: {}"), ov)
	require.Error(t, err)
	ov = valid()
	ov.Actions[0] = nil
	_, err = Apply([]byte("info: {}"), ov)
	require.ErrorIs(t, err, ErrMissingTarget)
	ov = valid()
	ov.Components = &highoverlay.Components{Actions: orderedmap.New[string, *highoverlay.ReusableAction]()}
	ov.Components.Actions.Set("nil", nil)
	_, err = Apply([]byte("info: {}"), ov)
	require.ErrorIs(t, err, ErrInvalidReusableAction)
	ov = valid()
	ov.Actions[0].Copy = "$.source"
	ov.Actions[0].Target = "$.targets[*]"
	_, err = Apply([]byte("source: {}\ntargets: [{}, []]"), ov)
	require.ErrorIs(t, err, ErrIncompatibleUpdate)
}

func TestApply_WideObjectMerge(t *testing.T) {
	target := &yaml.Node{Kind: yaml.MappingNode}
	for i := range 40 {
		target.Content = append(target.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: fmt.Sprintf("key%d", i)}, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "old"})
	}
	update := &yaml.Node{Kind: yaml.MappingNode}
	for i := 35; i < 45; i++ {
		update.Content = append(update.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: fmt.Sprintf("key%d", i)}, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "new"})
	}
	ov := &highoverlay.Overlay{Overlay: "1.2.0", Info: &highoverlay.Info{Title: "Wide update", Version: "1"}, Actions: []*highoverlay.Action{{Target: "$", Update: update}}}
	raw, err := yaml.Marshal(target)
	require.NoError(t, err)
	result, err := Apply(raw, ov)
	require.NoError(t, err)
	var values map[string]string
	require.NoError(t, yaml.Unmarshal(result.Bytes, &values))
	require.Len(t, values, 45)
	assert.Equal(t, "old", values["key34"])
	assert.Equal(t, "new", values["key35"])
	assert.Equal(t, "new", values["key44"])
}

func TestApply_WideInsertIntoEmptyObject(t *testing.T) {
	const properties = 200
	update := &yaml.Node{Kind: yaml.MappingNode}
	for i := range properties {
		update.Content = append(update.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: fmt.Sprintf("property%d", i)}, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "new"})
	}
	ov := &highoverlay.Overlay{Overlay: "1.2.0", Info: &highoverlay.Info{Title: "Wide insert", Version: "1"}, Actions: []*highoverlay.Action{{Target: "$.components.schemas", Update: update}}}
	result, err := Apply([]byte("components: {schemas: {}}"), ov)
	require.NoError(t, err)
	var doc struct {
		Components struct {
			Schemas map[string]string `yaml:"schemas"`
		} `yaml:"components"`
	}
	require.NoError(t, yaml.Unmarshal(result.Bytes, &doc))
	require.Len(t, doc.Components.Schemas, properties)
	assert.Equal(t, "new", doc.Components.Schemas["property199"])
}

func BenchmarkOverlayMergeInsert(b *testing.B) {
	for _, width := range []int{20, 200, 2000} {
		b.Run(fmt.Sprint(width), func(b *testing.B) {
			update := &yaml.Node{Kind: yaml.MappingNode}
			for i := range width {
				update.Content = append(update.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: fmt.Sprint(i)}, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "new"})
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				target := &yaml.Node{Kind: yaml.MappingNode}
				if err := mergeNode(target, update); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestApply_ReferenceWithoutComponents(t *testing.T) {
	ov := &highoverlay.Overlay{Overlay: "1.2.0", Info: &highoverlay.Info{Title: "Missing components", Version: "1"}, Actions: []*highoverlay.Action{{Ref: "#/components/actions/missing", Target: "$"}}}
	_, err := Apply([]byte("info: {}"), ov)
	require.ErrorIs(t, err, ErrInvalidActionReference)
}
