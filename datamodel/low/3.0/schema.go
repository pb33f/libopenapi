package v3

import (
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"strconv"
)

const (
	PropertiesLabel           = "properties"
	AdditionalPropertiesLabel = "additionalProperties"
	XMLLabel                  = "xml"
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
	Format               low.NodeReference[string]
	MaxItems             low.NodeReference[int]
	MinItems             low.NodeReference[int]
	UniqueItems          low.NodeReference[int]
	MaxProperties        low.NodeReference[int]
	MinProperties        low.NodeReference[int]
	Required             low.NodeReference[[]low.ValueReference[string]]
	Enum                 low.NodeReference[[]low.ValueReference[string]]
	Type                 low.NodeReference[string]
	AllOf                low.NodeReference[[]low.NodeReference[*Schema]]
	OneOf                low.NodeReference[[]low.NodeReference[*Schema]]
	AnyOf                low.NodeReference[[]low.NodeReference[*Schema]]
	Not                  low.NodeReference[[]low.NodeReference[*Schema]]
	Items                low.NodeReference[[]low.NodeReference[*Schema]]
	Properties           low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Schema]]
	AdditionalProperties low.NodeReference[any]
	Description          low.NodeReference[string]
	Default              low.NodeReference[any]
	Nullable             low.NodeReference[bool]
	Discriminator        low.NodeReference[*Discriminator]
	ReadOnly             low.NodeReference[bool]
	WriteOnly            low.NodeReference[bool]
	XML                  low.NodeReference[*XML]
	ExternalDocs         low.NodeReference[*ExternalDoc]
	Example              low.NodeReference[any]
	Deprecated           low.NodeReference[bool]
	Extensions           map[low.KeyReference[string]]low.ValueReference[any]
}

func (s *Schema) FindProperty(name string) *low.ValueReference[*Schema] {
	return low.FindItemInMap[*Schema](name, s.Properties.Value)
}

func (s *Schema) Build(root *yaml.Node, idx *index.SpecIndex) error {
	return s.BuildLevel(root, idx, 0)
}

func (s *Schema) BuildLevel(root *yaml.Node, idx *index.SpecIndex, level int) error {
	level++
	if level > 10 {
		return nil // we're done, son! too fricken deep.
	}

	s.extractExtensions(root)

	// handle example if set.
	_, expLabel, expNode := utils.FindKeyNodeFull(ExampleLabel, root.Content)
	if expNode != nil {
		s.Example = low.NodeReference[any]{Value: expNode.Value, KeyNode: expLabel, ValueNode: expNode}
	}

	_, addPLabel, addPNode := utils.FindKeyNodeFull(AdditionalPropertiesLabel, root.Content)
	if addPNode != nil {
		if utils.IsNodeMap(addPNode) {
			schema, serr := low.ExtractObjectRaw[*Schema](addPNode, idx)
			if serr != nil {
				return serr
			}
			s.AdditionalProperties = low.NodeReference[any]{Value: schema, KeyNode: addPLabel, ValueNode: addPNode}
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
		err := low.BuildModel(discNode, &discriminator)
		if err != nil {
			return err
		}
		s.Discriminator = low.NodeReference[*Discriminator]{Value: &discriminator, KeyNode: discLabel, ValueNode: discNode}
	}

	// handle externalDocs if set.
	_, extDocLabel, extDocNode := utils.FindKeyNodeFull(ExternalDocsLabel, root.Content)
	if extDocNode != nil {
		var exDoc ExternalDoc
		err := low.BuildModel(extDocNode, &exDoc)
		if err != nil {
			return err
		}
		err = exDoc.Build(extDocNode, idx)
		if err != nil {
			return err
		}
		s.ExternalDocs = low.NodeReference[*ExternalDoc]{Value: &exDoc, KeyNode: extDocLabel, ValueNode: extDocNode}
	}

	// handle xml if set.
	_, xmlLabel, xmlNode := utils.FindKeyNodeFull(XMLLabel, root.Content)
	if xmlNode != nil {
		var xml XML
		err := low.BuildModel(xmlNode, &xml)
		if err != nil {
			return err
		}
		// extract extensions if set.
		err = xml.Build(xmlNode)
		if err != nil {
			return err
		}
		s.XML = low.NodeReference[*XML]{Value: &xml, KeyNode: xmlLabel, ValueNode: xmlNode}
	}

	// handle properties
	_, propLabel, propsNode := utils.FindKeyNodeFull(PropertiesLabel, root.Content)
	if propsNode != nil {
		propertyMap := make(map[low.KeyReference[string]]low.ValueReference[*Schema])
		var currentProp *yaml.Node
		for i, prop := range propsNode.Content {
			if i%2 == 0 {
				currentProp = prop
				continue
			}

			// check our prop isn't reference
			if h, _, _ := utils.IsNodeRefValue(prop); h {
				ref := low.LocateRefNode(prop, idx)
				if ref != nil {
					prop = ref
				}
			}

			var property Schema
			err := low.BuildModel(prop, &property)
			if err != nil {
				return err
			}
			err = property.BuildLevel(prop, idx, level)
			if err != nil {
				return err
			}
			propertyMap[low.KeyReference[string]{
				Value:   currentProp.Value,
				KeyNode: currentProp,
			}] = low.ValueReference[*Schema]{
				Value:     &property,
				ValueNode: prop,
			}
		}
		s.Properties = low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Schema]]{
			Value:     propertyMap,
			KeyNode:   propLabel,
			ValueNode: propsNode,
		}

		// extract all sub-schemas
		var errors []error

		var allOf, anyOf, oneOf, not, items []low.NodeReference[*Schema]

		// make this async at some point to speed things up.
		allOfLabel, allOfValue := buildSchema(&allOf, AllOfLabel, root, level, &errors, idx)
		anyOfLabel, anyOfValue := buildSchema(&anyOf, AnyOfLabel, root, level, &errors, idx)
		oneOfLabel, oneOfValue := buildSchema(&oneOf, OneOfLabel, root, level, &errors, idx)
		notLabel, notValue := buildSchema(&not, NotLabel, root, level, &errors, idx)
		itemsLabel, itemsValue := buildSchema(&items, ItemsLabel, root, level, &errors, idx)

		if len(errors) > 0 {
			// todo fix this
			return errors[0]
		}
		if len(anyOf) > 0 {
			s.AnyOf = low.NodeReference[[]low.NodeReference[*Schema]]{
				Value:     anyOf,
				KeyNode:   anyOfLabel,
				ValueNode: anyOfValue,
			}
		}
		if len(oneOf) > 0 {
			s.OneOf = low.NodeReference[[]low.NodeReference[*Schema]]{
				Value:     oneOf,
				KeyNode:   oneOfLabel,
				ValueNode: oneOfValue,
			}
		}
		if len(allOf) > 0 {
			s.AllOf = low.NodeReference[[]low.NodeReference[*Schema]]{
				Value:     allOf,
				KeyNode:   allOfLabel,
				ValueNode: allOfValue,
			}
		}
		if len(not) > 0 {
			s.Not = low.NodeReference[[]low.NodeReference[*Schema]]{
				Value:     not,
				KeyNode:   notLabel,
				ValueNode: notValue,
			}
		}
		if len(items) > 0 {
			s.Items = low.NodeReference[[]low.NodeReference[*Schema]]{
				Value:     items,
				KeyNode:   itemsLabel,
				ValueNode: itemsValue,
			}
		}
	}
	return nil
}

