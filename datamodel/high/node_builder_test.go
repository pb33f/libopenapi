// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package high

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"reflect"
	"strings"
	"testing"
)

type key struct {
	Name             string `yaml:"name,omitempty"`
	ref              bool
	refStr           string
	ln               int
	nilval           bool
	v                any
	kn               *yaml.Node
	low.IsReferenced `yaml:"-"`
}

func (k key) GetKeyNode() *yaml.Node {
	if k.kn != nil {
		return k.kn
	}
	kn := utils.CreateStringNode("meddy")
	kn.Line = k.ln
	return kn
}

func (k key) GetValueUntyped() any {
	return k.v
}

func (k key) GetValueNode() *yaml.Node {
	return k.GetValueNodeUntyped()
}

func (k key) GetValueNodeUntyped() *yaml.Node {
	if k.nilval {
		return nil
	}
	kn := utils.CreateStringNode("maddy")
	kn.Line = k.ln
	return kn
}

func (k key) IsReference() bool {
	return k.ref
}

func (k key) GetReference() string {
	return k.refStr
}

func (k key) SetReference(ref string) {
	k.refStr = ref
}

func (k key) GoLowUntyped() any {
	return &k
}

func (k key) MarshalYAML() (interface{}, error) {
	return utils.CreateStringNode("pizza"), nil
}

func (k key) MarshalYAMLInline() (interface{}, error) {
	return utils.CreateStringNode("pizza-inline!"), nil
}

type plug struct {
	Name []string `yaml:"name,omitempty"`
}

type test1 struct {
	Thrig      map[string]*plug      `yaml:"thrig,omitempty"`
	Thing      string                `yaml:"thing,omitempty"`
	Thong      int                   `yaml:"thong,omitempty"`
	Thrum      int64                 `yaml:"thrum,omitempty"`
	Thang      float32               `yaml:"thang,omitempty"`
	Thung      float64               `yaml:"thung,omitempty"`
	Thyme      bool                  `yaml:"thyme,omitempty"`
	Thurm      any                   `yaml:"thurm,omitempty"`
	Thugg      *bool                 `yaml:"thugg,renderZero"`
	Thurr      *int64                `yaml:"thurr,omitempty"`
	Thral      *float64              `yaml:"thral,omitempty"`
	Throo      *float64              `yaml:"throo,renderZero,omitempty"`
	Tharg      []string              `yaml:"tharg,omitempty"`
	Type       []string              `yaml:"type,omitempty"`
	Throg      []*key                `yaml:"throg,omitempty"`
	Thrat      []interface{}         `yaml:"thrat,omitempty"`
	Thrag      []map[string][]string `yaml:"thrag,omitempty"`
	Thrug      map[string]string     `yaml:"thrug,omitempty"`
	Thoom      []map[string]string   `yaml:"thoom,omitempty"`
	Thomp      map[key]string        `yaml:"thomp,omitempty"`
	Thump      key                   `yaml:"thump,omitempty"`
	Thane      key                   `yaml:"thane,omitempty"`
	Thunk      key                   `yaml:"thunk,omitempty"`
	Thrim      *key                  `yaml:"thrim,omitempty"`
	Thril      map[string]*key       `yaml:"thril,omitempty"`
	Extensions map[string]any        `yaml:"-"`
	ignoreMe   string                `yaml:"-"`
	IgnoreMe   string                `yaml:"-"`
}

func (te *test1) GetExtensions() map[low.KeyReference[string]]low.ValueReference[any] {

	g := make(map[low.KeyReference[string]]low.ValueReference[any])

	for i := range te.Extensions {

		f := reflect.TypeOf(te.Extensions[i])
		switch f.Kind() {
		case reflect.String:
			vn := utils.CreateStringNode(te.Extensions[i].(string))
			vn.Line = 999999 // weighted to the bottom.
			g[low.KeyReference[string]{
				Value:   i,
				KeyNode: vn,
			}] = low.ValueReference[any]{
				ValueNode: vn,
				Value:     te.Extensions[i].(string),
			}
		case reflect.Map:
			kn := utils.CreateStringNode(i)
			var vn yaml.Node
			_ = vn.Decode(te.Extensions[i])

			kn.Line = 999999 // weighted to the bottom.
			g[low.KeyReference[string]{
				Value:   i,
				KeyNode: kn,
			}] = low.ValueReference[any]{
				ValueNode: &vn,
				Value:     te.Extensions[i],
			}
		}

	}
	return g
}

func (te *test1) MarshalYAML() (interface{}, error) {
	nb := NewNodeBuilder(te, te)
	return nb.Render(), nil
}

func (te *test1) GetKeyNode() *yaml.Node {
	kn := utils.CreateStringNode("meddy")
	kn.Line = 20
	return kn
}

func (te *test1) GoesLowUntyped() any {
	return te
}

