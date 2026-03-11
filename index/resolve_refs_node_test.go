package index

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestResolveRefsInNode_DuplicateSiblingRefsAreResolved(t *testing.T) {
	spec := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
paths: {}
components:
  schemas:
    Shared:
      type: object
      properties:
        id:
          type: string
root:
  first:
    $ref: "#/components/schemas/Shared"
  second:
    $ref: "#/components/schemas/Shared"
`

	var root yaml.Node
	err := yaml.Unmarshal([]byte(spec), &root)
	assert.NoError(t, err)

	target := findMappingValue(root.Content[0], "root")
	assert.NotNil(t, target)

	specIndex := NewSpecIndexWithConfig(&root, CreateOpenAPIIndexConfig())
	resolved := ResolveRefsInNode(target, specIndex)
	assert.NotNil(t, resolved)

	var decoded map[string]interface{}
	err = resolved.Decode(&decoded)
	assert.NoError(t, err)

	first, ok := decoded["first"].(map[string]interface{})
	assert.True(t, ok)
	second, ok := decoded["second"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "object", first["type"])
	assert.Equal(t, "object", second["type"])
	assert.NotContains(t, first, "$ref")
	assert.NotContains(t, second, "$ref")
}

func TestResolveRefsInNode_PreservesSiblingKeys(t *testing.T) {
	spec := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
paths: {}
components:
  schemas:
    Shared:
      type: object
      properties:
        id:
          type: string
root:
  schema:
    $ref: "#/components/schemas/Shared"
    description: keep sibling
`

	var root yaml.Node
	err := yaml.Unmarshal([]byte(spec), &root)
	assert.NoError(t, err)

	rootMap := findMappingValue(root.Content[0], "root")
	assert.NotNil(t, rootMap)
	target := findMappingValue(rootMap, "schema")
	assert.NotNil(t, target)

	specIndex := NewSpecIndexWithConfig(&root, CreateOpenAPIIndexConfig())
	resolved := ResolveRefsInNode(target, specIndex)
	assert.NotNil(t, resolved)

	var decoded map[string]interface{}
	err = resolved.Decode(&decoded)
	assert.NoError(t, err)
	assert.Equal(t, "object", decoded["type"])
	assert.Equal(t, "keep sibling", decoded["description"])
	assert.NotContains(t, decoded, "$ref")
}

func TestResolveRefsInNode_NilInputReturnsNil(t *testing.T) {
	assert.Nil(t, ResolveRefsInNode(nil, nil))
}

func TestResolveRefsInNode_NilIndexReturnsOriginalNode(t *testing.T) {
	node := &yaml.Node{Kind: yaml.MappingNode}
	assert.Same(t, node, ResolveRefsInNode(node, nil))
}

func TestResolveRefsInNode_SequenceItemsAreResolved(t *testing.T) {
	spec := `openapi: 3.0.0
components:
  schemas:
    Shared:
      type: object
root:
  items:
    - $ref: "#/components/schemas/Shared"
    - $ref: "#/components/schemas/Shared"
`
	var root yaml.Node
	err := yaml.Unmarshal([]byte(spec), &root)
	assert.NoError(t, err)

	rootMap := findMappingValue(root.Content[0], "root")
	assert.NotNil(t, rootMap)
	target := findMappingValue(rootMap, "items")
	assert.NotNil(t, target)

	specIndex := NewSpecIndexWithConfig(&root, CreateOpenAPIIndexConfig())
	resolved := ResolveRefsInNode(target, specIndex)
	assert.NotNil(t, resolved)

	var decoded []map[string]interface{}
	err = resolved.Decode(&decoded)
	assert.NoError(t, err)
	assert.Len(t, decoded, 2)
	assert.Equal(t, "object", decoded[0]["type"])
	assert.Equal(t, "object", decoded[1]["type"])
}

