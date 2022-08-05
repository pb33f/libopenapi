package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"strconv"
	"sync"
)

const (
	PropertiesLabel           = "properties"
	AdditionalPropertiesLabel = "additionalProperties"
	ExampleLabel              = "example"
	ItemsLabel                = "items"
	AllOfLabel                = "allOf"
	AnyOfLabel                = "anyOf"
	OneOfLabel                = "oneOf"
	NotLabel                  = "not"
	DiscriminatorLabel        = "discriminator"
)

type Schema struct {
	Title                low.NodeReference[string]
	MultipleOf           low.NodeReference[int]
	Maximum              low.NodeReference[int]
	ExclusiveMaximum     low.NodeReference[int]
	Minimum              low.NodeReference[int]
	ExclusiveMinimum     low.NodeReference[int]
	MaxLength            low.NodeReference[int]
	MinLength            low.NodeReference[int]
	Pattern              low.NodeReference[string]
	MaxItems             low.NodeReference[int]
	MinItems             low.NodeReference[int]
	UniqueItems          low.NodeReference[int]
	MaxProperties        low.NodeReference[int]
	MinProperties        low.NodeReference[int]
	Required             []low.NodeReference[string]
	Enum                 []low.NodeReference[string]
	Type                 low.NodeReference[string]
	AllOf                []low.NodeReference[*Schema]
	OneOf                []low.NodeReference[*Schema]
	AnyOf                []low.NodeReference[*Schema]
	Not                  []low.NodeReference[*Schema]
	Items                []low.NodeReference[*Schema]
	Properties           map[low.NodeReference[string]]*low.NodeReference[*Schema]
	AdditionalProperties low.NodeReference[any]
	Description          low.NodeReference[string]
	Default              low.NodeReference[any]
	Nullable             low.NodeReference[bool]
	Discriminator        low.NodeReference[*Discriminator]
	ReadOnly             low.NodeReference[bool]
	WriteOnly            low.NodeReference[bool]
	XML                  *low.NodeReference[*XML]
	ExternalDocs         *low.NodeReference[*ExternalDoc]
	Example              low.NodeReference[any]
	Deprecated           low.NodeReference[bool]
	Extensions           map[low.NodeReference[string]]low.NodeReference[any]
}

func (s *Schema) FindProperty(name string) *low.NodeReference[*Schema] {
	for k, v := range s.Properties {
		if k.Value == name {
			return v
		}
	}
	return nil
}

func (s *Schema) Build(root *yaml.Node, idx *index.SpecIndex, level int) error {
	level++
	if level > 50 {
		return nil // we're done, son! too fricken deep.
	}

	extensionMap, err := ExtractExtensions(root)
	if err != nil {
		return err
	}
	s.Extensions = extensionMap

	// handle example if set.
	_, expLabel, expNode := utils.FindKeyNodeFull(ExampleLabel, root.Content)
	if expNode != nil {
		s.Example = low.NodeReference[any]{Value: expNode.Value, KeyNode: expLabel, ValueNode: expNode}
	}

	_, addPLabel, addPNode := utils.FindKeyNodeFull(AdditionalPropertiesLabel, root.Content)
	if addPNode != nil {
		if utils.IsNodeMap(addPNode) {
			var props map[string]interface{}
			addPNode.Decode(&props)
			s.AdditionalProperties = low.NodeReference[any]{Value: props, KeyNode: addPLabel, ValueNode: addPNode}
		}

		if utils.IsNodeBoolValue(addPNode) {
			b, _ := strconv.ParseBool(addPNode.Value)
			s.AdditionalProperties = low.NodeReference[any]{Value: b, KeyNode: addPLabel, ValueNode: addPNode}
		}
	}

	// handle discriminator if set.
	_, discLabel, discNode := utils.FindKeyNodeFull(DiscriminatorLabel, root.Content)
	if discNode != nil {
		var discriminator Discriminator
		err = BuildModel(discNode, &discriminator)
		if err != nil {
			return err
		}
		s.Discriminator = low.NodeReference[*Discriminator]{Value: &discriminator, KeyNode: discLabel, ValueNode: discNode}
	}

	// handle properties
	_, propLabel, propsNode := utils.FindKeyNodeFull(PropertiesLabel, root.Content)
	if propsNode != nil {
		propertyMap := make(map[low.NodeReference[string]]*low.NodeReference[*Schema])
		var currentProp *yaml.Node
		for i, prop := range propsNode.Content {
			if i%2 == 0 {
				currentProp = prop
				continue
			}

			var property Schema
			err = BuildModel(prop, &property)
			if err != nil {
				return err
			}
			err = property.Build(prop, idx, level)
			if err != nil {
				return err
			}
			propertyMap[low.NodeReference[string]{
				Value:     currentProp.Value,
				KeyNode:   propLabel,
				ValueNode: propsNode,
			}] = &low.NodeReference[*Schema]{
				Value:     &property,
				KeyNode:   currentProp,
				ValueNode: prop,
			}
		}
		s.Properties = propertyMap

		// extract all sub-schemas
		var errors []error
		var wg sync.WaitGroup

		var allOf, anyOf, oneOf, not, items []low.NodeReference[*Schema]

		// make this async at some point to speed things up.
		buildSchema(&allOf, AllOfLabel, idx, root, level, &errors, &wg)
		buildSchema(&anyOf, AnyOfLabel, idx, root, level, &errors, &wg)
		buildSchema(&oneOf, OneOfLabel, idx, root, level, &errors, &wg)
		buildSchema(&not, NotLabel, idx, root, level, &errors, &wg)
		buildSchema(&items, ItemsLabel, idx, root, level, &errors, &wg)
		//wg.Wait()

		if len(errors) > 0 {
			// todo fix this
			return errors[0]
		}
		if len(anyOf) > 0 {
			s.AnyOf = anyOf
		}
		if len(oneOf) > 0 {
			s.OneOf = oneOf
		}
		if len(allOf) > 0 {
			s.AllOf = allOf
		}
		if len(not) > 0 {
			s.Not = not
		}
		if len(items) > 0 {
			s.Items = items
		}

	}

	return nil
}

func buildSchema(schemas *[]low.NodeReference[*Schema], attribute string, idx *index.SpecIndex, rootNode *yaml.Node, level int, errors *[]error, wg *sync.WaitGroup) {
	_, labelNode, valueNode := utils.FindKeyNodeFull(attribute, rootNode.Content)
	//wg.Add(1)
	if valueNode != nil {

		var build = func(kn *yaml.Node, vn *yaml.Node) *low.NodeReference[*Schema] {
			var schema Schema
			err := BuildModel(vn, &schema)
			if err != nil {
				*errors = append(*errors, err)
				return nil
			}
			err = schema.Build(vn, idx, level)
			if err != nil {
				*errors = append(*errors, err)
				return nil
			}
			return &low.NodeReference[*Schema]{
				Value:     &schema,
				KeyNode:   kn,
				ValueNode: vn,
			}
		}

		if utils.IsNodeMap(valueNode) {
			schema := build(labelNode, valueNode)
			if schema != nil {
				*schemas = append(*schemas, *schema)
			}
		}
		if utils.IsNodeArray(valueNode) {
			for _, vn := range valueNode.Content {
				schema := build(vn, vn)
				if schema != nil {
					*schemas = append(*schemas, *schema)
				}
			}
		}

	}
	//wg.Done()
}
