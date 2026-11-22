// Copyright 2023-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package specpath

import (
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func parseDocument(t *testing.T, src string) *yaml.Node {
	t.Helper()
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(src), &root))
	return &root
}

// at returns the value under each key in turn, starting from the top mapping of root.
func at(root *yaml.Node, keys ...string) *yaml.Node {
	node := root.Content[0]
	for _, key := range keys {
		var next *yaml.Node
		for i := 0; i+1 < len(node.Content); i += 2 {
			if node.Content[i].Value == key {
				next = node.Content[i+1]
				break
			}
		}
		if next == nil {
			return nil
		}
		node = next
	}
	return node
}

// describe names the node and path of each selection, as name@key/key.
func describe(selections []Selection, names map[*yaml.Node]string) []string {
	described := make([]string, len(selections))
	for i, selection := range selections {
		described[i] = names[selection.Node] + "@" + strings.Join(selection.Path, "/")
	}
	return described
}

func TestIsOpenAPIDocument(t *testing.T) {
	assert.True(t, IsOpenAPIDocument(parseDocument(t, "openapi: 3.1.0\n")))
	assert.True(t, IsOpenAPIDocument(parseDocument(t, "swagger: '2.0'\n").Content[0]))
	assert.False(t, IsOpenAPIDocument(parseDocument(t, "type: object\n")))
	assert.False(t, IsOpenAPIDocument(parseDocument(t, "- openapi\n")))
	assert.False(t, IsOpenAPIDocument(&yaml.Node{Kind: yaml.DocumentNode}))
	assert.False(t, IsOpenAPIDocument(&yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{nil, {Value: "x"}}}))
	assert.False(t, IsOpenAPIDocument(nil))
}

func TestTree_Path(t *testing.T) {
	root := parseDocument(t, `a:
  b:
    - c: &shared {d: 1}
    - *shared
e: *shared
`)
	tree := NewTree(root)
	shared := at(root, "a", "b").Content[0].Content[1]

	path, ok := tree.Path(root, shared)
	assert.True(t, ok)
	assert.Equal(t, []string{"a", "b", "c"}, path, "a sequence element sits under no key")

	path, ok = tree.Path(at(root, "a"), shared)
	assert.True(t, ok)
	assert.Equal(t, []string{"b", "c"}, path)

	path, ok = tree.Path(shared, shared)
	assert.True(t, ok)
	assert.Empty(t, path)

	_, ok = tree.Path(at(root, "e"), shared)
	assert.False(t, ok, "an alias does not hold its target")

	_, ok = tree.Path(root, &yaml.Node{})
	assert.False(t, ok, "a node outside the tree has no path")

	parent, _, keyed := tree.Parent(root)
	assert.Nil(t, parent)
	assert.False(t, keyed)
	parent, key, keyed := tree.Parent(at(root, "a"))
	assert.Equal(t, root.Content[0], parent)
	assert.Equal(t, "a", key)
	assert.True(t, keyed)
}

func TestTree_SkipsNilNodes(t *testing.T) {
	mapping := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{
		nil, {Kind: yaml.ScalarNode, Value: "skipped"},
		{Kind: yaml.ScalarNode, Value: "k"}, nil,
	}}
	sequence := &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{nil, mapping}}
	tree := NewTree(sequence)
	parent, _, _ := tree.Parent(mapping)
	assert.Equal(t, sequence, parent)
	parent, _, _ = tree.Parent(mapping.Content[1])
	assert.Nil(t, parent, "a value without a key is not linked")
}

