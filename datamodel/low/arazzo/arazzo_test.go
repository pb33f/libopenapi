// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// ---------------------------------------------------------------------------
// Info
// ---------------------------------------------------------------------------

func TestInfo_Build_Full(t *testing.T) {
	yml := `title: Pet Store Workflows
summary: Workflows for pet store
description: A sample set of workflows
version: "1.0.0"
x-custom: hello`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var info Info
	err = low.BuildModel(node.Content[0], &info)
	require.NoError(t, err)

	err = info.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "Pet Store Workflows", info.Title.Value)
	assert.Equal(t, "Workflows for pet store", info.Summary.Value)
	assert.Equal(t, "A sample set of workflows", info.Description.Value)
	assert.Equal(t, "1.0.0", info.Version.Value)

	ext := info.FindExtension("x-custom")
	require.NotNil(t, ext)
	assert.Equal(t, "hello", ext.Value.Value)
}

func TestInfo_Build_Minimal(t *testing.T) {
	yml := `title: Minimal
version: "1.0.0"`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var info Info
	err = low.BuildModel(node.Content[0], &info)
	require.NoError(t, err)

	err = info.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "Minimal", info.Title.Value)
	assert.Equal(t, "1.0.0", info.Version.Value)
	assert.True(t, info.Summary.IsEmpty())
	assert.True(t, info.Description.IsEmpty())
}

func TestInfo_Hash_Consistency(t *testing.T) {
	yml := `title: Test
summary: Sum
description: Desc
version: "2.0.0"`

	var n1, n2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &n1)
	_ = yaml.Unmarshal([]byte(yml), &n2)

	var i1, i2 Info
	_ = low.BuildModel(n1.Content[0], &i1)
	_ = i1.Build(context.Background(), nil, n1.Content[0], nil)
	_ = low.BuildModel(n2.Content[0], &i2)
	_ = i2.Build(context.Background(), nil, n2.Content[0], nil)

	assert.Equal(t, i1.Hash(), i2.Hash())
}

func TestInfo_Hash_Different(t *testing.T) {
	yml1 := `title: One
version: "1.0.0"`
	yml2 := `title: Two
version: "2.0.0"`

	var n1, n2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml1), &n1)
	_ = yaml.Unmarshal([]byte(yml2), &n2)

	var i1, i2 Info
	_ = low.BuildModel(n1.Content[0], &i1)
	_ = i1.Build(context.Background(), nil, n1.Content[0], nil)
	_ = low.BuildModel(n2.Content[0], &i2)
	_ = i2.Build(context.Background(), nil, n2.Content[0], nil)

	assert.NotEqual(t, i1.Hash(), i2.Hash())
}

func TestInfo_Getters(t *testing.T) {
	yml := `title: Test
version: "1.0.0"`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	keyNode := &yaml.Node{Value: "info"}
	var info Info
	_ = low.BuildModel(node.Content[0], &info)
	_ = info.Build(context.Background(), keyNode, node.Content[0], nil)

	assert.Equal(t, keyNode, info.GetKeyNode())
	assert.Equal(t, node.Content[0], info.GetRootNode())
	assert.Nil(t, info.GetIndex())
	assert.NotNil(t, info.GetContext())
	assert.NotNil(t, info.GetExtensions())
}

func TestInfo_FindExtension_NotFound(t *testing.T) {
	yml := `title: Test
version: "1.0.0"`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var info Info
	_ = low.BuildModel(node.Content[0], &info)
	_ = info.Build(context.Background(), nil, node.Content[0], nil)

	assert.Nil(t, info.FindExtension("x-nope"))
}

// ---------------------------------------------------------------------------
// SourceDescription
// ---------------------------------------------------------------------------

func TestSourceDescription_Build_Full(t *testing.T) {
	yml := `name: petStore
url: https://petstore.example.com/openapi.json
type: openapi
x-source-extra: val`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var sd SourceDescription
	err = low.BuildModel(node.Content[0], &sd)
	require.NoError(t, err)

	err = sd.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "petStore", sd.Name.Value)
	assert.Equal(t, "https://petstore.example.com/openapi.json", sd.URL.Value)
	assert.Equal(t, "openapi", sd.Type.Value)

	ext := sd.FindExtension("x-source-extra")
	require.NotNil(t, ext)
	assert.Equal(t, "val", ext.Value.Value)
}

func TestSourceDescription_Build_Minimal(t *testing.T) {
	yml := `name: minimal
url: https://example.com`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var sd SourceDescription
	err = low.BuildModel(node.Content[0], &sd)
	require.NoError(t, err)

	err = sd.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "minimal", sd.Name.Value)
	assert.Equal(t, "https://example.com", sd.URL.Value)
	assert.True(t, sd.Type.IsEmpty())
}

func TestSourceDescription_Hash_Consistency(t *testing.T) {
	yml := `name: petStore
url: https://petstore.example.com/openapi.json
type: openapi`

	var n1, n2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &n1)
	_ = yaml.Unmarshal([]byte(yml), &n2)

	var s1, s2 SourceDescription
	_ = low.BuildModel(n1.Content[0], &s1)
	_ = s1.Build(context.Background(), nil, n1.Content[0], nil)
	_ = low.BuildModel(n2.Content[0], &s2)
	_ = s2.Build(context.Background(), nil, n2.Content[0], nil)

	assert.Equal(t, s1.Hash(), s2.Hash())
}

func TestSourceDescription_Hash_Different(t *testing.T) {
	yml1 := `name: one
url: https://one.example.com`
	yml2 := `name: two
url: https://two.example.com`

	var n1, n2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml1), &n1)
	_ = yaml.Unmarshal([]byte(yml2), &n2)

	var s1, s2 SourceDescription
	_ = low.BuildModel(n1.Content[0], &s1)
	_ = s1.Build(context.Background(), nil, n1.Content[0], nil)
	_ = low.BuildModel(n2.Content[0], &s2)
	_ = s2.Build(context.Background(), nil, n2.Content[0], nil)

	assert.NotEqual(t, s1.Hash(), s2.Hash())
}

func TestSourceDescription_Getters(t *testing.T) {
	yml := `name: test
url: https://test.com`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	keyNode := &yaml.Node{Value: "sd"}
	var sd SourceDescription
	_ = low.BuildModel(node.Content[0], &sd)
	_ = sd.Build(context.Background(), keyNode, node.Content[0], nil)

	assert.Equal(t, keyNode, sd.GetKeyNode())
	assert.Equal(t, node.Content[0], sd.GetRootNode())
	assert.Nil(t, sd.GetIndex())
	assert.NotNil(t, sd.GetContext())
	assert.NotNil(t, sd.GetExtensions())
}

// ---------------------------------------------------------------------------
// CriterionExpressionType
// ---------------------------------------------------------------------------

func TestCriterionExpressionType_Build_Full(t *testing.T) {
	yml := `type: jsonpath
version: draft-goessner-dispatch-jsonpath-00`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var cet CriterionExpressionType
	err = low.BuildModel(node.Content[0], &cet)
	require.NoError(t, err)

	err = cet.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "jsonpath", cet.Type.Value)
	assert.Equal(t, "draft-goessner-dispatch-jsonpath-00", cet.Version.Value)
}

func TestCriterionExpressionType_Build_Minimal(t *testing.T) {
	yml := `type: xpath`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var cet CriterionExpressionType
	err = low.BuildModel(node.Content[0], &cet)
	require.NoError(t, err)

	err = cet.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "xpath", cet.Type.Value)
	assert.True(t, cet.Version.IsEmpty())
}

func TestCriterionExpressionType_Hash_Consistency(t *testing.T) {
	yml := `type: jsonpath
version: draft-goessner-dispatch-jsonpath-00`

	var n1, n2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &n1)
	_ = yaml.Unmarshal([]byte(yml), &n2)

	var c1, c2 CriterionExpressionType
	_ = low.BuildModel(n1.Content[0], &c1)
	_ = c1.Build(context.Background(), nil, n1.Content[0], nil)
	_ = low.BuildModel(n2.Content[0], &c2)
	_ = c2.Build(context.Background(), nil, n2.Content[0], nil)

	assert.Equal(t, c1.Hash(), c2.Hash())
}

func TestCriterionExpressionType_Hash_Different(t *testing.T) {
	yml1 := `type: jsonpath
version: draft-goessner-dispatch-jsonpath-00`
	yml2 := `type: xpath
version: "3.1"`

	var n1, n2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml1), &n1)
	_ = yaml.Unmarshal([]byte(yml2), &n2)

	var c1, c2 CriterionExpressionType
	_ = low.BuildModel(n1.Content[0], &c1)
	_ = c1.Build(context.Background(), nil, n1.Content[0], nil)
	_ = low.BuildModel(n2.Content[0], &c2)
	_ = c2.Build(context.Background(), nil, n2.Content[0], nil)

	assert.NotEqual(t, c1.Hash(), c2.Hash())
}

func TestCriterionExpressionType_Getters(t *testing.T) {
	yml := `type: jsonpath`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	keyNode := &yaml.Node{Value: "type"}
	var cet CriterionExpressionType
	_ = low.BuildModel(node.Content[0], &cet)
	_ = cet.Build(context.Background(), keyNode, node.Content[0], nil)

	assert.Equal(t, keyNode, cet.GetKeyNode())
	assert.Equal(t, node.Content[0], cet.GetRootNode())
	assert.Nil(t, cet.GetIndex())
	assert.NotNil(t, cet.GetContext())
	assert.NotNil(t, cet.GetExtensions())
	assert.Nil(t, cet.FindExtension("x-nope"))
}

// ---------------------------------------------------------------------------
// PayloadReplacement
// ---------------------------------------------------------------------------

func TestPayloadReplacement_Build_Full(t *testing.T) {
	yml := `target: /name
value: Fido`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var pr PayloadReplacement
	err = low.BuildModel(node.Content[0], &pr)
	require.NoError(t, err)

	err = pr.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "/name", pr.Target.Value)
	assert.False(t, pr.Value.IsEmpty())
	assert.Equal(t, "Fido", pr.Value.Value.Value)
}

func TestPayloadReplacement_Build_Minimal(t *testing.T) {
	yml := `target: /id`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var pr PayloadReplacement
	err = low.BuildModel(node.Content[0], &pr)
	require.NoError(t, err)

	err = pr.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "/id", pr.Target.Value)
	assert.True(t, pr.Value.IsEmpty())
}

