// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package high

import (
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high/nodes"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

type valueReferenceStruct struct {
	ref    bool
	refStr string
	Value  string `yaml:"value,omitempty"`
}

func (r valueReferenceStruct) IsReference() bool {
	return r.ref
}

func (r valueReferenceStruct) GetReference() string {
	return r.refStr
}

func (r *valueReferenceStruct) SetReference(ref string, _ *yaml.Node) {
	r.refStr = ref
}

func (r *valueReferenceStruct) GetReferenceNode() *yaml.Node {
	return nil
}

func (r valueReferenceStruct) MarshalYAML() (interface{}, error) {
	return utils.CreateStringNode("pizza"), nil
}

func (r valueReferenceStruct) MarshalYAMLInline() (interface{}, error) {
	return utils.CreateStringNode("pizza-inline!"), nil
}

func (r valueReferenceStruct) GoLowUntyped() any {
	return &r
}

func (r valueReferenceStruct) GetValueNode() *yaml.Node {
	n := utils.CreateEmptySequenceNode()
	n.Content = append(n.Content, utils.CreateEmptySequenceNode())
	return n
}

type plug struct {
	Name []string `yaml:"name,omitempty"`
}

type test1 struct {
	Thrig      *orderedmap.Map[string, *plug]                    `yaml:"thrig,omitempty"`
	Thing      string                                            `yaml:"thing,omitempty"`
	Thong      int                                               `yaml:"thong,omitempty"`
	Thrum      int64                                             `yaml:"thrum,omitempty"`
	Thang      float32                                           `yaml:"thang,omitempty"`
	Thung      float64                                           `yaml:"thung,omitempty"`
	Thyme      bool                                              `yaml:"thyme,omitempty"`
	Thurm      any                                               `yaml:"thurm,omitempty"`
	Thugg      *bool                                             `yaml:"thugg,renderZero"`
	Thurr      *int64                                            `yaml:"thurr,omitempty"`
	Thral      *float64                                          `yaml:"thral,omitempty"`
	Throo      *float64                                          `yaml:"throo,renderZero,omitempty"`
	Tharg      []string                                          `yaml:"tharg,omitempty"`
	Type       []string                                          `yaml:"type,omitempty"`
	Throg      []*valueReferenceStruct                           `yaml:"throg,omitempty"`
	Thrat      []interface{}                                     `yaml:"thrat,omitempty"`
	Thrag      []*orderedmap.Map[string, []string]               `yaml:"thrag,omitempty"`
	Thrug      *orderedmap.Map[string, string]                   `yaml:"thrug,omitempty"`
	Thoom      []*orderedmap.Map[string, string]                 `yaml:"thoom,omitempty"`
	Thomp      *orderedmap.Map[low.KeyReference[string], string] `yaml:"thomp,omitempty"`
	Thump      valueReferenceStruct                              `yaml:"thump,omitempty"`
	Thane      valueReferenceStruct                              `yaml:"thane,omitempty"`
	Thunk      valueReferenceStruct                              `yaml:"thunk,omitempty"`
	Thrim      *valueReferenceStruct                             `yaml:"thrim,omitempty"`
	Thril      *orderedmap.Map[string, *valueReferenceStruct]    `yaml:"thril,omitempty"`
	Extensions *orderedmap.Map[string, *yaml.Node]               `yaml:"-"`
	ignoreMe   string                                            `yaml:"-"`
	IgnoreMe   string                                            `yaml:"-"`
}

func (te *test1) GetExtensions() *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]] {
	g := orderedmap.New[low.KeyReference[string], low.ValueReference[*yaml.Node]]()

	i := 0
	for ext, node := range te.Extensions.FromOldest() {
		kn := utils.CreateStringNode(ext)
		kn.Line = 999999 + i // weighted to the bottom.

		g.Set(low.KeyReference[string]{
			Value:   ext,
			KeyNode: kn,
		}, low.ValueReference[*yaml.Node]{
			ValueNode: node,
			Value:     node,
		})
		i++
	}
	return g
}

