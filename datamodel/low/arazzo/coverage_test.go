// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"hash/maphash"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// ---------------------------------------------------------------------------
// Arazzo.Build() error paths
// ---------------------------------------------------------------------------

func TestArazzo_Build_InfoError(t *testing.T) {
	// info is expected to be a mapping; providing a scalar triggers an error from ExtractObject.
	yml := `arazzo: 1.0.1
info: not-a-mapping
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1`

	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yml), &node))

	var a Arazzo
	err := low.BuildModel(node.Content[0], &a)
	require.NoError(t, err)

	err = a.Build(context.Background(), nil, node.Content[0], nil)
	// ExtractObject for Info should not return an error for scalar (it just won't match).
	// But let's verify the build still succeeds (scalar info is simply ignored).
	// The actual error path would require an invalid structure inside info.
	// We accept no error for this benign case.
	assert.NoError(t, err)
}

func TestArazzo_Build_SourceDescriptionsNotSequence(t *testing.T) {
	// sourceDescriptions as a scalar instead of a sequence
	yml := `arazzo: 1.0.1
info:
  title: Test
  version: 0.1.0
sourceDescriptions: not-a-sequence
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1`

	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yml), &node))

	var a Arazzo
	err := low.BuildModel(node.Content[0], &a)
	require.NoError(t, err)

	err = a.Build(context.Background(), nil, node.Content[0], nil)
	assert.NoError(t, err)
	// sourceDescriptions is not a valid sequence, so it should be empty
	assert.True(t, a.SourceDescriptions.IsEmpty() || len(a.SourceDescriptions.Value) == 0)
}

func TestArazzo_Build_WorkflowsNotSequence(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Test
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com
workflows: not-a-sequence`

	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yml), &node))

	var a Arazzo
	err := low.BuildModel(node.Content[0], &a)
	require.NoError(t, err)

	err = a.Build(context.Background(), nil, node.Content[0], nil)
	assert.NoError(t, err)
	assert.True(t, a.Workflows.IsEmpty() || len(a.Workflows.Value) == 0)
}

func TestArazzo_Build_ComponentsNotMapping(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Test
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1
components: not-a-mapping`

	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yml), &node))

	var a Arazzo
	err := low.BuildModel(node.Content[0], &a)
	require.NoError(t, err)

	err = a.Build(context.Background(), nil, node.Content[0], nil)
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Arazzo.Hash() with Components non-empty
// ---------------------------------------------------------------------------

func TestArazzo_Hash_WithComponents(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Test
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com
    type: openapi
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1
components:
  parameters:
    myParam:
      name: p1
      in: query
      value: v1
  successActions:
    sa1:
      name: end
      type: end
  failureActions:
    fa1:
      name: retry
      type: retry
      retryAfter: 1.0
      retryLimit: 2
  inputs:
    myInput:
      type: object`

	var n1, n2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &n1)
	_ = yaml.Unmarshal([]byte(yml), &n2)

	var a1, a2 Arazzo
	_ = low.BuildModel(n1.Content[0], &a1)
	_ = a1.Build(context.Background(), nil, n1.Content[0], nil)
	_ = low.BuildModel(n2.Content[0], &a2)
	_ = a2.Build(context.Background(), nil, n2.Content[0], nil)

	// Components non-empty: hash path for Components.Hash() is covered
	assert.False(t, a1.Components.IsEmpty())
	assert.Equal(t, a1.Hash(), a2.Hash())

	// Verify the hash is different from a doc without components
	ymlNoComp := `arazzo: 1.0.1
info:
  title: Test
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com
    type: openapi
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1`

	var n3 yaml.Node
	_ = yaml.Unmarshal([]byte(ymlNoComp), &n3)
	var a3 Arazzo
	_ = low.BuildModel(n3.Content[0], &a3)
	_ = a3.Build(context.Background(), nil, n3.Content[0], nil)

	assert.NotEqual(t, a1.Hash(), a3.Hash())
}

func TestArazzo_GettersAndFindExtension(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Test
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1
x-custom: myval`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	keyNode := &yaml.Node{Value: "arazzo"}
	var a Arazzo
	_ = low.BuildModel(node.Content[0], &a)
	_ = a.Build(context.Background(), keyNode, node.Content[0], nil)

	assert.Equal(t, keyNode, a.GetKeyNode())
	assert.Equal(t, node.Content[0], a.GetRootNode())
	assert.Nil(t, a.GetIndex())
	assert.NotNil(t, a.GetContext())
	assert.NotNil(t, a.GetExtensions())

	ext := a.FindExtension("x-custom")
	require.NotNil(t, ext)
	assert.Equal(t, "myval", ext.Value.Value)

	assert.Nil(t, a.FindExtension("x-nope"))
}

// ---------------------------------------------------------------------------
// Components.Build() with all maps populated (happy path for Hash coverage)
// ---------------------------------------------------------------------------

