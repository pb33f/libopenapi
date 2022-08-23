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

	if low.IsCircular(root, idx) {
		return nil // circular references cannot be built.
	}

	if level > 30 {
		return fmt.Errorf("schema is too nested to continue: %d levels deep, is too deep", level) // we're done, son! too fricken deep.
	}
	level++
	if h, _, _ := utils.IsNodeRefValue(root); h {
		ref := low.LocateRefNode(root, idx)
		if ref != nil {
			root = ref
		} else {
			return fmt.Errorf("build schema failed: reference cannot be found: %s, line %d, col %d",
				root.Content[1].Value, root.Content[1].Line, root.Content[1].Column)
		}
	}

	s.extractExtensions(root)

	// handle example if set.
	_, expLabel, expNode := utils.FindKeyNodeFull(ExampleLabel, root.Content)
	if expNode != nil {
		s.Example = low.NodeReference[any]{Value: ExtractExampleValue(expNode), KeyNode: expLabel, ValueNode: expNode}
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
		_ = low.BuildModel(discNode, &discriminator)
		s.Discriminator = low.NodeReference[*Discriminator]{Value: &discriminator, KeyNode: discLabel, ValueNode: discNode}
	}

	// handle externalDocs if set.
	_, extDocLabel, extDocNode := utils.FindKeyNodeFull(ExternalDocsLabel, root.Content)
	if extDocNode != nil {
		var exDoc ExternalDoc
		_ = low.BuildModel(extDocNode, &exDoc)
		_ = exDoc.Build(extDocNode, idx) // throws no errors, can't check for one.
		s.ExternalDocs = low.NodeReference[*ExternalDoc]{Value: &exDoc, KeyNode: extDocLabel, ValueNode: extDocNode}
	}

	// handle xml if set.
	_, xmlLabel, xmlNode := utils.FindKeyNodeFull(XMLLabel, root.Content)
	if xmlNode != nil {
		var xml XML
		_ = low.BuildModel(xmlNode, &xml)
		// extract extensions if set.
		_ = xml.Build(xmlNode) // returns no errors, can't check for one.
		s.XML = low.NodeReference[*XML]{Value: &xml, KeyNode: xmlLabel, ValueNode: xmlNode}
	}

	// for property, build in a new thread!
	bChan := make(chan schemaBuildResult)
	eChan := make(chan error)

	var buildProperty = func(label *yaml.Node, value *yaml.Node, c chan schemaBuildResult, ec chan<- error) {
		// have we seen this before?
		seen := getSeenSchema(fmt.Sprintf("%d:%d", value.Line, value.Column))
		if seen != nil {
			c <- schemaBuildResult{
				k: low.KeyReference[string]{
					KeyNode: label,
					Value:   label.Value,
				},
				v: low.ValueReference[*Schema]{
					Value:     seen,
					ValueNode: value,
				},
			}
			return
		}
		p := new(Schema)
		_ = low.BuildModel(value, p)
		err := p.BuildLevel(value, idx, level)
		if err != nil {
			ec <- err
			return
		}
		c <- schemaBuildResult{
			k: low.KeyReference[string]{
				KeyNode: label,
				Value:   label.Value,
			},
			v: low.ValueReference[*Schema]{
				Value:     p,
				ValueNode: value,
			},
		}
	}

	// handle properties
	_, propLabel, propsNode := utils.FindKeyNodeFull(PropertiesLabel, root.Content)
	if propsNode != nil {
		propertyMap := make(map[low.KeyReference[string]]low.ValueReference[*Schema])
		var currentProp *yaml.Node
		totalProps := 0
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
				} else {
					return fmt.Errorf("schema properties build failed: cannot find reference %s, line %d, col %d",
						prop.Content[1].Value, prop.Content[1].Column, prop.Content[1].Line)
				}
			}
			totalProps++
			go buildProperty(currentProp, prop, bChan, eChan)
		}
		completedProps := 0
		for completedProps < totalProps {
			select {
			case err := <-eChan:
				return err
			case res := <-bChan:
				completedProps++
				propertyMap[res.k] = res.v
			}
		}
		s.Properties = low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Schema]]{
			Value:     propertyMap,
			KeyNode:   propLabel,
			ValueNode: propsNode,
		}
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

	return nil
}