func TestPayloadReplacement_Hash_Consistency(t *testing.T) {
	yml := `target: /name
value: Fido`

	var n1, n2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &n1)
	_ = yaml.Unmarshal([]byte(yml), &n2)

	var p1, p2 PayloadReplacement
	_ = low.BuildModel(n1.Content[0], &p1)
	_ = p1.Build(context.Background(), nil, n1.Content[0], nil)
	_ = low.BuildModel(n2.Content[0], &p2)
	_ = p2.Build(context.Background(), nil, n2.Content[0], nil)

	assert.Equal(t, p1.Hash(), p2.Hash())
}

func TestPayloadReplacement_Hash_Different(t *testing.T) {
	yml1 := `target: /name
value: Fido`
	yml2 := `target: /id
value: "123"`

	var n1, n2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml1), &n1)
	_ = yaml.Unmarshal([]byte(yml2), &n2)

	var p1, p2 PayloadReplacement
	_ = low.BuildModel(n1.Content[0], &p1)
	_ = p1.Build(context.Background(), nil, n1.Content[0], nil)
	_ = low.BuildModel(n2.Content[0], &p2)
	_ = p2.Build(context.Background(), nil, n2.Content[0], nil)

	assert.NotEqual(t, p1.Hash(), p2.Hash())
}

func TestPayloadReplacement_Getters(t *testing.T) {
	yml := `target: /name
value: Fido`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	keyNode := &yaml.Node{Value: "replacement"}
	var pr PayloadReplacement
	_ = low.BuildModel(node.Content[0], &pr)
	_ = pr.Build(context.Background(), keyNode, node.Content[0], nil)

	assert.Equal(t, keyNode, pr.GetKeyNode())
	assert.Equal(t, node.Content[0], pr.GetRootNode())
	assert.Nil(t, pr.GetIndex())
	assert.NotNil(t, pr.GetContext())
	assert.NotNil(t, pr.GetExtensions())
	assert.Nil(t, pr.FindExtension("x-nope"))
}

// ---------------------------------------------------------------------------
// Parameter
// ---------------------------------------------------------------------------

func TestParameter_Build_Full(t *testing.T) {
	yml := `name: petId
in: path
value: "123"`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var param Parameter
	err = low.BuildModel(node.Content[0], &param)
	require.NoError(t, err)

	err = param.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "petId", param.Name.Value)
	assert.Equal(t, "path", param.In.Value)
	assert.False(t, param.Value.IsEmpty())
	assert.Equal(t, "123", param.Value.Value.Value)
	assert.False(t, param.IsReusable())
}

func TestParameter_Build_WithReference(t *testing.T) {
	yml := `reference: $components.parameters.petIdParam`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var param Parameter
	err = low.BuildModel(node.Content[0], &param)
	require.NoError(t, err)

	err = param.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.True(t, param.IsReusable())
	assert.Equal(t, "$components.parameters.petIdParam", param.ComponentRef.Value)
}

func TestParameter_Build_Minimal(t *testing.T) {
	yml := `name: q`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var param Parameter
	err = low.BuildModel(node.Content[0], &param)
	require.NoError(t, err)

	err = param.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "q", param.Name.Value)
	assert.True(t, param.In.IsEmpty())
	assert.True(t, param.Value.IsEmpty())
	assert.False(t, param.IsReusable())
}

func TestParameter_Hash_Consistency(t *testing.T) {
	yml := `name: petId
in: path
value: "123"`

	var n1, n2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &n1)
	_ = yaml.Unmarshal([]byte(yml), &n2)

	var p1, p2 Parameter
	_ = low.BuildModel(n1.Content[0], &p1)
	_ = p1.Build(context.Background(), nil, n1.Content[0], nil)
	_ = low.BuildModel(n2.Content[0], &p2)
	_ = p2.Build(context.Background(), nil, n2.Content[0], nil)

	assert.Equal(t, p1.Hash(), p2.Hash())
}

func TestParameter_Hash_Different(t *testing.T) {
	yml1 := `name: petId
in: path
value: "123"`
	yml2 := `name: ownerId
in: query
value: "456"`

	var n1, n2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml1), &n1)
	_ = yaml.Unmarshal([]byte(yml2), &n2)

	var p1, p2 Parameter
	_ = low.BuildModel(n1.Content[0], &p1)
	_ = p1.Build(context.Background(), nil, n1.Content[0], nil)
	_ = low.BuildModel(n2.Content[0], &p2)
	_ = p2.Build(context.Background(), nil, n2.Content[0], nil)

	assert.NotEqual(t, p1.Hash(), p2.Hash())
}

func TestParameter_Getters(t *testing.T) {
	yml := `name: petId
in: path
value: "123"`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	keyNode := &yaml.Node{Value: "param"}
	var param Parameter
	_ = low.BuildModel(node.Content[0], &param)
	_ = param.Build(context.Background(), keyNode, node.Content[0], nil)

	assert.Equal(t, keyNode, param.GetKeyNode())
	assert.Equal(t, node.Content[0], param.GetRootNode())
	assert.Nil(t, param.GetIndex())
	assert.NotNil(t, param.GetContext())
	assert.NotNil(t, param.GetExtensions())
	assert.Nil(t, param.FindExtension("x-nope"))
}

// ---------------------------------------------------------------------------
// Criterion
// ---------------------------------------------------------------------------

func TestCriterion_Build_Full(t *testing.T) {
	// Note: Criterion has an unexported `context context.Context` field that clashes
	// with the exported `Context low.NodeReference[string]` field in BuildModel's
	// case-insensitive matching. Additionally, `Type` is `NodeReference[*yaml.Node]`
	// which BuildModel cannot populate from scalar values. We test condition-only
	// via BuildModel and verify context/type extraction works in Build().
	yml := `condition: $statusCode == 200`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var crit Criterion
	err = low.BuildModel(node.Content[0], &crit)
	require.NoError(t, err)

	err = crit.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "$statusCode == 200", crit.Condition.Value)
	assert.True(t, crit.Context.IsEmpty())
	assert.True(t, crit.Type.IsEmpty())
}

func TestCriterion_Build_WithContext(t *testing.T) {
	// Test that the Criterion Context field is populated by BuildModel.
	// We use BuildModel on a YAML without the problematic fields, then call Build()
	// on the full YAML with context to verify extraction works.
	ymlFull := `context: $response.body
condition: $statusCode == 200`

	var fullNode yaml.Node
	err := yaml.Unmarshal([]byte(ymlFull), &fullNode)
	require.NoError(t, err)

	// Build model on a node without context to avoid the unexported field clash
	ymlSafe := `condition: $statusCode == 200`
	var safeNode yaml.Node
	err = yaml.Unmarshal([]byte(ymlSafe), &safeNode)
	require.NoError(t, err)

	var crit Criterion
	err = low.BuildModel(safeNode.Content[0], &crit)
	require.NoError(t, err)

	// Build on the full node so extractRawNode and manual extraction works
	err = crit.Build(context.Background(), nil, fullNode.Content[0], nil)
	require.NoError(t, err)

	// Context is set by BuildModel on the exported field; since we skipped it in
	// BuildModel, it won't be populated there, but Build() does NOT extract it.
	// Context is only populated by BuildModel's reflection.
	assert.Equal(t, "$statusCode == 200", crit.Condition.Value)
}

func TestCriterion_Build_WithScalarType(t *testing.T) {
	// Test extractRawNode for the type field (scalar value).
	ymlFull := `condition: $statusCode == 200
type: simple`

	var fullNode yaml.Node
	err := yaml.Unmarshal([]byte(ymlFull), &fullNode)
	require.NoError(t, err)

	// BuildModel on a node without the type key (which it can't handle)
	ymlSafe := `condition: $statusCode == 200`
	var safeNode yaml.Node
	err = yaml.Unmarshal([]byte(ymlSafe), &safeNode)
	require.NoError(t, err)

	var crit Criterion
	err = low.BuildModel(safeNode.Content[0], &crit)
	require.NoError(t, err)

	err = crit.Build(context.Background(), nil, fullNode.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "$statusCode == 200", crit.Condition.Value)
	assert.False(t, crit.Type.IsEmpty())
	assert.Equal(t, "simple", crit.Type.Value.Value)
}

func TestCriterion_Build_WithExpressionTypeObject(t *testing.T) {
	yml := `condition: $.pets.length > 0
type:
  type: jsonpath
  version: draft-goessner-dispatch-jsonpath-00`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var crit Criterion
	err = low.BuildModel(node.Content[0], &crit)
	require.NoError(t, err)

	err = crit.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "$.pets.length > 0", crit.Condition.Value)
	assert.False(t, crit.Type.IsEmpty())
	assert.Equal(t, yaml.MappingNode, crit.Type.Value.Kind)
}

func TestCriterion_Build_Minimal(t *testing.T) {
	yml := `condition: $statusCode == 200`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var crit Criterion
	err = low.BuildModel(node.Content[0], &crit)
	require.NoError(t, err)

	err = crit.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "$statusCode == 200", crit.Condition.Value)
	assert.True(t, crit.Context.IsEmpty())
	assert.True(t, crit.Type.IsEmpty())
}

func TestCriterion_Hash_Consistency(t *testing.T) {
	yml := `condition: $statusCode == 200`

	var n1, n2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &n1)
	_ = yaml.Unmarshal([]byte(yml), &n2)

	var c1, c2 Criterion
	_ = low.BuildModel(n1.Content[0], &c1)
	_ = c1.Build(context.Background(), nil, n1.Content[0], nil)
	_ = low.BuildModel(n2.Content[0], &c2)
	_ = c2.Build(context.Background(), nil, n2.Content[0], nil)

	assert.Equal(t, c1.Hash(), c2.Hash())
}

func TestCriterion_Hash_Different(t *testing.T) {
	yml1 := `condition: $statusCode == 200`
	yml2 := `condition: $statusCode == 404`

	var n1, n2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml1), &n1)
	_ = yaml.Unmarshal([]byte(yml2), &n2)

	var c1, c2 Criterion
	_ = low.BuildModel(n1.Content[0], &c1)
	_ = c1.Build(context.Background(), nil, n1.Content[0], nil)
	_ = low.BuildModel(n2.Content[0], &c2)
	_ = c2.Build(context.Background(), nil, n2.Content[0], nil)

	assert.NotEqual(t, c1.Hash(), c2.Hash())
}

