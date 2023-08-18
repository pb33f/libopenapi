package utils

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
)

type Case int8

const (
	// OpenApi3 is used by all OpenAPI 3+ docs
	OpenApi3 = "openapi"

	// OpenApi2 is used by all OpenAPI 2 docs, formerly known as swagger.
	OpenApi2 = "swagger"

	// AsyncApi is used by akk AsyncAPI docs, all versions.
	AsyncApi = "asyncapi"

	PascalCase Case = iota
	CamelCase
	ScreamingSnakeCase
	SnakeCase
	KebabCase
	ScreamingKebabCase
	RegularCase
	UnknownCase
)

// FindNodes will find a node based on JSONPath, it accepts raw yaml/json as input.
func FindNodes(yamlData []byte, jsonPath string) ([]*yaml.Node, error) {
	jsonPath = FixContext(jsonPath)

	var node yaml.Node
	yaml.Unmarshal(yamlData, &node)

	path, err := yamlpath.NewPath(jsonPath)
	if err != nil {
		return nil, err
	}
	results, _ := path.Find(&node)
	return results, nil
}

// FindLastChildNode will find the last node in a tree, based on a starting node.
// Deprecated: This function is deprecated, use FindLastChildNodeWithLevel instead.
// this has the potential to cause a stack overflow, so use with caution. It will be removed later.
func FindLastChildNode(node *yaml.Node) *yaml.Node {
	s := len(node.Content) - 1
	if s < 0 {
		s = 0
	}
	if len(node.Content) > 0 && len(node.Content[s].Content) > 0 {
		return FindLastChildNode(node.Content[s])
	} else {
		if len(node.Content) > 0 {
			return node.Content[s]
		}
		return node
	}
}

// FindLastChildNodeWithLevel will find the last node in a tree, based on a starting node.
// Will stop searching after 100 levels, because that's just silly, we probably have a loop.
func FindLastChildNodeWithLevel(node *yaml.Node, level int) *yaml.Node {
	if level > 100 {
		return node // we've gone too far, give up.
	}
	s := len(node.Content) - 1
	if s < 0 {
		s = 0
	}
	if len(node.Content) > 0 && len(node.Content[s].Content) > 0 {
		level++
		return FindLastChildNodeWithLevel(node.Content[s], level)
	} else {
		if len(node.Content) > 0 {
			return node.Content[s]
		}
		return node
	}
}

// BuildPath will construct a JSONPath from a base and an array of strings.
func BuildPath(basePath string, segs []string) string {

	path := strings.Join(segs, ".")

	// trim that last period.
	if len(path) > 0 && path[len(path)-1] == '.' {
		path = path[:len(path)-1]
	}
	return fmt.Sprintf("%s.%s", basePath, path)
}

// FindNodesWithoutDeserializing will find a node based on JSONPath, without deserializing from yaml/json
func FindNodesWithoutDeserializing(node *yaml.Node, jsonPath string) ([]*yaml.Node, error) {
	jsonPath = FixContext(jsonPath)

	path, err := yamlpath.NewPath(jsonPath)
	if err != nil {
		return nil, err
	}
	results, _ := path.Find(node)
	return results, nil
}

// ConvertInterfaceIntoStringMap will convert an unknown input into a string map.
func ConvertInterfaceIntoStringMap(context interface{}) map[string]string {
	converted := make(map[string]string)
	if context != nil {
		if v, ok := context.(map[string]interface{}); ok {
			for k, n := range v {
				if s, okB := n.(string); okB {
					converted[k] = s
				}
			}
		}
		if v, ok := context.(map[string]string); ok {
			for k, n := range v {
				converted[k] = n
			}
		}
	}
	return converted
}

// ConvertInterfaceToStringArray will convert an unknown input map type into a string array/slice
func ConvertInterfaceToStringArray(raw interface{}) []string {
	if vals, ok := raw.(map[string]interface{}); ok {
		var s []string
		for _, v := range vals {
			if g, y := v.([]interface{}); y {
				for _, q := range g {
					s = append(s, fmt.Sprint(q))
				}
			}
		}
		return s
	}
	if vals, ok := raw.(map[string][]string); ok {
		var s []string
		for _, v := range vals {
			s = append(s, v...)
		}
		return s
	}
	return nil
}

// ConvertInterfaceArrayToStringArray will convert an unknown interface array type, into a string slice
func ConvertInterfaceArrayToStringArray(raw interface{}) []string {
	if vals, ok := raw.([]interface{}); ok {
		s := make([]string, len(vals))
		for i, v := range vals {
			s[i] = fmt.Sprint(v)
		}
		return s
	}
	if vals, ok := raw.([]string); ok {
		return vals
	}
	return nil
}

