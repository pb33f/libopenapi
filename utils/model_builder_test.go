package utils

import (
    "github.com/stretchr/testify/assert"
    "gopkg.in/yaml.v3"
    "testing"
)

type spank[t any] struct {
    life t
}

type hotdog struct {
    Name       string
    Beef       bool
    Fat        int
    Ketchup    float32
    Mustard    float64
    Grilled    spank[bool]
    NotGrilled spank[string]
}

func (h hotdog) Build(node *yaml.Node) {

}

func TestBuildModel(t *testing.T) {

    yml := `name: yummy
beef: true
fat: 200
ketchup: 200.45
mustard: 324938249028.98234892374892374923874823974
grilled: false
notGrilled: false`

    var rootNode yaml.Node
    mErr := yaml.Unmarshal([]byte(yml), &rootNode)
    assert.NoError(t, mErr)

    hd := hotdog{}
    cErr := BuildModel(&rootNode, &hd)
    assert.Equal(t, 200, hd.Fat)
    assert.Equal(t, true, hd.Beef)
    assert.Equal(t, "yummy", hd.Name)
    assert.Equal(t, float32(200.45), hd.Ketchup)
    assert.Equal(t, 324938249028.98234892374892374923874823974, hd.Mustard)

    assert.NoError(t, cErr)

}