type schemaBuildResult struct {
	k low.KeyReference[string]
	v low.ValueReference[*Schema]
}

func (s *Schema) extractExtensions(root *yaml.Node) {
	s.Extensions = low.ExtractExtensions(root)
}

func buildSchema(schemas *[]low.NodeReference[*Schema], attribute string, rootNode *yaml.Node, level int,
	errors *[]error, idx *index.SpecIndex) (labelNode *yaml.Node, valueNode *yaml.Node) {

	_, labelNode, valueNode = utils.FindKeyNodeFull(attribute, rootNode.Content)
	//wg.Add(1)
	if valueNode == nil {
		return nil, nil
	}

	if valueNode != nil {

		build := func(kn *yaml.Node, vn *yaml.Node) *low.NodeReference[*Schema] {
			schema := new(Schema)
			if h, _, _ := utils.IsNodeRefValue(vn); h {
				ref := low.LocateRefNode(vn, idx)
				if ref != nil {
					vn = ref
				} else {
					*errors = append(*errors, fmt.Errorf("build schema failed: reference cannot be found: %s, line %d, col %d",
						vn.Content[1].Value, vn.Content[1].Line, vn.Content[1].Column))
					return nil
				}
			}

			seen := getSeenSchema(fmt.Sprintf("%d:%d", vn.Line, vn.Column))
			if seen != nil {
				return &low.NodeReference[*Schema]{
					Value:     seen,
					KeyNode:   kn,
					ValueNode: vn,
				}
			}

			_ = low.BuildModel(vn, schema)

			// add schema before we build, so it doesn't get stuck in an infinite loop.
			addSeenSchema(fmt.Sprintf("%d:%d", vn.Line, vn.Column), schema)

			err := schema.BuildLevel(vn, idx, level)
			if err != nil {
				*errors = append(*errors, err)
				return nil
			}

			return &low.NodeReference[*Schema]{
				Value:     schema,
				KeyNode:   kn,
				ValueNode: vn,
			}
		}

		if utils.IsNodeMap(valueNode) {
			if h, _, _ := utils.IsNodeRefValue(valueNode); h {
				ref := low.LocateRefNode(valueNode, idx)
				if ref != nil {
					valueNode = ref
				} else {
					*errors = append(*errors, fmt.Errorf("build schema failed: reference cannot be found: %s, line %d, col %d",
						valueNode.Content[1].Value, valueNode.Content[1].Line, valueNode.Content[1].Column))
					return
				}
			}

			schema := build(labelNode, valueNode)
			if schema != nil {
				*schemas = append(*schemas, *schema)
			}
		}
		if utils.IsNodeArray(valueNode) {
			//fmt.Println("polymorphic looping sucks dude.")
			for _, vn := range valueNode.Content {
				if h, _, _ := utils.IsNodeRefValue(vn); h {
					ref := low.LocateRefNode(vn, idx)
					if ref != nil {
						vn = ref
					} else {
						*errors = append(*errors, fmt.Errorf("build schema failed: reference cannot be found: %s, line %d, col %d",
							vn.Content[1].Value, vn.Content[1].Line, vn.Content[1].Column))
					}
				}

				schema := build(vn, vn)
				if schema != nil {
					*schemas = append(*schemas, *schema)
				}
			}
		}

	}
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
		// check if schema has already been built.
		seen := getSeenSchema(fmt.Sprintf("%d:%d", schNode.Line, schNode.Column))
		if seen != nil {
			return &low.NodeReference[*Schema]{Value: seen, KeyNode: schLabel, ValueNode: schNode}, nil
		}

		var schema Schema
		_ = low.BuildModel(schNode, &schema)
		err := schema.Build(schNode, idx)
		addSeenSchema(fmt.Sprintf("%d:%d", schNode.Line, schNode.Column), &schema)
		if err != nil {
			return nil, err
		}

		return &low.NodeReference[*Schema]{Value: &schema, KeyNode: schLabel, ValueNode: schNode}, nil
	}
	return nil, nil
}