// ExtractValueFromInterfaceMap pulls out an unknown value from a map using a string key
func ExtractValueFromInterfaceMap(name string, raw interface{}) interface{} {

	if propMap, ok := raw.(map[string]interface{}); ok {
		if props, okn := propMap[name].([]interface{}); okn {
			return props
		} else {
			return propMap[name]
		}
	}
	if propMap, ok := raw.(map[string][]string); ok {
		return propMap[name]
	}

	return nil
}

// FindFirstKeyNode will locate the first key and value yaml.Node based on a key.
func FindFirstKeyNode(key string, nodes []*yaml.Node, depth int) (keyNode *yaml.Node, valueNode *yaml.Node) {
	if depth > 40 {
		return nil, nil
	}
	for i, v := range nodes {
		if key != "" && key == v.Value {
			if i+1 >= len(nodes) {
				return v, NodeAlias(nodes[i]) // this is the node we need.
			}
			return v, NodeAlias(nodes[i+1]) // next node is what we need.
		}
		if len(v.Content) > 0 {
			depth++
			x, y := FindFirstKeyNode(key, v.Content, depth)
			if x != nil && y != nil {
				return x, y
			}
		}
	}
	return nil, nil
}

// KeyNodeResult is a result from a KeyNodeSearch performed by the FindAllKeyNodesWithPath
type KeyNodeResult struct {
	KeyNode   *yaml.Node
	ValueNode *yaml.Node
	Parent    *yaml.Node
	Path      []yaml.Node
}

// KeyNodeSearch keeps a track of everything we have found on our adventure down the trees.
type KeyNodeSearch struct {
	Key             string
	Ignore          []string
	Results         []*KeyNodeResult
	AllowExtensions bool
}

// FindKeyNodeTop is a non-recursive search of top level nodes for a key, will not look at content.
// Returns the key and value
func FindKeyNodeTop(key string, nodes []*yaml.Node) (keyNode *yaml.Node, valueNode *yaml.Node) {
	for i, v := range nodes {
		if i%2 != 0 {
			continue
		}

		if strings.EqualFold(key, v.Value) {
			return v, nodes[i+1] // next node is what we need.
		}
	}
	return nil, nil
}

// FindKeyNode is a non-recursive search of a *yaml.Node Content for a child node with a key.
// Returns the key and value
func FindKeyNode(key string, nodes []*yaml.Node) (keyNode *yaml.Node, valueNode *yaml.Node) {

	//numNodes := len(nodes)
	for i, v := range nodes {
		if i%2 == 0 && key == v.Value {
			return v, nodes[i+1] // next node is what we need.
		}
		for x, j := range v.Content {
			if key == j.Value {
				if IsNodeMap(v) {
					if x+1 == len(v.Content) {
						return v, v.Content[x]
					}
					return v, v.Content[x+1] // next node is what we need.

				}
				if IsNodeArray(v) {
					return v, v.Content[x]
				}
			}
		}
	}
	return nil, nil
}

// FindKeyNodeFull is an overloaded version of FindKeyNode. This version however returns keys, labels and values.
// generally different things are required from different node trees, so depending on what this function is looking at
// it will return different things.
func FindKeyNodeFull(key string, nodes []*yaml.Node) (keyNode *yaml.Node, labelNode *yaml.Node, valueNode *yaml.Node) {
	for i := range nodes {
		if i%2 == 0 && key == nodes[i].Value {
			return nodes[i], nodes[i], nodes[i+1] // next node is what we need.
		}
	}
	for _, v := range nodes {
		for x := range v.Content {
			if key == v.Content[x].Value {
				if IsNodeMap(v) {
					if x+1 == len(v.Content) {
						return v, v.Content[x], NodeAlias(v.Content[x])
					}
					return v, v.Content[x], NodeAlias(v.Content[x+1])
				}
				if IsNodeArray(v) {
					return v, v.Content[x], NodeAlias(v.Content[x])
				}
			}
		}
	}
	return nil, nil, nil
}

// FindKeyNodeFullTop is an overloaded version of FindKeyNodeFull. This version only looks at the top
// level of the node and not the children.
func FindKeyNodeFullTop(key string, nodes []*yaml.Node) (keyNode *yaml.Node, labelNode *yaml.Node, valueNode *yaml.Node) {
	for i := range nodes {
		if i%2 != 0 {
			continue
		}
		if i%2 == 0 && key == nodes[i].Value {
			return nodes[i], nodes[i], NodeAlias(nodes[i+1]) // next node is what we need.
		}
	}
	return nil, nil, nil
}

