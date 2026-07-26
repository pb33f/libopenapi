package orderedmap_test

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestOrderedMap_ToYamlNode(t *testing.T) {
	type args struct {
		om  any
		low any
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "simple ordered map",
			args: args{
				om: orderedmap.ToOrderedMap(map[string]string{
					"one":   "two",
					"three": "four",
				}),
			},
			want: `one: two
three: four
`,
		},
		{
			name: "simple ordered map with low representation",
			args: args{
				om: orderedmap.ToOrderedMap(map[string]string{
					"one":   "two",
					"three": "four",
				}),
				low: low.NodeReference[*orderedmap.Map[*low.KeyReference[string], *low.ValueReference[string]]]{
					Value: orderedmap.ToOrderedMap(map[*low.KeyReference[string]]*low.ValueReference[string]{
						{Value: "one", KeyNode: utils.CreateStringNode("one")}: {Value: "two", ValueNode: utils.CreateStringNode("two")},
					}),
					ValueNode: utils.CreateYamlNode(orderedmap.ToOrderedMap(map[string]string{
						"one":   "two",
						"three": "four",
					})),
				},
			},
			want: `one: two
three: four
`,
		},
		{
			name: "ordered map with KeyReference",
			args: args{
				om: orderedmap.ToOrderedMap(map[*low.KeyReference[string]]string{
					{
						KeyNode: utils.CreateStringNode("one"),
					}: "two",
					{
						KeyNode: utils.CreateStringNode("three"),
					}: "four",
				}),
			},
			want: `one: two
three: four
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nb := new(high.NodeBuilder)

			node := tt.args.om.(orderedmap.MapToYamlNoder).ToYamlNode(nb, tt.args.low)
			b, err := yaml.Marshal(node)
			require.NoError(t, err)
			require.Equal(t, tt.want, string(b))
		})
	}
}

type findValueUntyped interface {
	FindValueUntyped(k string) any
}

func TestOrderedMap_FindValueUntyped(t *testing.T) {
	type args struct {
		om  any
		key string
	}
	tests := []struct {
		name string
		args args
		want any
	}{
		{
			name: "find value in simple ordered map",
			args: args{
				om: orderedmap.ToOrderedMap(map[string]string{
					"one":   "two",
					"three": "four",
				}),
				key: "one",
			},
			want: "two",
		},
		{
			name: "unable to find value in simple ordered map",
			args: args{
				om: orderedmap.ToOrderedMap(map[string]string{
					"one":   "two",
					"three": "four",
				}),
				key: "five",
			},
			want: nil,
		},
		{
			name: "find value in ordered map with KeyReference",
			args: args{
				om: orderedmap.ToOrderedMap(map[*low.KeyReference[string]]string{
					{Value: "one"}:   "two",
					{Value: "three"}: "four",
				}),
				key: "three",
			},
			want: "four",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := tt.args.om.(findValueUntyped).FindValueUntyped(tt.args.key)
			require.Equal(t, tt.want, value)
		})
	}
}

// lowWithValueNode supplies the original YAML node that ToYamlNode consults to recover the
// authored quoting style of each key.
type lowWithValueNode struct{ node *yaml.Node }

func (l lowWithValueNode) GetValueNode() *yaml.Node { return l.node }

// ToYamlNode looks each key up in the original node to carry its style across. A key that is
// not present there (one added to the high model after parsing) simply has no authored style,
// and must render with the default rather than fail the lookup.
func TestOrderedMap_ToYamlNode_KeyAbsentFromOriginalNode(t *testing.T) {
	// The original document quoted "kept" but knows nothing about "added".
	var original yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(`"kept": one`), &original))
	require.NotNil(t, original.Content)

	om := orderedmap.New[string, string]()
	om.Set("kept", "one")
	om.Set("added", "two")

	node := om.ToYamlNode(new(high.NodeBuilder), lowWithValueNode{node: original.Content[0]})
	rendered, err := yaml.Marshal(node)
	require.NoError(t, err)

	// The quoted key keeps its style; the one with no counterpart renders plainly.
	require.Equal(t, `"kept": one
added: two
`, string(rendered))
}

// A low model that carries no value node at all leaves every key without an authored style.
func TestOrderedMap_ToYamlNode_NilValueNode(t *testing.T) {
	om := orderedmap.New[string, string]()
	om.Set("one", "two")

	node := om.ToYamlNode(new(high.NodeBuilder), lowWithValueNode{node: nil})
	rendered, err := yaml.Marshal(node)
	require.NoError(t, err)
	require.Equal(t, "one: two\n", string(rendered))
}