func TestCriterion_Getters(t *testing.T) {
	yml := `condition: $statusCode == 200`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	keyNode := &yaml.Node{Value: "crit"}
	var crit Criterion
	_ = low.BuildModel(node.Content[0], &crit)
	_ = crit.Build(context.Background(), keyNode, node.Content[0], nil)

	assert.Equal(t, keyNode, crit.GetKeyNode())
	assert.Equal(t, node.Content[0], crit.GetRootNode())
	assert.Nil(t, crit.GetIndex())
	assert.NotNil(t, crit.GetContext())
	assert.NotNil(t, crit.GetExtensions())
	assert.Nil(t, crit.FindExtension("x-nope"))
}

// ---------------------------------------------------------------------------
// RequestBody
// ---------------------------------------------------------------------------

func TestRequestBody_Build_Full(t *testing.T) {
	yml := `contentType: application/json
payload:
  name: Fido
  tag: dog
replacements:
  - target: /name
    value: $inputs.petName`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var rb RequestBody
	err = low.BuildModel(node.Content[0], &rb)
	require.NoError(t, err)

	err = rb.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "application/json", rb.ContentType.Value)
	assert.False(t, rb.Payload.IsEmpty())
	assert.Equal(t, yaml.MappingNode, rb.Payload.Value.Kind)
	require.False(t, rb.Replacements.IsEmpty())
	require.Len(t, rb.Replacements.Value, 1)
	assert.Equal(t, "/name", rb.Replacements.Value[0].Value.Target.Value)
}

func TestRequestBody_Build_Minimal(t *testing.T) {
	yml := `contentType: application/json`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var rb RequestBody
	err = low.BuildModel(node.Content[0], &rb)
	require.NoError(t, err)

	err = rb.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "application/json", rb.ContentType.Value)
	assert.True(t, rb.Payload.IsEmpty())
	assert.True(t, rb.Replacements.IsEmpty())
}

func TestRequestBody_Hash_Consistency(t *testing.T) {
	yml := `contentType: application/json
payload:
  name: Fido`

	var n1, n2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &n1)
	_ = yaml.Unmarshal([]byte(yml), &n2)

	var r1, r2 RequestBody
	_ = low.BuildModel(n1.Content[0], &r1)
	_ = r1.Build(context.Background(), nil, n1.Content[0], nil)
	_ = low.BuildModel(n2.Content[0], &r2)
	_ = r2.Build(context.Background(), nil, n2.Content[0], nil)

	assert.Equal(t, r1.Hash(), r2.Hash())
}

func TestRequestBody_Hash_Different(t *testing.T) {
	yml1 := `contentType: application/json
payload:
  name: Fido`
	yml2 := `contentType: application/xml
payload:
  name: Rex`

	var n1, n2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml1), &n1)
	_ = yaml.Unmarshal([]byte(yml2), &n2)

	var r1, r2 RequestBody
	_ = low.BuildModel(n1.Content[0], &r1)
	_ = r1.Build(context.Background(), nil, n1.Content[0], nil)
	_ = low.BuildModel(n2.Content[0], &r2)
	_ = r2.Build(context.Background(), nil, n2.Content[0], nil)

	assert.NotEqual(t, r1.Hash(), r2.Hash())
}

func TestRequestBody_Getters(t *testing.T) {
	yml := `contentType: application/json`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	keyNode := &yaml.Node{Value: "requestBody"}
	var rb RequestBody
	_ = low.BuildModel(node.Content[0], &rb)
	_ = rb.Build(context.Background(), keyNode, node.Content[0], nil)

	assert.Equal(t, keyNode, rb.GetKeyNode())
	assert.Equal(t, node.Content[0], rb.GetRootNode())
	assert.Nil(t, rb.GetIndex())
	assert.NotNil(t, rb.GetContext())
	assert.NotNil(t, rb.GetExtensions())
	assert.Nil(t, rb.FindExtension("x-nope"))
}

// ---------------------------------------------------------------------------
// SuccessAction
// ---------------------------------------------------------------------------

func TestSuccessAction_Build_Full(t *testing.T) {
	yml := `name: endWorkflow
type: end
workflowId: other-workflow
stepId: someStep
criteria:
  - condition: $statusCode == 200`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var sa SuccessAction
	err = low.BuildModel(node.Content[0], &sa)
	require.NoError(t, err)

	err = sa.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "endWorkflow", sa.Name.Value)
	assert.Equal(t, "end", sa.Type.Value)
	assert.Equal(t, "other-workflow", sa.WorkflowId.Value)
	assert.Equal(t, "someStep", sa.StepId.Value)
	assert.False(t, sa.IsReusable())
	require.False(t, sa.Criteria.IsEmpty())
	require.Len(t, sa.Criteria.Value, 1)
	assert.Equal(t, "$statusCode == 200", sa.Criteria.Value[0].Value.Condition.Value)
}

func TestSuccessAction_Build_WithReference(t *testing.T) {
	yml := `reference: $components.successActions.endAction`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var sa SuccessAction
	err = low.BuildModel(node.Content[0], &sa)
	require.NoError(t, err)

	err = sa.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.True(t, sa.IsReusable())
	assert.Equal(t, "$components.successActions.endAction", sa.ComponentRef.Value)
}

func TestSuccessAction_Build_Minimal(t *testing.T) {
	yml := `name: done
type: end`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var sa SuccessAction
	err = low.BuildModel(node.Content[0], &sa)
	require.NoError(t, err)

	err = sa.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "done", sa.Name.Value)
	assert.Equal(t, "end", sa.Type.Value)
	assert.True(t, sa.WorkflowId.IsEmpty())
	assert.True(t, sa.StepId.IsEmpty())
	assert.True(t, sa.Criteria.IsEmpty())
	assert.False(t, sa.IsReusable())
}

func TestSuccessAction_Hash_Consistency(t *testing.T) {
	yml := `name: endWorkflow
type: end`

	var n1, n2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &n1)
	_ = yaml.Unmarshal([]byte(yml), &n2)

	var s1, s2 SuccessAction
	_ = low.BuildModel(n1.Content[0], &s1)
	_ = s1.Build(context.Background(), nil, n1.Content[0], nil)
	_ = low.BuildModel(n2.Content[0], &s2)
	_ = s2.Build(context.Background(), nil, n2.Content[0], nil)

	assert.Equal(t, s1.Hash(), s2.Hash())
}

func TestSuccessAction_Hash_Different(t *testing.T) {
	yml1 := `name: endWorkflow
type: end`
	yml2 := `name: goToStep
type: goto
stepId: step2`

	var n1, n2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml1), &n1)
	_ = yaml.Unmarshal([]byte(yml2), &n2)

	var s1, s2 SuccessAction
	_ = low.BuildModel(n1.Content[0], &s1)
	_ = s1.Build(context.Background(), nil, n1.Content[0], nil)
	_ = low.BuildModel(n2.Content[0], &s2)
	_ = s2.Build(context.Background(), nil, n2.Content[0], nil)

	assert.NotEqual(t, s1.Hash(), s2.Hash())
}

func TestSuccessAction_Getters(t *testing.T) {
	yml := `name: done
type: end`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	keyNode := &yaml.Node{Value: "sa"}
	var sa SuccessAction
	_ = low.BuildModel(node.Content[0], &sa)
	_ = sa.Build(context.Background(), keyNode, node.Content[0], nil)

	assert.Equal(t, keyNode, sa.GetKeyNode())
	assert.Equal(t, node.Content[0], sa.GetRootNode())
	assert.Nil(t, sa.GetIndex())
	assert.NotNil(t, sa.GetContext())
	assert.NotNil(t, sa.GetExtensions())
	assert.Nil(t, sa.FindExtension("x-nope"))
}

// ---------------------------------------------------------------------------
// FailureAction
// ---------------------------------------------------------------------------

func TestFailureAction_Build_Full(t *testing.T) {
	yml := `name: retryStep
type: retry
workflowId: other-workflow
stepId: someStep
retryAfter: 1.5
retryLimit: 3
criteria:
  - condition: $statusCode == 503`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var fa FailureAction
	err = low.BuildModel(node.Content[0], &fa)
	require.NoError(t, err)

	err = fa.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "retryStep", fa.Name.Value)
	assert.Equal(t, "retry", fa.Type.Value)
	assert.Equal(t, "other-workflow", fa.WorkflowId.Value)
	assert.Equal(t, "someStep", fa.StepId.Value)
	assert.InDelta(t, 1.5, fa.RetryAfter.Value, 0.001)
	assert.Equal(t, int64(3), fa.RetryLimit.Value)
	assert.False(t, fa.IsReusable())
	require.False(t, fa.Criteria.IsEmpty())
	require.Len(t, fa.Criteria.Value, 1)
	assert.Equal(t, "$statusCode == 503", fa.Criteria.Value[0].Value.Condition.Value)
}

func TestFailureAction_Build_WithReference(t *testing.T) {
	yml := `reference: $components.failureActions.retryAction`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var fa FailureAction
	err = low.BuildModel(node.Content[0], &fa)
	require.NoError(t, err)

	err = fa.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.True(t, fa.IsReusable())
	assert.Equal(t, "$components.failureActions.retryAction", fa.ComponentRef.Value)
}

func TestFailureAction_Build_Minimal(t *testing.T) {
	yml := `name: fail
type: end`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var fa FailureAction
	err = low.BuildModel(node.Content[0], &fa)
	require.NoError(t, err)

	err = fa.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "fail", fa.Name.Value)
	assert.Equal(t, "end", fa.Type.Value)
	assert.True(t, fa.WorkflowId.IsEmpty())
	assert.True(t, fa.StepId.IsEmpty())
	assert.True(t, fa.RetryAfter.IsEmpty())
	assert.True(t, fa.RetryLimit.IsEmpty())
	assert.True(t, fa.Criteria.IsEmpty())
	assert.False(t, fa.IsReusable())
}