func TestSelections(t *testing.T) {
	main := parseDocument(t, "openapi: 3.1.0\n")
	wrapper := parseDocument(t, "$ref: owner.yaml\n")
	catalog := parseDocument(t, "entries:\n  owner:\n    $ref: owner.yaml\n  other:\n    $ref: other.yaml\n")
	owner := parseDocument(t, "const: 1\n")
	other := parseDocument(t, "const: 2\n")
	roots := map[string]*yaml.Node{"wrapper.yaml": wrapper, "catalog.yaml": catalog, "owner.yaml": owner, "other.yaml": other}
	relative := map[string]bool{"wrapper.yaml": true, "catalog.yaml": true, "owner.yaml": true, "other.yaml": true}
	refs := []Reference{
		{From: "main.yaml", Holder: main.Content[0], SourcePath: []string{"components", "schemas", "A"}, Target: "wrapper.yaml", Selected: wrapper.Content[0]},
		{From: "main.yaml", Holder: main.Content[0], SourcePath: []string{"components", "schemas", "B"}, Target: "catalog.yaml", Selected: at(catalog, "entries", "owner")},
		{From: "wrapper.yaml", Holder: wrapper.Content[0], Target: "owner.yaml", Selected: owner.Content[0]},
		{From: "catalog.yaml", Holder: at(catalog, "entries", "owner"), SourcePath: []string{"entries", "owner"}, Target: "owner.yaml", Selected: owner.Content[0]},
		{From: "catalog.yaml", Holder: at(catalog, "entries", "other"), SourcePath: []string{"entries", "other"}, Target: "other.yaml", Selected: other.Content[0]},
	}

	names := map[*yaml.Node]string{wrapper.Content[0]: "wrapper", owner.Content[0]: "owner", other.Content[0]: "other"}

	selections := Selections(refs, roots, relative)
	assert.Equal(t, []string{"wrapper@components/schemas/A"}, describe(selections["wrapper.yaml"], names))
	assert.ElementsMatch(t, []string{"owner@components/schemas/A", "owner@components/schemas/B"},
		describe(selections["owner.yaml"], names), "a reference in a selected node takes effect at the path of the selection")
	assert.Equal(t, []string{"other@entries/other"}, describe(selections["other.yaml"], names),
		"a reference nothing selected reaches keeps its own source path")
}

func TestSelections_FallbackPropagates(t *testing.T) {
	first := parseDocument(t, "x:\n  $ref: second.yaml\n")
	second := parseDocument(t, "$ref: third.yaml\n")
	third := parseDocument(t, "const: 1\n")
	roots := map[string]*yaml.Node{"first.yaml": first, "second.yaml": second, "third.yaml": third}
	relative := map[string]bool{"first.yaml": true, "second.yaml": true, "third.yaml": true}
	refs := []Reference{
		{From: "second.yaml", Holder: second.Content[0], Target: "third.yaml", Selected: third.Content[0]},
		{From: "first.yaml", Holder: at(first, "x"), SourcePath: []string{"x"}, Target: "second.yaml", Selected: second.Content[0]},
	}
	names := map[*yaml.Node]string{second.Content[0]: "second", third.Content[0]: "third"}

	for _, order := range [][]Reference{refs, {refs[1], refs[0]}} {
		selections := Selections(order, roots, relative)
		assert.Equal(t, []string{"second@x"}, describe(selections["second.yaml"], names))
		assert.Equal(t, []string{"third@x"}, describe(selections["third.yaml"], names),
			"a document nothing references falls back first and passes its path on")
	}
}

func TestSelections_ReferencedCycleFallsBack(t *testing.T) {
	a := parseDocument(t, "$ref: b.yaml\n")
	b := parseDocument(t, "$ref: a.yaml\n")
	roots := map[string]*yaml.Node{"a.yaml": a, "b.yaml": b}
	relative := map[string]bool{"a.yaml": true, "b.yaml": true}
	refs := []Reference{
		{From: "a.yaml", Holder: a.Content[0], Target: "b.yaml", Selected: b.Content[0]},
		{From: "b.yaml", Holder: b.Content[0], Target: "a.yaml", Selected: a.Content[0]},
	}
	names := map[*yaml.Node]string{a.Content[0]: "a", b.Content[0]: "b"}

	selections := Selections(refs, roots, relative)
	assert.Equal(t, []string{"b@"}, describe(selections["b.yaml"], names))
	assert.Equal(t, []string{"a@"}, describe(selections["a.yaml"], names))
}

func TestSelections_CycleStops(t *testing.T) {
	a := parseDocument(t, "next:\n  $ref: b.yaml\n")
	b := parseDocument(t, "next:\n  $ref: a.yaml\n")
	roots := map[string]*yaml.Node{"a.yaml": a, "b.yaml": b}
	relative := map[string]bool{"a.yaml": true, "b.yaml": true}
	refs := []Reference{
		{From: "main.yaml", SourcePath: []string{"components", "schemas", "A"}, Target: "a.yaml", Selected: a.Content[0]},
		{From: "a.yaml", Holder: at(a, "next"), SourcePath: []string{"next"}, Target: "b.yaml", Selected: b.Content[0]},
		{From: "b.yaml", Holder: at(b, "next"), SourcePath: []string{"next"}, Target: "a.yaml", Selected: a.Content[0]},
	}

	names := map[*yaml.Node]string{a.Content[0]: "a", b.Content[0]: "b"}

	selections := Selections(refs, roots, relative)
	assert.Equal(t, []string{"a@components/schemas/A", "a@components/schemas/A/next/next"}, describe(selections["a.yaml"], names))
	assert.Equal(t, []string{"b@components/schemas/A/next"}, describe(selections["b.yaml"], names),
		"a chain stops before it takes a reference a second time")
}

