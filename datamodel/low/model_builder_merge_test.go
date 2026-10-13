package low

import (
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestBuildModelPreservesMergedSource(t *testing.T) {
	const source = `first: &first
  name: inherited # original value
  fat: 2
second: &second
  name: later
  ketchup: 3
sources: &sources [*first, *second]
item:
  <<: *sources
  fat: 5
`
	var document yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(source), &document))
	_, item := utils.FindKeyNodeTop("item", document.Content[0].Content)
	require.NotNil(t, item)
	before, err := yaml.Marshal(&document)
	require.NoError(t, err)
	_, first := utils.FindKeyNodeTop("first", document.Content[0].Content)
	nameKey, nameValue := utils.FindKeyNodeTop("name", first.Content)
	var result hotdog
	require.NoError(t, BuildModel(item, &result))
	assert.Equal(t, "inherited", result.Name.Value)
	assert.Equal(t, 5, result.Fat.Value)
	assert.Equal(t, float32(3), result.Ketchup.Value)
	assert.Same(t, nameKey, result.Name.KeyNode)
	assert.Same(t, nameValue, result.Name.ValueNode)
	assert.Equal(t, "# original value", result.Name.ValueNode.LineComment)
	after, err := yaml.Marshal(&document)
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after), "model construction must not replace shared source content")
}

func TestBuildModelMergedMapFields(t *testing.T) {
	const source = `defaults: &defaults
  inherited: original
  local: overridden
mapped: &mapped
  <<: *defaults
  local: selected
a: *mapped
b: *mapped
c: *mapped
d: *mapped
e: *mapped
`
	var document yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(source), &document))
	before, err := yaml.Marshal(&document)
	require.NoError(t, err)
	var model struct {
		A *orderedmap.Map[string, NodeReference[*yaml.Node]]
		B *orderedmap.Map[string, NodeReference[string]]
		C *orderedmap.Map[KeyReference[string], ValueReference[string]]
		D KeyReference[*orderedmap.Map[KeyReference[string], ValueReference[string]]]
		E NodeReference[*orderedmap.Map[KeyReference[string], ValueReference[string]]]
	}
	require.NoError(t, BuildModel(document.Content[0], &model))
	t.Run("node values", func(t *testing.T) {
		inherited, ok := model.A.Get("inherited")
		require.True(t, ok)
		assert.Equal(t, "original", inherited.Value.Value)
		local, ok := model.A.Get("local")
		require.True(t, ok)
		assert.Equal(t, "selected", local.Value.Value)
	})
	t.Run("string values", func(t *testing.T) {
		inherited, ok := model.B.Get("inherited")
		require.True(t, ok)
		assert.Equal(t, "original", inherited.Value)
		local, ok := model.B.Get("local")
		require.True(t, ok)
		assert.Equal(t, "selected", local.Value)
	})
	for name, values := range map[string]*orderedmap.Map[KeyReference[string], ValueReference[string]]{
		"plain map": model.C, "key reference": model.D.Value, "node reference": model.E.Value,
	} {
		t.Run(name, func(t *testing.T) {
			require.NotNil(t, values)
			got := make(map[string]string)
			for key, value := range values.FromOldest() {
				got[key.Value] = value.Value
			}
			assert.Equal(t, map[string]string{"inherited": "original", "local": "selected"}, got)
		})
	}
	after, err := yaml.Marshal(&document)
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after))
}
