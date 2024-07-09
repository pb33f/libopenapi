package utils

import (
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

type petstore []byte

var once sync.Once

var psBytes petstore

func getPetstore() petstore {
	once.Do(func() {
		psBytes, _ = os.ReadFile("../test_specs/petstorev3.json")
	})
	return psBytes
}

func TestRenderCodeSnippet(t *testing.T) {
	code := []string{"hey", "ho", "let's", "go!"}
	startNode := &yaml.Node{
		Line: 1,
	}
	rendered := RenderCodeSnippet(startNode, code, 1, 3)
	assert.Equal(t, "hey\nho\nlet's\n", rendered)
}

func TestRenderCodeSnippet_BelowStart(t *testing.T) {
	code := []string{"hey", "ho", "let's", "go!"}
	startNode := &yaml.Node{
		Line: 0,
	}
	rendered := RenderCodeSnippet(startNode, code, 1, 3)
	assert.Equal(t, "hey\nho\nlet's\n", rendered)
}

func TestFindNodes(t *testing.T) {
	nodes, err := FindNodes(getPetstore(), "$.info.contact")
	assert.NoError(t, err)
	assert.NotNil(t, nodes)
	assert.Len(t, nodes, 1)
}

func TestFindNodes_BadPath(t *testing.T) {
	nodes, err := FindNodes(getPetstore(), "I am not valid")
	assert.Error(t, err)
	assert.Nil(t, nodes)
}

func TestFindLastChildNode(t *testing.T) {
	nodes, _ := FindNodes(getPetstore(), "$.info")
	lastNode := FindLastChildNode(nodes[0])
	assert.Equal(t, "1.0.11", lastNode.Value) // should be the version.
	lastNodeDouble := FindLastChildNodeWithLevel(nodes[0], 0)
	assert.Equal(t, lastNode, lastNodeDouble)
}

func TestFindLastChildNode_WithKids(t *testing.T) {
	nodes, _ := FindNodes(getPetstore(), "$.paths./pet")
	lastNode := FindLastChildNode(nodes[0])
	lastNodeDouble := FindLastChildNodeWithLevel(nodes[0], 0)
	assert.Equal(t, lastNode, lastNodeDouble)
	assert.Equal(t, "read:pets", lastNode.Value)
}

func TestFindLastChildNode_NotFound(t *testing.T) {
	node := &yaml.Node{
		Value: "same",
	}
	lastNode := FindLastChildNode(node)
	assert.Equal(t, "same", lastNode.Value) // should be the same node
	lastNodeDouble := FindLastChildNodeWithLevel(node, 0)
	assert.Equal(t, lastNode, lastNodeDouble)
}

func genLoop(count int) *yaml.Node {
	if count > 200 {
		return nil
	}
	count++
	return &yaml.Node{
		Value: "same",
		Content: []*yaml.Node{
			genLoop(count),
		},
	}
}

func TestFindLastChildNode_TooDeep(t *testing.T) {
	node := genLoop(0)
	lastNodeDouble := FindLastChildNodeWithLevel(node, 0)
	assert.NotNil(t, lastNodeDouble)
}

func TestBuildPath(t *testing.T) {
	assert.Equal(t, "$.fresh.fish.and.chicken.nuggets",
		BuildPath("$.fresh.fish", []string{"and", "chicken", "nuggets"}))
}

func TestBuildPath_WithTrailingPeriod(t *testing.T) {
	assert.Equal(t, "$.fresh.fish.and.chicken.nuggets",
		BuildPath("$.fresh.fish", []string{"and", "chicken", "nuggets", ""}))
}

func TestFindNodesWithoutDeserializing(t *testing.T) {
	root, _ := FindNodes(getPetstore(), "$")
	nodes, err := FindNodesWithoutDeserializing(root[0], "$.info.contact")
	assert.NoError(t, err)
	assert.NotNil(t, nodes)
	assert.Len(t, nodes, 1)
}

func TestFindNodesWithoutDeserializing_InvalidPath(t *testing.T) {
	root, _ := FindNodes(getPetstore(), "$")
	nodes, err := FindNodesWithoutDeserializing(root[0], "I love a good curry")
	assert.Error(t, err)
	assert.Nil(t, nodes)
}

func TestConvertInterfaceIntoStringMap(t *testing.T) {
	var d interface{}
	n := make(map[string]string)
	n["melody"] = "baby girl"
	d = n
	parsed := ConvertInterfaceIntoStringMap(d)
	assert.Equal(t, "baby girl", parsed["melody"])
}

func TestConvertInterfaceIntoStringMap_Float64(t *testing.T) {
	var d interface{}
	n := make(map[string]interface{})
	n["melody"] = 5.9
	d = n
	parsed := ConvertInterfaceIntoStringMap(d)
	assert.Equal(t, "5.9", parsed["melody"])
}

func TestConvertInterfaceIntoStringMap_Bool(t *testing.T) {
	var d interface{}
	n := make(map[string]interface{})
	n["melody"] = true
	d = n
	parsed := ConvertInterfaceIntoStringMap(d)
	assert.Equal(t, "true", parsed["melody"])
}

func TestConvertInterfaceIntoStringMap_int64(t *testing.T) {
	var d interface{}
	n := make(map[string]interface{})
	n["melody"] = int64(12345)
	d = n
	parsed := ConvertInterfaceIntoStringMap(d)
	assert.Equal(t, "12345", parsed["melody"])
}

func TestConvertInterfaceIntoStringMap_int(t *testing.T) {
	var d interface{}
	n := make(map[string]interface{})
	n["melody"] = 12345
	d = n
	parsed := ConvertInterfaceIntoStringMap(d)
	assert.Equal(t, "12345", parsed["melody"])
}

func TestConvertInterfaceIntoStringMap_NoType(t *testing.T) {
	var d interface{}
	n := make(map[string]interface{})
	n["melody"] = "baby girl"
	d = n
	parsed := ConvertInterfaceIntoStringMap(d)
	assert.Equal(t, "baby girl", parsed["melody"])
}

func TestConvertInterfaceToStringArray(t *testing.T) {
	var d interface{}
	n := make(map[string][]string)
	n["melody"] = []string{"melody", "is", "my", "baby"}
	d = n
	parsed := ConvertInterfaceToStringArray(d)
	assert.Equal(t, "baby", parsed[3])
}

func TestConvertInterfaceToStringArray_NoType(t *testing.T) {
	var d interface{}
	m := make([]interface{}, 4)
	n := make(map[string]interface{})
	m[0] = "melody"
	m[1] = "is"
	m[2] = "my"
	m[3] = "baby"
	n["melody"] = m
	d = n
	parsed := ConvertInterfaceToStringArray(d)
	assert.Equal(t, "baby", parsed[3])
}

func TestConvertInterfaceToStringArray_Invalid(t *testing.T) {
	var d interface{} = "I am a carrot"
	parsed := ConvertInterfaceToStringArray(d)
	assert.Nil(t, parsed)
}

func TestConvertInterfaceArrayToStringArray(t *testing.T) {
	var d interface{}
	m := []string{"maddox", "is", "my", "little", "champion"}
	d = m
	parsed := ConvertInterfaceArrayToStringArray(d)
	assert.Equal(t, "little", parsed[3])
}

func TestConvertInterfaceArrayToStringArray_NoType(t *testing.T) {
	var d interface{}
	m := make([]interface{}, 4)
	m[0] = "melody"
	m[1] = "is"
	m[2] = "my"
	m[3] = "baby"
	d = m
	parsed := ConvertInterfaceArrayToStringArray(d)
	assert.Equal(t, "baby", parsed[3])
}

func TestConvertInterfaceArrayToStringArray_Invalid(t *testing.T) {
	var d interface{} = "weed is good"
	parsed := ConvertInterfaceArrayToStringArray(d)
	assert.Nil(t, parsed)
}

func TestExtractValueFromInterfaceMap(t *testing.T) {
	var d interface{}
	m := make(map[string][]string)
	m["melody"] = []string{"is", "my", "baby"}
	d = m
	parsed := ExtractValueFromInterfaceMap("melody", d)
	assert.Equal(t, "baby", parsed.([]string)[2])
}

func TestExtractValueFromInterfaceMap_NoType(t *testing.T) {
	var d interface{}
	m := make(map[string]interface{})
	n := make([]interface{}, 3)
	n[0] = "maddy"
	n[1] = "the"
	n[2] = "champion"
	m["maddy"] = n
	d = m
	parsed := ExtractValueFromInterfaceMap("maddy", d)
	assert.Equal(t, "champion", parsed.([]interface{})[2])
}

func TestExtractValueFromInterfaceMap_Flat(t *testing.T) {
	var d interface{}
	m := make(map[string]interface{})
	m["maddy"] = "niblet"
	d = m
	parsed := ExtractValueFromInterfaceMap("maddy", d)
	assert.Equal(t, "niblet", parsed)
}

func TestExtractValueFromInterfaceMap_NotFound(t *testing.T) {
	var d interface{} = "not a map"
	parsed := ExtractValueFromInterfaceMap("melody", d)
	assert.Nil(t, parsed)
}

func TestFindFirstKeyNode(t *testing.T) {
	nodes, _ := FindNodes(getPetstore(), "$")
	key, value := FindFirstKeyNode("operationId", nodes, 0)
	assert.NotNil(t, key)
	assert.NotNil(t, value)
	assert.Equal(t, 55, key.Line)
}

func TestFindFirstKeyNode_NotFound(t *testing.T) {
	nodes, _ := FindNodes(getPetstore(), "$")
	key, value := FindFirstKeyNode("i-do-not-exist-in-the-doc", nodes, 0)
	assert.Nil(t, key)
	assert.Nil(t, value)
}

func TestFindFirstKeyNode_TooDeep(t *testing.T) {
	a, b := FindFirstKeyNode("", nil, 900)
	assert.Nil(t, a)
	assert.Nil(t, b)
}

func TestFindFirstKeyNode_ValueIsKey(t *testing.T) {
	a := &yaml.Node{
		Value: "chicken",
	}

	b := &yaml.Node{
		Value:   "nuggets",
		Content: []*yaml.Node{a},
	}

	c, d := FindFirstKeyNode("nuggets", []*yaml.Node{b}, 0)
	assert.NotNil(t, c)
	assert.NotNil(t, d)
	assert.Equal(t, c, d)
}

func TestFindFirstKeyNode_Map(t *testing.T) {
	nodes, _ := FindNodes(getPetstore(), "$")
	key, value := FindFirstKeyNode("pet", nodes, 0)
	assert.NotNil(t, key)
	assert.NotNil(t, value)
	assert.Equal(t, 27, key.Line)
}

func TestFindKeyNodeTop(t *testing.T) {
	nodes, _ := FindNodes(getPetstore(), "$")
	k, v := FindKeyNodeTop("info", nodes[0].Content)
	assert.NotNil(t, k)
	assert.NotNil(t, v)
	assert.Equal(t, 3, k.Line)
}

func TestFindKeyNodeTopSingleNode(t *testing.T) {
	a := &yaml.Node{
		Value: "chicken",
	}

	c, k := FindKeyNodeTop("chicken", []*yaml.Node{a})
	assert.Equal(t, "chicken", c.Value)
	assert.Equal(t, "chicken", k.Value)
}

func TestFindKeyNodeTop_NotFound(t *testing.T) {
	nodes, _ := FindNodes(getPetstore(), "$")
	k, v := FindKeyNodeTop("i am a giant potato", nodes[0].Content)
	assert.Nil(t, k)
	assert.Nil(t, v)
}

func TestFindKeyNode(t *testing.T) {
	nodes, _ := FindNodes(getPetstore(), "$")
	k, v := FindKeyNode("/pet", nodes[0].Content)
	assert.NotNil(t, k)
	assert.NotNil(t, v)
	assert.Equal(t, 47, k.Line)
}

func TestFindKeyNodeOffByOne(t *testing.T) {
	k, v := FindKeyNode("key", []*yaml.Node{
		{
			Value: "key",
			Line:  999,
		},
	})
	assert.NotNil(t, k)
	assert.NotNil(t, v)
	assert.Equal(t, 999, k.Line)
}

func TestFindKeyNode_ValueIsKey(t *testing.T) {
	a := &yaml.Node{
		Value: "chicken",
	}

	b := &yaml.Node{
		Tag:     "!!map",
		Value:   "nuggets",
		Content: []*yaml.Node{a},
	}

	c, d := FindKeyNode("nuggets", []*yaml.Node{b, a})
	assert.Equal(t, "nuggets", c.Value)
	assert.Equal(t, "chicken", d.Value)

	e := &yaml.Node{
		Value: "pizza",
	}
	f := &yaml.Node{
		Value: "pie",
	}
	b.Content = append(b.Content, e, f)

	c, d = FindKeyNode("pie", []*yaml.Node{b, a})
	assert.Equal(t, "nuggets", c.Value)
	assert.Equal(t, "pie", d.Value)

	b.Tag = "!!seq"

	c, d = FindKeyNode("pie", []*yaml.Node{b, a})
	assert.Equal(t, "nuggets", c.Value)
	assert.Equal(t, "pie", d.Value)
}

func TestFindExtensionNodes(t *testing.T) {
	a := &yaml.Node{
		Value: "x-coffee",
	}
	b := &yaml.Node{
		Value: "required",
	}
	c := &yaml.Node{
		Content: []*yaml.Node{a, b},
	}
	exts := FindExtensionNodes(c.Content)
	assert.Len(t, exts, 1)
	assert.Equal(t, "required", exts[0].Value.Value)
}

func TestFindKeyNodeFull(t *testing.T) {
	a := &yaml.Node{
		Value: "fish",
	}
	b := &yaml.Node{
		Value: "paste",
	}

	c, d, e := FindKeyNodeFull("fish", []*yaml.Node{a, b})
	assert.Equal(t, "fish", c.Value)
	assert.Equal(t, "fish", d.Value)
	assert.Equal(t, "paste", e.Value)
}

func TestFindKeyNodeFull_NoValue(t *testing.T) {
	a := &yaml.Node{
		Value: "openapi",
	}

	c, d, e := FindKeyNodeFull("openapi", []*yaml.Node{a})
	assert.Equal(t, "openapi", c.Value)
	assert.Equal(t, "openapi", d.Value)
	assert.Equal(t, "openapi", e.Value)
}

func TestFindKeyNodeFull_MapValueIsLastNode(t *testing.T) {
	f := &yaml.Node{
		Value: "cheese",
	}
	h := &yaml.Node{
		Tag:     "!!map",
		Value:   "deserts", // this is invalid btw, but helps with mechanical understanding
		Content: []*yaml.Node{f},
	}

	c, d, e := FindKeyNodeFull("cheese", []*yaml.Node{h})
	assert.Equal(t, "deserts", c.Value)
	assert.Equal(t, "cheese", d.Value)
	assert.Equal(t, "cheese", e.Value)
}

func TestFindKeyNodeFull_Map(t *testing.T) {
	f := &yaml.Node{
		Value: "cheese",
	}
	g := &yaml.Node{
		Value: "cake",
	}
	h := &yaml.Node{
		Tag:     "!!map",
		Value:   "deserts", // this is invalid btw, but helps with mechanical understanding
		Content: []*yaml.Node{f, g},
	}

	c, d, e := FindKeyNodeFull("cheese", []*yaml.Node{h})
	assert.Equal(t, "deserts", c.Value)
	assert.Equal(t, "cheese", d.Value)
	assert.Equal(t, "cake", e.Value)
}

func TestFindKeyNodeFull_Array(t *testing.T) {
	f := &yaml.Node{
		Value: "cheese",
	}
	g := &yaml.Node{
		Value: "cake",
	}
	h := &yaml.Node{
		Tag:     "!!seq",
		Value:   "deserts", // this is invalid btw, but helps with mechanical understanding
		Content: []*yaml.Node{f, g},
	}

	c, d, e := FindKeyNodeFull("cheese", []*yaml.Node{h})
	assert.Equal(t, "deserts", c.Value)
	assert.Equal(t, "cheese", d.Value)
	assert.Equal(t, "cheese", e.Value)
}

func TestFindKeyNodeFull_Nothing(t *testing.T) {
	c, d, e := FindKeyNodeFull("cheese", []*yaml.Node{})
	assert.Nil(t, c)
	assert.Nil(t, d)
	assert.Nil(t, e)
}

func TestFindKeyNode_NotFound(t *testing.T) {
	nodes, _ := FindNodes(getPetstore(), "$")
	k, v := FindKeyNode("I am not anything at all", nodes[0].Content)
	assert.Nil(t, k)
	assert.Nil(t, v)
}

func TestFindKeyFullNodeTop(t *testing.T) {
	a := &yaml.Node{
		Value: "fish",
	}
	b := &yaml.Node{
		Value: "paste",
	}

	c, d, e := FindKeyNodeFullTop("fish", []*yaml.Node{a, b})
	assert.Equal(t, "fish", c.Value)
	assert.Equal(t, "fish", d.Value)
	assert.Equal(t, "paste", e.Value)
}

func TestFindKeyFullNode_NotFound(t *testing.T) {
	a := &yaml.Node{
		Value: "fish",
	}
	b := &yaml.Node{
		Value: "paste",
	}

	c, d, e := FindKeyNodeFullTop("lemons", []*yaml.Node{a, b})
	assert.Nil(t, c)
	assert.Nil(t, d)
	assert.Nil(t, e)
}

func TestMakeTagReadable(t *testing.T) {
	n := &yaml.Node{
		Tag: "!!map",
	}
	assert.Equal(t, ObjectLabel, MakeTagReadable(n))
	n.Tag = "!!seq"
	assert.Equal(t, ArrayLabel, MakeTagReadable(n))
	n.Tag = "!!str"
	assert.Equal(t, StringLabel, MakeTagReadable(n))
	n.Tag = "!!int"
	assert.Equal(t, IntegerLabel, MakeTagReadable(n))
	n.Tag = "!!float"
	assert.Equal(t, NumberLabel, MakeTagReadable(n))
	n.Tag = "!!bool"
	assert.Equal(t, BooleanLabel, MakeTagReadable(n))
	n.Tag = "mr potato man is here"
	assert.Equal(t, "unknown", MakeTagReadable(n))
}

func TestIsNodeMap(t *testing.T) {
	n := &yaml.Node{
		Tag: "!!map",
	}
	assert.True(t, IsNodeMap(n))
	n.Tag = "!!pizza"
	assert.False(t, IsNodeMap(n))
}

func TestIsNodeMap_Nil(t *testing.T) {
	assert.False(t, IsNodeMap(nil))
}

func TestIsNodePolyMorphic(t *testing.T) {
	n := &yaml.Node{
		Content: []*yaml.Node{
			{
				Value: "anyOf",
			},
		},
	}
	assert.True(t, IsNodePolyMorphic(n))
	n.Content[0].Value = "cakes"
	assert.False(t, IsNodePolyMorphic(n))
}

func TestIsNodeArray(t *testing.T) {
	n := &yaml.Node{
		Tag: "!!seq",
	}
	assert.True(t, IsNodeArray(n))
	n.Tag = "!!pizza"
	assert.False(t, IsNodeArray(n))
}

func TestIsNodeArray_Nil(t *testing.T) {
	assert.False(t, IsNodeArray(nil))
}

func TestIsNodeStringValue(t *testing.T) {
	n := &yaml.Node{
		Tag: "!!str",
	}
	assert.True(t, IsNodeStringValue(n))
	n.Tag = "!!pizza"
	assert.False(t, IsNodeStringValue(n))
}

func TestIsNodeStringValue_Nil(t *testing.T) {
	assert.False(t, IsNodeStringValue(nil))
}

func TestIsNodeIntValue(t *testing.T) {
	n := &yaml.Node{
		Tag: "!!int",
	}
	assert.True(t, IsNodeIntValue(n))
	n.Tag = "!!pizza"
	assert.False(t, IsNodeIntValue(n))
}

func TestIsNodeIntValue_Nil(t *testing.T) {
	assert.False(t, IsNodeIntValue(nil))
}

func TestIsNodeFloatValue(t *testing.T) {
	n := &yaml.Node{
		Tag: "!!float",
	}
	assert.True(t, IsNodeFloatValue(n))
	n.Tag = "!!pizza"
	assert.False(t, IsNodeFloatValue(n))
}

func TestIsNodeNumberValue(t *testing.T) {
	n := &yaml.Node{
		Tag: "!!float",
	}
	assert.True(t, IsNodeNumberValue(n))
	n.Tag = "!!pizza"
	assert.False(t, IsNodeNumberValue(n))

	n = &yaml.Node{
		Tag: "!!int",
	}
	assert.True(t, IsNodeNumberValue(n))
	n.Tag = "!!pizza"
	assert.False(t, IsNodeNumberValue(n))
	assert.False(t, IsNodeNumberValue(nil))
}

func TestIsNodeFloatValue_Nil(t *testing.T) {
	assert.False(t, IsNodeFloatValue(nil))
}

func TestIsNodeBoolValue(t *testing.T) {
	n := &yaml.Node{
		Tag: "!!bool",
	}
	assert.True(t, IsNodeBoolValue(n))
	n.Tag = "!!pizza"
	assert.False(t, IsNodeBoolValue(n))
}

func TestIsNodeBoolValue_Nil(t *testing.T) {
	assert.False(t, IsNodeBoolValue(nil))
}

func TestFixContext(t *testing.T) {
	assert.Equal(t, "$.nuggets[12].name", FixContext("(root).nuggets.12.name"))
}

func TestFixContext_HttpCode(t *testing.T) {
	assert.Equal(t, "$.nuggets.404.name", FixContext("(root).nuggets.404.name"))
}

func TestIsJSON(t *testing.T) {
	assert.True(t, IsJSON("{'hello':'there'}"))
	assert.False(t, IsJSON("potato shoes"))
	assert.False(t, IsJSON(""))
}

func TestIsYAML(t *testing.T) {
	assert.True(t, IsYAML("hello:\n  there:\n    my-name: is quobix"))
	assert.True(t, IsYAML("potato shoes"))
	assert.False(t, IsYAML("{'hello':'there'}"))
	assert.False(t, IsYAML(""))
	assert.False(t, IsYAML("8908: hello: yeah: \n12309812: :123"))
}

func TestConvertYAMLtoJSON(t *testing.T) {
	str, err := ConvertYAMLtoJSON([]byte("hello: there"))
	assert.NoError(t, err)
	assert.NotNil(t, str)
	assert.Equal(t, "{\"hello\":\"there\"}", string(str))

	str, err = ConvertYAMLtoJSON([]byte("gonna: break: you:\nyeah:yeah:yeah"))
	assert.Error(t, err)
	assert.Nil(t, str)
}

func TestIsHttpVerb(t *testing.T) {
	assert.True(t, IsHttpVerb("get"))
	assert.True(t, IsHttpVerb("post"))
	assert.False(t, IsHttpVerb("nuggets"))
}

func TestConvertComponentIdIntoFriendlyPathSearch(t *testing.T) {
	segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/chicken/chips/pizza/cake")
	assert.Equal(t, "$.chicken.chips['pizza'].cake", path)
	assert.Equal(t, "cake", segment)
}

func TestConvertComponentIdIntoFriendlyPathSearch_SuperCrazy(t *testing.T) {
	segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/paths/~1crazy~1ass~1references/get/responses/404/content/application~1xml;%20charset=utf-8/schema")
	assert.Equal(t, "$.paths['/crazy/ass/references'].get.responses['404'].content['application/xml; charset=utf-8'].schema", path)
	assert.Equal(t, "schema", segment)
}

func TestConvertComponentIdIntoFriendlyPathSearch_Crazy(t *testing.T) {
	segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/components/schemas/gpg-key/properties/subkeys/examples/0/expires_at")
	assert.Equal(t, "$.components.schemas['gpg-key'].properties['subkeys'].examples[0].expires_at", path)
	assert.Equal(t, "expires_at", segment)
}

func BenchmarkConvertComponentIdIntoFriendlyPathSearch_Crazy(t *testing.B) {
	for n := 0; n < t.N; n++ {
		segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/components/schemas/gpg-key/properties/subkeys/examples/0/expires_at")
		assert.Equal(t, "$.components.schemas.gpg-key.properties['subkeys'].examples[0].expires_at", path)
		assert.Equal(t, "expires_at", segment)
	}
}

func BenchmarkConvertComponentIdIntoFriendlyPathSearch_Plural(t *testing.B) {
	for n := 0; n < t.N; n++ {
		segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/components/schemas/gpg-key/properties/subkeys/examples/0/expires_at")
		assert.Equal(t, "$.components.schemas['gpg-key'].properties['subkeys'].examples[0].expires_at", path)
		assert.Equal(t, "expires_at", segment)
	}
}

func TestConvertComponentIdIntoFriendlyPathSearch_Simple(t *testing.T) {
	segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/~1fresh~1pizza/get")
	assert.Equal(t, "$['/fresh/pizza'].get", path)
	assert.Equal(t, "get", segment)
}

func TestConvertComponentIdIntoFriendlyPathSearch_Plural(t *testing.T) {
	segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/components/schemas/FreshMan/properties/subkeys/examples/0/expires_at")
	assert.Equal(t, "$.components.schemas['FreshMan'].properties['subkeys'].examples[0].expires_at", path)
	assert.Equal(t, "expires_at", segment)
}

func TestConvertComponentIdIntoFriendlyPathSearch_Params(t *testing.T) {
	segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/why/0")
	assert.Equal(t, "$.why[0]", path)
	assert.Equal(t, "0", segment)
}

func TestConvertComponentIdIntoFriendlyPathSearch_Crazy_Github(t *testing.T) {
	segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/paths/~1crazy~1ass~1references/get/responses/404/content/application~1xml;%20charset=utf-8/schema")
	assert.Equal(t, "$.paths['/crazy/ass/references'].get.responses['404'].content['application/xml; charset=utf-8'].schema", path)
	assert.Equal(t, "schema", segment)
}

func TestConvertComponentIdIntoFriendlyPathSearch_Crazy_DigitalOcean(t *testing.T) {
	segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/paths/~1v2~1customers~1my~1invoices~1%7Binvoice_uuid%7D/get/parameters/0")
	assert.Equal(t, "$.paths['/v2/customers/my/invoices/{invoice_uuid}'].get.parameters[0]", path)
	assert.Equal(t, "0", segment)
}

func TestConvertComponentIdIntoFriendlyPathSearch_Crazy_DigitalOcean_More(t *testing.T) {
	segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/paths/~1v2~1certificates/post/responses/201/content/application~1json/examples/Custom%20Certificate")
	assert.Equal(t, "$.paths['/v2/certificates'].post.responses['201'].content['application/json'].examples['Custom Certificate']", path)
	assert.Equal(t, "Custom Certificate", segment)
}

func TestConvertComponentIdIntoFriendlyPathSearch_CrazyShort(t *testing.T) {
	segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/paths/~1crazy~1ass~1references")
	assert.Equal(t, "$.paths['/crazy/ass/references']", path)
	assert.Equal(t, "/crazy/ass/references", segment)
}

func TestConvertComponentIdIntoFriendlyPathSearch_Short(t *testing.T) {
	segment, path := ConvertComponentIdIntoFriendlyPathSearch("/~1crazy~1ass~1references")
	assert.Equal(t, "$['/crazy/ass/references']", path)
	assert.Equal(t, "/crazy/ass/references", segment)
}

func TestConvertComponentIdIntoFriendlyPathSearch_Array(t *testing.T) {
	segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/paths/~1crazy~1ass~1references/get/parameters/0")
	assert.Equal(t, "$.paths['/crazy/ass/references'].get.parameters[0]", path)
	assert.Equal(t, "0", segment)
}

func TestConvertComponentIdIntoFriendlyPathSearch_Slashes(t *testing.T) {
	_, path := ConvertComponentIdIntoFriendlyPathSearch(`#/nice/\rice/\and/\spice`)
	assert.Equal(t, "$.nice.rice.and.spice", path)
}

func TestConvertComponentIdIntoFriendlyPathSearch_HTTPCode(t *testing.T) {
	segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/paths/~1crazy~1ass~1references/get/responses/404")
	assert.Equal(t, "$.paths['/crazy/ass/references'].get.responses['404']", path)
	assert.Equal(t, "404", segment)
}

func TestConvertComponentIdIntoPath(t *testing.T) {
	segment, path := ConvertComponentIdIntoPath("$.chicken.chips.pizza.cake")
	assert.Equal(t, "#/chicken/chips/pizza/cake", path)
	assert.Equal(t, "cake", segment)
}

func TestConvertComponentIdIntoPath_NoHash(t *testing.T) {
	segment, path := ConvertComponentIdIntoPath("chicken.chips.pizza.cake")
	assert.Equal(t, "#/chicken/chips/pizza/cake", path)
	assert.Equal(t, "cake", segment)
}

func TestConvertComponentIdIntoPath_Alt1(t *testing.T) {
	segment, path := ConvertComponentIdIntoPath("$.chicken.chips['pizza'].cakes[0].burgers[2]")
	assert.Equal(t, "#/chicken/chips/pizza/cakes/0/burgers/2", path)
	assert.Equal(t, "burgers[2]", segment)
}

func TestConvertComponentIdIntoPath_Alt2(t *testing.T) {
	_, path := ConvertComponentIdIntoPath("chicken.chips['pizza'].cakes[0].burgers[2]")
	assert.Equal(t, "#/chicken/chips/pizza/cakes/0/burgers/2", path)
}

func TestConvertComponentIdIntoPath_Alt3(t *testing.T) {
	_, path := ConvertComponentIdIntoPath("chicken.chips['/one/two/pizza'].cakes[0].burgers[2]")
	assert.Equal(t, "#/chicken/chips/~1one~1two~1pizza/cakes/0/burgers/2", path)
}

func TestDetectCase(t *testing.T) {
	assert.Equal(t, PascalCase, DetectCase("PizzaPie"))
	assert.Equal(t, CamelCase, DetectCase("anyoneForTennis"))
	assert.Equal(t, ScreamingSnakeCase, DetectCase("I_LOVE_BEER"))
	assert.Equal(t, ScreamingKebabCase, DetectCase("I-LOVE-BURGERS"))
	assert.Equal(t, SnakeCase, DetectCase("snakes_on_a_plane"))
	assert.Equal(t, KebabCase, DetectCase("chicken-be-be-beef-or-pork"))
	assert.Equal(t, RegularCase, DetectCase("kebab-TimeIn_london-TOWN"))
	assert.Equal(t, UnknownCase, DetectCase(""))
}

func TestIsNodeRefValue(t *testing.T) {
	f := &yaml.Node{
		Value: "$ref",
	}
	g := &yaml.Node{
		Value: "'#/somewhere/out-there'",
	}
	h := &yaml.Node{
		Tag:     "!!map",
		Content: []*yaml.Node{f, g},
	}

	ref, node, val := IsNodeRefValue(h)

	assert.True(t, ref)
	assert.Equal(t, "$ref", node.Value)
	assert.Equal(t, "'#/somewhere/out-there'", val)
}

func TestIsNodeAlias(t *testing.T) {
	yml := `things:
  &anchorA
  - Stuff
  - Junk
thangs: *anchorA`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	ref, a := IsNodeAlias(node.Content[0].Content[3])

	assert.True(t, a)
	assert.Len(t, ref.Content, 2)
}

func TestNodeAlias(t *testing.T) {
	yml := `things:
  &anchorA
  - Stuff
  - Junk
thangs: *anchorA`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	ref := NodeAlias(node.Content[0].Content[3])

	assert.Len(t, ref.Content, 2)
}

func TestNodeAlias_Nil(t *testing.T) {
	ref := NodeAlias(nil)
	assert.Nil(t, ref)
}

func TestNodeAlias_IsNodeAlias_Nil(t *testing.T) {
	_, isAlias := IsNodeAlias(nil)
	assert.False(t, isAlias)
}

func TestNodeAlias_IsNodeAlias_False(t *testing.T) {
	yml := `things:
  - Stuff
  - Junk
thangs: none`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	_, isAlias := IsNodeAlias(node.Content[0].Content[3])
	assert.False(t, isAlias)
}

func TestCheckForMergeNodes(t *testing.T) {
	yml := `x-common-definitions:
  life_cycle_types: &life_cycle_types_def
    type: string
    enum: ["Onboarding", "Monitoring", "Re-Assessment"]
    description: The type of life cycle
<<: *life_cycle_types_def`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	mainNode := node.Content[0]

	CheckForMergeNodes(mainNode)

	_, _, enumVal := FindKeyNodeFullTop("enum", mainNode.Content)
	_, _, descriptionVal := FindKeyNodeFullTop("description", mainNode.Content)

	assert.Equal(t, "The type of life cycle", descriptionVal.Value)
	assert.Len(t, enumVal.Content, 3)
}

func TestCheckForMergeNodes_Empty_NoPanic(t *testing.T) {
	CheckForMergeNodes(nil)
}

func TestIsNodeRefValue_False(t *testing.T) {
	f := &yaml.Node{
		Value: "woof",
	}
	g := &yaml.Node{
		Value: "dog",
	}
	h := &yaml.Node{
		Tag:     "!!map",
		Content: []*yaml.Node{f, g},
	}

	ref, node, val := IsNodeRefValue(h)

	assert.False(t, ref)
	assert.Nil(t, node)
	assert.Empty(t, val)
}

func TestIsNodeRefValue_Nil(t *testing.T) {
	ref, node, val := IsNodeRefValue(nil)

	assert.False(t, ref)
	assert.Nil(t, node)
	assert.Empty(t, val)
}

func TestCheckEnumForDuplicates_Success(t *testing.T) {
	yml := "- yes\n- no\n- crisps"
	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)
	assert.Len(t, CheckEnumForDuplicates(rootNode.Content[0].Content), 0)
}

