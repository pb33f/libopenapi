// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package high

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    "github.com/stretchr/testify/assert"
    "gopkg.in/yaml.v3"
    "strings"
    "testing"
)

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
    Thoom      []map[string]string `yaml:"thoom"`
    Thomp      map[string]string   `yaml:"thomp"`
    Extensions map[string]any      `yaml:"-"`
    ignoreMe   string              `yaml:"-"`
    IgnoreMe   string              `yaml:"-"`
}

func (te *test1) GetExtensions() map[low.KeyReference[string]]low.ValueReference[any] {

    g := make(map[low.KeyReference[string]]low.ValueReference[any])

    for i := range te.Extensions {
        vn := CreateStringNode(te.Extensions[i].(string))
        vn.Line = 999999 // weighted to the bottom.
        g[low.KeyReference[string]{
            Value:   i,
            KeyNode: vn,
        }] = low.ValueReference[any]{
            ValueNode: vn,
            Value:     te.Extensions[i].(string),
        }
    }
    return g
}

func (te *test1) MarshalYAML() (interface{}, error) {
    nb := NewNodeBuilder(te, te)
    return nb.Render(), nil
}

func (te *test1) GetKeyNode() *yaml.Node {
    kn := CreateStringNode("test1")
    kn.Line = 20
    return kn
}

func TestNewNodeBuilder_Components(t *testing.T) {

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
        Thomp: map[string]string{
            "meddy": "princess",
        },
        Extensions: map[string]any{
            "x-pizza": "time",
        },
    }

    nb := NewNodeBuilder(&t1, &t1)
    node := nb.Render()

    data, _ := yaml.Marshal(node)

    desired := `thing: ding
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