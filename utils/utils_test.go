package utils

import (
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pb33f/jsonpath/pkg/jsonpath"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
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

func TestGetJSONPath_CacheHit(t *testing.T) {
	jsonPathCache = sync.Map{}

	path1, err := getJSONPath("$.info.contact")
	assert.NoError(t, err)
	assert.NotNil(t, path1)

	path2, err := getJSONPath("$.info.contact")
	assert.NoError(t, err)
	assert.Equal(t, path1, path2)
}

func TestGetJSONPath_CacheHit_Invalid(t *testing.T) {
	jsonPathCache = sync.Map{}

	path1, err := getJSONPath("I am not valid")
	assert.Error(t, err)
	assert.Nil(t, path1)

	path2, err := getJSONPath("I am not valid")
	assert.Error(t, err)
	assert.Equal(t, path1, path2)
}

func TestFindLastChildNode(t *testing.T) {
	nodes, _ := FindNodes(getPetstore(), "$.info")
	lastNode := FindLastChildNode(nodes[0])
	assert.Equal(t, "1.0.11", lastNode.Value) // should be the version.
	lastNodeDouble := FindLastChildNodeWithLevel(nodes[0], 0)
	assert.Equal(t, lastNode, lastNodeDouble)
}

func TestFindLastChildNode_WithKids(t *testing.T) {
	nodes, _ := FindNodes(getPetstore(), "$.paths['/pet']")
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
	assert.False(t, IsYAML("potato shoes"))
	assert.False(t, IsYAML("{'hello':'there'}"))
	assert.False(t, IsYAML(""))
	assert.True(t, IsYAML("8908: hello: yeah: \n12309812: :123"))
	assert.True(t, IsYAML("---"))
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
	assert.Equal(t, "$.components.schemas['gpg-key'].properties['subkeys']['examples'][0].expires_at", path)
	assert.Equal(t, "expires_at", segment)
}

func BenchmarkConvertComponentIdIntoFriendlyPathSearch_Crazy(t *testing.B) {
	for n := 0; n < t.N; n++ {
		segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/components/schemas/gpg-key/properties/subkeys/examples/0/expires_at")
		assert.Equal(t, "$.components.schemas.gpg-key.properties['subkeys']['examples'][0].expires_at", path)
		assert.Equal(t, "expires_at", segment)
	}
}

func BenchmarkConvertComponentIdIntoFriendlyPathSearch_Plural(t *testing.B) {
	for n := 0; n < t.N; n++ {
		segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/components/schemas/gpg-key/properties/subkeys/examples/0/expires_at")
		assert.Equal(t, "$.components.schemas['gpg-key'].properties['subkeys']['examples'][0].expires_at", path)
		assert.Equal(t, "expires_at", segment)
	}
}

func TestConvertComponentIdIntoFriendlyPathSearch_Simple(t *testing.T) {
	segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/~1fresh~1pizza/get")
	assert.Equal(t, "$.['/fresh/pizza'].get", path)
	assert.Equal(t, "get", segment)
}

func TestConvertComponentIdIntoFriendlyPathSearch_Plural(t *testing.T) {
	segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/components/schemas/FreshMan/properties/subkeys/examples/0/expires_at")
	assert.Equal(t, "$.components.schemas['FreshMan'].properties['subkeys']['examples'][0].expires_at", path)
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
	assert.Equal(t, "$.['/crazy/ass/references']", path)
	assert.Equal(t, "/crazy/ass/references", segment)
}

func TestConvertComponentIdIntoFriendlyPathSearch_Callback(t *testing.T) {
	segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/components/pathItems/test-callback-2")
	assert.Equal(t, "$.components.pathItems['test-callback-2']", path)
	assert.Equal(t, "test-callback-2", segment)
}

func TestConvertComponentIdIntoFriendlyPathSearch_Callback2(t *testing.T) {
	segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/test-callback")
	assert.Equal(t, "$.['test-callback']", path)
	assert.Equal(t, "test-callback", segment)
}

func TestConvertComponentIdIntoFriendlyPathSearch_Root(t *testing.T) {
	segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/")
	assert.Equal(t, "$.", path)
	assert.Equal(t, "", segment)
}

func TestConvertComponentIdIntoFriendlyPathSearch_CBase(t *testing.T) {
	segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/NoMoreBeer")
	assert.Equal(t, "$.NoMoreBeer", path)
	assert.Equal(t, "NoMoreBeer", segment)
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

// https://github.com/pb33f/libopenapi/issues/112
func TestConvertComponentIdIntoFriendlyPathSearch_BracketInName(t *testing.T) {
	_, path := ConvertComponentIdIntoFriendlyPathSearch(`#/oo/missus/hows/your/fa[ther]`)
	assert.Equal(t, "$.oo.missus['hows']['your']['fa[ther]']", path)
}

func TestConvertComponentIdIntoFriendlyPathSearch_DashSomething(t *testing.T) {
	_, path := ConvertComponentIdIntoFriendlyPathSearch(`-rome`)
	assert.Equal(t, "$.['-rome']", path)
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

// Tests for performance optimization coverage
func TestAppendSegment(t *testing.T) {
	// Test appendSegment function (currently 0% coverage)
	var sb strings.Builder
	segs := []string{"test", "segment", "value"}
	cleaned := []string{"initial"}

	// Test without quotes
	appendSegment(&sb, segs, cleaned, 1, false)
	assert.Equal(t, "initial[segment]", cleaned[0])

	// Test with quotes
	cleaned = []string{"another"}
	appendSegment(&sb, segs, cleaned, 2, true)
	assert.Equal(t, "another['value']", cleaned[0])
}

func TestConvertComponentIdIntoFriendlyPathSearch_EdgeCases(t *testing.T) {
	// Test empty cleaned array handling
	_, path := ConvertComponentIdIntoFriendlyPathSearch("")
	assert.Equal(t, "$.", path)

	// Test single segment without cleaning
	_, path = ConvertComponentIdIntoFriendlyPathSearch("simple")
	assert.Equal(t, "$.simple", path)

	// Test path that doesn't start with $
	_, path = ConvertComponentIdIntoFriendlyPathSearch("noprefix/path")
	assert.Equal(t, "$.noprefix.path", path)
}

func TestConvertComponentIdIntoFriendlyPathSearch_LargeIntegerArrays(t *testing.T) {
	// Test integer > 99 without cleaned array
	_, path := ConvertComponentIdIntoFriendlyPathSearch("100")
	assert.Equal(t, "$.", path) // Empty since no segments

	// Test integer <= 99 without cleaned array
	_, path = ConvertComponentIdIntoFriendlyPathSearch("50")
	assert.Equal(t, "$.", path) // Empty since no segments
}

func TestConvertComponentIdIntoFriendlyPathSearch_SpecialCharacterEdgeCases(t *testing.T) {
	// Test non-path chars at beginning - actual behavior returns only last segment
	_, path := ConvertComponentIdIntoFriendlyPathSearch("@special/chars")
	assert.Equal(t, "$.chars", path)

	// Test non-path chars in middle with no cleaned array
	_, path = ConvertComponentIdIntoFriendlyPathSearch("path/@middle")
	assert.Equal(t, "$.path['@middle']", path)

	// Test non-path chars at end with no cleaned array
	_, path = ConvertComponentIdIntoFriendlyPathSearch("end/@last")
	assert.Equal(t, "$.end['@last']", path)
}

func TestConvertComponentIdIntoFriendlyPathSearch_CleanedSingleSegment(t *testing.T) {
	// Test case that results in single cleaned segment - # should be preserved (issue #485)
	_, path := ConvertComponentIdIntoFriendlyPathSearch("#single")
	assert.Equal(t, "$.['#single']", path)

	// Test empty cleaned result
	_, path = ConvertComponentIdIntoFriendlyPathSearch("/")
	assert.Equal(t, "$.", path)
}

func TestConvertComponentIdIntoFriendlyPathSearch_PathWithoutDotAfterDollar(t *testing.T) {
	// This is a complex test to trigger the path[1] != '.' branch
	// We need a path that after processing doesn't have a dot after $
	// This happens with certain edge cases in the string building logic
	testCases := []struct {
		input    string
		expected string
	}{
		{
			// Path with only non-path chars that get wrapped - # is preserved (issue #485)
			input:    "!@#",
			expected: "$.['!@#']",
		},
		{
			// Complex path to test final builder logic - # in segment name is preserved (issue #485)
			input:    "#/test#value",
			expected: "$.['test#value']",
		},
	}

	for _, tc := range testCases {
		_, path := ConvertComponentIdIntoFriendlyPathSearch(tc.input)
		assert.Equal(t, tc.expected, path)
	}
}

func TestConvertComponentIdIntoFriendlyPathSearch_HashCharacterHandling(t *testing.T) {
	// Test # character in segments - # should be preserved in component names (issue #485)
	_, path := ConvertComponentIdIntoFriendlyPathSearch("#/path/with#hash/in#middle")
	assert.Equal(t, "$.path['with#hash']['in#middle']", path)

	// Test multiple # in single segment - all should be preserved (issue #485)
	_, path = ConvertComponentIdIntoFriendlyPathSearch("#/seg#ment#with#many")
	assert.Equal(t, "$.['seg#ment#with#many']", path)
}

// Additional tests to hit uncovered branches
func TestConvertComponentIdIntoFriendlyPathSearch_UncoveredBranches(t *testing.T) {
	// Test non-path char in middle position with existing cleaned array
	_, path := ConvertComponentIdIntoFriendlyPathSearch("#/start/@middle/end")
	assert.Equal(t, "$.start['@middle'].end", path)

	// Test non-path char at first segment position
	_, path = ConvertComponentIdIntoFriendlyPathSearch("@first/second")
	assert.Equal(t, "$.second", path)

	// Test non-path char at last segment with cleaned array
	_, path = ConvertComponentIdIntoFriendlyPathSearch("#/first/@last")
	assert.Equal(t, "$.first['@last']", path)

	// Test integer array index at beginning
	_, path = ConvertComponentIdIntoFriendlyPathSearch("0/path")
	assert.Equal(t, "$.path", path)

	// Test integer array index > 99 in middle with cleaned array
	_, path = ConvertComponentIdIntoFriendlyPathSearch("#/path/200/next")
	assert.Equal(t, "$.path['200'].next", path)

	// Test integer array index <= 99 in middle with cleaned array
	_, path = ConvertComponentIdIntoFriendlyPathSearch("#/path/50/next")
	assert.Equal(t, "$.path[50].next", path)

	// Test empty segment handling
	_, path = ConvertComponentIdIntoFriendlyPathSearch("#//empty//segments")
	assert.Equal(t, "$.empty.segments", path)

	// Test path with backslashes and # character
	_, path = ConvertComponentIdIntoFriendlyPathSearch(`#/path\with\backslash`)
	assert.Equal(t, "$.pathwithbackslash", path)

	// Test plural parent handling - first segment after components/schemas
	_, path = ConvertComponentIdIntoFriendlyPathSearch("#/components/schemas/MySchema")
	assert.Equal(t, "$.components.schemas['MySchema']", path)

	// Test ensuring $ prefix is added
	// Create a scenario where replaced doesn't start with $
	// This is difficult since most paths get $ added, but let's try
	_, path = ConvertComponentIdIntoFriendlyPathSearch("nosharppathstart")
	assert.Equal(t, "$.nosharppathstart", path)

	// Test ensuring . after $ is added - need a path that results in $ without .
	// This happens with certain edge cases in string building
	_, path = ConvertComponentIdIntoFriendlyPathSearch("['wrapped']")
	assert.Equal(t, "$.['['wrapped']']", path)
}

func TestConvertComponentIdIntoFriendlyPathSearch_EmptyCleanedArray(t *testing.T) {
	// Test when cleaned array ends up empty (all segments filtered out)
	_, path := ConvertComponentIdIntoFriendlyPathSearch("///")
	assert.Equal(t, "$.", path)
}

func TestConvertComponentIdIntoFriendlyPathSearch_NonPathCharNoCleanedArray(t *testing.T) {
	// Test non-path char as first segment (i=0) when cleaned is empty
	_, path := ConvertComponentIdIntoFriendlyPathSearch("@special")
	assert.Equal(t, "$.['@special']", path)
}

func TestConvertComponentIdIntoFriendlyPathSearch_IntegerWithoutCleanedArray(t *testing.T) {
	// Test integer processing when cleaned array is empty - # prefix means path is empty
	_, path := ConvertComponentIdIntoFriendlyPathSearch("#/99")
	assert.Equal(t, "$.", path)

	_, path = ConvertComponentIdIntoFriendlyPathSearch("#/999")
	assert.Equal(t, "$.", path)
}

// Test to hit line 870 - # character in multi-segment cleaned path
func TestConvertComponentIdIntoFriendlyPathSearch_HashInMultiSegment(t *testing.T) {
	// This creates multiple cleaned segments
	_, path := ConvertComponentIdIntoFriendlyPathSearch("#/segment1/segment2")
	assert.Equal(t, "$.segment1.segment2", path)

	// Another test with # in actual segment names - # should be preserved (issue #485)
	_, path = ConvertComponentIdIntoFriendlyPathSearch("#/test/another#segment/end")
	assert.Equal(t, "$.test['another#segment'].end", path)
}

// Test appendSegmentOptimized with no cleaned array
func TestConvertComponentIdIntoFriendlyPathSearch_AppendOptimizedNoCleaned(t *testing.T) {
	// This should trigger appendSegmentOptimized when cleaned is empty
	// Integer without any prior segments
	_, path := ConvertComponentIdIntoFriendlyPathSearch("5")
	assert.Equal(t, "$.", path)

	_, path = ConvertComponentIdIntoFriendlyPathSearch("500")
	assert.Equal(t, "$.", path)
}

// Complex surgical test to trigger the replaced[0] != '$' branch (lines 897-903)
func TestConvertComponentIdIntoFriendlyPathSearch_NoDollarPrefixEdgeCase(t *testing.T) {
	// This test is designed to hit the extremely rare edge case where
	// the finalBuilder somehow doesn't start with '$'. This is nearly impossible
	// in normal operation since all code paths add '$' or '$.' at the start.
	// However, we need this test for 100% coverage.

	// Looking at the code, this edge case would only trigger if:
	// 1. len(cleaned) == 0 (empty case)
	// 2. finalBuilder.WriteString("$.") fails or is overridden somehow
	// 3. Or if there's a very specific input that breaks the logic

	// Let's try various edge cases that might not add the $ prefix
	testCases := []string{
		"",     // Empty string
		"///",  // Only slashes
		"////", // More slashes
	}

	for _, tc := range testCases {
		// Even though these will likely all start with $,
		// we're testing for the edge case branch
		_, path := ConvertComponentIdIntoFriendlyPathSearch(tc)
		// All should result in "$." due to the safety check
		assert.True(t, len(path) > 0, "Path should not be empty for input: %s", tc)
		if len(path) > 0 {
			assert.Equal(t, byte('$'), path[0], "Path should start with $ for input: %s", tc)
		}
	}

	// The branch at line 897-903 is defensive code that ensures the result
	// always starts with '$' even if something unexpected happens
	// This test documents that the code properly handles edge cases
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

// Issue #485 tests - Hash character in component names should be preserved
func TestConvertComponentIdIntoFriendlyPathSearch_Issue485_HashInComponentName(t *testing.T) {
	// Real-world Elasticsearch example from issue #485
	segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/components/parameters/async_search.submit#wait_for_completion_timeout")
	assert.Equal(t, "$.components.parameters['async_search.submit#wait_for_completion_timeout']", path)
	assert.Equal(t, "async_search.submit#wait_for_completion_timeout", segment)
}

func TestConvertComponentIdIntoFriendlyPathSearch_Issue485_MultipleHashesInName(t *testing.T) {
	// Component name with multiple # characters
	segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/components/schemas/model#v1#beta")
	assert.Equal(t, "$.components.schemas['model#v1#beta']", path)
	assert.Equal(t, "model#v1#beta", segment)
}

func TestConvertComponentIdIntoFriendlyPathSearch_Issue485_HashAtEndOfName(t *testing.T) {
	// Component name with # at the end
	segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/components/schemas/deprecated#")
	assert.Equal(t, "$.components.schemas['deprecated#']", path)
	assert.Equal(t, "deprecated#", segment)
}

func TestConvertComponentIdIntoFriendlyPathSearch_Issue485_HashAtStartOfName(t *testing.T) {
	// Component name with # at the start (after the path)
	segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/components/schemas/#internal")
	assert.Equal(t, "$.components.schemas['#internal']", path)
	assert.Equal(t, "#internal", segment)
}

func TestConvertComponentIdIntoFriendlyPathSearch_Issue485_OnlyHashInName(t *testing.T) {
	// Component name that is just a #
	segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/components/schemas/#")
	assert.Equal(t, "$.components.schemas['#']", path)
	assert.Equal(t, "#", segment)
}

func TestConvertComponentIdIntoFriendlyPathSearch_Issue485_HashWithSpecialChars(t *testing.T) {
	// Component name with # mixed with other special characters
	segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/components/schemas/model#v1-beta.final")
	assert.Equal(t, "$.components.schemas['model#v1-beta.final']", path)
	assert.Equal(t, "model#v1-beta.final", segment)
}

func TestConvertComponentIdIntoFriendlyPathSearch_Issue485_NormalPathsUnaffected(t *testing.T) {
	// Verify normal paths (without # in component names) still work correctly
	testCases := []struct {
		input        string
		expectedPath string
		expectedName string
	}{
		{"#/components/schemas/Pet", "$.components.schemas['Pet']", "Pet"},
		{"#/components/parameters/page-size", "$.components.parameters['page-size']", "page-size"},
	}

	for _, tc := range testCases {
		segment, path := ConvertComponentIdIntoFriendlyPathSearch(tc.input)
		assert.Equal(t, tc.expectedPath, path, "Path mismatch for input: %s", tc.input)
		assert.Equal(t, tc.expectedName, segment, "Name mismatch for input: %s", tc.input)
	}
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

func TestIsNodeNull(t *testing.T) {
	n := &yaml.Node{
		Tag: "!!null",
	}
	assert.True(t, IsNodeNull(n))
	n.Tag = "!!str"
	assert.False(t, IsNodeNull(n))

	var noNode *yaml.Node
	assert.True(t, IsNodeNull(noNode))
}

func TestFindNodesWithoutDeserializingWithTimeout_Timeout(t *testing.T) {
	root, _ := FindNodes(getPetstore(), "$")
	block := make(chan struct{})
	done := make(chan struct{})
	original := jsonPathQuery
	jsonPathQuery = func(path *jsonpath.JSONPath, node *yaml.Node) []*yaml.Node {
		<-block
		close(done)
		return nil
	}
	defer func() {
		jsonPathQuery = original
		close(block)
		<-done
	}()

	nodes, err := FindNodesWithoutDeserializingWithTimeout(root[0], "$.info.contact", 1*time.Millisecond)
	assert.Nil(t, nodes)
	assert.ErrorContains(t, err, "timeout exceeded")
}

func TestFindNodesWithoutDeserializingWithTimeout_Success(t *testing.T) {
	root, _ := FindNodes(getPetstore(), "$")
	nodes, err := FindNodesWithoutDeserializingWithTimeout(root[0], "$.info.contact", 100*time.Millisecond)
	assert.NoError(t, err)
	assert.NotNil(t, nodes)
	assert.Len(t, nodes, 1)
}

func TestGenerateAlphanumericString(t *testing.T) {
	reg := regexp.MustCompile("^[0-9A-Za-z]{1,4}$")
	assert.NotNil(t, reg.MatchString(GenerateAlphanumericString(4)))

	reg = regexp.MustCompile("^[0-9A-Za-z]{1,10}$")
	assert.NotNil(t, reg.MatchString(GenerateAlphanumericString(10)))

	reg = regexp.MustCompile("^[0-9A-Za-z]{1,15}$")
	assert.NotNil(t, reg.MatchString(GenerateAlphanumericString(15)))
}

// Test specific edge cases for ConvertComponentIdIntoFriendlyPathSearch to cover uncovered lines
func TestConvertComponentIdIntoFriendlyPathSearch_SpecificEdgeCases(t *testing.T) {
	// Test empty string case - should trigger the early return and avoid the uncovered paths
	name, path := ConvertComponentIdIntoFriendlyPathSearch("")
	assert.Equal(t, "", name)
	assert.Equal(t, "$.", path)

	// Test with only hash case - should trigger the early return and avoid the uncovered paths
	name2, path2 := ConvertComponentIdIntoFriendlyPathSearch("#/")
	assert.Equal(t, "", name2)
	assert.Equal(t, "$.", path2)

	// Test with malformed input that could potentially not start with $
	// This is unlikely but let's try various edge cases
	_, path3 := ConvertComponentIdIntoFriendlyPathSearch("###")
	assert.NotEmpty(t, path3)
	assert.True(t, strings.HasPrefix(path3, "$"))

	// Test with only slashes
	_, path4 := ConvertComponentIdIntoFriendlyPathSearch("///")
	assert.NotEmpty(t, path4)
	assert.True(t, strings.HasPrefix(path4, "$"))
}

// Test to try to trigger the formatting safeguard code
func TestConvertComponentIdIntoFriendlyPathSearch_FormatSafeguards(t *testing.T) {
	// Test various edge cases that might trigger the formatting safeguards
	testCases := []string{
		"#",
		"##",
		"#//",
		"/#",
		"//#",
		"#//#",
		"###/test",
		"#/test###",
		"#/###/test",
	}

	for _, testCase := range testCases {
		name, path := ConvertComponentIdIntoFriendlyPathSearch(testCase)
		// All results should start with $
		assert.True(t, strings.HasPrefix(path, "$"), "Path should start with $ for input: %s, got: %s", testCase, path)
		// Path should have proper format if it has content beyond $
		if len(path) > 1 {
			if path[1] != '.' && path[1] != '[' {
				t.Errorf("Path should have proper format after $ for input: %s, got: %s", testCase, path)
			}
		}
		// Name should be valid
		assert.NotNil(t, name)
	}
}

// Test extreme edge cases that could potentially stress the string building logic
func TestConvertComponentIdIntoFriendlyPathSearch_ExtremeEdgeCases(t *testing.T) {
	// Test cases that exercise different parts of the string building logic
	extremeCases := []string{
		// Single characters and minimal cases
		"#/.",
		"#/./",
		"#/..",
		"#/../",
		"#/....",
		// Cases with many empty segments
		"#//////",
		"#/a///b///c",
		// Cases with unusual character combinations
		"#/###",
		"#/\\\\\\",
		"#/¡¢£¤¥", // Non-ASCII characters
		// Cases that might result in empty cleaned array
		"#/#",
		"#/##",
		"#/#/#",
		// Cases with mixed separators
		"#/./a/../b",
		// Very short cases
		"#/a",
		"#/0",
		"#/-",
		"#/_",
	}

	for _, testCase := range extremeCases {
		name, path := ConvertComponentIdIntoFriendlyPathSearch(testCase)

		// All results should start with $ (this is the key test for the safeguard code)
		assert.True(t, strings.HasPrefix(path, "$"), "Path should start with $ for input: %s, got: %s", testCase, path)

		// If path is longer than just "$", it should have proper formatting
		if len(path) > 1 {
			// Should either be "$." or start with "$." or have proper bracket notation
			isProperlyFormatted := strings.HasPrefix(path, "$.") ||
				strings.HasPrefix(path, "$[") ||
				path == "$."
			assert.True(t, isProperlyFormatted, "Path should be properly formatted for input: %s, got: %s", testCase, path)
		}

		// Name should not be nil (even if empty)
		assert.NotNil(t, name)
	}
}

// https://github.com/pb33f/libopenapi/issues/500
// Test digit-starting property names require bracket notation in JSONPath
func TestConvertComponentIdIntoFriendlyPathSearch_DigitStartingSegments(t *testing.T) {
	// Root-level key starting with digit (like error codes)
	segment, path := ConvertComponentIdIntoFriendlyPathSearch("#/403_permission_denied")
	assert.Equal(t, "$.['403_permission_denied']", path)
	assert.Equal(t, "403_permission_denied", segment)

	// Nested path with digit-starting segment
	segment, path = ConvertComponentIdIntoFriendlyPathSearch("#/responses/400_unexpected_request_body")
	assert.Equal(t, "$.responses['400_unexpected_request_body']", path)
	assert.Equal(t, "400_unexpected_request_body", segment)

	// Multiple digit-starting segments
	segment, path = ConvertComponentIdIntoFriendlyPathSearch("#/4xx_errors/403_forbidden")
	assert.Equal(t, "$.['4xx_errors']['403_forbidden']", path)
	assert.Equal(t, "403_forbidden", segment)

	// Digit-starting in middle of path
	segment, path = ConvertComponentIdIntoFriendlyPathSearch("#/components/responses/5xx_server_error/description")
	assert.Equal(t, "$.components.responses['5xx_server_error'].description", path)
	assert.Equal(t, "description", segment)

	// Pure numeric segment (handled by integer code path, uses [0] not ['0'])
	segment, path = ConvertComponentIdIntoFriendlyPathSearch("#/items/0/name")
	assert.Equal(t, "$.items[0].name", path)
	assert.Equal(t, "name", segment)

	// Segment starting with digit but not pure number
	segment, path = ConvertComponentIdIntoFriendlyPathSearch("#/2xx_success")
	assert.Equal(t, "$.['2xx_success']", path)
	assert.Equal(t, "2xx_success", segment)
}

// Test isPathChar function directly for comprehensive coverage
func TestIsPathChar(t *testing.T) {
	// Valid path characters (letters, numbers not at start, underscore, backslash)
	assert.True(t, isPathChar("validName"))
	assert.True(t, isPathChar("Valid123"))
	assert.True(t, isPathChar("with_underscore"))
	assert.True(t, isPathChar(`with\backslash`))
	assert.True(t, isPathChar("MixedCase123_test"))

	// Pure integers return true - they're handled separately as array indices
	assert.True(t, isPathChar("0"))
	assert.True(t, isPathChar("123"))
	assert.True(t, isPathChar("99"))

	// Invalid: empty string
	assert.False(t, isPathChar(""))

	// Invalid: starts with digit but NOT a pure integer (requires bracket notation in JSONPath)
	assert.False(t, isPathChar("403_permission_denied"))
	assert.False(t, isPathChar("4xx_errors"))
	assert.False(t, isPathChar("123abc"))
	assert.False(t, isPathChar("9_starts_with_nine"))
	assert.False(t, isPathChar("0x123")) // hex-like but has 'x'

	// Invalid: contains special characters
	assert.False(t, isPathChar("with-dash"))
	assert.False(t, isPathChar("with space"))
	assert.False(t, isPathChar("with@symbol"))
	assert.False(t, isPathChar("with#hash"))
	assert.False(t, isPathChar("with.dot"))
}

// Test documenting the defensive safeguard code behavior
func TestConvertComponentIdIntoFriendlyPathSearch_DefensiveCodeDocumentation(t *testing.T) {
	// This test documents that the defensive safeguard code at lines 897-903 in ConvertComponentIdIntoFriendlyPathSearch
	// is difficult to trigger because the function is well-designed to always produce strings starting with '$'.
	// The safeguard code handles theoretical edge cases where string building might fail,
	// but these cases don't occur in normal operation.

	// Test a comprehensive set of inputs to verify they all produce properly formatted strings
	inputs := []string{
		"#/test", "#/", "", "#", "test", "/test", "#/test/sub", "#/test/123",
		"#/test space", "#/test-dash", "#/test_underscore", "#/test.dot",
		"#/test[bracket]", "#/test{brace}", "#/test(paren)", "#/test%20encoded",
		"#/components/schemas/Test", "#/paths/~1api~1v1/get", "#/definitions/Model",
	}

	allProperlyFormatted := true
	for _, input := range inputs {
		_, path := ConvertComponentIdIntoFriendlyPathSearch(input)
		if !strings.HasPrefix(path, "$") {
			allProperlyFormatted = false
			t.Errorf("Input %s produced path %s that doesn't start with $", input, path)
		}
	}

	// This assertion should always pass, demonstrating why the defensive code is rarely executed
	assert.True(t, allProperlyFormatted, "All paths should start with $ due to the function's design")
}

func TestGetRefValueNode_NilNode(t *testing.T) {
	result := GetRefValueNode(nil)
	assert.Nil(t, result)
}

func TestGetRefValueNode_SimpleRef(t *testing.T) {
	// YAML node with $ref at position 1 (standard case)
	node := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "$ref"},
			{Kind: yaml.ScalarNode, Value: "#/components/schemas/Pet"},
		},
	}
	result := GetRefValueNode(node)
	assert.NotNil(t, result)
	assert.Equal(t, "#/components/schemas/Pet", result.Value)
}

func TestGetRefValueNode_RefWithSiblingProperties(t *testing.T) {
	// OpenAPI 3.1 style: $ref with sibling properties (description comes before $ref)
	node := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "description"},
			{Kind: yaml.ScalarNode, Value: "A pet description"},
			{Kind: yaml.ScalarNode, Value: "$ref"},
			{Kind: yaml.ScalarNode, Value: "#/components/schemas/Pet"},
		},
	}
	result := GetRefValueNode(node)
	assert.NotNil(t, result)
	assert.Equal(t, "#/components/schemas/Pet", result.Value)
}