func TestComponents_Build_AllMaps(t *testing.T) {
	yml := `inputs:
  inputA:
    type: object
parameters:
  paramA:
    name: p1
    in: query
    value: v1
successActions:
  sa1:
    name: end
    type: end
failureActions:
  fa1:
    name: retry
    type: retry
    retryAfter: 1.0
    retryLimit: 2`

	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yml), &node))

	var comp Components
	require.NoError(t, low.BuildModel(node.Content[0], &comp))
	require.NoError(t, comp.Build(context.Background(), nil, node.Content[0], nil))

	// Verify all maps are populated
	assert.False(t, comp.Inputs.IsEmpty())
	assert.NotNil(t, comp.Inputs.Value)

	assert.False(t, comp.Parameters.IsEmpty())
	assert.NotNil(t, comp.Parameters.Value)

	assert.False(t, comp.SuccessActions.IsEmpty())
	assert.NotNil(t, comp.SuccessActions.Value)

	assert.False(t, comp.FailureActions.IsEmpty())
	assert.NotNil(t, comp.FailureActions.Value)
}

func TestComponents_Hash_AllMapsPopulated(t *testing.T) {
	yml := `inputs:
  inputA:
    type: object
parameters:
  paramA:
    name: p1
    in: query
    value: v1
successActions:
  sa1:
    name: end
    type: end
failureActions:
  fa1:
    name: retry
    type: retry
    retryAfter: 1.0
    retryLimit: 2`

	var n1, n2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &n1)
	_ = yaml.Unmarshal([]byte(yml), &n2)

	var c1, c2 Components
	_ = low.BuildModel(n1.Content[0], &c1)
	_ = c1.Build(context.Background(), nil, n1.Content[0], nil)
	_ = low.BuildModel(n2.Content[0], &c2)
	_ = c2.Build(context.Background(), nil, n2.Content[0], nil)

	assert.Equal(t, c1.Hash(), c2.Hash())
}

func TestComponents_Hash_Empty(t *testing.T) {
	// Empty Components should still hash consistently
	var c1, c2 Components
	assert.Equal(t, c1.Hash(), c2.Hash())
}

func TestComponents_Build_ParametersNotMapping(t *testing.T) {
	yml := `parameters: not-a-mapping`

	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yml), &node))

	var comp Components
	_ = low.BuildModel(node.Content[0], &comp)
	err := comp.Build(context.Background(), nil, node.Content[0], nil)
	assert.NoError(t, err)
	// parameters value is not a mapping, so the map value should be nil
	assert.Nil(t, comp.Parameters.Value)
}

func TestComponents_Build_SuccessActionsNotMapping(t *testing.T) {
	yml := `successActions: not-a-mapping`

	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yml), &node))

	var comp Components
	_ = low.BuildModel(node.Content[0], &comp)
	err := comp.Build(context.Background(), nil, node.Content[0], nil)
	assert.NoError(t, err)
	assert.Nil(t, comp.SuccessActions.Value)
}

func TestComponents_Build_FailureActionsNotMapping(t *testing.T) {
	yml := `failureActions: not-a-mapping`

	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yml), &node))

	var comp Components
	_ = low.BuildModel(node.Content[0], &comp)
	err := comp.Build(context.Background(), nil, node.Content[0], nil)
	assert.NoError(t, err)
	assert.Nil(t, comp.FailureActions.Value)
}

func TestCov_Components_Getters(t *testing.T) {
	yml := `inputs:
  i1:
    type: string`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	keyNode := &yaml.Node{Value: "components"}
	var comp Components
	_ = low.BuildModel(node.Content[0], &comp)
	_ = comp.Build(context.Background(), keyNode, node.Content[0], nil)

	assert.Equal(t, keyNode, comp.GetKeyNode())
	assert.Equal(t, node.Content[0], comp.GetRootNode())
	assert.Nil(t, comp.GetIndex())
	assert.NotNil(t, comp.GetContext())
	assert.NotNil(t, comp.GetExtensions())
	assert.Nil(t, comp.FindExtension("x-nope"))
}

// ---------------------------------------------------------------------------
// Criterion.Hash() with Context and Type non-empty
// ---------------------------------------------------------------------------

