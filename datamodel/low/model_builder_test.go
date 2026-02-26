package low

import (
	"sync"
	"testing"

	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

type hotdog struct {
	Name            NodeReference[string]
	ValueName       ValueReference[string]
	Fat             NodeReference[int]
	Ketchup         NodeReference[float32]
	Mustard         NodeReference[float64]
	Grilled         NodeReference[bool]
	MaxTemp         NodeReference[int]
	MaxTempHigh     NodeReference[int64]
	MaxTempAlt      []NodeReference[int]
	Drinks          []NodeReference[string]
	Sides           []NodeReference[float32]
	BigSides        []NodeReference[float64]
	Temps           []NodeReference[int]
	HighTemps       []NodeReference[int64]
	Buns            []NodeReference[bool]
	UnknownElements NodeReference[*yaml.Node]
	LotsOfUnknowns  []NodeReference[*yaml.Node]
	Where           *orderedmap.Map[string, NodeReference[*yaml.Node]]
	There           *orderedmap.Map[string, NodeReference[string]]
	AllTheThings    NodeReference[*orderedmap.Map[KeyReference[string], ValueReference[string]]]
}

func TestBuildModel_Mismatch(t *testing.T) {
	yml := `crisps: are tasty`

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	hd := hotdog{}
	cErr := BuildModel(&rootNode, &hd)
	assert.NoError(t, cErr)
	assert.Empty(t, hd.Name)
}

func TestBuildModel(t *testing.T) {
	yml := `name: yummy
valueName: yammy
beef: true
fat: 200
ketchup: 200.45
mustard: 324938249028.98234892374892374923874823974
grilled: true
maxTemp: 250
maxTempAlt: [1,2,3,4,5]
maxTempHigh: 7392837462032342
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
highTemps: 
  - 827349283744710
  - 11732849090192923
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
there:
  oh: yeah
  care: bear
allTheThings:
  beer: isGood
  cake: isNice`

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	hd := hotdog{}
	cErr := BuildModel(rootNode.Content[0], &hd)
	assert.Equal(t, 200, hd.Fat.Value)
	assert.Equal(t, 4, hd.Fat.ValueNode.Line)
	assert.Equal(t, true, hd.Grilled.Value)
	assert.Equal(t, "yummy", hd.Name.Value)
	assert.Equal(t, "yammy", hd.ValueName.Value)
	assert.Equal(t, float32(200.45), hd.Ketchup.Value)
	assert.Len(t, hd.Drinks, 3)
	assert.Len(t, hd.Sides, 4)
	assert.Len(t, hd.BigSides, 4)
	assert.Len(t, hd.Temps, 2)
	assert.Len(t, hd.HighTemps, 2)
	assert.Equal(t, int64(11732849090192923), hd.HighTemps[1].Value)
	assert.Len(t, hd.MaxTempAlt, 5)
	assert.Equal(t, int64(7392837462032342), hd.MaxTempHigh.Value)
	assert.Equal(t, 2, hd.Temps[1].Value)
	assert.Equal(t, 27, hd.Temps[1].ValueNode.Line)

	var unknownElements map[string]any
	_ = hd.UnknownElements.Value.Decode(&unknownElements)

	assert.Len(t, unknownElements, 2)
	assert.Len(t, hd.LotsOfUnknowns, 3)
	assert.Equal(t, 2, orderedmap.Len(hd.Where))
	assert.Equal(t, 2, orderedmap.Len(hd.There))
	assert.Equal(t, "bear", hd.There.GetOrZero("care").Value)
	assert.Equal(t, 324938249028.98234892374892374923874823974, hd.Mustard.Value)

	allTheThings := hd.AllTheThings.Value
	for k, v := range allTheThings.FromOldest() {
		if k.Value == "beer" {
			assert.Equal(t, "isGood", v.Value)
		}
		if k.Value == "cake" {
			assert.Equal(t, "isNice", v.Value)
		}
	}
	assert.NoError(t, cErr)
}

func TestBuildModel_UseCopyNotRef(t *testing.T) {
	yml := `cake: -99999`

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	hd := hotdog{}
	cErr := BuildModel(&rootNode, hd)
	assert.Error(t, cErr)
	assert.Empty(t, hd.Name)
}

func TestBuildModel_UseUnsupportedPrimitive(t *testing.T) {
	// Exported field with a primitive Go type (string) that has no NodeReference wrapper.
	type notSupported struct {
		Cake string
	}
	ns := notSupported{}
	yml := `cake: party`

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	cErr := BuildModel(rootNode.Content[0], &ns)
	assert.Error(t, cErr)
	assert.Empty(t, ns.Cake)
}

func TestBuildModel_SkipsUnexportedFields(t *testing.T) {
	// Unexported fields should be silently skipped, even if they match a YAML key.
	type hasUnexported struct {
		context string //nolint:unused
	}
	h := hasUnexported{}
	yml := `context: hello`

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	cErr := BuildModel(rootNode.Content[0], &h)
	assert.NoError(t, cErr)
}

func TestBuildModel_UsingInternalConstructs(t *testing.T) {
	type internal struct {
		Extensions NodeReference[string]
		PathItems  NodeReference[string]
		Thing      NodeReference[string]
	}

	yml := `extensions: one
pathItems: two
thing: yeah`

	ins := new(internal)
	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	// try a null build
	try := BuildModel(nil, ins)
	assert.NoError(t, try)

	cErr := BuildModel(rootNode.Content[0], ins)
	assert.NoError(t, cErr)
	assert.Empty(t, ins.PathItems.Value)
	assert.Empty(t, ins.Extensions.Value)
	assert.Equal(t, "yeah", ins.Thing.Value)
}

func TestSetField_MapHelperWrapped(t *testing.T) {
	type internal struct {
		Thing KeyReference[*orderedmap.Map[KeyReference[string], ValueReference[string]]]
	}

	yml := `thing: 
  what: not
  chip: chop
  lip: lop`

	ins := new(internal)
	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	try := BuildModel(rootNode.Content[0], ins)
	assert.NoError(t, try)
	assert.Equal(t, 3, orderedmap.Len(ins.Thing.Value))
}

func TestSetField_MapHelper(t *testing.T) {
	type internal struct {
		Thing *orderedmap.Map[KeyReference[string], ValueReference[string]]
	}

	yml := `thing: 
  what: not
  chip: chop
  lip: lop`

	ins := new(internal)
	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	try := BuildModel(rootNode.Content[0], ins)
	assert.NoError(t, try)
	assert.Equal(t, 3, orderedmap.Len(ins.Thing))
}

func TestSetField_ArrayHelper(t *testing.T) {
	type internal struct {
		Thing NodeReference[[]ValueReference[string]]
	}

	yml := `thing: 
  - nice
  - rice
  - slice`

	ins := new(internal)
	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	try := BuildModel(rootNode.Content[0], ins)
	assert.NoError(t, try)
	assert.Len(t, ins.Thing.Value, 3)
}

func TestSetField_Enum_Helper(t *testing.T) {
	type internal struct {
		Thing NodeReference[[]ValueReference[*yaml.Node]]
	}

	yml := `thing: 
  - nice
  - rice
  - slice`

	ins := new(internal)
	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	try := BuildModel(rootNode.Content[0], ins)
	assert.NoError(t, try)
	assert.Len(t, ins.Thing.Value, 3)
}

func TestSetField_Default_Helper(t *testing.T) {
	type cake struct {
		thing int
	}

	// this should be ignored, no custom objects in here my friend.
	type internal struct {
		Thing cake
	}

	yml := `thing: 
  type: cake`

	ins := new(internal)
	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	try := BuildModel(rootNode.Content[0], ins)
	assert.NoError(t, try)
	assert.Equal(t, 0, ins.Thing.thing)
}

func TestHandleSlicesOfInts(t *testing.T) {
	type internal struct {
		Thing NodeReference[[]ValueReference[*yaml.Node]]
	}

	yml := `thing:
  - 5
  - 1.234`

	ins := new(internal)
	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	try := BuildModel(rootNode.Content[0], ins)
	assert.NoError(t, try)

	var thing0 int64
	_ = ins.Thing.GetValue()[0].Value.Decode(&thing0)

	var thing1 float64
	_ = ins.Thing.GetValue()[1].Value.Decode(&thing1)

	assert.Equal(t, int64(5), thing0)
	assert.Equal(t, 1.234, thing1)
}

func TestHandleSlicesOfBools(t *testing.T) {
	type internal struct {
		Thing NodeReference[[]ValueReference[*yaml.Node]]
	}

	yml := `thing:
  - true
  - false`

	ins := new(internal)
	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	try := BuildModel(rootNode.Content[0], ins)

	var thing0 bool
	_ = ins.Thing.GetValue()[0].Value.Decode(&thing0)

	var thing1 bool
	_ = ins.Thing.GetValue()[1].Value.Decode(&thing1)

	assert.NoError(t, try)
	assert.Equal(t, true, thing0)
	assert.Equal(t, false, thing1)
}

func TestSetField_Ignore(t *testing.T) {
	type Complex struct{}
	type internal struct {
		Thing *Complex
	}

	yml := `thing: 
  - nice
  - rice
  - slice`

	ins := new(internal)
	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	try := BuildModel(&rootNode, ins)
	assert.NoError(t, try)
	assert.Nil(t, ins.Thing)
}

func TestBuildModelAsync(t *testing.T) {
	type internal struct {
		Thing KeyReference[*orderedmap.Map[KeyReference[string], ValueReference[string]]]
	}

	yml := `thing: 
  what: not
  chip: chop
  lip: lop`

	ins := new(internal)
	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	var wg sync.WaitGroup
	var errors []error
	wg.Add(1)
	BuildModelAsync(rootNode.Content[0], ins, &wg, &errors)
	wg.Wait()
	assert.Equal(t, 3, orderedmap.Len(ins.Thing.Value))
}

func TestSetField_NilValueNode(t *testing.T) {
	assert.NotPanics(t, func() {
		SetField(nil, nil, nil)
	})
}

func TestBuildModelAsync_HandlesError(t *testing.T) {
	errs := []error{}
	wg := sync.WaitGroup{}
	wg.Add(1)
	BuildModelAsync(utils.CreateStringNode("cake"), "cake", &wg, &errs)
	assert.NotEmpty(t, errs)
}