func (te *test1) MarshalYAML() (interface{}, error) {
	panic("MarshalYAML")
	nb := NewNodeBuilder(te, te)
	return nb.Render(), nil
}

func (te *test1) GetKeyNode() *yaml.Node {
	panic("GetKeyNode")
	kn := utils.CreateStringNode("meddy")
	kn.Line = 20
	return kn
}

func (te *test1) GetValueNode() *yaml.Node {
	kn := utils.CreateStringNode("meddy")
	kn.Line = 20
	return kn
}

func (te *test1) GoesLowUntyped() any {
	panic("GoesLowUntyped")
	return te
}

type test2 struct {
	Thrat      *valueReferenceStruct                             `yaml:"throg,omitempty"`
	Thrig      *orderedmap.Map[string, *plug]                    `yaml:"thrig,omitempty"`
	Thing      string                                            `yaml:"thing,omitempty"`
	Thong      int                                               `yaml:"thong,omitempty"`
	Thrum      int64                                             `yaml:"thrum,omitempty"`
	Thang      float32                                           `yaml:"thang,omitempty"`
	Thung      float64                                           `yaml:"thung,omitempty"`
	Thyme      bool                                              `yaml:"thyme,omitempty"`
	Thurm      any                                               `yaml:"thurm,omitempty"`
	Thugg      *bool                                             `yaml:"thugg,renderZero"`
	Thurr      *int64                                            `yaml:"thurr,omitempty"`
	Thral      *float64                                          `yaml:"thral,omitempty"`
	Throo      *float64                                          `yaml:"throo,renderZero,omitempty"`
	Tharg      []string                                          `yaml:"tharg,omitempty"`
	Type       []string                                          `yaml:"type,omitempty"`
	Throg      []*valueReferenceStruct                           `yaml:"throg,omitempty"`
	Throj      *valueReferenceStruct                             `yaml:"throg,omitempty"`
	Thrag      []*orderedmap.Map[string, []string]               `yaml:"thrag,omitempty"`
	Thrug      *orderedmap.Map[string, string]                   `yaml:"thrug,omitempty"`
	Thoom      []*orderedmap.Map[string, string]                 `yaml:"thoom,omitempty"`
	Thomp      *orderedmap.Map[low.KeyReference[string], string] `yaml:"thomp,omitempty"`
	Thump      valueReferenceStruct                              `yaml:"thump,omitempty"`
	Thane      valueReferenceStruct                              `yaml:"thane,omitempty"`
	Thunk      valueReferenceStruct                              `yaml:"thunk,omitempty"`
	Thrim      *valueReferenceStruct                             `yaml:"thrim,omitempty"`
	Thril      *orderedmap.Map[string, *valueReferenceStruct]    `yaml:"thril,omitempty"`
	Extensions *orderedmap.Map[string, *yaml.Node]               `yaml:"-"`
	ignoreMe   string                                            `yaml:"-"`
	IgnoreMe   string                                            `yaml:"-"`
}

func (t test2) IsReference() bool {
	return true
}

func (t test2) GetReference() string {
	return "aggghhh"
}

func (t test2) SetReference(ref string, _ *yaml.Node) {
}

func (t test2) GetReferenceNode() *yaml.Node {
	return nil
}

func (t test2) MarshalYAML() (interface{}, error) {
	return utils.CreateStringNode("pizza"), nil
}

func (t test2) MarshalYAMLInline() (interface{}, error) {
	return utils.CreateStringNode("pizza-inline!"), nil
}

func (t test2) GoLowUntyped() any {
	return &t
}

func (t test2) GetValue() *yaml.Node {
	return nil
}

type structLowTest struct {
	Name string `yaml:"name,omitempty"`
}

type nonEmptyExample struct{}

var nonEmptyExampleCallCount int

func (nonEmptyExample) IsEmpty() bool {
	nonEmptyExampleCallCount++
	return false
}