func TestCheckEnumForDuplicates_Fail(t *testing.T) {
	yml := "- yes\n- no\n- crisps\n- no"
	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)
	assert.Len(t, CheckEnumForDuplicates(rootNode.Content[0].Content), 1)
}

func TestCheckEnumForDuplicates_FailMultiple(t *testing.T) {
	yml := "- yes\n- no\n- crisps\n- no\n- rice\n- yes\n- no"

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)
	assert.Len(t, CheckEnumForDuplicates(rootNode.Content[0].Content), 3)
}

func TestConvertComponentIdIntoFriendlyPathSearch_Brackets(t *testing.T) {
	segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/components/schemas/OhNoWhy[HaveYouDoneThis]")
	assert.Equal(t, "$.components.schemas['OhNoWhy[HaveYouDoneThis]']", path)
	assert.Equal(t, "OhNoWhy[HaveYouDoneThis]", segment)
}

func TestDetermineYAMLWhitespaceLength(t *testing.T) {
	someBytes, _ := os.ReadFile("../test_specs/burgershop.openapi.yaml")
	assert.Equal(t, 2, DetermineWhitespaceLength(string(someBytes)))
}

func TestDetermineJSONWhitespaceLength(t *testing.T) {
	someBytes, _ := os.ReadFile("../test_specs/petstorev3.json")
	assert.Equal(t, 2, DetermineWhitespaceLength(string(someBytes)))
}