func TestNewNodeBuilder(t *testing.T) {

	b := true
	c := int64(12345)
	d := 1234.1234

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
		Thoom: []map[string]string{
			{
				"maddy": "champion",
			},
			{
				"ember": "naughty",
			},
		},
		Thomp: map[key]string{
			{ln: 1}: "princess",
		},
		Thane: key{ // this is going to be ignored, needs to be a ValueReference
			ln:     2,
			ref:    true,
			refStr: "ripples",
		},
		Thrug: map[string]string{
			"chicken": "nuggets",
		},
		Thump: key{Name: "I will be ignored", ln: 3},
		Thunk: key{ln: 4, nilval: true},
		Extensions: map[string]any{
			"x-pizza": "time",
		},
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

	t1 := key{
		Name:   "cotton",
		ref:    true,
		refStr: "#/my/heart",
		ln:     2,
	}

	nb := NewNodeBuilder(&t1, &t1)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `$ref: '#/my/heart'`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_Extensions(t *testing.T) {

	t1 := test1{
		Thing: "ding",
		Extensions: map[string]any{
			"x-pizza": "time",
			"x-money": "time",
		},
		Thong: 1,
	}

	nb := NewNodeBuilder(&t1, &t1)
	node := nb.Render()

	data, _ := yaml.Marshal(node)
	assert.Len(t, data, 49)
}

func TestNewNodeBuilder_LowValueNode(t *testing.T) {

	t1 := test1{
		Thing: "ding",
		Extensions: map[string]any{
			"x-pizza": "time",
			"x-money": "time",
		},
		Thong: 1,
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

	nodeEnty := NodeEntry{}
	nb := NewNodeBuilder(&t1, &t1)
	node := nb.AddYAMLNode(nil, &nodeEnty)
	assert.Nil(t, node)
}

func TestNewNodeBuilder_EmptyString(t *testing.T) {
	t1 := new(test1)
	nodeEnty := NodeEntry{}
	nb := NewNodeBuilder(t1, t1)
	node := nb.AddYAMLNode(nil, &nodeEnty)
	assert.Nil(t, node)
}

func TestNewNodeBuilder_EmptyStringRenderZero(t *testing.T) {
	t1 := new(test1)
	nodeEnty := NodeEntry{RenderZero: true, Value: ""}
	nb := NewNodeBuilder(t1, t1)
	m := utils.CreateEmptyMapNode()
	node := nb.AddYAMLNode(m, &nodeEnty)
	assert.NotNil(t, node)
}

func TestNewNodeBuilder_Bool(t *testing.T) {
	t1 := new(test1)
	nb := NewNodeBuilder(t1, t1)
	nodeEnty := NodeEntry{}
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
	nodeEnty := NodeEntry{Tag: "p", Value: 12, Key: "p"}
	node := nb.AddYAMLNode(p, &nodeEnty)
	assert.NotNil(t, node)
	assert.Len(t, node.Content, 2)
	assert.Equal(t, "12", node.Content[1].Value)
}

func TestNewNodeBuilder_Int64(t *testing.T) {
	t1 := new(test1)
	nb := NewNodeBuilder(t1, t1)
	p := utils.CreateEmptyMapNode()
	nodeEnty := NodeEntry{Tag: "p", Value: int64(234556), Key: "p"}
	node := nb.AddYAMLNode(p, &nodeEnty)
	assert.NotNil(t, node)
	assert.Len(t, node.Content, 2)
	assert.Equal(t, "234556", node.Content[1].Value)
}

func TestNewNodeBuilder_Float32(t *testing.T) {
	t1 := new(test1)
	nb := NewNodeBuilder(t1, t1)
	p := utils.CreateEmptyMapNode()
	nodeEnty := NodeEntry{Tag: "p", Value: float32(1234.23), Key: "p"}
	node := nb.AddYAMLNode(p, &nodeEnty)
	assert.NotNil(t, node)
	assert.Len(t, node.Content, 2)
	assert.Equal(t, "1234.23", node.Content[1].Value)
}

func TestNewNodeBuilder_Float64(t *testing.T) {
	t1 := new(test1)
	nb := NewNodeBuilder(t1, t1)
	p := utils.CreateEmptyMapNode()
	nodeEnty := NodeEntry{Tag: "p", Value: 1234.232323, Key: "p", StringValue: "1234.232323"}
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

	t1 := test1{
		Thrug: map[string]string{
			"dump": "trump",
		},
	}

	type test1low struct {
		Thrug key   `yaml:"thrug"`
		Thugg *bool `yaml:"thugg"`
	}

	t2 := test1low{
		Thrug: key{
			v: map[string]string{
				"dump": "trump",
			},
			ln: 2,
		},
	}

	nb := NewNodeBuilder(&t1, &t2)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	desired := `thrug:
    dump: trump`

	assert.Equal(t, desired, strings.TrimSpace(string(data)))
}

func TestNewNodeBuilder_MapKeyHasValueThatHasValue(t *testing.T) {

	t1 := test1{
		Thomp: map[key]string{
			{v: "who"}: "princess",
		},
	}

	type test1low struct {
		Thomp key   `yaml:"thomp"`
		Thugg *bool `yaml:"thugg"`
	}

	t2 := test1low{
		Thomp: key{
			v: map[key]string{
				{
					v: key{
						v:  "ice",
						kn: utils.CreateStringNode("limes"),
					},
					kn: utils.CreateStringNode("chimes"),
					ln: 6}: "princess",
			},
			ln: 2,
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

	t1 := test1{
		Thomp: map[key]string{
			{v: "who"}: "princess",
		},
	}

	type test1low struct {
		Thomp low.NodeReference[map[key]string] `yaml:"thomp"`
		Thugg *bool                             `yaml:"thugg"`
	}

	g := low.NodeReference[map[key]string]{
		Value: map[key]string{
			{v: "my", kn: utils.CreateStringNode("limes")}: "princess",
		},
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

func TestNewNodeBuilder_MapKeyHasValueThatHasValueMatchKeyNode(t *testing.T) {

	t1 := test1{
		Thomp: map[key]string{
			{v: "who"}: "princess",
		},
	}

	type test1low struct {
		Thomp low.NodeReference[map[key]string] `yaml:"thomp"`
		Thugg *bool                             `yaml:"thugg"`
	}

	g := low.NodeReference[map[key]string]{
		Value: map[key]string{
			{v: "my", kn: utils.CreateStringNode("limes")}: "princess",
		},
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

func TestNewNodeBuilder_MapKeyHasValueThatHasValueMatch_NoWrap(t *testing.T) {

	t1 := test1{
		Thomp: map[key]string{
			{v: "who"}: "princess",
		},
	}

	type test1low struct {
		Thomp map[key]string `yaml:"thomp"`
		Thugg *bool          `yaml:"thugg"`
	}

	t2 := test1low{
		Thomp: map[key]string{
			{v: "my", kn: utils.CreateStringNode("meddy")}: "princess",
		},
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
	nodeEnty := NodeEntry{Value: 1234.232323, Key: "p"}
	node := nb.AddYAMLNode(p, &nodeEnty)
	assert.NotNil(t, node)
	assert.Len(t, node.Content, 0)
}

func TestNewNodeBuilder_ExtensionMap(t *testing.T) {

	t1 := test1{
		Thing: "ding",
		Extensions: map[string]any{
			"x-pizza": map[string]string{
				"dump": "trump",
			},
			"x-money": "time",
		},
		Thong: 1,
	}

	nb := NewNodeBuilder(&t1, &t1)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	assert.Len(t, data, 60)
}

func TestNewNodeBuilder_MapKeyHasValueThatHasValueMismatch(t *testing.T) {

	t1 := test1{
		Extensions: map[string]any{
			"x-pizza": map[string]string{
				"dump": "trump",
			},
			"x-cake": map[string]string{
				"maga": "nomore",
			},
		},
		Thril: map[string]*key{
			"princess": {v: "who", Name: "beef", ln: 2},
			"heavy":    {v: "who", Name: "industries", ln: 3},
		},
	}

	nb := NewNodeBuilder(&t1, nil)
	node := nb.Render()

	data, _ := yaml.Marshal(node)

	assert.Len(t, data, 94)
}

func TestNewNodeBuilder_SliceRef(t *testing.T) {

	c := key{ref: true, refStr: "#/red/robin/yummmmm", Name: "milky"}
	ty := []*key{&c}
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

	c := key{ref: true, refStr: "#/red/robin/yummmmm", Name: "milky"}
	ty := []*key{&c}
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

type testRender struct {
}

func (t testRender) MarshalYAML() (interface{}, error) {
	return utils.CreateStringNode("testy!"), nil
}

type testRenderRawNode struct {
}

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

	ty := key{}
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

	c := key{ref: false, Name: "milky"}
	ty := []*key{&c}
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
		Thrim: &key{
			ref:    true,
			refStr: "#/cash/money",
			Name:   "pizza",
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

	t1 := test1{
		Thurm: low.NodeReference[string]{
			Reference:     "#/cash/money",
			ReferenceNode: true,
			KeyNode:       fkn,
			ValueNode:     fkn,
		},
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

	a := []map[string][]string{
		{"pizza": {"beer", "wine"}},
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

	t1 := test1{
		Thrig: map[string]*plug{
			"pork": {Name: []string{"gammon", "bacon"}},
		},
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

	m := make(map[low.KeyReference[string]]low.ValueReference[*key])

	m[low.KeyReference[string]{
		KeyNode: utils.CreateStringNode("pizza"),
		Value:   "pizza",
	}] = low.ValueReference[*key]{
		ValueNode: utils.CreateStringNode("beer"),
		Value:     &key{},
	}

	d := make(map[string]*key)
	d["pizza"] = &key{}

	type t1low struct {
		Thril low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*key]]
		Thugg *bool `yaml:"thugg"`
	}

	t1 := test1{
		Thril: d,
	}

	t2 := t1low{
		Thril: low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*key]]{
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