func TestSelections_LongChainInOneDocument(t *testing.T) {
	catalog := parseDocument(t, "A:\n  $ref: '#/B'\nB:\n  $ref: '#/C'\nC:\n  $ref: '#/D'\nD:\n  $ref: owner.yaml\n")
	owner := parseDocument(t, "const: 1\n")
	roots := map[string]*yaml.Node{"catalog.yaml": catalog, "owner.yaml": owner}
	relative := map[string]bool{"catalog.yaml": true, "owner.yaml": true}
	refs := []Reference{
		{From: "main.yaml", SourcePath: []string{"components", "schemas", "A"}, Target: "catalog.yaml", Selected: at(catalog, "A")},
		{From: "catalog.yaml", Holder: at(catalog, "A"), SourcePath: []string{"A"}, Target: "catalog.yaml", Selected: at(catalog, "B")},
		{From: "catalog.yaml", Holder: at(catalog, "B"), SourcePath: []string{"B"}, Target: "catalog.yaml", Selected: at(catalog, "C")},
		{From: "catalog.yaml", Holder: at(catalog, "C"), SourcePath: []string{"C"}, Target: "catalog.yaml", Selected: at(catalog, "D")},
		{From: "catalog.yaml", Holder: at(catalog, "D"), SourcePath: []string{"D"}, Target: "owner.yaml", Selected: owner.Content[0]},
	}
	names := map[*yaml.Node]string{owner.Content[0]: "owner"}

	selections := Selections(refs, roots, relative)
	assert.Equal(t, []string{"owner@components/schemas/A"}, describe(selections["owner.yaml"], names),
		"a chain is as long as the references it takes, whatever the number of documents")
}

func TestSelections_FallbackLeavesRootedDocuments(t *testing.T) {
	owner := parseDocument(t, "const: 1\n")
	unrelated := parseDocument(t, "foo:\n  $ref: owner.yaml\nbar:\n  $ref: other.yaml\n")
	other := parseDocument(t, "const: 2\n")
	roots := map[string]*yaml.Node{"owner.yaml": owner, "unrelated.yaml": unrelated, "other.yaml": other}
	relative := map[string]bool{"owner.yaml": true, "unrelated.yaml": true, "other.yaml": true}
	refs := []Reference{
		{From: "main.yaml", SourcePath: []string{"components", "schemas", "Owner"}, Target: "owner.yaml", Selected: owner.Content[0]},
		{From: "unrelated.yaml", Holder: at(unrelated, "foo"), SourcePath: []string{"foo"}, Target: "owner.yaml", Selected: owner.Content[0]},
		{From: "unrelated.yaml", Holder: at(unrelated, "bar"), SourcePath: []string{"bar"}, Target: "other.yaml", Selected: other.Content[0]},
	}
	names := map[*yaml.Node]string{owner.Content[0]: "owner", other.Content[0]: "other"}

	selections := Selections(refs, roots, relative)
	assert.Equal(t, []string{"owner@components/schemas/Owner"}, describe(selections["owner.yaml"], names),
		"a document nothing references does not add a context to one an OpenAPI description reaches")
	assert.Equal(t, []string{"other@bar"}, describe(selections["other.yaml"], names))
}

func TestSelections_SameSelectionOnce(t *testing.T) {
	owner := parseDocument(t, "const: 1\n")
	roots := map[string]*yaml.Node{"owner.yaml": owner}
	relative := map[string]bool{"owner.yaml": true}
	ref := Reference{From: "main.yaml", SourcePath: []string{"components", "schemas", "Owner"}, Target: "owner.yaml", Selected: owner.Content[0]}
	names := map[*yaml.Node]string{owner.Content[0]: "owner"}

	selections := Selections([]Reference{ref, ref}, roots, relative)
	assert.Equal(t, []string{"owner@components/schemas/Owner"}, describe(selections["owner.yaml"], names))
}