func TestDetermineJSONWhitespaceLength_None(t *testing.T) {
	someBytes := []byte(`{"hello": "world"}`)
	assert.Equal(t, 0, DetermineWhitespaceLength(string(someBytes)))
}

func TestFindFirstKeyNode_MergeTest(t *testing.T) {
	yml := []byte(`openapi: 3.0.3

x-a: &anchor
  important-field: true

x-b:
  <<: *anchor
`)

	var rootNode yaml.Node
	_ = yaml.Unmarshal(yml, &rootNode)

	k, v := FindFirstKeyNode("important-field", rootNode.Content[0].Content[5].Content, 0)
	assert.NotNil(t, k)
	assert.NotNil(t, v)
	assert.Equal(t, "true", v.Value)

}

func TestFindKeyNodeFull_MergeTest(t *testing.T) {
	yml := []byte(`openapi: 3.0.3

x-a: &anchor
  important-field: true

x-b:
  <<: *anchor
`)

	var rootNode yaml.Node
	_ = yaml.Unmarshal(yml, &rootNode)

	k, l, v := FindKeyNodeFull("servers", rootNode.Content[0].Content)
	assert.Nil(t, k)
	assert.Nil(t, v)
	assert.Nil(t, l)

}

func TestFindFirstKeyNode_DoubleMerge(t *testing.T) {
	yml := []byte(`openapi: 3.0.3

t-k: &anchorB
  important-field: a nice string
x-a: &anchorA
  <<: *anchorB
x-b:
  <<: *anchorA
`)

	var rootNode yaml.Node
	_ = yaml.Unmarshal(yml, &rootNode)

	k, v := FindFirstKeyNode("important-field", rootNode.Content[0].Content[7].Content, 0)
	assert.NotNil(t, k)
	assert.NotNil(t, v)
	assert.Equal(t, "a nice string", v.Value)

}