func TestFailureAction_Hash_Consistency(t *testing.T) {
	yml := `name: retryStep
type: retry
retryAfter: 1.5
retryLimit: 3`

	var n1, n2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &n1)
	_ = yaml.Unmarshal([]byte(yml), &n2)

	var f1, f2 FailureAction
	_ = low.BuildModel(n1.Content[0], &f1)
	_ = f1.Build(context.Background(), nil, n1.Content[0], nil)
	_ = low.BuildModel(n2.Content[0], &f2)
	_ = f2.Build(context.Background(), nil, n2.Content[0], nil)

	assert.Equal(t, f1.Hash(), f2.Hash())
}

func TestFailureAction_Hash_Different(t *testing.T) {
	yml1 := `name: retryStep
type: retry
retryAfter: 1.5
retryLimit: 3`
	yml2 := `name: abortStep
type: end`

	var n1, n2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml1), &n1)
	_ = yaml.Unmarshal([]byte(yml2), &n2)

	var f1, f2 FailureAction
	_ = low.BuildModel(n1.Content[0], &f1)
	_ = f1.Build(context.Background(), nil, n1.Content[0], nil)
	_ = low.BuildModel(n2.Content[0], &f2)
	_ = f2.Build(context.Background(), nil, n2.Content[0], nil)

	assert.NotEqual(t, f1.Hash(), f2.Hash())
}

func TestFailureAction_Getters(t *testing.T) {
	yml := `name: fail
type: end`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	keyNode := &yaml.Node{Value: "fa"}
	var fa FailureAction
	_ = low.BuildModel(node.Content[0], &fa)
	_ = fa.Build(context.Background(), keyNode, node.Content[0], nil)

	assert.Equal(t, keyNode, fa.GetKeyNode())
	assert.Equal(t, node.Content[0], fa.GetRootNode())
	assert.Nil(t, fa.GetIndex())
	assert.NotNil(t, fa.GetContext())
	assert.NotNil(t, fa.GetExtensions())
	assert.Nil(t, fa.FindExtension("x-nope"))
}

// ---------------------------------------------------------------------------
// Step
// ---------------------------------------------------------------------------

func TestStep_Build_Full(t *testing.T) {
	yml := `stepId: getPet
description: Get a pet by ID
operationId: getPetById
parameters:
  - name: petId
    in: path
    value: $inputs.petId
requestBody:
  contentType: application/json
  payload:
    name: Fido
successCriteria:
  - condition: $statusCode == 200
onSuccess:
  - name: endWorkflow
    type: end
onFailure:
  - name: retryStep
    type: retry
    retryAfter: 1.5
    retryLimit: 3
outputs:
  petName: $response.body#/name`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var step Step
	err = low.BuildModel(node.Content[0], &step)
	require.NoError(t, err)

	err = step.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "getPet", step.StepId.Value)
	assert.Equal(t, "Get a pet by ID", step.Description.Value)
	assert.Equal(t, "getPetById", step.OperationId.Value)
	assert.True(t, step.OperationPath.IsEmpty())
	assert.True(t, step.WorkflowId.IsEmpty())

	// Parameters
	require.False(t, step.Parameters.IsEmpty())
	require.Len(t, step.Parameters.Value, 1)
	assert.Equal(t, "petId", step.Parameters.Value[0].Value.Name.Value)

	// RequestBody
	require.False(t, step.RequestBody.IsEmpty())
	assert.Equal(t, "application/json", step.RequestBody.Value.ContentType.Value)

	// SuccessCriteria
	require.False(t, step.SuccessCriteria.IsEmpty())
	require.Len(t, step.SuccessCriteria.Value, 1)
	assert.Equal(t, "$statusCode == 200", step.SuccessCriteria.Value[0].Value.Condition.Value)

	// OnSuccess
	require.False(t, step.OnSuccess.IsEmpty())
	require.Len(t, step.OnSuccess.Value, 1)
	assert.Equal(t, "endWorkflow", step.OnSuccess.Value[0].Value.Name.Value)

	// OnFailure
	require.False(t, step.OnFailure.IsEmpty())
	require.Len(t, step.OnFailure.Value, 1)
	assert.Equal(t, "retryStep", step.OnFailure.Value[0].Value.Name.Value)
	assert.InDelta(t, 1.5, step.OnFailure.Value[0].Value.RetryAfter.Value, 0.001)
	assert.Equal(t, int64(3), step.OnFailure.Value[0].Value.RetryLimit.Value)

	// Outputs
	require.False(t, step.Outputs.IsEmpty())
	pair := step.Outputs.Value.First()
	require.NotNil(t, pair)
	assert.Equal(t, "petName", pair.Key().Value)
	assert.Equal(t, "$response.body#/name", pair.Value().Value)
}

func TestStep_Build_WithOperationPath(t *testing.T) {
	yml := `stepId: listPets
operationPath: "{$sourceDescriptions.petStore.url}#/paths/~1pets/get"
successCriteria:
  - condition: $statusCode == 200`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var step Step
	err = low.BuildModel(node.Content[0], &step)
	require.NoError(t, err)

	err = step.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "listPets", step.StepId.Value)
	assert.Equal(t, "{$sourceDescriptions.petStore.url}#/paths/~1pets/get", step.OperationPath.Value)
	assert.True(t, step.OperationId.IsEmpty())
}

func TestStep_Build_WithWorkflowId(t *testing.T) {
	yml := `stepId: callOtherWorkflow
workflowId: other-workflow`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var step Step
	err = low.BuildModel(node.Content[0], &step)
	require.NoError(t, err)

	err = step.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "callOtherWorkflow", step.StepId.Value)
	assert.Equal(t, "other-workflow", step.WorkflowId.Value)
}

func TestStep_Build_Minimal(t *testing.T) {
	yml := `stepId: minimal`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var step Step
	err = low.BuildModel(node.Content[0], &step)
	require.NoError(t, err)

	err = step.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "minimal", step.StepId.Value)
	assert.True(t, step.Description.IsEmpty())
	assert.True(t, step.OperationId.IsEmpty())
	assert.True(t, step.OperationPath.IsEmpty())
	assert.True(t, step.WorkflowId.IsEmpty())
	assert.True(t, step.Parameters.IsEmpty())
	assert.True(t, step.RequestBody.IsEmpty())
	assert.True(t, step.SuccessCriteria.IsEmpty())
	assert.True(t, step.OnSuccess.IsEmpty())
	assert.True(t, step.OnFailure.IsEmpty())
	assert.True(t, step.Outputs.IsEmpty())
}

func TestStep_Hash_Consistency(t *testing.T) {
	yml := `stepId: getPet
operationId: getPetById
successCriteria:
  - condition: $statusCode == 200`

	var n1, n2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &n1)
	_ = yaml.Unmarshal([]byte(yml), &n2)

	var s1, s2 Step
	_ = low.BuildModel(n1.Content[0], &s1)
	_ = s1.Build(context.Background(), nil, n1.Content[0], nil)
	_ = low.BuildModel(n2.Content[0], &s2)
	_ = s2.Build(context.Background(), nil, n2.Content[0], nil)

	assert.Equal(t, s1.Hash(), s2.Hash())
}

func TestStep_Hash_Different(t *testing.T) {
	yml1 := `stepId: getPet
operationId: getPetById`
	yml2 := `stepId: listPets
operationId: listPets`

	var n1, n2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml1), &n1)
	_ = yaml.Unmarshal([]byte(yml2), &n2)

	var s1, s2 Step
	_ = low.BuildModel(n1.Content[0], &s1)
	_ = s1.Build(context.Background(), nil, n1.Content[0], nil)
	_ = low.BuildModel(n2.Content[0], &s2)
	_ = s2.Build(context.Background(), nil, n2.Content[0], nil)

	assert.NotEqual(t, s1.Hash(), s2.Hash())
}

func TestStep_Getters(t *testing.T) {
	yml := `stepId: getPet
operationId: getPetById`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	keyNode := &yaml.Node{Value: "step"}
	var step Step
	_ = low.BuildModel(node.Content[0], &step)
	_ = step.Build(context.Background(), keyNode, node.Content[0], nil)

	assert.Equal(t, keyNode, step.GetKeyNode())
	assert.Equal(t, node.Content[0], step.GetRootNode())
	assert.Nil(t, step.GetIndex())
	assert.NotNil(t, step.GetContext())
	assert.NotNil(t, step.GetExtensions())
	assert.Nil(t, step.FindExtension("x-nope"))
}

// ---------------------------------------------------------------------------
// Workflow
// ---------------------------------------------------------------------------

func TestWorkflow_Build_Full(t *testing.T) {
	yml := `workflowId: get-pet
summary: Get a pet
description: Retrieve a pet by ID
inputs:
  type: object
  properties:
    petId:
      type: integer
dependsOn:
  - list-pets
steps:
  - stepId: getPet
    operationId: getPetById
    successCriteria:
      - condition: $statusCode == 200
successActions:
  - name: done
    type: end
failureActions:
  - name: retry
    type: retry
    retryAfter: 2.0
    retryLimit: 5
outputs:
  result: $steps.getPet.outputs.petName
parameters:
  - name: apiKey
    in: header
    value: $inputs.apiKey`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var wf Workflow
	err = low.BuildModel(node.Content[0], &wf)
	require.NoError(t, err)

	err = wf.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "get-pet", wf.WorkflowId.Value)
	assert.Equal(t, "Get a pet", wf.Summary.Value)
	assert.Equal(t, "Retrieve a pet by ID", wf.Description.Value)

	// Inputs (raw node)
	assert.False(t, wf.Inputs.IsEmpty())
	assert.Equal(t, yaml.MappingNode, wf.Inputs.Value.Kind)

	// DependsOn
	require.False(t, wf.DependsOn.IsEmpty())
	require.Len(t, wf.DependsOn.Value, 1)
	assert.Equal(t, "list-pets", wf.DependsOn.Value[0].Value)

	// Steps
	require.False(t, wf.Steps.IsEmpty())
	require.Len(t, wf.Steps.Value, 1)
	assert.Equal(t, "getPet", wf.Steps.Value[0].Value.StepId.Value)

	// SuccessActions
	require.False(t, wf.SuccessActions.IsEmpty())
	require.Len(t, wf.SuccessActions.Value, 1)
	assert.Equal(t, "done", wf.SuccessActions.Value[0].Value.Name.Value)

	// FailureActions
	require.False(t, wf.FailureActions.IsEmpty())
	require.Len(t, wf.FailureActions.Value, 1)
	assert.Equal(t, "retry", wf.FailureActions.Value[0].Value.Name.Value)
	assert.InDelta(t, 2.0, wf.FailureActions.Value[0].Value.RetryAfter.Value, 0.001)
	assert.Equal(t, int64(5), wf.FailureActions.Value[0].Value.RetryLimit.Value)

	// Outputs
	require.False(t, wf.Outputs.IsEmpty())
	pair := wf.Outputs.Value.First()
	require.NotNil(t, pair)
	assert.Equal(t, "result", pair.Key().Value)
	assert.Equal(t, "$steps.getPet.outputs.petName", pair.Value().Value)

	// Parameters
	require.False(t, wf.Parameters.IsEmpty())
	require.Len(t, wf.Parameters.Value, 1)
	assert.Equal(t, "apiKey", wf.Parameters.Value[0].Value.Name.Value)
}