type ExtensionNode struct {
	Key   *yaml.Node
	Value *yaml.Node
}

func FindExtensionNodes(nodes []*yaml.Node) []*ExtensionNode {
	var extensions []*ExtensionNode
	for i, v := range nodes {
		if i%2 == 0 && strings.HasPrefix(v.Value, "x-") {
			if i+1 < len(nodes) {
				extensions = append(extensions, &ExtensionNode{
					Key:   v,
					Value: NodeAlias(nodes[i+1]),
				})
			}
		}
	}
	return extensions
}

var ObjectLabel = "object"
var IntegerLabel = "integer"
var NumberLabel = "number"
var StringLabel = "string"
var BinaryLabel = "binary"
var ArrayLabel = "array"
var BooleanLabel = "boolean"
var SchemaSource = "https://json-schema.org/draft/2020-12/schema"
var SchemaId = "https://pb33f.io/openapi-changes/schema"

func MakeTagReadable(node *yaml.Node) string {
	switch node.Tag {
	case "!!map":
		return ObjectLabel
	case "!!seq":
		return ArrayLabel
	case "!!str":
		return StringLabel
	case "!!int":
		return IntegerLabel
	case "!!float":
		return NumberLabel
	case "!!bool":
		return BooleanLabel
	}
	return "unknown"
}

// IsNodeMap checks if the node is a map type
func IsNodeMap(node *yaml.Node) bool {
	if node == nil {
		return false
	}
	n := NodeAlias(node)
	return n.Tag == "!!map"
}

// IsNodeAlias checks if the node is an alias, and lifts out the anchor
func IsNodeAlias(node *yaml.Node) (*yaml.Node, bool) {
	if node == nil {
		return nil, false
	}
	if node.Kind == yaml.AliasNode {
		node = node.Alias
		return node, true
	}
	return node, false
}

// NodeAlias checks if the node is an alias, and lifts out the anchor
func NodeAlias(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.AliasNode {
		node = node.Alias
		return node
	}
	return node
}

// IsNodePolyMorphic will return true if the node contains polymorphic keys.
func IsNodePolyMorphic(node *yaml.Node) bool {
	n := NodeAlias(node)
	for i, v := range n.Content {
		if i%2 == 0 {
			if v.Value == "anyOf" || v.Value == "oneOf" || v.Value == "allOf" {
				return true
			}
		}
	}
	return false
}

// IsNodeArray checks if a node is an array type
func IsNodeArray(node *yaml.Node) bool {
	if node == nil {
		return false
	}
	n := NodeAlias(node)
	return n.Tag == "!!seq"
}

// IsNodeStringValue checks if a node is a string value
func IsNodeStringValue(node *yaml.Node) bool {
	if node == nil {
		return false
	}
	n := NodeAlias(node)
	return n.Tag == "!!str"
}

// IsNodeIntValue will check if a node is an int value
func IsNodeIntValue(node *yaml.Node) bool {
	if node == nil {
		return false
	}
	n := NodeAlias(node)
	return n.Tag == "!!int"
}

// IsNodeFloatValue will check is a node is a float value.
func IsNodeFloatValue(node *yaml.Node) bool {
	if node == nil {
		return false
	}
	n := NodeAlias(node)
	return n.Tag == "!!float"
}

// IsNodeNumberValue will check if a node can be parsed as a float value.
func IsNodeNumberValue(node *yaml.Node) bool {
	if node == nil {
		return false
	}
	return IsNodeIntValue(node) || IsNodeFloatValue(node)
}

// IsNodeBoolValue will check is a node is a bool
func IsNodeBoolValue(node *yaml.Node) bool {
	if node == nil {
		return false
	}
	n := NodeAlias(node)
	return n.Tag == "!!bool"
}

func IsNodeRefValue(node *yaml.Node) (bool, *yaml.Node, string) {

	if node == nil {
		return false, nil, ""
	}
	n := NodeAlias(node)
	for i, r := range n.Content {
		if i%2 == 0 {
			if r.Value == "$ref" {
				return true, r, n.Content[i+1].Value
			}
		}
	}
	return false, nil, ""
}

