package v3

import (
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"sync"
)

const (
	PropertiesLabel    = "properties"
	ItemsLabel         = "items"
	AllOfLabel         = "allOf"
	AnyOfLabel         = "anyOf"
	OneOfLabel         = "oneOf"
	NotLabel           = "not"
	DiscriminatorLabel = "discriminator"
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
	AllOf                *low.NodeReference[*Schema]
	OneOf                *low.NodeReference[*Schema]
	AnyOf                *low.NodeReference[*Schema]
	Not                  *low.NodeReference[*Schema]
	Items                *low.NodeReference[*Schema]
	Properties           map[low.NodeReference[string]]*low.NodeReference[*Schema]
	AdditionalProperties low.NodeReference[any]
	Description          low.NodeReference[string]
	Default              low.NodeReference[any]
	Nullable             low.NodeReference[bool]
	Discriminator        *low.NodeReference[*Discriminator]
	ReadOnly             low.NodeReference[bool]
	WriteOnly            low.NodeReference[bool]
	XML                  *low.NodeReference[*XML]
	ExternalDocs         *low.NodeReference[*ExternalDoc]
	Example              low.NodeReference[any]
	Deprecated           low.NodeReference[bool]
	Extensions           map[low.NodeReference[string]]low.NodeReference[any]
}

func (s *Schema) Build(root *yaml.Node, idx *index.SpecIndex, level int) error {
	level++
	if level > 50 {
		return nil // we done, son! too fricken deep.
	}

	extensionMap, err := datamodel.ExtractExtensions(root)
	if err != nil {
		return err
	}
	s.Extensions = extensionMap

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
			err = datamodel.BuildModel(prop, &property)
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

		var allOf, anyOf, oneOf, not, items low.NodeReference[*Schema]
		allOf = low.NodeReference[*Schema]{Value: &Schema{}}
		anyOf = low.NodeReference[*Schema]{Value: &Schema{}}
		oneOf = low.NodeReference[*Schema]{Value: &Schema{}}
		not = low.NodeReference[*Schema]{Value: &Schema{}}
		items = low.NodeReference[*Schema]{Value: &Schema{}}
		go buildSchema(&allOf, AllOfLabel, idx, root, level, &errors, &wg)
		go buildSchema(&anyOf, AnyOfLabel, idx, root, level, &errors, &wg)
		go buildSchema(&oneOf, OneOfLabel, idx, root, level, &errors, &wg)
		go buildSchema(&not, NotLabel, idx, root, level, &errors, &wg)
		go buildSchema(&items, ItemsLabel, idx, root, level, &errors, &wg)
		wg.Wait()

		if len(errors) > 0 {
			// todo fix this
			return errors[0]
		}
		if anyOf.KeyNode != nil {
			s.AnyOf = &anyOf
		}
		if oneOf.KeyNode != nil {
			s.OneOf = &oneOf
		}
		if allOf.KeyNode != nil {
			s.AllOf = &allOf
		}
		if not.KeyNode != nil {
			s.Not = &not
		}
		if items.KeyNode != nil {
			s.Items = &items
		}

	}

	return nil
}

func buildSchema(schema *low.NodeReference[*Schema], attribute string, idx *index.SpecIndex, rootNode *yaml.Node, level int, errors *[]error, wg *sync.WaitGroup) {
	_, labelNode, valueNode := utils.FindKeyNodeFull(attribute, rootNode.Content)
	if valueNode != nil {
		wg.Add(1)
		err := datamodel.BuildModel(valueNode, &schema.Value)
		if err != nil {
			*errors = append(*errors, err)
			return
		}
		err = schema.Value.Build(rootNode, idx, level)
		if err != nil {
			*errors = append(*errors, err)
			return
		}
		schema.KeyNode = labelNode
		schema.ValueNode = valueNode
		wg.Done()
	}
}