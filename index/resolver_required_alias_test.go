package index

import (
	"fmt"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

func TestResolver_RequiredReferenceMissingTarget(t *testing.T) {
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("type: object\n"), &node))
	idx := NewSpecIndex(&node)
	resolver := NewResolver(idx)
	infinite, _ := resolver.isInfiniteCircularDependency(&Reference{
		FullDefinition:        "#/Item",
		RequiredRefProperties: map[string][]string{"#/Missing": {"id"}},
	}, make(map[string]bool), nil)
	require.False(t, infinite)
}

func TestResolver_RequiredChildWithSelfReference(t *testing.T) {
	for _, tc := range []struct {
		name     string
		required string
		infinite bool
	}{
		{"required next node", "[id, next]", true},
		{"optional next node", "[id]", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := fmt.Sprintf(`openapi: 3.1.0
info: {title: Linked list, version: '1'}
paths: {}
components:
  schemas:
    List:
      type: object
      required: [head]
      properties:
        head: {$ref: '#/components/schemas/Node'}
    Node:
      type: object
      required: %s
      properties:
        id: {type: integer}
        next: {$ref: '#/components/schemas/Node'}
`, tc.required)
			var doc yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(spec), &doc))
			idx := NewSpecIndex(&doc)
			list, _ := idx.SearchIndexForReference("#/components/schemas/List")
			require.NotNil(t, list)
			infinite, _ := NewResolver(idx).isInfiniteCircularDependency(list, make(map[string]bool), nil)
			require.Equal(t, tc.infinite, infinite)
		})
	}
}

func TestResolver_CircularJourneySnapshot(t *testing.T) {
	original := &Reference{FullDefinition: "#/Loop"}
	journey := make([]*Reference, 1, 8)
	journey[0] = original
	resolver := &Resolver{}
	result := resolver.buildCircularReferenceResult(original, original, journey, 0)
	journey[0] = &Reference{FullDefinition: "#/Sibling"}
	journey = append(journey, &Reference{FullDefinition: "#/Child"})
	require.Equal(t, []*Reference{original, original}, result.Journey)
}

func TestRequiredAliasPreservesLocalFragment(t *testing.T) {
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("$ref: '#/$defs/ID'\n$defs:\n  ID: {type: string}\n"), &node))
	for _, location := range []string{"/spec/schemas/alias.yaml", "https://example.com/schemas/alias.yaml"} {
		targets := extractDefinitionRequiredRefProperties(node.Content[0], map[string][]string{}, location, nil)
		require.Contains(t, targets, location+"#/$defs/ID")
	}
}

func TestRequiredPropertyWithoutDefinition(t *testing.T) {
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(`type: object
required: [missing, id]
properties:
  id: {$ref: '#/$defs/ID'}
`), &node))
	targets := extractDefinitionRequiredRefProperties(node.Content[0], map[string][]string{}, "https://example.com/schema.yaml", nil)
	require.Equal(t, map[string][]string{"https://example.com/schema.yaml#/$defs/ID": {"id"}}, targets)
}

func TestRelativeSchemaFileUsesRemoteOwner(t *testing.T) {
	resolver := NewResolver(NewSpecIndex(nil))
	for _, ref := range []*Reference{
		{FullDefinition: "https://example.com/schemas/owner.yaml#/components/schemas/Pet"},
		{FullDefinition: "#/components/schemas/Pet", RemoteLocation: "https://example.com/schemas/owner.yaml"},
	} {
		definition, full := resolver.buildRelativeLookupDefinitions(ref, "child.yaml", "")
		require.Equal(t, "child.yaml", definition)
		require.Equal(t, "https://example.com/schemas/child.yaml", full)
	}
}