// FixContext will clean up a JSONpath string to be correctly traversable.
func FixContext(context string) string {
	tokens := strings.Split(context, ".")
	var cleaned = []string{}

	for i, t := range tokens {
		if v, err := strconv.Atoi(t); err == nil {
			if v < 200 { // codes start here
				if cleaned[i-1] != "" {
					cleaned[i-1] += fmt.Sprintf("[%v]", t)
				}
			} else {
				cleaned = append(cleaned, t)
			}
			continue
		}
		cleaned = append(cleaned, strings.ReplaceAll(t, "(root)", "$"))
	}

	return strings.Join(cleaned, ".")
}

// IsJSON will tell you if a string is JSON or not.
func IsJSON(testString string) bool {
	if testString == "" {
		return false
	}
	runes := []rune(strings.TrimSpace(testString))
	if runes[0] == '{' && runes[len(runes)-1] == '}' {
		return true
	}
	return false
}

// IsYAML will tell you if a string is YAML or not.
func IsYAML(testString string) bool {
	if testString == "" {
		return false
	}
	if IsJSON(testString) {
		return false
	}
	var n interface{}
	err := yaml.Unmarshal([]byte(testString), &n)
	if err != nil {
		return false
	}
	_, err = yaml.Marshal(n)
	return err == nil
}

// ConvertYAMLtoJSON will do exactly what you think it will. It will deserialize YAML into serialized JSON.
func ConvertYAMLtoJSON(yamlData []byte) ([]byte, error) {
	var decodedYaml map[string]interface{}
	err := yaml.Unmarshal(yamlData, &decodedYaml)
	if err != nil {
		return nil, err
	}
	// if the data can be decoded, it can be encoded (that's my view anyway). no need for an error check.
	jsonData, _ := json.Marshal(decodedYaml)
	return jsonData, nil
}

// ConvertYAMLtoJSONPretty will do exactly what you think it will. It will deserialize YAML into serialized JSON.
// However, this version will apply prefix/indentation to the JSON.
func ConvertYAMLtoJSONPretty(yamlData []byte, prefix string, indent string) ([]byte, error) {
	var decodedYaml map[string]interface{}
	err := yaml.Unmarshal(yamlData, &decodedYaml)
	if err != nil {
		return nil, err
	}
	// if the data can be decoded, it can be encoded (that's my view anyway). no need for an error check.
	jsonData, _ := json.MarshalIndent(decodedYaml, prefix, indent)
	return jsonData, nil
}

// IsHttpVerb will check if an operation is valid or not.
func IsHttpVerb(verb string) bool {
	verbs := []string{"get", "post", "put", "patch", "delete", "options", "trace", "head"}
	for _, v := range verbs {
		if verb == v {
			return true
		}
	}
	return false
}

// define bracket name expression
var bracketNameExp = regexp.MustCompile("^(\\w+)\\[(\\w+)\\]$")

func ConvertComponentIdIntoFriendlyPathSearch(id string) (string, string) {
	segs := strings.Split(id, "/")
	name, _ := url.QueryUnescape(strings.ReplaceAll(segs[len(segs)-1], "~1", "/"))
	var cleaned []string

	// check for strange spaces, chars and if found, wrap them up, clean them and create a new cleaned path.
	for i := range segs {
		pathCharExp, _ := regexp.MatchString("[%=;~.]", segs[i])
		if pathCharExp {
			segs[i], _ = url.QueryUnescape(strings.ReplaceAll(segs[i], "~1", "/"))
			segs[i] = fmt.Sprintf("['%s']", segs[i])
			if len(cleaned) > 0 {
				cleaned[len(cleaned)-1] = fmt.Sprintf("%s%s", segs[i-1], segs[i])
				continue
			}
		} else {

			// check for brackets in the name, and if found, rewire the path to encapsulate them
			// correctly. https://github.com/pb33f/libopenapi/issues/112
			brackets := bracketNameExp.FindStringSubmatch(segs[i])

			if len(brackets) > 0 {
				segs[i] = bracketNameExp.ReplaceAllString(segs[i], "['$1[$2]']")
				if len(cleaned) > 0 {
					cleaned[len(cleaned)-1] = fmt.Sprintf("%s%s", segs[i-1], segs[i])
					continue
				}
			}

			intVal, err := strconv.ParseInt(segs[i], 10, 32)
			if err == nil && intVal <= 99 {
				segs[i] = fmt.Sprintf("[%d]", intVal)
				cleaned[len(cleaned)-1] = fmt.Sprintf("%s%s", cleaned[len(cleaned)-1], segs[i])
				continue
			}
			if err == nil && intVal > 99 {
				segs[i] = fmt.Sprintf("['%d']", intVal)
				cleaned[len(cleaned)-1] = fmt.Sprintf("%s%s", cleaned[len(cleaned)-1], segs[i])
				continue
			}
			cleaned = append(cleaned, segs[i])
		}
	}
	_, err := strconv.ParseInt(name, 10, 32)
	var replaced string
	if err != nil {
		replaced = strings.ReplaceAll(fmt.Sprintf("%s",
			strings.Join(cleaned, ".")), "#", "$")
	} else {
		replaced = strings.ReplaceAll(fmt.Sprintf("%s",
			strings.Join(cleaned, ".")), "#", "$")
	}

	if len(replaced) > 0 {
		if replaced[0] != '$' {
			replaced = fmt.Sprintf("$%s", replaced)
		}
	}
	return name, replaced
}