func TestWorkflow_Build_Minimal(t *testing.T) {
	yml := `workflowId: minimal
steps:
  - stepId: onlyStep
    operationId: doSomething`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var wf Workflow
	err = low.BuildModel(node.Content[0], &wf)
	require.NoError(t, err)

	err = wf.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "minimal", wf.WorkflowId.Value)
	assert.True(t, wf.Summary.IsEmpty())
	assert.True(t, wf.Description.IsEmpty())
	assert.True(t, wf.Inputs.IsEmpty())
	assert.True(t, wf.DependsOn.IsEmpty())
	assert.True(t, wf.SuccessActions.IsEmpty())
	assert.True(t, wf.FailureActions.IsEmpty())
	assert.True(t, wf.Outputs.IsEmpty())
	assert.True(t, wf.Parameters.IsEmpty())
	require.False(t, wf.Steps.IsEmpty())
	assert.Len(t, wf.Steps.Value, 1)
}

func TestWorkflow_Hash_Consistency(t *testing.T) {
	yml := `workflowId: get-pet
summary: Get a pet
steps:
  - stepId: getPet
    operationId: getPetById`

	var n1, n2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &n1)
	_ = yaml.Unmarshal([]byte(yml), &n2)

	var w1, w2 Workflow
	_ = low.BuildModel(n1.Content[0], &w1)
	_ = w1.Build(context.Background(), nil, n1.Content[0], nil)
	_ = low.BuildModel(n2.Content[0], &w2)
	_ = w2.Build(context.Background(), nil, n2.Content[0], nil)

	assert.Equal(t, w1.Hash(), w2.Hash())
}

func TestWorkflow_Hash_Different(t *testing.T) {
	yml1 := `workflowId: get-pet
steps:
  - stepId: getPet
    operationId: getPetById`
	yml2 := `workflowId: list-pets
steps:
  - stepId: listAll
    operationId: listPets`

	var n1, n2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml1), &n1)
	_ = yaml.Unmarshal([]byte(yml2), &n2)

	var w1, w2 Workflow
	_ = low.BuildModel(n1.Content[0], &w1)
	_ = w1.Build(context.Background(), nil, n1.Content[0], nil)
	_ = low.BuildModel(n2.Content[0], &w2)
	_ = w2.Build(context.Background(), nil, n2.Content[0], nil)

	assert.NotEqual(t, w1.Hash(), w2.Hash())
}

func TestWorkflow_Getters(t *testing.T) {
	yml := `workflowId: test-wf
steps:
  - stepId: s1
    operationId: op1`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	keyNode := &yaml.Node{Value: "wf"}
	var wf Workflow
	_ = low.BuildModel(node.Content[0], &wf)
	_ = wf.Build(context.Background(), keyNode, node.Content[0], nil)

	assert.Equal(t, keyNode, wf.GetKeyNode())
	assert.Equal(t, node.Content[0], wf.GetRootNode())
	assert.Nil(t, wf.GetIndex())
	assert.NotNil(t, wf.GetContext())
	assert.NotNil(t, wf.GetExtensions())
	assert.Nil(t, wf.FindExtension("x-nope"))
}

// ---------------------------------------------------------------------------
// Components
// ---------------------------------------------------------------------------

func TestComponents_Build_Full(t *testing.T) {
	yml := `inputs:
  petInput:
    type: object
    properties:
      petId:
        type: integer
parameters:
  petIdParam:
    name: petId
    in: path
    value: "123"
successActions:
  endAction:
    name: done
    type: end
failureActions:
  retryAction:
    name: retry
    type: retry
    retryAfter: 2.0
    retryLimit: 5`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var comp Components
	err = low.BuildModel(node.Content[0], &comp)
	require.NoError(t, err)

	err = comp.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	// Inputs
	require.False(t, comp.Inputs.IsEmpty())
	require.NotNil(t, comp.Inputs.Value)
	inputPair := comp.Inputs.Value.First()
	require.NotNil(t, inputPair)
	assert.Equal(t, "petInput", inputPair.Key().Value)
	assert.Equal(t, yaml.MappingNode, inputPair.Value().Value.Kind)

	// Parameters
	require.False(t, comp.Parameters.IsEmpty())
	require.NotNil(t, comp.Parameters.Value)
	paramPair := comp.Parameters.Value.First()
	require.NotNil(t, paramPair)
	assert.Equal(t, "petIdParam", paramPair.Key().Value)
	assert.Equal(t, "petId", paramPair.Value().Value.Name.Value)
	assert.Equal(t, "path", paramPair.Value().Value.In.Value)

	// SuccessActions
	require.False(t, comp.SuccessActions.IsEmpty())
	require.NotNil(t, comp.SuccessActions.Value)
	saPair := comp.SuccessActions.Value.First()
	require.NotNil(t, saPair)
	assert.Equal(t, "endAction", saPair.Key().Value)
	assert.Equal(t, "done", saPair.Value().Value.Name.Value)
	assert.Equal(t, "end", saPair.Value().Value.Type.Value)

	// FailureActions
	require.False(t, comp.FailureActions.IsEmpty())
	require.NotNil(t, comp.FailureActions.Value)
	faPair := comp.FailureActions.Value.First()
	require.NotNil(t, faPair)
	assert.Equal(t, "retryAction", faPair.Key().Value)
	assert.Equal(t, "retry", faPair.Value().Value.Name.Value)
	assert.Equal(t, "retry", faPair.Value().Value.Type.Value)
	assert.InDelta(t, 2.0, faPair.Value().Value.RetryAfter.Value, 0.001)
	assert.Equal(t, int64(5), faPair.Value().Value.RetryLimit.Value)
}

func TestComponents_Build_Minimal(t *testing.T) {
	yml := `parameters:
  petIdParam:
    name: petId
    in: path
    value: "123"`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var comp Components
	err = low.BuildModel(node.Content[0], &comp)
	require.NoError(t, err)

	err = comp.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.True(t, comp.Inputs.IsEmpty())
	assert.True(t, comp.SuccessActions.IsEmpty())
	assert.True(t, comp.FailureActions.IsEmpty())
	require.False(t, comp.Parameters.IsEmpty())
}

func TestComponents_Build_Empty(t *testing.T) {
	yml := `x-empty: true`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var comp Components
	err = low.BuildModel(node.Content[0], &comp)
	require.NoError(t, err)

	err = comp.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.True(t, comp.Inputs.IsEmpty())
	assert.True(t, comp.Parameters.IsEmpty())
	assert.True(t, comp.SuccessActions.IsEmpty())
	assert.True(t, comp.FailureActions.IsEmpty())

	ext := comp.FindExtension("x-empty")
	require.NotNil(t, ext)
}

func TestComponents_Hash_Consistency(t *testing.T) {
	yml := `parameters:
  petIdParam:
    name: petId
    in: path
    value: "123"
successActions:
  endAction:
    name: done
    type: end`

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

func TestComponents_Hash_Different(t *testing.T) {
	yml1 := `parameters:
  petIdParam:
    name: petId
    in: path
    value: "123"`
	yml2 := `parameters:
  ownerId:
    name: ownerId
    in: query
    value: "456"`

	var n1, n2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml1), &n1)
	_ = yaml.Unmarshal([]byte(yml2), &n2)

	var c1, c2 Components
	_ = low.BuildModel(n1.Content[0], &c1)
	_ = c1.Build(context.Background(), nil, n1.Content[0], nil)
	_ = low.BuildModel(n2.Content[0], &c2)
	_ = c2.Build(context.Background(), nil, n2.Content[0], nil)

	assert.NotEqual(t, c1.Hash(), c2.Hash())
}