func (s *Schema) extractExtensions(root *yaml.Node) {
	s.Extensions = low.ExtractExtensions(root)
}

func buildSchema(schemas *[]low.NodeReference[*Schema], attribute string, rootNode *yaml.Node, level int,
	errors *[]error, idx *index.SpecIndex) (labelNode *yaml.Node, valueNode *yaml.Node) {

	_, labelNode, valueNode = utils.FindKeyNodeFull(attribute, rootNode.Content)
	//wg.Add(1)
	if valueNode != nil {
		var build = func(kn *yaml.Node, vn *yaml.Node) *low.NodeReference[*Schema] {
			var schema Schema
			err := low.BuildModel(vn, &schema)
			if err != nil {
				*errors = append(*errors, err)
				return nil
			}
			err = schema.BuildLevel(vn, idx, level)
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
	return labelNode, valueNode
}

func ExtractSchema(root *yaml.Node, idx *index.SpecIndex) (*low.NodeReference[*Schema], error) {
	var schLabel, schNode *yaml.Node
	errStr := "schema build failed: reference '%s' cannot be found at line %d, col %d"
	if rf, rl, _ := utils.IsNodeRefValue(root); rf {
		// locate reference in index.
		ref := low.LocateRefNode(root, idx)
		if ref != nil {
			schNode = ref
			schLabel = rl
		} else {
			return nil, fmt.Errorf(errStr,
				root.Content[1].Value, root.Content[1].Line, root.Content[1].Column)
		}
	} else {
		_, schLabel, schNode = utils.FindKeyNodeFull(SchemaLabel, root.Content)
		if schNode != nil {
			if h, _, _ := utils.IsNodeRefValue(schNode); h {
				ref := low.LocateRefNode(schNode, idx)
				if ref != nil {
					schNode = ref
				} else {
					return nil, fmt.Errorf(errStr,
						schNode.Content[1].Value, schNode.Content[1].Line, schNode.Content[1].Column)
				}
			}
		}
	}

	if schNode != nil {
		var schema Schema
		err := low.BuildModel(schNode, &schema)
		if err != nil {
			return nil, err
		}
		err = schema.Build(schNode, idx)
		if err != nil {
			return nil, err
		}
		return &low.NodeReference[*Schema]{Value: &schema, KeyNode: schLabel, ValueNode: schNode}, nil
	}
	return nil, nil
}