func TestResolveRefsInNode_NonLocalRefIsPreserved(t *testing.T) {
	spec := `openapi: 3.0.0
components:
  schemas:
    Shared:
      type: object
root:
  schema:
    $ref: "https://example.com/openapi.yaml#/components/schemas/Remote"
    nested:
      $ref: "#/components/schemas/Shared"
`
	var root yaml.Node
	err := yaml.Unmarshal([]byte(spec), &root)
	assert.NoError(t, err)

	rootMap := findMappingValue(root.Content[0], "root")
	assert.NotNil(t, rootMap)
	target := findMappingValue(rootMap, "schema")
	assert.NotNil(t, target)

	specIndex := NewSpecIndexWithConfig(&root, CreateOpenAPIIndexConfig())
	resolved := ResolveRefsInNode(target, specIndex)
	assert.NotNil(t, resolved)

	var decoded map[string]interface{}
	err = resolved.Decode(&decoded)
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com/openapi.yaml#/components/schemas/Remote", decoded["$ref"])

	nested, ok := decoded["nested"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "object", nested["type"])
}

func TestResolveRefsInNode_UnresolvedLocalRefFallsBackToOriginalMapping(t *testing.T) {
	spec := `openapi: 3.0.0
components:
  schemas:
    Shared:
      type: object
root:
  schema:
    $ref: "#/components/schemas/Missing"
    nested:
      $ref: "#/components/schemas/Shared"
`
	var root yaml.Node
	err := yaml.Unmarshal([]byte(spec), &root)
	assert.NoError(t, err)

	rootMap := findMappingValue(root.Content[0], "root")
	assert.NotNil(t, rootMap)
	target := findMappingValue(rootMap, "schema")
	assert.NotNil(t, target)

	specIndex := NewSpecIndexWithConfig(&root, CreateOpenAPIIndexConfig())
	resolved := ResolveRefsInNode(target, specIndex)
	assert.NotNil(t, resolved)

	var decoded map[string]interface{}
	err = resolved.Decode(&decoded)
	assert.NoError(t, err)
	assert.Equal(t, "#/components/schemas/Missing", decoded["$ref"])

	nested, ok := decoded["nested"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "object", nested["type"])
}

func TestResolveRefsInNode_CircularRefFallsBack(t *testing.T) {
	spec := `openapi: 3.0.0
components:
  schemas:
    Loop:
      $ref: "#/components/schemas/Loop"
root:
  schema:
    $ref: "#/components/schemas/Loop"
    description: keep sibling
`
	var root yaml.Node
	err := yaml.Unmarshal([]byte(spec), &root)
	assert.NoError(t, err)

	rootMap := findMappingValue(root.Content[0], "root")
	assert.NotNil(t, rootMap)
	target := findMappingValue(rootMap, "schema")
	assert.NotNil(t, target)

	specIndex := NewSpecIndexWithConfig(&root, CreateOpenAPIIndexConfig())
	resolved := ResolveRefsInNode(target, specIndex)
	assert.NotNil(t, resolved)

	var decoded map[string]interface{}
	err = resolved.Decode(&decoded)
	assert.NoError(t, err)
	assert.Equal(t, "#/components/schemas/Loop", decoded["$ref"])
	assert.Equal(t, "keep sibling", decoded["description"])
}

func TestResolveRefsInNode_DocumentNodeMergeBranch(t *testing.T) {
	var mappedDoc yaml.Node
	err := yaml.Unmarshal([]byte(`type: object
properties:
  id:
    type: string
`), &mappedDoc)
	assert.NoError(t, err)

	var targetDoc yaml.Node
	err = yaml.Unmarshal([]byte(`schema:
  $ref: "#/components/schemas/DocRef"
  description: merged
`), &targetDoc)
	assert.NoError(t, err)
	target := findMappingValue(targetDoc.Content[0], "schema")
	assert.NotNil(t, target)

	cfg := CreateClosedAPIIndexConfig()
	idx := NewSpecIndexWithConfig(&mappedDoc, cfg)
	idx.SetMappedReferences(map[string]*Reference{
		"#/components/schemas/DocRef": {
			FullDefinition: "#/components/schemas/DocRef",
			Node:           &mappedDoc,
			Index:          idx,
		},
	})

	resolved := ResolveRefsInNode(target, idx)
	assert.NotNil(t, resolved)

	var decoded map[string]interface{}
	err = resolved.Decode(&decoded)
	assert.NoError(t, err)
	assert.Equal(t, "object", decoded["type"])
	assert.Equal(t, "merged", decoded["description"])
}