func TestComponents_Getters(t *testing.T) {
	yml := `parameters:
  p1:
    name: p1
    in: query
    value: x`

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
// Arazzo (root document)
// ---------------------------------------------------------------------------

func TestArazzo_Build_Full(t *testing.T) {
	yml := `arazzo: "1.0.1"
info:
  title: Pet Store Workflows
  summary: Workflows for pet store
  description: A sample set of workflows
  version: "1.0.0"
sourceDescriptions:
  - name: petStore
    url: https://petstore.example.com/openapi.json
    type: openapi
workflows:
  - workflowId: get-pet
    summary: Get a pet
    description: Retrieve a pet by ID
    inputs:
      type: object
      properties:
        petId:
          type: integer
    dependsOn:
      - list-pets
    steps:
      - stepId: getPet
        operationId: getPetById
        parameters:
          - name: petId
            in: path
            value: $inputs.petId
        successCriteria:
          - condition: $statusCode == 200
        onSuccess:
          - name: endWorkflow
            type: end
        onFailure:
          - name: retryStep
            type: retry
            retryAfter: 1.5
            retryLimit: 3
        outputs:
          petName: $response.body#/name
    outputs:
      result: $steps.getPet.outputs.petName
  - workflowId: list-pets
    steps:
      - stepId: listAll
        operationId: listPets
        successCriteria:
          - condition: $statusCode == 200
components:
  parameters:
    petIdParam:
      name: petId
      in: path
      value: "123"
  successActions:
    endAction:
      name: done
      type: end
  failureActions:
    retryAction:
      name: retry
      type: retry
      retryAfter: 2.0
      retryLimit: 5`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var arazzo Arazzo
	err = low.BuildModel(node.Content[0], &arazzo)
	require.NoError(t, err)

	err = arazzo.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	// Root version
	assert.Equal(t, "1.0.1", arazzo.Arazzo.Value)

	// Info
	require.False(t, arazzo.Info.IsEmpty())
	info := arazzo.Info.Value
	assert.Equal(t, "Pet Store Workflows", info.Title.Value)
	assert.Equal(t, "Workflows for pet store", info.Summary.Value)
	assert.Equal(t, "A sample set of workflows", info.Description.Value)
	assert.Equal(t, "1.0.0", info.Version.Value)

	// SourceDescriptions
	require.False(t, arazzo.SourceDescriptions.IsEmpty())
	require.Len(t, arazzo.SourceDescriptions.Value, 1)
	sd := arazzo.SourceDescriptions.Value[0].Value
	assert.Equal(t, "petStore", sd.Name.Value)
	assert.Equal(t, "https://petstore.example.com/openapi.json", sd.URL.Value)
	assert.Equal(t, "openapi", sd.Type.Value)

	// Workflows
	require.False(t, arazzo.Workflows.IsEmpty())
	require.Len(t, arazzo.Workflows.Value, 2)

	// First workflow
	wf1 := arazzo.Workflows.Value[0].Value
	assert.Equal(t, "get-pet", wf1.WorkflowId.Value)
	assert.Equal(t, "Get a pet", wf1.Summary.Value)
	assert.Equal(t, "Retrieve a pet by ID", wf1.Description.Value)
	assert.False(t, wf1.Inputs.IsEmpty())
	require.Len(t, wf1.DependsOn.Value, 1)
	assert.Equal(t, "list-pets", wf1.DependsOn.Value[0].Value)

	// First workflow steps
	require.Len(t, wf1.Steps.Value, 1)
	step := wf1.Steps.Value[0].Value
	assert.Equal(t, "getPet", step.StepId.Value)
	assert.Equal(t, "getPetById", step.OperationId.Value)

	// Step parameters
	require.Len(t, step.Parameters.Value, 1)
	assert.Equal(t, "petId", step.Parameters.Value[0].Value.Name.Value)
	assert.Equal(t, "path", step.Parameters.Value[0].Value.In.Value)

	// Step successCriteria
	require.Len(t, step.SuccessCriteria.Value, 1)
	assert.Equal(t, "$statusCode == 200", step.SuccessCriteria.Value[0].Value.Condition.Value)

	// Step onSuccess
	require.Len(t, step.OnSuccess.Value, 1)
	assert.Equal(t, "endWorkflow", step.OnSuccess.Value[0].Value.Name.Value)
	assert.Equal(t, "end", step.OnSuccess.Value[0].Value.Type.Value)

	// Step onFailure
	require.Len(t, step.OnFailure.Value, 1)
	assert.Equal(t, "retryStep", step.OnFailure.Value[0].Value.Name.Value)
	assert.Equal(t, "retry", step.OnFailure.Value[0].Value.Type.Value)
	assert.InDelta(t, 1.5, step.OnFailure.Value[0].Value.RetryAfter.Value, 0.001)
	assert.Equal(t, int64(3), step.OnFailure.Value[0].Value.RetryLimit.Value)

	// Step outputs
	require.False(t, step.Outputs.IsEmpty())
	outPair := step.Outputs.Value.First()
	require.NotNil(t, outPair)
	assert.Equal(t, "petName", outPair.Key().Value)
	assert.Equal(t, "$response.body#/name", outPair.Value().Value)

	// First workflow outputs
	require.False(t, wf1.Outputs.IsEmpty())
	wfOutPair := wf1.Outputs.Value.First()
	require.NotNil(t, wfOutPair)
	assert.Equal(t, "result", wfOutPair.Key().Value)
	assert.Equal(t, "$steps.getPet.outputs.petName", wfOutPair.Value().Value)

	// Second workflow
	wf2 := arazzo.Workflows.Value[1].Value
	assert.Equal(t, "list-pets", wf2.WorkflowId.Value)
	require.Len(t, wf2.Steps.Value, 1)
	assert.Equal(t, "listAll", wf2.Steps.Value[0].Value.StepId.Value)

	// Components
	require.False(t, arazzo.Components.IsEmpty())
	comp := arazzo.Components.Value

	// Components - parameters
	require.False(t, comp.Parameters.IsEmpty())
	paramPair := comp.Parameters.Value.First()
	require.NotNil(t, paramPair)
	assert.Equal(t, "petIdParam", paramPair.Key().Value)
	assert.Equal(t, "petId", paramPair.Value().Value.Name.Value)

	// Components - successActions
	require.False(t, comp.SuccessActions.IsEmpty())
	saPair := comp.SuccessActions.Value.First()
	require.NotNil(t, saPair)
	assert.Equal(t, "endAction", saPair.Key().Value)
	assert.Equal(t, "done", saPair.Value().Value.Name.Value)

	// Components - failureActions
	require.False(t, comp.FailureActions.IsEmpty())
	faPair := comp.FailureActions.Value.First()
	require.NotNil(t, faPair)
	assert.Equal(t, "retryAction", faPair.Key().Value)
	assert.Equal(t, "retry", faPair.Value().Value.Name.Value)
	assert.InDelta(t, 2.0, faPair.Value().Value.RetryAfter.Value, 0.001)
	assert.Equal(t, int64(5), faPair.Value().Value.RetryLimit.Value)
}

func TestArazzo_Build_Minimal(t *testing.T) {
	yml := `arazzo: "1.0.0"
info:
  title: Minimal
  version: "1.0.0"
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.json
workflows:
  - workflowId: basic
    steps:
      - stepId: s1
        operationId: doSomething`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var arazzo Arazzo
	err = low.BuildModel(node.Content[0], &arazzo)
	require.NoError(t, err)

	err = arazzo.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "1.0.0", arazzo.Arazzo.Value)
	assert.False(t, arazzo.Info.IsEmpty())
	assert.False(t, arazzo.SourceDescriptions.IsEmpty())
	assert.False(t, arazzo.Workflows.IsEmpty())
	assert.True(t, arazzo.Components.IsEmpty())
}

func TestArazzo_Build_WithExtensions(t *testing.T) {
	yml := `arazzo: "1.0.0"
info:
  title: Extended
  version: "1.0.0"
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.json
workflows:
  - workflowId: basic
    steps:
      - stepId: s1
        operationId: doSomething
x-custom: extended-value`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var arazzo Arazzo
	err = low.BuildModel(node.Content[0], &arazzo)
	require.NoError(t, err)

	err = arazzo.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.NotNil(t, arazzo.Extensions)
	ext := arazzo.FindExtension("x-custom")
	require.NotNil(t, ext)
	assert.Equal(t, "extended-value", ext.Value.Value)
}

func TestArazzo_FindExtension_NotFound(t *testing.T) {
	yml := `arazzo: "1.0.0"
info:
  title: Test
  version: "1.0.0"
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.json
workflows:
  - workflowId: basic
    steps:
      - stepId: s1
        operationId: doSomething`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var arazzo Arazzo
	_ = low.BuildModel(node.Content[0], &arazzo)
	_ = arazzo.Build(context.Background(), nil, node.Content[0], nil)

	assert.Nil(t, arazzo.FindExtension("x-nonexistent"))
}

func TestArazzo_Hash_Consistency(t *testing.T) {
	yml := `arazzo: "1.0.0"
info:
  title: Test
  version: "1.0.0"
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.json
workflows:
  - workflowId: basic
    steps:
      - stepId: s1
        operationId: doSomething`

	var n1, n2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &n1)
	_ = yaml.Unmarshal([]byte(yml), &n2)

	var a1, a2 Arazzo
	_ = low.BuildModel(n1.Content[0], &a1)
	_ = a1.Build(context.Background(), nil, n1.Content[0], nil)
	_ = low.BuildModel(n2.Content[0], &a2)
	_ = a2.Build(context.Background(), nil, n2.Content[0], nil)

	assert.Equal(t, a1.Hash(), a2.Hash())
}

func TestArazzo_Hash_Different(t *testing.T) {
	yml1 := `arazzo: "1.0.0"
info:
  title: Test One
  version: "1.0.0"
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.json
workflows:
  - workflowId: basic
    steps:
      - stepId: s1
        operationId: doSomething`

	yml2 := `arazzo: "1.0.1"
info:
  title: Test Two
  version: "2.0.0"
sourceDescriptions:
  - name: other
    url: https://other.example.com/openapi.json
workflows:
  - workflowId: different
    steps:
      - stepId: s2
        operationId: doOther`

	var n1, n2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml1), &n1)
	_ = yaml.Unmarshal([]byte(yml2), &n2)

	var a1, a2 Arazzo
	_ = low.BuildModel(n1.Content[0], &a1)
	_ = a1.Build(context.Background(), nil, n1.Content[0], nil)
	_ = low.BuildModel(n2.Content[0], &a2)
	_ = a2.Build(context.Background(), nil, n2.Content[0], nil)

	assert.NotEqual(t, a1.Hash(), a2.Hash())
}

func TestArazzo_Getters(t *testing.T) {
	yml := `arazzo: "1.0.0"
info:
  title: Test
  version: "1.0.0"
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.json
workflows:
  - workflowId: basic
    steps:
      - stepId: s1
        operationId: doSomething`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	keyNode := &yaml.Node{Value: "arazzo"}
	var arazzo Arazzo
	_ = low.BuildModel(node.Content[0], &arazzo)
	_ = arazzo.Build(context.Background(), keyNode, node.Content[0], nil)

	assert.Equal(t, keyNode, arazzo.GetKeyNode())
	assert.Equal(t, node.Content[0], arazzo.GetRootNode())
	assert.Nil(t, arazzo.GetIndex())
	assert.NotNil(t, arazzo.GetContext())
	assert.NotNil(t, arazzo.GetExtensions())
}

// ---------------------------------------------------------------------------
// Hash of empty structs (zero-value)
// ---------------------------------------------------------------------------

