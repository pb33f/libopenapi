// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

// Package high contains a set of high-level models that represent OpenAPI 2 and 3 documents.
// These high-level models (porcelain) are used by applications directly, rather than the low-level models
// plumbing) that are used to compose high level models.
//
// High level models are simple to navigate, strongly typed, precise representations of the OpenAPI schema
// that are created from an OpenAPI specification.
//
// All high level objects contains a 'GoLow' method. This 'GoLow' method will return the low-level model that
// was used to create it, which provides an engineer as much low level detail about the raw spec used to create
// those models, things like key/value breakdown of each value, lines, column, source comments etc.
package high

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"gopkg.in/yaml.v3"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// GoesLow is used to represent any high-level model. All high level models meet this interface and can be used to
// extract low-level models from any high-level model.
type GoesLow[T any] interface {

	// GoLow returns the low-level object that was used to create the high-level object. This allows consumers
	// to dive-down into the plumbing API at any point in the model.
	GoLow() T
}

// ExtractExtensions is a convenience method for converting low-level extension definitions, to a high level map[string]any
// definition that is easier to consume in applications.
func ExtractExtensions(extensions map[low.KeyReference[string]]low.ValueReference[any]) map[string]any {
	extracted := make(map[string]any)
	for k, v := range extensions {
		extracted[k.Value] = v.Value
	}
	return extracted
}

// UnpackExtensions is a convenience function that makes it easy and simple to unpack an objects extensions
// into a complex type, provided as a generic. This function is for high-level models that implement `GoesLow()`
// and for low-level models that support extensions via `HasExtensions`.
//
// This feature will be upgraded at some point to hold a registry of types and extension mappings to make this
// functionality available a little more automatically.
// You can read more about the discussion here: https://github.com/pb33f/libopenapi/issues/8
//
// `T` represents the Type you want to unpack into
// `R` represents the LOW type of the object that contains the extensions (not the high)
// `low` represents the HIGH type of the object that contains the extensions.
//
// to use:
//  schema := schemaProxy.Schema() // any high-level object that has extensions
//  extensions, err := UnpackExtensions[MyComplexType, low.Schema](schema)
func UnpackExtensions[T any, R low.HasExtensions[T]](low GoesLow[R]) (map[string]*T, error) {
	m := make(map[string]*T)
	ext := low.GoLow().GetExtensions()
	for i := range ext {
		key := i.Value
		g := new(T)
		valueNode := ext[i].ValueNode
		err := valueNode.Decode(g)
		if err != nil {
			return nil, err
		}
		m[key] = g
	}
	return m, nil
}

// MarshalExtensions is a convenience function that makes it easy and simple to marshal an objects extensions into a
// map that can then correctly rendered back down in to YAML.
func MarshalExtensions(parent *yaml.Node, extensions map[string]any) {
	for k := range extensions {
		AddYAMLNode(parent, k, extensions[k])
	}
}

type NodeEntry struct {
	Key   string
	Value any
	Line  int
}

type NodeBuilder struct {
	Nodes []*NodeEntry
	High  any
	Low   any
}

func NewNodeBuilder(high any, low any) *NodeBuilder {
	// create a new node builder
	nb := &NodeBuilder{High: high, Low: low}

	// extract fields from the high level object and add them into our node builder.
	// this will allow us to extract the line numbers from the low level object as well.
	v := reflect.ValueOf(high).Elem()
	num := v.NumField()
	for i := 0; i < num; i++ {
		nb.add(v.Type().Field(i).Name)
	}
	return nb
}