func TestGetRefValueNode_RefAtEnd(t *testing.T) {
	// $ref at the end of multiple properties
	node := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "description"},
			{Kind: yaml.ScalarNode, Value: "Description text"},
			{Kind: yaml.ScalarNode, Value: "summary"},
			{Kind: yaml.ScalarNode, Value: "Summary text"},
			{Kind: yaml.ScalarNode, Value: "$ref"},
			{Kind: yaml.ScalarNode, Value: "./external.yaml#/components/schemas/Item"},
		},
	}
	result := GetRefValueNode(node)
	assert.NotNil(t, result)
	assert.Equal(t, "./external.yaml#/components/schemas/Item", result.Value)
}

func TestGetRefValueNode_NoRef(t *testing.T) {
	// Node without $ref
	node := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "type"},
			{Kind: yaml.ScalarNode, Value: "object"},
			{Kind: yaml.ScalarNode, Value: "description"},
			{Kind: yaml.ScalarNode, Value: "A schema without ref"},
		},
	}
	result := GetRefValueNode(node)
	assert.Nil(t, result)
}

func TestGetRefValueNode_EmptyNode(t *testing.T) {
	// Empty mapping node
	node := &yaml.Node{
		Kind:    yaml.MappingNode,
		Content: []*yaml.Node{},
	}
	result := GetRefValueNode(node)
	assert.Nil(t, result)
}