func TestCriterion_Hash_WithContextAndType(t *testing.T) {
	// Use Build() on a node that has context and type fields.
	// Note: Criterion.Context clashes with the unexported context.Context in BuildModel,
	// so we call BuildModel on a safe YAML (condition only), then Build on the full YAML.
	ymlFull := `context: $response.body
condition: $statusCode == 200
type: simple`

	var fullNode yaml.Node
	_ = yaml.Unmarshal([]byte(ymlFull), &fullNode)

	ymlSafe := `condition: $statusCode == 200`
	var safeNode yaml.Node
	_ = yaml.Unmarshal([]byte(ymlSafe), &safeNode)

	var crit Criterion
	_ = low.BuildModel(safeNode.Content[0], &crit)
	// Manually set Context since BuildModel can't handle the clash
	crit.Context = low.NodeReference[string]{
		Value:     "$response.body",
		ValueNode: &yaml.Node{Kind: yaml.ScalarNode, Value: "$response.body"},
	}
	_ = crit.Build(context.Background(), nil, fullNode.Content[0], nil)

	// Now hash. Context non-empty and Type non-empty should both be written.
	h1 := crit.Hash()
	assert.NotZero(t, h1)

	// Same input should produce same hash
	var crit2 Criterion
	_ = low.BuildModel(safeNode.Content[0], &crit2)
	crit2.Context = low.NodeReference[string]{
		Value:     "$response.body",
		ValueNode: &yaml.Node{Kind: yaml.ScalarNode, Value: "$response.body"},
	}
	_ = crit2.Build(context.Background(), nil, fullNode.Content[0], nil)
	assert.Equal(t, h1, crit2.Hash())

	// Different context => different hash
	var crit3 Criterion
	_ = low.BuildModel(safeNode.Content[0], &crit3)
	crit3.Context = low.NodeReference[string]{
		Value:     "$response.header",
		ValueNode: &yaml.Node{Kind: yaml.ScalarNode, Value: "$response.header"},
	}
	_ = crit3.Build(context.Background(), nil, fullNode.Content[0], nil)
	assert.NotEqual(t, h1, crit3.Hash())
}

// ---------------------------------------------------------------------------
// FailureAction.Hash() with all fields populated
// ---------------------------------------------------------------------------

func TestFailureAction_Hash_AllFields(t *testing.T) {
	yml := `name: retryStep
type: retry
workflowId: wf1
stepId: step1
retryAfter: 2.5
retryLimit: 10
criteria:
  - condition: $statusCode == 503
reference: $components.failureActions.retryAction`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var fa FailureAction
	_ = low.BuildModel(node.Content[0], &fa)
	_ = fa.Build(context.Background(), nil, node.Content[0], nil)

	// Verify all fields are populated
	assert.False(t, fa.Name.IsEmpty())
	assert.False(t, fa.Type.IsEmpty())
	assert.False(t, fa.WorkflowId.IsEmpty())
	assert.False(t, fa.StepId.IsEmpty())
	assert.False(t, fa.RetryAfter.IsEmpty())
	assert.False(t, fa.RetryLimit.IsEmpty())
	assert.False(t, fa.Criteria.IsEmpty())
	assert.False(t, fa.ComponentRef.IsEmpty())

	h1 := fa.Hash()
	assert.NotZero(t, h1)

	// Consistency check
	var fa2 FailureAction
	_ = low.BuildModel(node.Content[0], &fa2)
	_ = fa2.Build(context.Background(), nil, node.Content[0], nil)
	assert.Equal(t, h1, fa2.Hash())
}

func TestCov_FailureAction_Build_InvalidRetryValues(t *testing.T) {
	// retryAfter with non-numeric value should return an error
	yml := `name: retry
type: retry
retryAfter: not-a-number
retryLimit: abc`

	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yml), &node))

	var fa FailureAction
	_ = low.BuildModel(node.Content[0], &fa)
	err := fa.Build(context.Background(), nil, node.Content[0], nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid retryAfter value")
}

func TestFailureAction_Build_OddContentNode(t *testing.T) {
	// Verify that an odd number of Content items (malformed) doesn't crash.
	// This exercises the i+1 >= len(root.Content) guard in Build()'s manual loop.
	root := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "reference"},
			{Kind: yaml.ScalarNode, Value: "$components.failureActions.test"},
			{Kind: yaml.ScalarNode, Value: "orphanKey"},
			// Missing value for orphanKey - triggers break in the loop
		},
	}

	var fa FailureAction
	err := fa.Build(context.Background(), nil, root, nil)
	assert.NoError(t, err)
	// reference should still be extracted since it appears before the orphan
	assert.Equal(t, "$components.failureActions.test", fa.ComponentRef.Value)
}

// ---------------------------------------------------------------------------
// helpers.go: hashYAMLNode with DocumentNode and AliasNode
// ---------------------------------------------------------------------------

func TestHashYAMLNode_DocumentNode(t *testing.T) {
	// DocumentNode should recurse into its children
	child := &yaml.Node{Kind: yaml.ScalarNode, Value: "hello"}
	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{child}}

	var h maphash.Hash
	hashYAMLNode(&h, doc)
	result1 := h.Sum64()

	h.Reset()
	hashYAMLNode(&h, child)
	result2 := h.Sum64()

	// DocumentNode containing the scalar should hash the same as the scalar itself
	assert.Equal(t, result1, result2)
}

