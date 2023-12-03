package orderedmap_test

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
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