func (n *NodeBuilder) add(key string) {

	// only operate on exported fields.
	if unicode.IsLower(rune(key[0])) {
		return
	}

	// if the key is 'Extensions' then we need to extract the keys from the map
	// and add them to the node builder.
	if key == "Extensions" {
		extensions := reflect.ValueOf(n.High).Elem().FieldByName(key)
		for _, e := range extensions.MapKeys() {
			v := extensions.MapIndex(e)

			extKey := e.String()
			extValue := v.Interface()
			nodeEntry := &NodeEntry{Key: extKey, Value: extValue}

			if !reflect.ValueOf(n.Low).IsZero() {
				fieldValue := reflect.ValueOf(n.Low).Elem().FieldByName("Extensions")
				f := fieldValue.Interface()
				value := reflect.ValueOf(f)
				switch value.Kind() {
				case reflect.Map:
					if j, ok := n.Low.(low.HasExtensionsUntyped); ok {
						originalExtensions := j.GetExtensions()
						for k := range originalExtensions {
							if k.Value == extKey {
								nodeEntry.Line = originalExtensions[k].ValueNode.Line
							}
						}
					}
				default:
					panic("not supported yet")
				}
			}
			n.Nodes = append(n.Nodes, nodeEntry)
		}
		// done, extensions are handled separately.
		return
	}

	// find the field with the tag supplied.
	field, _ := reflect.TypeOf(n.High).Elem().FieldByName(key)
	tag := string(field.Tag.Get("yaml"))
	tagName := strings.Split(tag, ",")[0]

	// extract the value of the field
	fieldValue := reflect.ValueOf(n.High).Elem().FieldByName(key)
	f := fieldValue.Interface()
	value := reflect.ValueOf(f)

	// create a new node entry
	nodeEntry := &NodeEntry{Key: tagName}

	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		nodeEntry.Value = strconv.FormatInt(value.Int(), 10)
	case reflect.String:
		nodeEntry.Value = value.String()
	case reflect.Bool:
		nodeEntry.Value = value.Bool()
	case reflect.Ptr:
		nodeEntry.Value = f
	case reflect.Map:
		nodeEntry.Value = f
	default:
		panic("not supported yet")
	}

	// if there is no low level object, then we cannot extract line numbers,
	// so skip and default to 0, which means a new entry to the spec.
	// this will place new content and the top of the rendered object.
	if !reflect.ValueOf(n.Low).IsZero() {
		lowFieldValue := reflect.ValueOf(n.Low).Elem().FieldByName(key)
		fLow := lowFieldValue.Interface()
		value = reflect.ValueOf(fLow)
		switch value.Kind() {
		case reflect.Struct:
			nb := value.Interface().(low.HasValueNodeUntyped).GetValueNode()
			if nb != nil {
				nodeEntry.Line = nb.Line
			}
		default:
			// everything else, weight it to the bottom of the rendered object.
			// this is things that we have no way of knowing where they should be placed.
			nodeEntry.Line = 9999
		}
	}
	n.Nodes = append(n.Nodes, nodeEntry)
}

func (n *NodeBuilder) Render() *yaml.Node {
	// order nodes by line number, retain original order
	sort.Slice(n.Nodes, func(i, j int) bool {
		return n.Nodes[i].Line < n.Nodes[j].Line
	})
	m := CreateEmptyMapNode()
	for i := range n.Nodes {
		node := n.Nodes[i]
		AddYAMLNode(m, node.Key, node.Value)
	}
	return m
}

func AddYAMLNode(parent *yaml.Node, key string, value any) *yaml.Node {

	if value == nil {
		return parent
	}

	// check the type
	t := reflect.TypeOf(value)
	var l *yaml.Node
	if key != "" {
		l = CreateStringNode(key)
	}
	var valueNode *yaml.Node
	switch t.Kind() {
	case reflect.Struct:
		panic("no way dude, why?")
	case reflect.Ptr:
		rawRender, _ := value.(Renderable).MarshalYAML()
		if rawRender != nil {
			valueNode = rawRender.(*yaml.Node)
		} else {
			return parent
		}
	default:
		var rawNode yaml.Node
		err := rawNode.Encode(value)
		if err != nil {
			return parent
		} else {
			valueNode = &rawNode
		}
	}
	if l != nil {
		parent.Content = append(parent.Content, l, valueNode)
	} else {
		parent.Content = valueNode.Content
	}

	return parent
}

func CreateEmptyMapNode() *yaml.Node {
	n := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
	}
	return n
}

func CreateStringNode(str string) *yaml.Node {
	n := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: str,
	}
	return n
}

func CreateIntNode(val int) *yaml.Node {
	i := strconv.Itoa(val)
	n := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!int",
		Value: i,
	}
	return n
}

type Renderable interface {
	MarshalYAML() (interface{}, error)
}