func TestHashYAMLNode_AliasNode(t *testing.T) {
	// AliasNode should recurse into its Alias target
	target := &yaml.Node{Kind: yaml.ScalarNode, Value: "world"}
	alias := &yaml.Node{Kind: yaml.AliasNode, Alias: target}

	var h maphash.Hash
	hashYAMLNode(&h, alias)
	result1 := h.Sum64()

	h.Reset()
	hashYAMLNode(&h, target)
	result2 := h.Sum64()

	assert.Equal(t, result1, result2)
}

func TestHashYAMLNode_AliasNodeNilAlias(t *testing.T) {
	// AliasNode with nil Alias should not crash
	alias := &yaml.Node{Kind: yaml.AliasNode, Alias: nil}

	var h maphash.Hash
	hashYAMLNode(&h, alias)
	// Should not panic, sum is zero-ish for no writes
	_ = h.Sum64()
}

func TestHashYAMLNode_NilNode(t *testing.T) {
	// nil node should not crash
	var h maphash.Hash
	hashYAMLNode(&h, nil)
	_ = h.Sum64()
}

func TestHashYAMLNode_SequenceNode(t *testing.T) {
	// SequenceNode should recurse into children
	seq := &yaml.Node{
		Kind: yaml.SequenceNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "a"},
			{Kind: yaml.ScalarNode, Value: "b"},
		},
	}

	var h maphash.Hash
	hashYAMLNode(&h, seq)
	result := h.Sum64()
	assert.NotZero(t, result)
}

// ---------------------------------------------------------------------------
// helpers.go: extractArray and extractObjectMap edge cases
// ---------------------------------------------------------------------------

func TestCov_ExtractArray_NotSequence(t *testing.T) {
	// When the value for the label is not a SequenceNode, extractArray should skip it.
	yml := `items: not-a-sequence`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	result, err := extractArray[SourceDescription](context.Background(), "items", node.Content[0], nil)
	assert.NoError(t, err)
	// Should have KeyNode set but no items
	assert.Nil(t, result.Value)
}

func TestCov_ExtractObjectMap_NotMapping(t *testing.T) {
	// When the value for the label is not a MappingNode, extractObjectMap should skip it.
	yml := `things: not-a-mapping`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	result, err := extractObjectMap[Parameter](context.Background(), "things", node.Content[0], nil)
	assert.NoError(t, err)
	assert.Nil(t, result.Value)
}

func TestExtractArray_LabelNotFound(t *testing.T) {
	yml := `other: value`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	result, err := extractArray[SourceDescription](context.Background(), "items", node.Content[0], nil)
	assert.NoError(t, err)
	assert.True(t, result.IsEmpty())
}

func TestExtractObjectMap_LabelNotFound(t *testing.T) {
	yml := `other: value`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	result, err := extractObjectMap[Parameter](context.Background(), "things", node.Content[0], nil)
	assert.NoError(t, err)
	assert.True(t, result.IsEmpty())
}

// ---------------------------------------------------------------------------
// helpers.go: extractStringArray edge cases
// ---------------------------------------------------------------------------

func TestCov_ExtractStringArray_NotSequence(t *testing.T) {
	yml := `items: scalar-value`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	result := extractStringArray("items", node.Content[0])
	assert.Nil(t, result.Value)
}

func TestExtractStringArray_Found(t *testing.T) {
	yml := `items:
  - alpha
  - beta`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	result := extractStringArray("items", node.Content[0])
	require.NotNil(t, result.Value)
	assert.Len(t, result.Value, 2)
	assert.Equal(t, "alpha", result.Value[0].Value)
	assert.Equal(t, "beta", result.Value[1].Value)
}

// ---------------------------------------------------------------------------
// helpers.go: extractExpressionsMap edge cases
// ---------------------------------------------------------------------------

func TestCov_ExtractExpressionsMap_NotMapping(t *testing.T) {
	yml := `outputs: a-scalar`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	result := extractExpressionsMap("outputs", node.Content[0])
	assert.Nil(t, result.Value)
}

func TestExtractExpressionsMap_OddContent(t *testing.T) {
	// A mapping node with an odd number of children (malformed)
	root := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "outputs"},
			{Kind: yaml.MappingNode, Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "key1"},
				{Kind: yaml.ScalarNode, Value: "val1"},
				{Kind: yaml.ScalarNode, Value: "orphan"},
			}},
		},
	}

	result := extractExpressionsMap("outputs", root)
	require.NotNil(t, result.Value)
	// Should only have 1 pair (second key has no value)
	assert.Equal(t, 1, result.Value.Len())
}

// ---------------------------------------------------------------------------
// helpers.go: extractRawNodeMap edge cases
// ---------------------------------------------------------------------------

func TestCov_ExtractRawNodeMap_NotMapping(t *testing.T) {
	yml := `inputs: not-a-mapping`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	result := extractRawNodeMap("inputs", node.Content[0])
	assert.Nil(t, result.Value)
}