func TestFindKeyNodeTop_DoubleMerge(t *testing.T) {
	yml := []byte(`openapi: 3.0.3

t-k: &anchorB
  important-field: a nice string
x-a: &anchorA
  <<: *anchorB
x-b:
  <<: *anchorA
`)

	var rootNode yaml.Node
	_ = yaml.Unmarshal(yml, &rootNode)

	k, v := FindKeyNodeTop("important-field", rootNode.Content[0].Content[7].Content)
	assert.NotNil(t, k)
	assert.NotNil(t, v)
	assert.Equal(t, "a nice string", v.Value)

}

func TestFindKeyNode_DoubleMerge(t *testing.T) {
	yml := []byte(`openapi: 3.0.3

t-k: &anchorB
  important-field: a nice string
x-a: &anchorA
  <<: *anchorB
x-b:
  <<: *anchorA
`)

	var rootNode yaml.Node
	_ = yaml.Unmarshal(yml, &rootNode)

	k, v := FindKeyNode("important-field", rootNode.Content[0].Content[7].Content)
	assert.NotNil(t, k)
	assert.NotNil(t, v)
	assert.Equal(t, "a nice string", v.Value)

}

func TestFindKeyNodeFull_DoubleMerge(t *testing.T) {
	yml := []byte(`openapi: 3.0.3
any-thing: &anchorH
  important-field: a nice string
t-k: &anchorB
  panda:
    <<: *anchorH
x-a: &anchorA
  <<: *anchorB
x-b:
  <<: *anchorA

`)

	var rootNode yaml.Node
	ee := yaml.Unmarshal(yml, &rootNode)
	assert.NoError(t, ee)

	k, l, v := FindKeyNodeFull("important-field", rootNode.Content[0].Content[9].Content)
	assert.NotNil(t, l)
	assert.NotNil(t, k)
	assert.NotNil(t, v)
	assert.Equal(t, "a nice string", v.Value)

}