type pointerFieldStruct struct {
	Example *nonEmptyExample `yaml:"example,omitempty"`
}

func TestNewNodeBuilder_SliceRef_Inline_HasValue(t *testing.T) {
	ty := []interface{}{utils.CreateEmptySequenceNode()}
	t1 := test1{
		Thrat: ty,
	}

	t2 := test2{
		Thrat: &valueReferenceStruct{Value: renderZero},
	}

	nb := NewNodeBuilder(&t1, &t2)
	nb.Resolve = true
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `thrat:
    - []`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder(t *testing.T) {
	b := true
	c := int64(12345)
	d := 1234.1234

	thoom1 := orderedmap.New[string, string]()
	thoom1.Set("maddy", "champion")

	thoom2 := orderedmap.New[string, string]()
	thoom2.Set("ember", "naughty")

	thomp := orderedmap.New[low.KeyReference[string], string]()
	thomp.Set(low.KeyReference[string]{
		Value:   "meddy",
		KeyNode: utils.CreateStringNode("meddy"),
	}, "princess")

	thrug := orderedmap.New[string, string]()
	thrug.Set("chicken", "nuggets")

	ext := orderedmap.New[string, *yaml.Node]()
	ext.Set("x-pizza", utils.CreateStringNode("time"))

	t1 := test1{
		ignoreMe: "I should never be seen!",
		Thing:    "ding",
		Thong:    1,
		Thurm:    nil,
		Thrum:    1234567,
		Thang:    2.2,
		Thung:    3.33333,
		Thyme:    true,
		Thugg:    &b,
		Thurr:    &c,
		Thral:    &d,
		Tharg:    []string{"chicken", "nuggets"},
		Type:     []string{"chicken"},
		Thoom: []*orderedmap.Map[string, string]{
			thoom1,
			thoom2,
		},
		Thomp: thomp,
		Thane: valueReferenceStruct{ // this is going to be ignored, needs to be a ValueReference
			Value: "ripples",
		},
		Thrug:      thrug,
		Thump:      valueReferenceStruct{Value: "I will be ignored"},
		Thunk:      valueReferenceStruct{},
		Extensions: ext,
	}

	nb := NewNodeBuilder(&t1, nil)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `thing: ding
thong: 1
thrum: 1234567
thang: 2.2
thung: 3.33
thyme: true
thugg: true
thurr: 12345
thral: 1234.1234
tharg:
    - chicken
    - nuggets
type: chicken
thrug:
    chicken: nuggets
thoom:
    - maddy: champion
    - ember: naughty
thomp:
    meddy: princess
x-pizza: time`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_float_noprecision(t *testing.T) {
	var throo float64 = 3
	t1 := test1{
		Throo: &throo,
		Thung: 3,
	}

	nb := NewNodeBuilder(&t1, nil)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `thung: 3
throo: 3`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_Type(t *testing.T) {
	t1 := test1{
		Type: []string{"chicken", "soup"},
	}

	nb := NewNodeBuilder(&t1, &t1)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `type:
    - chicken
    - soup`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_IsReferenced(t *testing.T) {
	t1 := &low.ValueReference[string]{
		Value: "cotton",
	}
	t1.SetReference("#/my/heart", nil)

	nb := NewNodeBuilder(t1, t1)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `$ref: '#/my/heart'`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_Extensions(t *testing.T) {
	ext := orderedmap.New[string, *yaml.Node]()
	ext.Set("x-pizza", utils.CreateStringNode("time"))
	ext.Set("x-money", utils.CreateStringNode("time"))

	t1 := test1{
		Thing:      "ding",
		Extensions: ext,
		Thong:      1,
	}

	nb := NewNodeBuilder(&t1, &t1)
	node := nb.Render()

	data, _ := yaml.Marshal(node)
	assert.Len(t, data, 49)
}

func TestNewNodeBuilder_LowValueNode(t *testing.T) {
	ext := orderedmap.New[string, *yaml.Node]()
	ext.Set("x-pizza", utils.CreateStringNode("time"))
	ext.Set("x-money", utils.CreateStringNode("time"))

	t1 := test1{
		Thing:      "ding",
		Extensions: ext,
		Thong:      1,
	}

	nb := NewNodeBuilder(&t1, &t1)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	assert.Len(t, data, 49)
}

func TestNewNodeBuilder_NoValue(t *testing.T) {
	t1 := test1{
		Thing: "",
	}

	nodeEnty := nodes.NodeEntry{}
	nb := NewNodeBuilder(&t1, &t1)
	node := nb.AddYAMLNode(nil, &nodeEnty)
	assert.Nil(t, node)
}

func TestNewNodeBuilder_EmptyString(t *testing.T) {
	t1 := new(test1)
	nodeEnty := nodes.NodeEntry{}
	nb := NewNodeBuilder(t1, t1)
	node := nb.AddYAMLNode(nil, &nodeEnty)
	assert.Nil(t, node)
}

func TestNewNodeBuilder_EmptyStringRenderZero(t *testing.T) {
	t1 := new(test1)
	nodeEnty := nodes.NodeEntry{RenderZero: true, Value: ""}
	nb := NewNodeBuilder(t1, t1)
	m := utils.CreateEmptyMapNode()
	node := nb.AddYAMLNode(m, &nodeEnty)
	assert.NotNil(t, node)
}

func TestNewNodeBuilder_Bool(t *testing.T) {
	t1 := new(test1)
	nb := NewNodeBuilder(t1, t1)
	nodeEnty := nodes.NodeEntry{}
	node := nb.AddYAMLNode(nil, &nodeEnty)
	assert.Nil(t, node)
}

func TestNewNodeBuilder_BoolRenderZero(t *testing.T) {
	type yui struct {
		Thrit bool `yaml:"thrit,renderZero"`
	}
	t1 := new(yui)
	t1.Thrit = false
	nb := NewNodeBuilder(t1, t1)
	r := nb.Render()
	assert.NotNil(t, r)
}

func TestNewNodeBuilder_Int(t *testing.T) {
	t1 := new(test1)
	nb := NewNodeBuilder(t1, t1)
	p := utils.CreateEmptyMapNode()
	nodeEnty := nodes.NodeEntry{Tag: "p", Value: 12, Key: "p"}
	node := nb.AddYAMLNode(p, &nodeEnty)
	assert.NotNil(t, node)
	assert.Len(t, node.Content, 2)
	assert.Equal(t, "12", node.Content[1].Value)
}

func TestNewNodeBuilder_Int64(t *testing.T) {
	t1 := new(test1)
	nb := NewNodeBuilder(t1, t1)
	p := utils.CreateEmptyMapNode()
	nodeEnty := nodes.NodeEntry{Tag: "p", Value: int64(234556), Key: "p"}
	node := nb.AddYAMLNode(p, &nodeEnty)
	assert.NotNil(t, node)
	assert.Len(t, node.Content, 2)
	assert.Equal(t, "234556", node.Content[1].Value)
}

func TestNewNodeBuilder_Float32(t *testing.T) {
	t1 := new(test1)
	nb := NewNodeBuilder(t1, t1)
	p := utils.CreateEmptyMapNode()
	nodeEnty := nodes.NodeEntry{Tag: "p", Value: float32(1234.23), Key: "p"}
	node := nb.AddYAMLNode(p, &nodeEnty)
	assert.NotNil(t, node)
	assert.Len(t, node.Content, 2)
	assert.Equal(t, "1234.23", node.Content[1].Value)
}

func TestNewNodeBuilder_Float64(t *testing.T) {
	t1 := new(test1)
	nb := NewNodeBuilder(t1, t1)
	p := utils.CreateEmptyMapNode()
	nodeEnty := nodes.NodeEntry{Tag: "p", Value: 1234.232323, Key: "p", StringValue: "1234.232323"}
	node := nb.AddYAMLNode(p, &nodeEnty)
	assert.NotNil(t, node)
	assert.Len(t, node.Content, 2)
	assert.Equal(t, "1234.232323", node.Content[1].Value)
}

func TestNewNodeBuilder_EmptyNode(t *testing.T) {
	t1 := new(test1)
	nb := NewNodeBuilder(t1, t1)
	nb.Nodes = nil
	m := nb.Render()
	assert.Len(t, m.Content, 0)
}

func TestNewNodeBuilder_MapKeyHasValue(t *testing.T) {
	thrug := orderedmap.New[string, string]()
	thrug.Set("dump", "trump")

	t1 := test1{
		Thrug: thrug,
	}

	type test1low struct {
		Thrug *orderedmap.Map[*low.KeyReference[string], *low.ValueReference[string]] `yaml:"thrug"`
		Thugg *bool                                                                   `yaml:"thugg"`
		Throo *float32                                                                `yaml:"throo"`
	}

	thrugLow := orderedmap.New[*low.KeyReference[string], *low.ValueReference[string]]()
	thrugLow.Set(&low.KeyReference[string]{
		Value:   "dump",
		KeyNode: utils.CreateStringNode("dump"),
	}, &low.ValueReference[string]{
		Value:     "trump",
		ValueNode: utils.CreateStringNode("trump"),
	})

	t2 := test1low{
		Thrug: thrugLow,
	}

	nb := NewNodeBuilder(&t1, &t2)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `thrug:
    dump: trump`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_MapKeyHasValueThatHasValue(t *testing.T) {
	thomp := orderedmap.New[low.KeyReference[string], string]()
	thomp.Set(low.KeyReference[string]{Value: "meddy", KeyNode: utils.CreateStringNode("meddy")}, "princess")

	t1 := test1{
		Thomp: thomp,
	}

	type test1low struct {
		Thomp low.ValueReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[string]]] `yaml:"thomp"`
		Thugg *bool                                                                                     `yaml:"thugg"`
		Throo *float32                                                                                  `yaml:"throo"`
	}

	valueMap := orderedmap.New[low.KeyReference[string], low.ValueReference[string]]()
	valueMap.Set(low.KeyReference[string]{
		Value:   "ice",
		KeyNode: utils.CreateStringNode("ice"),
	}, low.ValueReference[string]{
		Value: "princess",
	})

	t2 := test1low{
		Thomp: low.ValueReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[string]]]{
			Value: valueMap,
		},
	}

	nb := NewNodeBuilder(&t1, &t2)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `thomp:
    meddy: princess`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_MapKeyHasValueThatHasValueMatch(t *testing.T) {
	thomp := orderedmap.New[low.KeyReference[string], string]()
	thomp.Set(low.KeyReference[string]{Value: "meddy", KeyNode: utils.CreateStringNode("meddy")}, "princess")

	t1 := test1{
		Thomp: thomp,
	}

	type test1low struct {
		Thomp low.NodeReference[*orderedmap.Map[low.KeyReference[string], string]] `yaml:"thomp"`
		Thugg *bool                                                                `yaml:"thugg"`
		Throo *float32                                                             `yaml:"throo"`
	}

	valMap := orderedmap.New[low.KeyReference[string], string]()
	valMap.Set(low.KeyReference[string]{
		Value:   "meddy",
		KeyNode: utils.CreateStringNode("meddy"),
	}, "princess")

	g := low.NodeReference[*orderedmap.Map[low.KeyReference[string], string]]{
		Value: valMap,
	}

	t2 := test1low{
		Thomp: g,
	}

	nb := NewNodeBuilder(&t1, &t2)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `thomp:
    meddy: princess`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_MissingLabel(t *testing.T) {
	t1 := new(test1)
	nb := NewNodeBuilder(t1, t1)
	p := utils.CreateEmptyMapNode()
	nodeEnty := nodes.NodeEntry{Value: 1234.232323, Key: "p"}
	node := nb.AddYAMLNode(p, &nodeEnty)
	assert.NotNil(t, node)
	assert.Len(t, node.Content, 0)
}

func TestNewNodeBuilder_ExtensionMap(t *testing.T) {
	ext := orderedmap.New[string, *yaml.Node]()
	pizza := orderedmap.New[string, string]()
	pizza.Set("dump", "trump")
	ext.Set("x-pizza", utils.CreateYamlNode(pizza))
	ext.Set("x-money", utils.CreateStringNode("time"))

	t1 := test1{
		Thing:      "ding",
		Extensions: ext,
		Thong:      1,
	}

	nb := NewNodeBuilder(&t1, &t1)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	assert.Len(t, data, 60)
}

func TestNewNodeBuilder_MapKeyHasValueThatHasValueMismatch(t *testing.T) {
	ext := orderedmap.New[string, *yaml.Node]()
	pizza := orderedmap.New[string, string]()
	pizza.Set("dump", "trump")
	ext.Set("x-pizza", utils.CreateYamlNode(pizza))
	cake := orderedmap.New[string, string]()
	cake.Set("maga", "nomore")
	ext.Set("x-cake", utils.CreateYamlNode(cake))

	thril := orderedmap.New[string, *valueReferenceStruct]()
	thril.Set("princess", &valueReferenceStruct{Value: "who"})
	thril.Set("heavy", &valueReferenceStruct{Value: "who"})

	t1 := test1{
		Extensions: ext,
		Thril:      thril,
	}

	nb := NewNodeBuilder(&t1, nil)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	assert.Equal(t, `thril:
    princess: pizza
    heavy: pizza
x-pizza:
    dump: trump
x-cake:
    maga: nomore`, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_SliceRef(t *testing.T) {
	c := valueReferenceStruct{ref: true, refStr: "#/red/robin/yummmmm", Value: "milky"}
	ty := []*valueReferenceStruct{&c}
	t1 := test1{
		Throg: ty,
	}

	nb := NewNodeBuilder(&t1, &t1)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `throg:
    - $ref: '#/red/robin/yummmmm'`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_SliceRef_Inline(t *testing.T) {
	c := valueReferenceStruct{Value: "milky"}
	ty := []*valueReferenceStruct{&c}
	t1 := test1{
		Throg: ty,
	}

	nb := NewNodeBuilder(&t1, &t1)
	nb.Resolve = true
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `throg:
    - pizza-inline!`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_SliceRef_InlineNull(t *testing.T) {
	c := valueReferenceStruct{Value: "milky"}
	ty := []*valueReferenceStruct{&c}
	t1 := test1{
		Throg: ty,
	}

	t2 := test1{
		Throg: []*valueReferenceStruct{},
	}

	nb := NewNodeBuilder(&t1, &t2)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := "throg:\n    - pizza"

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

type testRender struct{}

func (t testRender) MarshalYAML() (interface{}, error) {
	return utils.CreateStringNode("testy!"), nil
}

type testRenderRawNode struct{}

func (t testRenderRawNode) MarshalYAML() (interface{}, error) {
	return yaml.Node{Kind: yaml.ScalarNode, Value: "zesty!"}, nil
}

func TestNewNodeBuilder_SliceRef_Inline_NotCompatible(t *testing.T) {
	ty := []interface{}{testRender{}}
	t1 := test1{
		Thrat: ty,
	}

	nb := NewNodeBuilder(&t1, &t1)
	nb.Resolve = true
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `thrat:
    - testy!`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_SliceRef_Inline_NotCompatible_NotPointer(t *testing.T) {
	ty := []interface{}{testRenderRawNode{}}
	t1 := test1{
		Thrat: ty,
	}

	nb := NewNodeBuilder(&t1, &t1)
	nb.Resolve = true
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `thrat:
    - zesty!`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_PointerRef_Inline_NotCompatible_RawNode(t *testing.T) {
	ty := testRenderRawNode{}
	t1 := test1{
		Thurm: &ty,
	}

	nb := NewNodeBuilder(&t1, &t1)
	nb.Resolve = true
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `thurm: zesty!`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_PointerRef_Inline_NotCompatible(t *testing.T) {
	ty := valueReferenceStruct{}
	t1 := test1{
		Thurm: &ty,
	}

	nb := NewNodeBuilder(&t1, &t1)
	nb.Resolve = true
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `thurm: pizza-inline!`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_SliceNoRef(t *testing.T) {
	c := valueReferenceStruct{Value: "milky"}
	ty := []*valueReferenceStruct{&c}
	t1 := test1{
		Throg: ty,
	}

	nb := NewNodeBuilder(&t1, &t1)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `throg:
    - pizza`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_TestStructAny(t *testing.T) {
	t1 := test1{
		Thurm: low.ValueReference[any]{
			ValueNode: utils.CreateStringNode("beer"),
		},
	}

	nb := NewNodeBuilder(&t1, &t1)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `thurm: beer`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_TestStructString(t *testing.T) {
	t1 := test1{
		Thurm: low.ValueReference[string]{
			ValueNode: utils.CreateStringNode("beer"),
		},
	}

	nb := NewNodeBuilder(&t1, &t1)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `thurm: beer`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_TestStructPointer(t *testing.T) {
	t1 := test1{
		Thrim: &valueReferenceStruct{
			ref:    true,
			refStr: "#/cash/money",
			Value:  "pizza",
		},
	}

	nb := NewNodeBuilder(&t1, &t1)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `thrim:
    $ref: '#/cash/money'`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_TestStructRef(t *testing.T) {
	fkn := utils.CreateStringNode("pizzaBurgers")
	fkn.Line = 22

	thurm := low.NodeReference[string]{
		KeyNode:   fkn,
		ValueNode: fkn,
	}
	thurm.SetReference("#/cash/money", nil)

	t1 := test1{
		Thurm: thurm,
	}

	nb := NewNodeBuilder(&t1, &t1)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `thurm: pizzaBurgers`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_TestStructDefaultEncode(t *testing.T) {
	f := 1
	t1 := test1{
		Thurm: &f,
	}

	nb := NewNodeBuilder(&t1, &t1)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `thurm: 1`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_TestSliceMapSliceStruct(t *testing.T) {
	pizza := orderedmap.New[string, []string]()
	pizza.Set("pizza", []string{"beer", "wine"})
	a := []*orderedmap.Map[string, []string]{
		pizza,
	}

	t1 := test1{
		Thrag: a,
	}

	nb := NewNodeBuilder(&t1, &t1)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `thrag:
    - pizza:
        - beer
        - wine`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_TestRenderZero(t *testing.T) {
	f := false
	t1 := test1{
		Thugg: &f,
	}

	nb := NewNodeBuilder(&t1, &t1)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `thugg: false`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_TestRenderZero_Float(t *testing.T) {
	f := 0.0
	t1 := test1{
		Throo: &f,
	}

	nb := NewNodeBuilder(&t1, &t1)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `throo: 0`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_TestRenderZero_Float_NotZero(t *testing.T) {
	f := 0.12
	t1 := test1{
		Throo: &f,
	}

	nb := NewNodeBuilder(&t1, &t1)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `throo: 0.12`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_TestRenderServerVariableSimulation(t *testing.T) {
	thrig := orderedmap.New[string, *plug]()
	thrig.Set("pork", &plug{Name: []string{"gammon", "bacon"}})

	t1 := test1{
		Thrig: thrig,
	}

	nb := NewNodeBuilder(&t1, &t1)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `thrig:
    pork:
        name:
            - gammon
            - bacon`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_ShouldHaveNotDoneTestsLikeThisOhWell(t *testing.T) {
	m := orderedmap.New[low.KeyReference[string], low.ValueReference[*valueReferenceStruct]]()

	m.Set(low.KeyReference[string]{
		KeyNode: utils.CreateStringNode("pizza"),
		Value:   "pizza",
	}, low.ValueReference[*valueReferenceStruct]{
		ValueNode: utils.CreateStringNode("beer"),
		Value:     &valueReferenceStruct{},
	})

	d := orderedmap.New[string, *valueReferenceStruct]()
	d.Set("pizza", &valueReferenceStruct{})

	type t1low struct {
		Thril low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*valueReferenceStruct]]]
		Thugg *bool    `yaml:"thugg"`
		Throo *float32 `yaml:"throo"`
	}

	t1 := test1{
		Thril: d,
	}

	t2 := t1low{
		Thril: low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*valueReferenceStruct]]]{
			Value:     m,
			ValueNode: utils.CreateStringNode("beer"),
		},
	}

	nb := NewNodeBuilder(&t1, &t2)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `thril:
    pizza: pizza`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

type zeroer struct {
	AlwaysZero string `yaml:"thing"`
}

func (z zeroer) IsZero() bool {
	return true
}

func TestNodeBuilder_IsZeroer(t *testing.T) {
	type test struct {
		Thing zeroer `yaml:"thing"`
	}

	t1 := test{
		Thing: zeroer{
			AlwaysZero: "will never render",
		},
	}

	nb := NewNodeBuilder(&t1, nil)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	assert.Equal(t, "{}", strings.TrimSpace(string(data)))
}

func TestNodeBuilder_GetStringLowValue(t *testing.T) {
	type test struct {
		Thing string `yaml:"thing"`
	}
	type testLow struct {
		Thing low.ValueReference[string] `yaml:"thing"`
	}

	t1 := test{
		Thing: "thing",
	}
	lowNode := utils.CreateStringNode("thing")
	lowNode.Style = yaml.DoubleQuotedStyle
	t1Low := testLow{
		Thing: low.ValueReference[string]{
			Value:     "thing",
			ValueNode: lowNode,
		},
	}

	nb := NewNodeBuilder(&t1, &t1Low)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	assert.Equal(t, `thing: "thing"`, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_Float64_Negative(t *testing.T) {
	floatNum := -3.33333
	t1 := test1{
		Thral: &floatNum,
	}

	nb := NewNodeBuilder(&t1, &t1)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `thral: -3.33333`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_Int64_Negative(t *testing.T) {
	intNum := int64(-3)
	t1 := test1{
		Thurr: &intNum,
	}

	nb := NewNodeBuilder(&t1, &t1)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `thurr: -3`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_DescriptionOmitEmpty(t *testing.T) {
	t1 := struct {
		Blah string `yaml:"description"`
	}{
		Blah: "",
	}

	t2 := struct {
		Blah string `yaml:"description,omitempty"`
	}{
		Blah: "",
	}

	nb := NewNodeBuilder(&t1, &t1)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `description: ""` // response body needs this capability

	assert.Equal(t, desired, strings.TrimSpace(string(data)))

	nb = NewNodeBuilder(&t2, &t2)
	node = nb.Render()

	data, _ = yaml.Marshal(node)

	desired = `{}`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNodeBuilder_LowStructValueAccess(t *testing.T) {
	high := &structLowTest{
		Name: "libopenapi",
	}
	low := structLowTest{
		Name: "libopenapi",
	}

	nb := NewNodeBuilder(high, low)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	assert.Equal(t, "name: libopenapi", strings.TrimSpace(string(data)))
}

func TestNodeBuilder_LowPointerIsNotEmpty(t *testing.T) {
	nonEmptyExampleCallCount = 0
	high := &pointerFieldStruct{}
	low := &pointerFieldStruct{
		Example: &nonEmptyExample{},
	}

	nb := NewNodeBuilder(high, low)

	assert.Equal(t, 1, nonEmptyExampleCallCount)

	node := nb.Render()
	data, _ := yaml.Marshal(node)

	output := strings.TrimSpace(string(data))
	assert.Equal(t, "{}", output)
}