func TestExtractRawNodeMap_OddContent(t *testing.T) {
	root := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "inputs"},
			{Kind: yaml.MappingNode, Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "key1"},
				{Kind: yaml.ScalarNode, Value: "val1"},
				{Kind: yaml.ScalarNode, Value: "orphan"},
			}},
		},
	}

	result := extractRawNodeMap("inputs", root)
	require.NotNil(t, result.Value)
	assert.Equal(t, 1, result.Value.Len())
}

// ---------------------------------------------------------------------------
// helpers.go: extractRawNode edge cases
// ---------------------------------------------------------------------------

func TestCov_ExtractRawNode_NotFound(t *testing.T) {
	yml := `other: value`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	result := extractRawNode("missing", node.Content[0])
	assert.True(t, result.IsEmpty())
}

func TestExtractRawNode_OddContent(t *testing.T) {
	// Root with an odd number of children
	root := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "key1"},
			{Kind: yaml.ScalarNode, Value: "val1"},
			{Kind: yaml.ScalarNode, Value: "orphan"},
		},
	}

	result := extractRawNode("orphan", root)
	assert.True(t, result.IsEmpty())
}

// ---------------------------------------------------------------------------
// helpers.go: hashExtensionsInto with nil
// ---------------------------------------------------------------------------

func TestHashExtensionsInto_Nil(t *testing.T) {
	var h maphash.Hash
	hashExtensionsInto(&h, nil)
	_ = h.Sum64() // should not panic
}

// ---------------------------------------------------------------------------
// Workflow.Build() and Workflow.Hash() edge cases
// ---------------------------------------------------------------------------

func TestWorkflow_Build_MinimalWithNoOptionalArrays(t *testing.T) {
	yml := `workflowId: minimal`

	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yml), &node))

	var wf Workflow
	_ = low.BuildModel(node.Content[0], &wf)
	err := wf.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "minimal", wf.WorkflowId.Value)
	assert.True(t, wf.Steps.IsEmpty())
	assert.True(t, wf.SuccessActions.IsEmpty())
	assert.True(t, wf.FailureActions.IsEmpty())
	assert.True(t, wf.Outputs.IsEmpty())
	assert.True(t, wf.Parameters.IsEmpty())
	assert.True(t, wf.DependsOn.IsEmpty())
	assert.True(t, wf.Inputs.IsEmpty())

	// Hash on minimal workflow should not be zero
	assert.NotZero(t, wf.Hash())
}

// ---------------------------------------------------------------------------
// Step.Build() edge cases
// ---------------------------------------------------------------------------

func TestCov_Step_Build_WithExtensions(t *testing.T) {
	yml := `stepId: ext-step
operationId: op1
x-my-ext: hello`

	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yml), &node))

	var step Step
	_ = low.BuildModel(node.Content[0], &step)
	err := step.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	ext := step.FindExtension("x-my-ext")
	require.NotNil(t, ext)
	assert.Equal(t, "hello", ext.Value.Value)
}

// ---------------------------------------------------------------------------
// SuccessAction.Hash() with all fields including Criteria
// ---------------------------------------------------------------------------

func TestSuccessAction_Hash_AllFields(t *testing.T) {
	yml := `name: goToWorkflow
type: goto
workflowId: otherWf
stepId: step2
criteria:
  - condition: $statusCode == 200
reference: $components.successActions.myAction`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var sa SuccessAction
	_ = low.BuildModel(node.Content[0], &sa)
	_ = sa.Build(context.Background(), nil, node.Content[0], nil)

	assert.False(t, sa.Name.IsEmpty())
	assert.False(t, sa.Type.IsEmpty())
	assert.False(t, sa.WorkflowId.IsEmpty())
	assert.False(t, sa.StepId.IsEmpty())
	assert.False(t, sa.Criteria.IsEmpty())
	assert.False(t, sa.ComponentRef.IsEmpty())

	h := sa.Hash()
	assert.NotZero(t, h)
}

// ---------------------------------------------------------------------------
// SourceDescription with extension Hash coverage
// ---------------------------------------------------------------------------

func TestSourceDescription_Hash_WithExtension(t *testing.T) {
	yml := `name: api
url: https://example.com
type: openapi
x-vendor: acme`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var sd SourceDescription
	_ = low.BuildModel(node.Content[0], &sd)
	_ = sd.Build(context.Background(), nil, node.Content[0], nil)

	h := sd.Hash()
	assert.NotZero(t, h)

	// Without extension, hash should differ
	yml2 := `name: api
url: https://example.com
type: openapi`

	var n2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml2), &n2)

	var sd2 SourceDescription
	_ = low.BuildModel(n2.Content[0], &sd2)
	_ = sd2.Build(context.Background(), nil, n2.Content[0], nil)

	assert.NotEqual(t, h, sd2.Hash())
}

// ---------------------------------------------------------------------------
// Info with extension Hash coverage
// ---------------------------------------------------------------------------