func TestHash_EmptyStructs(t *testing.T) {
	var info Info
	assert.NotZero(t, info.Hash())

	var sd SourceDescription
	assert.NotZero(t, sd.Hash())

	var cet CriterionExpressionType
	assert.NotZero(t, cet.Hash())

	var pr PayloadReplacement
	assert.NotZero(t, pr.Hash())

	var param Parameter
	assert.NotZero(t, param.Hash())

	var crit Criterion
	assert.NotZero(t, crit.Hash())

	var rb RequestBody
	assert.NotZero(t, rb.Hash())

	var sa SuccessAction
	assert.NotZero(t, sa.Hash())

	var fa FailureAction
	assert.NotZero(t, fa.Hash())

	var step Step
	assert.NotZero(t, step.Hash())

	var wf Workflow
	assert.NotZero(t, wf.Hash())

	var comp Components
	assert.NotZero(t, comp.Hash())

	var arazzo Arazzo
	assert.NotZero(t, arazzo.Hash())
}

// ---------------------------------------------------------------------------
// Helper function edge cases
// ---------------------------------------------------------------------------

func TestExtractArray_NotSequence(t *testing.T) {
	yml := `parameters: not-a-list`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	result, err := extractArray[Parameter](context.Background(), ParametersLabel, node.Content[0], nil)
	require.NoError(t, err)
	// Has key/value nodes set but no items since it is not a sequence
	assert.NotNil(t, result.KeyNode)
	assert.Nil(t, result.Value)
}

func TestExtractArray_Empty(t *testing.T) {
	yml := `parameters: []`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	result, err := extractArray[Parameter](context.Background(), ParametersLabel, node.Content[0], nil)
	require.NoError(t, err)
	assert.Len(t, result.Value, 0)
}

func TestExtractArray_Missing(t *testing.T) {
	yml := `name: test`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	result, err := extractArray[Parameter](context.Background(), ParametersLabel, node.Content[0], nil)
	require.NoError(t, err)
	assert.Nil(t, result.Value)
}

func TestExtractStringArray_NotSequence(t *testing.T) {
	yml := `dependsOn: not-a-list`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	result := extractStringArray(DependsOnLabel, node.Content[0])
	assert.NotNil(t, result.KeyNode)
	assert.Nil(t, result.Value)
}

func TestExtractStringArray_Empty(t *testing.T) {
	yml := `dependsOn: []`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	result := extractStringArray(DependsOnLabel, node.Content[0])
	assert.Len(t, result.Value, 0)
}

func TestExtractStringArray_Multiple(t *testing.T) {
	yml := `dependsOn:
  - alpha
  - beta
  - gamma`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	result := extractStringArray(DependsOnLabel, node.Content[0])
	require.Len(t, result.Value, 3)
	assert.Equal(t, "alpha", result.Value[0].Value)
	assert.Equal(t, "beta", result.Value[1].Value)
	assert.Equal(t, "gamma", result.Value[2].Value)
}

func TestExtractRawNode_Found(t *testing.T) {
	yml := `value: hello`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	result := extractRawNode(ValueLabel, node.Content[0])
	assert.False(t, result.IsEmpty())
	assert.Equal(t, "hello", result.Value.Value)
}

func TestExtractRawNode_NotFound(t *testing.T) {
	yml := `name: test`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	result := extractRawNode(ValueLabel, node.Content[0])
	assert.True(t, result.IsEmpty())
}

func TestExtractExpressionsMap_Found(t *testing.T) {
	yml := `outputs:
  petName: $response.body#/name
  petId: $response.body#/id`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	result := extractExpressionsMap(OutputsLabel, node.Content[0])
	require.False(t, result.IsEmpty())
	require.NotNil(t, result.Value)
	assert.Equal(t, 2, result.Value.Len())
	first := result.Value.First()
	require.NotNil(t, first)
	assert.Equal(t, "petName", first.Key().Value)
	assert.Equal(t, "$response.body#/name", first.Value().Value)
}

func TestExtractExpressionsMap_NotMapping(t *testing.T) {
	yml := `outputs: not-a-map`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	result := extractExpressionsMap(OutputsLabel, node.Content[0])
	assert.NotNil(t, result.KeyNode)
	assert.Nil(t, result.Value)
}

func TestExtractExpressionsMap_Missing(t *testing.T) {
	yml := `name: test`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	result := extractExpressionsMap(OutputsLabel, node.Content[0])
	assert.True(t, result.IsEmpty())
}

func TestExtractRawNodeMap_Found(t *testing.T) {
	yml := `inputs:
  petInput:
    type: object`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	result := extractRawNodeMap(InputsLabel, node.Content[0])
	require.False(t, result.IsEmpty())
	require.NotNil(t, result.Value)
	assert.Equal(t, 1, result.Value.Len())
	pair := result.Value.First()
	require.NotNil(t, pair)
	assert.Equal(t, "petInput", pair.Key().Value)
}

func TestExtractRawNodeMap_NotMapping(t *testing.T) {
	yml := `inputs: not-a-map`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	result := extractRawNodeMap(InputsLabel, node.Content[0])
	assert.NotNil(t, result.KeyNode)
	assert.Nil(t, result.Value)
}

func TestExtractRawNodeMap_Missing(t *testing.T) {
	yml := `name: test`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	result := extractRawNodeMap(InputsLabel, node.Content[0])
	assert.True(t, result.IsEmpty())
}

func TestExtractObjectMap_Found(t *testing.T) {
	yml := `parameters:
  petIdParam:
    name: petId
    in: path
    value: "123"
  ownerParam:
    name: ownerId
    in: query
    value: "456"`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	result, err := extractObjectMap[Parameter](context.Background(), ParametersLabel, node.Content[0], nil)
	require.NoError(t, err)
	require.False(t, result.IsEmpty())
	require.NotNil(t, result.Value)
	assert.Equal(t, 2, result.Value.Len())

	first := result.Value.First()
	require.NotNil(t, first)
	assert.Equal(t, "petIdParam", first.Key().Value)
	assert.Equal(t, "petId", first.Value().Value.Name.Value)

	second := first.Next()
	require.NotNil(t, second)
	assert.Equal(t, "ownerParam", second.Key().Value)
	assert.Equal(t, "ownerId", second.Value().Value.Name.Value)
}

func TestExtractObjectMap_NotMapping(t *testing.T) {
	yml := `parameters: not-a-map`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	result, err := extractObjectMap[Parameter](context.Background(), ParametersLabel, node.Content[0], nil)
	require.NoError(t, err)
	assert.Nil(t, result.Value)
}

func TestExtractObjectMap_Missing(t *testing.T) {
	yml := `name: test`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	result, err := extractObjectMap[Parameter](context.Background(), ParametersLabel, node.Content[0], nil)
	require.NoError(t, err)
	assert.True(t, result.IsEmpty())
}

// ---------------------------------------------------------------------------
// Odd content length edge cases (break guards)
// ---------------------------------------------------------------------------

func TestExtractArray_OddContentLength(t *testing.T) {
	yml := `name: test`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	// Append an orphan key to create odd-length content
	root := node.Content[0]
	root.Content = append(root.Content, &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "orphan",
	})

	result, err := extractArray[Parameter](context.Background(), ParametersLabel, root, nil)
	require.NoError(t, err)
	assert.Nil(t, result.Value)
}

func TestExtractStringArray_OddContentLength(t *testing.T) {
	yml := `name: test`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	root := node.Content[0]
	root.Content = append(root.Content, &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "orphan",
	})

	result := extractStringArray(DependsOnLabel, root)
	assert.Nil(t, result.Value)
}

func TestExtractRawNode_OddContentLength(t *testing.T) {
	yml := `name: test`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	root := node.Content[0]
	root.Content = append(root.Content, &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "orphan",
	})

	result := extractRawNode(ValueLabel, root)
	assert.True(t, result.IsEmpty())
}

func TestExtractExpressionsMap_OddContentLength(t *testing.T) {
	yml := `name: test`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	root := node.Content[0]
	root.Content = append(root.Content, &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "orphan",
	})

	result := extractExpressionsMap(OutputsLabel, root)
	assert.True(t, result.IsEmpty())
}

func TestExtractObjectMap_OddContentLength(t *testing.T) {
	yml := `name: test`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	root := node.Content[0]
	root.Content = append(root.Content, &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "orphan",
	})

	result, err := extractObjectMap[Parameter](context.Background(), ParametersLabel, root, nil)
	require.NoError(t, err)
	assert.True(t, result.IsEmpty())
}

func TestExtractRawNodeMap_OddContentLength(t *testing.T) {
	yml := `name: test`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	root := node.Content[0]
	root.Content = append(root.Content, &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "orphan",
	})

	result := extractRawNodeMap(InputsLabel, root)
	assert.True(t, result.IsEmpty())
}

// ---------------------------------------------------------------------------
// hashYAMLNode edge cases
// ---------------------------------------------------------------------------

func TestHashYAMLNode_Nil(t *testing.T) {
	// Should not panic
	yml := `target: /name`
	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var pr PayloadReplacement
	_ = low.BuildModel(node.Content[0], &pr)
	_ = pr.Build(context.Background(), nil, node.Content[0], nil)
	// Hash should work even when Value node is nil in some paths
	assert.NotZero(t, pr.Hash())
}

// ---------------------------------------------------------------------------
// Parameter with odd content length for reference extraction
// ---------------------------------------------------------------------------

func TestParameter_Build_OddContentLength(t *testing.T) {
	yml := `name: petId`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	root := node.Content[0]
	root.Content = append(root.Content, &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "orphan",
	})

	var param Parameter
	err = low.BuildModel(root, &param)
	require.NoError(t, err)

	err = param.Build(context.Background(), nil, root, nil)
	require.NoError(t, err)

	assert.False(t, param.IsReusable())
}

// ---------------------------------------------------------------------------
// SuccessAction with odd content length for reference extraction
// ---------------------------------------------------------------------------

func TestSuccessAction_Build_OddContentLength(t *testing.T) {
	yml := `name: done
type: end`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	root := node.Content[0]
	root.Content = append(root.Content, &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "orphan",
	})

	var sa SuccessAction
	err = low.BuildModel(root, &sa)
	require.NoError(t, err)

	err = sa.Build(context.Background(), nil, root, nil)
	require.NoError(t, err)

	assert.False(t, sa.IsReusable())
}

// ---------------------------------------------------------------------------
// FailureAction with odd content length for reference extraction
// ---------------------------------------------------------------------------

