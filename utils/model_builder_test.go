package utils

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    "github.com/stretchr/testify/assert"
    "gopkg.in/yaml.v3"
    "testing"
)

type hotdog struct {
    Name            low.NodeReference[string]
    Fat             low.NodeReference[int]
    Ketchup         low.NodeReference[float32]
    Mustard         low.NodeReference[float64]
    Grilled         low.NodeReference[bool]
    MaxTemp         low.NodeReference[int]
    Drinks          []low.NodeReference[string]
    Sides           []low.NodeReference[float32]
    BigSides        []low.NodeReference[float64]
    Temps           []low.NodeReference[int]
    Buns            []low.NodeReference[bool]
    UnknownElements low.ObjectReference
    LotsOfUnknowns  []low.ObjectReference
    Where           map[string]low.ObjectReference
}

func (h hotdog) Build(node *yaml.Node) {

}

func TestBuildModel(t *testing.T) {

    yml := `name: yummy
beef: true
fat: 200
ketchup: 200.45
mustard: 324938249028.98234892374892374923874823974
grilled: true
maxTemp: 250
drinks:
  - nice
  - rice
  - spice
sides:
  - 0.23
  - 22.23
  - 99.45
  - 22311.2234
bigSides:
  - 98237498.9872349872349872349872347982734927342983479234234234234234234
  - 9827347234234.982374982734987234987
  - 234234234.234982374982347982374982374982347
  - 987234987234987234982734.987234987234987234987234987234987234987234982734982734982734987234987234987234987
temps: 
  - 1
  - 2
buns:
 - true
 - false
unknownElements:
  well:
    whoKnows: not me?
  doYou:
    love: beerToo?
lotsOfUnknowns:
  - wow:
      what: aTrip
  - amazing:
      french: fries
  - amazing:
      french: fries
where:
  things:
    are:
      wild: out here
  howMany:
    bears: 200
`

    var rootNode yaml.Node
    mErr := yaml.Unmarshal([]byte(yml), &rootNode)
    assert.NoError(t, mErr)

    hd := hotdog{}
    cErr := BuildModel(&rootNode, &hd)
    assert.Equal(t, 200, hd.Fat.Value)
    assert.Equal(t, 3, hd.Fat.Node.Line)
    assert.Equal(t, true, hd.Grilled.Value)
    assert.Equal(t, "yummy", hd.Name.Value)
    assert.Equal(t, float32(200.45), hd.Ketchup.Value)
    assert.Len(t, hd.Drinks, 3)
    assert.Len(t, hd.Sides, 4)
    assert.Len(t, hd.BigSides, 4)
    assert.Len(t, hd.Temps, 2)
    assert.Equal(t, 2, hd.Temps[1].Value)
    assert.Equal(t, 24, hd.Temps[1].Node.Line)
    assert.Len(t, hd.UnknownElements.Value, 2)
    assert.Len(t, hd.LotsOfUnknowns, 3)
    assert.Len(t, hd.Where, 2)
    assert.Equal(t, 324938249028.98234892374892374923874823974, hd.Mustard.Value)
    assert.NoError(t, cErr)
}