func TestFindKeyNodeFullTop_DoubleMerge(t *testing.T) {
	yml := []byte(`openapi: 3.0.3
any-thing: &anchorH
  important-field: a nice string
t-k: &anchorB
  <<: *anchorH
x-a: &anchorA
  <<: *anchorB
x-b:
  <<: *anchorA

`)

	var rootNode yaml.Node
	ee := yaml.Unmarshal(yml, &rootNode)
	assert.NoError(t, ee)

	k, l, v := FindKeyNodeFullTop("important-field", rootNode.Content[0].Content[9].Content)
	assert.NotNil(t, l)
	assert.NotNil(t, k)
	assert.NotNil(t, v)
	assert.Equal(t, "a nice string", v.Value)

}

func TestNodeMerge(t *testing.T) {
	yml := []byte(`openapi: 3.0.3
any-thing: &anchorH
  important-field: a nice string
t-k: &anchorB
  <<: *anchorH
x-a: &anchorA
  <<: *anchorB
x-b:
  <<: *anchorA

`)

	var rootNode yaml.Node
	ee := yaml.Unmarshal(yml, &rootNode)
	assert.NoError(t, ee)

	n := NodeMerge(rootNode.Content[0].Content[9].Content)
	assert.NotNil(t, n)
	assert.Equal(t, "a nice string", n.Content[1].Value)
}

func TestNodeMerge_NoNodes(t *testing.T) {
	n := NodeMerge(nil)
	assert.Nil(t, n)
}