func ConvertComponentIdIntoPath(id string) (string, string) {
	segs := strings.Split(id, "/")
	name := segs[len(segs)-1]

	return name, strings.ReplaceAll(fmt.Sprintf("%s.%s",
		strings.Join(segs[:len(segs)-1], "."), name), "#", "$")
}

func RenderCodeSnippet(startNode *yaml.Node, specData []string, before, after int) string {

	buf := new(strings.Builder)

	startLine := startNode.Line - before
	endLine := startNode.Line + after

	if startLine < 0 {
		startLine = 0
	}

	if endLine >= len(specData) {
		endLine = len(specData) - 1
	}

	delta := endLine - startLine

	for i := 0; i < delta; i++ {
		l := startLine + i
		if l < len(specData) {
			line := specData[l]
			buf.WriteString(fmt.Sprintf("%s\n", line))
		}
	}

	return buf.String()
}

func DetectCase(input string) Case {
	trim := strings.TrimSpace(input)
	if trim == "" {
		return UnknownCase
	}

	pascalCase := regexp.MustCompile("^[A-Z][a-z]+(?:[A-Z][a-z]+)*$")
	camelCase := regexp.MustCompile("^[a-z]+(?:[A-Z][a-z]+)*$")
	screamingSnakeCase := regexp.MustCompile("^[A-Z]+(_[A-Z]+)*$")
	snakeCase := regexp.MustCompile("^[a-z]+(_[a-z]+)*$")
	kebabCase := regexp.MustCompile("^[a-z]+(-[a-z]+)*$")
	screamingKebabCase := regexp.MustCompile("^[A-Z]+(-[A-Z]+)*$")
	if pascalCase.MatchString(trim) {
		return PascalCase
	}
	if camelCase.MatchString(trim) {
		return CamelCase
	}
	if screamingSnakeCase.MatchString(trim) {
		return ScreamingSnakeCase
	}
	if snakeCase.MatchString(trim) {
		return SnakeCase
	}
	if kebabCase.MatchString(trim) {
		return KebabCase
	}
	if screamingKebabCase.MatchString(trim) {
		return ScreamingKebabCase
	}
	return RegularCase
}

// CheckEnumForDuplicates will check an array of nodes to check if there are any duplicate values.
func CheckEnumForDuplicates(seq []*yaml.Node) []*yaml.Node {
	var res []*yaml.Node
	seen := make(map[string]*yaml.Node)

	for _, enum := range seq {
		if seen[enum.Value] != nil {
			res = append(res, enum)
			continue
		}
		seen[enum.Value] = enum
	}
	return res
}

// DetermineWhitespaceLength will determine the length of the whitespace for a JSON or YAML file.
func DetermineWhitespaceLength(input string) int {
	exp := regexp.MustCompile(`\n( +)`)
	whiteSpace := exp.FindAllStringSubmatch(input, -1)
	var filtered []string
	for i := range whiteSpace {
		filtered = append(filtered, whiteSpace[i][1])
	}
	sort.Strings(filtered)
	if len(filtered) > 0 {
		return len(filtered[0])
	} else {
		return 0
	}
}

// CheckForMergeNodes will check the top level of the schema for merge nodes. If any are found, then the merged nodes
// will be appended to the end of the rest of the nodes in the schema.
// Note: this is a destructive operation, so the in-memory node structure will be modified
func CheckForMergeNodes(node *yaml.Node) {
	if node == nil {
		return
	}
	total := len(node.Content)
	for i := 0; i < total; i++ {
		mn := node.Content[i]
		if i%2 == 0 {
			if mn.Tag == "!!merge" {
				an := node.Content[i+1].Alias
				if an != nil {
					node.Content = append(node.Content, an.Content...) // append the merged nodes
					total = len(node.Content)
					i += 2
				}
			}
		}
	}
}