func TestResolveRefsInNode_ScalarResolvedRefFallsBackToOriginalMapping(t *testing.T) {
	var root yaml.Node
	err := yaml.Unmarshal([]byte("openapi: 3.0.0"), &root)
	assert.NoError(t, err)

	cfg := CreateClosedAPIIndexConfig()
	idx := NewSpecIndexWithConfig(&root, cfg)
	idx.SetMappedReferences(map[string]*Reference{
		"#/components/schemas/ScalarRef": {
			FullDefinition: "#/components/schemas/ScalarRef",
			Node:           &yaml.Node{Kind: yaml.ScalarNode, Value: "value"},
			Index:          idx,
		},
	})

	var targetDoc yaml.Node
	err = yaml.Unmarshal([]byte(`schema:
  $ref: "#/components/schemas/ScalarRef"
  description: keep
`), &targetDoc)
	assert.NoError(t, err)
	target := findMappingValue(targetDoc.Content[0], "schema")
	assert.NotNil(t, target)

	resolved := ResolveRefsInNode(target, idx)
	assert.NotNil(t, resolved)

	var decoded map[string]interface{}
	err = resolved.Decode(&decoded)
	assert.NoError(t, err)
	assert.Equal(t, "#/components/schemas/ScalarRef", decoded["$ref"])
	assert.Equal(t, "keep", decoded["description"])
}

func TestResolveRefsInNode_DefaultBranchForScalarNode(t *testing.T) {
	var root yaml.Node
	err := yaml.Unmarshal([]byte("openapi: 3.0.0"), &root)
	assert.NoError(t, err)
	idx := NewSpecIndexWithConfig(&root, CreateClosedAPIIndexConfig())

	scalar := &yaml.Node{Kind: yaml.ScalarNode, Value: "literal"}
	out := resolveRefsInNode(scalar, idx, map[string]struct{}{})
	assert.Same(t, scalar, out)
}

func TestResolveRefsInNode_InternalNilGuards(t *testing.T) {
	var root yaml.Node
	err := yaml.Unmarshal([]byte("openapi: 3.0.0"), &root)
	assert.NoError(t, err)
	idx := NewSpecIndexWithConfig(&root, CreateClosedAPIIndexConfig())

	assert.Nil(t, resolveRefsInNode(nil, idx, map[string]struct{}{}))

	node := &yaml.Node{Kind: yaml.MappingNode}
	assert.Same(t, node, resolveRefsInNode(node, nil, map[string]struct{}{}))
}

func TestMergeResolvedMappingWithSiblings_OverridesAndSkipsNilKeys(t *testing.T) {
	resolved := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "type"},
			{Kind: yaml.ScalarNode, Value: "object"},
		},
	}
	siblings := []*yaml.Node{
		{Kind: yaml.ScalarNode, Value: "type"},
		{Kind: yaml.ScalarNode, Value: "string"},
		nil,
		{Kind: yaml.ScalarNode, Value: "ignored"},
		{Kind: yaml.ScalarNode, Value: "description"},
		{Kind: yaml.ScalarNode, Value: "ok"},
	}

	merged := mergeResolvedMappingWithSiblings(resolved, siblings)
	assert.NotNil(t, merged)

	var decoded map[string]interface{}
	err := merged.Decode(&decoded)
	assert.NoError(t, err)
	assert.Equal(t, "string", decoded["type"])
	assert.Equal(t, "ok", decoded["description"])
}

func findMappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i] != nil && node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}
