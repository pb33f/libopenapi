package index

import (
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