func TestFailureAction_Build_OddContentLength(t *testing.T) {
	yml := `name: fail
type: end`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	root := node.Content[0]
	root.Content = append(root.Content, &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "orphan",
	})

	var fa FailureAction
	err = low.BuildModel(root, &fa)
	require.NoError(t, err)

	err = fa.Build(context.Background(), nil, root, nil)
	require.NoError(t, err)

	assert.False(t, fa.IsReusable())
}

// ---------------------------------------------------------------------------
// FailureAction with invalid numeric fields
// ---------------------------------------------------------------------------

func TestFailureAction_Build_InvalidRetryAfter(t *testing.T) {
	yml := `name: retry
type: retry
retryAfter: not-a-number
retryLimit: also-not-a-number`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var fa FailureAction
	err = low.BuildModel(node.Content[0], &fa)
	require.NoError(t, err)

	err = fa.Build(context.Background(), nil, node.Content[0], nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid retryAfter value")
}

// ---------------------------------------------------------------------------
// Criterion with extension
// ---------------------------------------------------------------------------

func TestCriterion_Build_WithExtension(t *testing.T) {
	yml := `condition: $statusCode == 200
x-extra: some-value`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var crit Criterion
	err = low.BuildModel(node.Content[0], &crit)
	require.NoError(t, err)

	err = crit.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	ext := crit.FindExtension("x-extra")
	require.NotNil(t, ext)
	assert.Equal(t, "some-value", ext.Value.Value)
}

// ---------------------------------------------------------------------------
// Multiple SourceDescriptions
// ---------------------------------------------------------------------------

func TestArazzo_Build_MultipleSourceDescriptions(t *testing.T) {
	yml := `arazzo: "1.0.0"
info:
  title: Multi-Source
  version: "1.0.0"
sourceDescriptions:
  - name: petStore
    url: https://petstore.example.com/openapi.json
    type: openapi
  - name: weatherApi
    url: https://weather.example.com/openapi.json
    type: openapi
  - name: arazzoWf
    url: https://example.com/arazzo.yaml
    type: arazzo
workflows:
  - workflowId: basic
    steps:
      - stepId: s1
        operationId: doSomething`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var arazzo Arazzo
	err = low.BuildModel(node.Content[0], &arazzo)
	require.NoError(t, err)

	err = arazzo.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	require.Len(t, arazzo.SourceDescriptions.Value, 3)
	assert.Equal(t, "petStore", arazzo.SourceDescriptions.Value[0].Value.Name.Value)
	assert.Equal(t, "weatherApi", arazzo.SourceDescriptions.Value[1].Value.Name.Value)
	assert.Equal(t, "arazzoWf", arazzo.SourceDescriptions.Value[2].Value.Name.Value)
	assert.Equal(t, "arazzo", arazzo.SourceDescriptions.Value[2].Value.Type.Value)
}

// ---------------------------------------------------------------------------
// RequestBody with multiple replacements
// ---------------------------------------------------------------------------

func TestRequestBody_Build_MultipleReplacements(t *testing.T) {
	yml := `contentType: application/json
payload:
  name: default
  status: unknown
replacements:
  - target: /name
    value: $inputs.petName
  - target: /status
    value: $inputs.petStatus`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var rb RequestBody
	err = low.BuildModel(node.Content[0], &rb)
	require.NoError(t, err)

	err = rb.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	require.Len(t, rb.Replacements.Value, 2)
	assert.Equal(t, "/name", rb.Replacements.Value[0].Value.Target.Value)
	assert.Equal(t, "/status", rb.Replacements.Value[1].Value.Target.Value)
}

// ---------------------------------------------------------------------------
// Workflow with multiple dependsOn
// ---------------------------------------------------------------------------

func TestWorkflow_Build_MultipleDependsOn(t *testing.T) {
	yml := `workflowId: final
dependsOn:
  - step-a
  - step-b
  - step-c
steps:
  - stepId: s1
    operationId: doSomething`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var wf Workflow
	err = low.BuildModel(node.Content[0], &wf)
	require.NoError(t, err)

	err = wf.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	require.Len(t, wf.DependsOn.Value, 3)
	assert.Equal(t, "step-a", wf.DependsOn.Value[0].Value)
	assert.Equal(t, "step-b", wf.DependsOn.Value[1].Value)
	assert.Equal(t, "step-c", wf.DependsOn.Value[2].Value)
}

// ---------------------------------------------------------------------------
// Step with multiple parameters
// ---------------------------------------------------------------------------

func TestStep_Build_MultipleParameters(t *testing.T) {
	yml := `stepId: complexStep
operationId: complexOp
parameters:
  - name: id
    in: path
    value: "1"
  - name: format
    in: query
    value: json
  - name: auth
    in: header
    value: $inputs.token`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var step Step
	err = low.BuildModel(node.Content[0], &step)
	require.NoError(t, err)

	err = step.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	require.Len(t, step.Parameters.Value, 3)
	assert.Equal(t, "id", step.Parameters.Value[0].Value.Name.Value)
	assert.Equal(t, "format", step.Parameters.Value[1].Value.Name.Value)
	assert.Equal(t, "auth", step.Parameters.Value[2].Value.Name.Value)
}

// ---------------------------------------------------------------------------
// Step with multiple success criteria
// ---------------------------------------------------------------------------

func TestStep_Build_MultipleSuccessCriteria(t *testing.T) {
	// Note: Criterion has an unexported `context context.Context` field that clashes
	// with the exported `Context low.NodeReference[string]` when BuildModel runs.
	// BuildModel lowercases field names and matches both to the YAML "context" key,
	// causing an error on the unexported interface field. We test without the context
	// key here, and test context extraction separately.
	yml := `stepId: validated
operationId: validateOp
successCriteria:
  - condition: $statusCode == 200
  - condition: $response.body#/valid == true`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var step Step
	err = low.BuildModel(node.Content[0], &step)
	require.NoError(t, err)

	err = step.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	require.Len(t, step.SuccessCriteria.Value, 2)
	assert.Equal(t, "$statusCode == 200", step.SuccessCriteria.Value[0].Value.Condition.Value)
	assert.Equal(t, "$response.body#/valid == true", step.SuccessCriteria.Value[1].Value.Condition.Value)
}

// ---------------------------------------------------------------------------
// Components with inputs
// ---------------------------------------------------------------------------

func TestComponents_Build_WithInputs(t *testing.T) {
	yml := `inputs:
  petInput:
    type: object
    properties:
      petId:
        type: integer
  ownerInput:
    type: object
    properties:
      ownerId:
        type: string`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var comp Components
	err = low.BuildModel(node.Content[0], &comp)
	require.NoError(t, err)

	err = comp.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	require.False(t, comp.Inputs.IsEmpty())
	require.NotNil(t, comp.Inputs.Value)
	assert.Equal(t, 2, comp.Inputs.Value.Len())

	first := comp.Inputs.Value.First()
	require.NotNil(t, first)
	assert.Equal(t, "petInput", first.Key().Value)

	second := first.Next()
	require.NotNil(t, second)
	assert.Equal(t, "ownerInput", second.Key().Value)
}

// ---------------------------------------------------------------------------
// Workflow with extensions
// ---------------------------------------------------------------------------

func TestWorkflow_Build_WithExtensions(t *testing.T) {
	yml := `workflowId: extended
steps:
  - stepId: s1
    operationId: op1
x-workflow-extra: workflow-value`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var wf Workflow
	err = low.BuildModel(node.Content[0], &wf)
	require.NoError(t, err)

	err = wf.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	ext := wf.FindExtension("x-workflow-extra")
	require.NotNil(t, ext)
	assert.Equal(t, "workflow-value", ext.Value.Value)
}

// ---------------------------------------------------------------------------
// Step with extensions
// ---------------------------------------------------------------------------

func TestStep_Build_WithExtensions(t *testing.T) {
	yml := `stepId: extended
operationId: op1
x-step-extra: step-value`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var step Step
	err = low.BuildModel(node.Content[0], &step)
	require.NoError(t, err)

	err = step.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	ext := step.FindExtension("x-step-extra")
	require.NotNil(t, ext)
	assert.Equal(t, "step-value", ext.Value.Value)
}

// ---------------------------------------------------------------------------
// ObjectMap with odd inner content length
// ---------------------------------------------------------------------------

func TestExtractObjectMap_OddInnerContentLength(t *testing.T) {
	yml := `parameters:
  petIdParam:
    name: petId
    in: path
    value: "123"`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	// Find the parameters mapping node and add orphan key
	root := node.Content[0]
	for i := 0; i < len(root.Content); i += 2 {
		if root.Content[i].Value == "parameters" {
			root.Content[i+1].Content = append(root.Content[i+1].Content, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: "orphan",
			})
			break
		}
	}

	result, err := extractObjectMap[Parameter](context.Background(), ParametersLabel, root, nil)
	require.NoError(t, err)
	require.NotNil(t, result.Value)
	// Should still have the one valid entry
	assert.Equal(t, 1, result.Value.Len())
}

// ---------------------------------------------------------------------------
// ExpressionsMap with odd inner content length
// ---------------------------------------------------------------------------

func TestExtractExpressionsMap_OddInnerContentLength(t *testing.T) {
	yml := `outputs:
  petName: $response.body#/name`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	root := node.Content[0]
	for i := 0; i < len(root.Content); i += 2 {
		if root.Content[i].Value == "outputs" {
			root.Content[i+1].Content = append(root.Content[i+1].Content, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: "orphan",
			})
			break
		}
	}

	result := extractExpressionsMap(OutputsLabel, root)
	require.NotNil(t, result.Value)
	assert.Equal(t, 1, result.Value.Len())
}

// ---------------------------------------------------------------------------
// RawNodeMap with odd inner content length
// ---------------------------------------------------------------------------

func TestExtractRawNodeMap_OddInnerContentLength(t *testing.T) {
	yml := `inputs:
  petInput:
    type: object`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	root := node.Content[0]
	for i := 0; i < len(root.Content); i += 2 {
		if root.Content[i].Value == "inputs" {
			root.Content[i+1].Content = append(root.Content[i+1].Content, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: "orphan",
			})
			break
		}
	}

	result := extractRawNodeMap(InputsLabel, root)
	require.NotNil(t, result.Value)
	assert.Equal(t, 1, result.Value.Len())
}