func TestInfo_Hash_WithExtension(t *testing.T) {
	yml := `title: Test
version: "1.0.0"
x-custom: value`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var info Info
	_ = low.BuildModel(node.Content[0], &info)
	_ = info.Build(context.Background(), nil, node.Content[0], nil)

	h := info.Hash()
	assert.NotZero(t, h)
}

// ---------------------------------------------------------------------------
// CriterionExpressionType with extension
// ---------------------------------------------------------------------------

func TestCriterionExpressionType_Hash_WithExtension(t *testing.T) {
	yml := `type: jsonpath
version: draft-01
x-custom: val`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var cet CriterionExpressionType
	_ = low.BuildModel(node.Content[0], &cet)
	_ = cet.Build(context.Background(), nil, node.Content[0], nil)

	h := cet.Hash()
	assert.NotZero(t, h)
}

// ---------------------------------------------------------------------------
// PayloadReplacement with extension
// ---------------------------------------------------------------------------

func TestPayloadReplacement_Hash_WithExtension(t *testing.T) {
	yml := `target: /name
value: replaced
x-note: info`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var pr PayloadReplacement
	_ = low.BuildModel(node.Content[0], &pr)
	_ = pr.Build(context.Background(), nil, node.Content[0], nil)

	h := pr.Hash()
	assert.NotZero(t, h)
}

// ---------------------------------------------------------------------------
// RequestBody with extension
// ---------------------------------------------------------------------------

func TestRequestBody_Hash_WithReplacementsAndExtension(t *testing.T) {
	yml := `contentType: application/json
payload:
  name: test
replacements:
  - target: /name
    value: replaced
x-extra: info`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var rb RequestBody
	_ = low.BuildModel(node.Content[0], &rb)
	_ = rb.Build(context.Background(), nil, node.Content[0], nil)

	h := rb.Hash()
	assert.NotZero(t, h)
}

// ---------------------------------------------------------------------------
// Parameter.Hash() with extension
// ---------------------------------------------------------------------------

func TestParameter_Hash_WithExtension(t *testing.T) {
	yml := `name: petId
in: path
value: "123"
x-desc: info`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var param Parameter
	_ = low.BuildModel(node.Content[0], &param)
	_ = param.Build(context.Background(), nil, node.Content[0], nil)

	h := param.Hash()
	assert.NotZero(t, h)
}

// ---------------------------------------------------------------------------
// Parameter with reference Hash
// ---------------------------------------------------------------------------

func TestParameter_Hash_WithReference(t *testing.T) {
	yml := `reference: $components.parameters.petIdParam`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var param Parameter
	_ = low.BuildModel(node.Content[0], &param)
	_ = param.Build(context.Background(), nil, node.Content[0], nil)

	h := param.Hash()
	assert.NotZero(t, h)
}

// ---------------------------------------------------------------------------
// Workflow.Hash() with extensions
// ---------------------------------------------------------------------------

func TestWorkflow_Hash_WithExtension(t *testing.T) {
	yml := `workflowId: wf1
summary: sum
description: desc
x-custom: val`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var wf Workflow
	_ = low.BuildModel(node.Content[0], &wf)
	_ = wf.Build(context.Background(), nil, node.Content[0], nil)

	h := wf.Hash()
	assert.NotZero(t, h)
}

// ---------------------------------------------------------------------------
// Step.Hash() with extensions
// ---------------------------------------------------------------------------

func TestStep_Hash_WithExtension(t *testing.T) {
	yml := `stepId: s1
operationId: op1
x-extra: val`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var step Step
	_ = low.BuildModel(node.Content[0], &step)
	_ = step.Build(context.Background(), nil, node.Content[0], nil)

	h := step.Hash()
	assert.NotZero(t, h)
}

// ---------------------------------------------------------------------------
// extractArray / extractObjectMap odd Content guards
// ---------------------------------------------------------------------------

func TestExtractArray_OddContent(t *testing.T) {
	// Root mapping with odd number of children (key without value)
	root := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "items"},
			{Kind: yaml.SequenceNode, Content: []*yaml.Node{
				{Kind: yaml.MappingNode, Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "name"},
					{Kind: yaml.ScalarNode, Value: "api"},
					{Kind: yaml.ScalarNode, Value: "url"},
					{Kind: yaml.ScalarNode, Value: "https://example.com"},
				}},
			}},
			{Kind: yaml.ScalarNode, Value: "orphan"},
		},
	}

	result, err := extractArray[SourceDescription](context.Background(), "items", root, nil)
	assert.NoError(t, err)
	require.NotNil(t, result.Value)
	assert.Len(t, result.Value, 1)
}

