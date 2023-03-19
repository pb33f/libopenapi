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
    Name             string `yaml:"name"`
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

type test1 struct {
    Thing      string              `yaml:"thing"`
    Thong      int                 `yaml:"thong"`
    Thrum      int64               `yaml:"thrum"`
    Thang      float32             `yaml:"thang"`
    Thung      float64             `yaml:"thung"`
    Thyme      bool                `yaml:"thyme"`
    Thurm      any                 `yaml:"thurm"`
    Thugg      *bool               `yaml:"thugg"`
    Thurr      *int64              `yaml:"thurr"`
    Thral      *float64            `yaml:"thral"`
    Tharg      []string            `yaml:"tharg"`
    Type       []string            `yaml:"type"`
    Throg      []*key              `yaml:"throg"`
    Thrug      map[string]string   `yaml:"thrug"`
    Thoom      []map[string]string `yaml:"thoom"`
    Thomp      map[key]string      `yaml:"thomp"`
    Thump      key                 `yaml:"thump"`
    Thane      key                 `yaml:"thane"`
    Thunk      key                 `yaml:"thunk"`
    Thrim      *key                `yaml:"thrim"`
    Thril      map[string]*key     `yaml:"thril"`
    Extensions map[string]any      `yaml:"-"`
    ignoreMe   string              `yaml:"-"`
    IgnoreMe   string              `yaml:"-"`
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
        Thurm:    &test1{},
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

    nb := NewNodeBuilder(&t1, &t1)
    node := nb.Render()

    data, _ := yaml.Marshal(node)

    desired := `thrug:
    chicken: nuggets
thomp:
    meddy: princess
thing: ding
thong: "1"
thrum: "1234567"
thang: 2.20
thung: 3.33333
thyme: true
thugg: true
thurr: 12345
thral: 1234.1234
tharg:
    - chicken
    - nuggets
type: chicken
thoom:
    - maddy: champion
    - ember: naughty
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
    assert.Len(t, data, 51)
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

    assert.Len(t, data, 51)
}

func TestNewNodeBuilder_NoValue(t *testing.T) {

    t1 := test1{
        Thing: "",
    }

    nb := NewNodeBuilder(&t1, &t1)
    node := nb.AddYAMLNode(nil, "", "", nil, 0)
    assert.Nil(t, node)
}

func TestNewNodeBuilder_EmptyString(t *testing.T) {
    t1 := new(test1)
    nb := NewNodeBuilder(t1, t1)
    node := nb.AddYAMLNode(nil, "", "", "", 0)
    assert.Nil(t, node)
}

func TestNewNodeBuilder_Bool(t *testing.T) {
    t1 := new(test1)
    nb := NewNodeBuilder(t1, t1)
    node := nb.AddYAMLNode(nil, "", "", false, 0)
    assert.Nil(t, node)
}

func TestNewNodeBuilder_Int(t *testing.T) {
    t1 := new(test1)
    nb := NewNodeBuilder(t1, t1)
    p := utils.CreateEmptyMapNode()
    node := nb.AddYAMLNode(p, "p", "p", 12, 0)
    assert.NotNil(t, node)
    assert.Len(t, node.Content, 2)
    assert.Equal(t, "12", node.Content[1].Value)
}

func TestNewNodeBuilder_Int64(t *testing.T) {
    t1 := new(test1)
    nb := NewNodeBuilder(t1, t1)
    p := utils.CreateEmptyMapNode()
    node := nb.AddYAMLNode(p, "p", "p", int64(234556), 0)
    assert.NotNil(t, node)
    assert.Len(t, node.Content, 2)
    assert.Equal(t, "234556", node.Content[1].Value)
}

func TestNewNodeBuilder_Float32(t *testing.T) {
    t1 := new(test1)
    nb := NewNodeBuilder(t1, t1)
    p := utils.CreateEmptyMapNode()
    node := nb.AddYAMLNode(p, "p", "p", float32(1234.23), 0)
    assert.NotNil(t, node)
    assert.Len(t, node.Content, 2)
    assert.Equal(t, "1234.23", node.Content[1].Value)
}

func TestNewNodeBuilder_Float64(t *testing.T) {
    t1 := new(test1)
    nb := NewNodeBuilder(t1, t1)
    p := utils.CreateEmptyMapNode()
    node := nb.AddYAMLNode(p, "p", "p", 1234.232323, 0)
    assert.NotNil(t, node)
    assert.Len(t, node.Content, 2)
    assert.Equal(t, "1234.232323", node.Content[1].Value)
}

func TestNewNodeBuilder_MapKeyHasValue(t *testing.T) {

    t1 := test1{
        Thrug: map[string]string{
            "dump": "trump",
        },
    }

    type test1low struct {
        Thrug key `yaml:"thrug"`
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
        Thomp key `yaml:"thomp"`
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
        Thomp key `yaml:"thomp"`
    }

    t2 := test1low{
        Thomp: key{
            v: map[key]string{
                {
                    v: key{
                        v:  "ice",
                        kn: utils.CreateStringNode("limes"),
                    },
                    kn: utils.CreateStringNode("meddy"),
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

func TestNewNodeBuilder_MissingLabel(t *testing.T) {

    t1 := new(test1)
    nb := NewNodeBuilder(t1, t1)
    p := utils.CreateEmptyMapNode()
    node := nb.AddYAMLNode(p, "", "p", 1234.232323, 0)
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

    assert.Len(t, data, 62)
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


