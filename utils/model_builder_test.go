package utils

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    "github.com/stretchr/testify/assert"
    "gopkg.in/yaml.v3"
    "testing"
)

type hotdog struct {
    Name    low.NodeReference[string]
    Beef    low.NodeReference[bool]
    Fat     low.NodeReference[int]
    Ketchup low.NodeReference[float32]
    Mustard low.NodeReference[float64]
    Grilled low.NodeReference[bool]
    MaxTemp low.NodeReference[int]
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
maxTemp: 250
`

    var rootNode yaml.Node
    mErr := yaml.Unmarshal([]byte(yml), &rootNode)
    assert.NoError(t, mErr)

    hd := hotdog{}
    cErr := BuildModel(&rootNode, &hd)
    assert.Equal(t, 200, hd.Fat.Value)
    assert.Equal(t, 3, hd.Fat.Node.Line)
    assert.Equal(t, true, hd.Beef.Value)
    assert.Equal(t, "yummy", hd.Name.Value)
    assert.Equal(t, float32(200.45), hd.Ketchup.Value)
    assert.Equal(t, 324938249028.98234892374892374923874823974, hd.Mustard.Value)
    assert.NoError(t, cErr)

}