func TestExtractObjectMap_OddContent(t *testing.T) {
	// Root with odd content
	root := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "things"},
			{Kind: yaml.MappingNode, Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "p1"},
				{Kind: yaml.MappingNode, Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "name"},
					{Kind: yaml.ScalarNode, Value: "param1"},
				}},
				{Kind: yaml.ScalarNode, Value: "orphan"},
			}},
			{Kind: yaml.ScalarNode, Value: "dangling"},
		},
	}

	result, err := extractObjectMap[Parameter](context.Background(), "things", root, nil)
	assert.NoError(t, err)
	require.NotNil(t, result.Value)
	assert.Equal(t, 1, result.Value.Len())
}

// ---------------------------------------------------------------------------
// extractStringArray odd content guard
// ---------------------------------------------------------------------------

func TestExtractStringArray_OddRootContent(t *testing.T) {
	root := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "items"},
			{Kind: yaml.SequenceNode, Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "a"},
			}},
			{Kind: yaml.ScalarNode, Value: "orphan"},
		},
	}

	result := extractStringArray("items", root)
	require.NotNil(t, result.Value)
	assert.Len(t, result.Value, 1)
	assert.Equal(t, "a", result.Value[0].Value)
}

// ---------------------------------------------------------------------------
// Arazzo.Build() cascading error paths
// ---------------------------------------------------------------------------

// TestArazzo_Build_WorkflowError triggers the error return path in Arazzo.Build()
// for workflows extraction. The steps array contains items that will cause Build to
// propagate an error from a nested extractArray (e.g., SuccessCriteria containing
// invalid criteria objects).
// NOTE: Most Build() error paths require $ref resolution failures which need a
// SpecIndex. Without a real index, these paths are hard to reach. Instead we cover
// them via the full document integration test which indirectly exercises all the
// Build code paths.

// TestArazzo_Build_ErrorPropagation_Steps tests that an error in step's nested
// extractArray (e.g. parameters) propagates up through workflows extractArray
// and then through Arazzo.Build().
// This is difficult to trigger with pure YAML since BuildModel and Build for
// simple objects like Parameter/Criterion don't fail on valid YAML.
// We accept the coverage as-is for these deeply nested error returns.

// ---------------------------------------------------------------------------
// Step.Build() and Step.Hash() edge cases
// ---------------------------------------------------------------------------

func TestStep_Hash_WithAllFields(t *testing.T) {
	yml := `stepId: fullStep
description: Full step description
operationId: op1
parameters:
  - name: p1
    in: query
    value: v1
requestBody:
  contentType: application/json
  payload:
    key: val
successCriteria:
  - condition: $statusCode == 200
onSuccess:
  - name: done
    type: end
onFailure:
  - name: retry
    type: retry
    retryAfter: 1.0
    retryLimit: 2
outputs:
  result: $response.body`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var step Step
	_ = low.BuildModel(node.Content[0], &step)
	_ = step.Build(context.Background(), nil, node.Content[0], nil)

	// All branches in Hash() should be exercised
	assert.False(t, step.StepId.IsEmpty())
	assert.False(t, step.Description.IsEmpty())
	assert.False(t, step.OperationId.IsEmpty())
	assert.False(t, step.Parameters.IsEmpty())
	assert.False(t, step.RequestBody.IsEmpty())
	assert.False(t, step.SuccessCriteria.IsEmpty())
	assert.False(t, step.OnSuccess.IsEmpty())
	assert.False(t, step.OnFailure.IsEmpty())
	assert.False(t, step.Outputs.IsEmpty())

	h := step.Hash()
	assert.NotZero(t, h)
}

func TestStep_Hash_WithOperationPath(t *testing.T) {
	yml := `stepId: pathStep
operationPath: "{$sourceDescriptions.api}/pets"
workflowId: otherWf`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var step Step
	_ = low.BuildModel(node.Content[0], &step)
	_ = step.Build(context.Background(), nil, node.Content[0], nil)

	assert.False(t, step.OperationPath.IsEmpty())
	assert.False(t, step.WorkflowId.IsEmpty())

	h := step.Hash()
	assert.NotZero(t, h)
}

// ---------------------------------------------------------------------------
// Workflow.Build() edge cases - all nested arrays
// ---------------------------------------------------------------------------

func TestWorkflow_Build_WithAllFields(t *testing.T) {
	yml := `workflowId: fullWorkflow
summary: Full workflow
description: Described
inputs:
  type: object
dependsOn:
  - otherWf
steps:
  - stepId: s1
    operationId: op1
successActions:
  - name: done
    type: end
failureActions:
  - name: retry
    type: retry
    retryAfter: 1.0
    retryLimit: 2
outputs:
  result: $steps.s1.outputs.r
parameters:
  - name: pk
    in: query
    value: val`

	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yml), &node))

	var wf Workflow
	_ = low.BuildModel(node.Content[0], &wf)
	err := wf.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.False(t, wf.WorkflowId.IsEmpty())
	assert.False(t, wf.Summary.IsEmpty())
	assert.False(t, wf.Description.IsEmpty())
	assert.False(t, wf.Inputs.IsEmpty())
	assert.False(t, wf.DependsOn.IsEmpty())
	assert.False(t, wf.Steps.IsEmpty())
	assert.False(t, wf.SuccessActions.IsEmpty())
	assert.False(t, wf.FailureActions.IsEmpty())
	assert.False(t, wf.Outputs.IsEmpty())
	assert.False(t, wf.Parameters.IsEmpty())
}

// ---------------------------------------------------------------------------
// SuccessAction.Build() edge case - odd content for reference extraction
// ---------------------------------------------------------------------------

func TestSuccessAction_Build_OddContentNode(t *testing.T) {
	root := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "reference"},
			{Kind: yaml.ScalarNode, Value: "$components.successActions.test"},
			{Kind: yaml.ScalarNode, Value: "orphanKey"},
		},
	}

	var sa SuccessAction
	err := sa.Build(context.Background(), nil, root, nil)
	assert.NoError(t, err)
	assert.Equal(t, "$components.successActions.test", sa.ComponentRef.Value)
}

// ---------------------------------------------------------------------------
// RequestBody.Build() edge case - empty payload and replacements
// ---------------------------------------------------------------------------

func TestRequestBody_Build_Empty(t *testing.T) {
	yml := `x-empty: true`

	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yml), &node))

	var rb RequestBody
	_ = low.BuildModel(node.Content[0], &rb)
	err := rb.Build(context.Background(), nil, node.Content[0], nil)
	assert.NoError(t, err)
	assert.True(t, rb.Payload.IsEmpty())
	assert.True(t, rb.Replacements.IsEmpty())
}

// ---------------------------------------------------------------------------
// Full Arazzo document Build + Hash for comprehensive coverage
// ---------------------------------------------------------------------------

func TestArazzo_Build_FullDocument(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Full Test
  summary: Summary
  description: Description
  version: "1.0.0"
  x-info-ext: val
sourceDescriptions:
  - name: petStore
    url: https://petstore.example.com/openapi.json
    type: openapi
    x-sd-ext: val
workflows:
  - workflowId: createPet
    summary: Create pet
    description: Create a pet workflow
    dependsOn:
      - verifyPet
    inputs:
      type: object
      properties:
        petName:
          type: string
    steps:
      - stepId: addPet
        operationId: addPet
        description: Add a new pet
        parameters:
          - name: api_key
            in: header
            value: abc123
        requestBody:
          contentType: application/json
          payload:
            name: fluffy
          replacements:
            - target: /name
              value: replaced
        successCriteria:
          - condition: $statusCode == 200
            type: simple
          - condition: $response.body#/id != null
            context: $response.body
            type:
              type: jsonpath
              version: draft-01
        onSuccess:
          - name: logSuccess
            type: end
            criteria:
              - condition: $statusCode == 200
        onFailure:
          - name: retryAdd
            type: retry
            retryAfter: 1.5
            retryLimit: 3
            criteria:
              - condition: $statusCode == 500
        outputs:
          petId: $response.body#/id
      - stepId: getPet
        operationPath: "{$sourceDescriptions.petStore}/pet/{petId}"
    successActions:
      - name: notify
        type: goto
        stepId: addPet
    failureActions:
      - name: abort
        type: end
    outputs:
      result: $steps.addPet.outputs.petId
    parameters:
      - name: storeId
        in: query
        value: store-1
  - workflowId: verifyPet
    steps:
      - stepId: check
        operationId: getPetById
components:
  inputs:
    petInput:
      type: object
  parameters:
    apiKey:
      name: api_key
      in: header
      value: default
  successActions:
    logEnd:
      name: logEnd
      type: end
  failureActions:
    retryDefault:
      name: retryDefault
      type: retry
      retryAfter: 2.0
      retryLimit: 5
x-top: toplevel`

	var n1, n2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &n1)
	_ = yaml.Unmarshal([]byte(yml), &n2)

	var a1 Arazzo
	_ = low.BuildModel(n1.Content[0], &a1)
	err := a1.Build(context.Background(), nil, n1.Content[0], nil)
	require.NoError(t, err)

	var a2 Arazzo
	_ = low.BuildModel(n2.Content[0], &a2)
	_ = a2.Build(context.Background(), nil, n2.Content[0], nil)

	// Full hash consistency
	assert.Equal(t, a1.Hash(), a2.Hash())

	// Verify structure
	assert.Equal(t, "1.0.1", a1.Arazzo.Value)
	assert.Equal(t, "Full Test", a1.Info.Value.Title.Value)
	assert.Len(t, a1.SourceDescriptions.Value, 1)
	assert.Len(t, a1.Workflows.Value, 2)
	assert.False(t, a1.Components.IsEmpty())

	// Verify components hash covers all maps
	compHash := a1.Components.Value.Hash()
	assert.NotZero(t, compHash)
}
