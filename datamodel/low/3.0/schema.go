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
	AllOf                low.NodeReference[[]low.ValueReference[*SchemaProxy]]
	OneOf                low.NodeReference[[]low.ValueReference[*SchemaProxy]]
	AnyOf                low.NodeReference[[]low.ValueReference[*SchemaProxy]]
	Not                  low.NodeReference[[]low.ValueReference[*SchemaProxy]]
	Items                low.NodeReference[[]low.ValueReference[*SchemaProxy]]
	Properties           low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*SchemaProxy]]
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

func (s *Schema) FindProperty(name string) *low.ValueReference[*SchemaProxy] {
	return low.FindItemInMap[*SchemaProxy](name, s.Properties.Value)
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
			return fmt.Errorf("build schema failed: reference cannot be found: '%s', line %d, col %d",
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
	bChan := make(chan schemaProxyBuildResult)

	var buildProperty = func(label *yaml.Node, value *yaml.Node, c chan schemaProxyBuildResult) {
		c <- schemaProxyBuildResult{
			k: low.KeyReference[string]{
				KeyNode: label,
				Value:   label.Value,
			},
			v: low.ValueReference[*SchemaProxy]{
				Value:     &SchemaProxy{kn: label, vn: value, idx: idx},
				ValueNode: value,
			},
		}
	}

	// handle properties
	_, propLabel, propsNode := utils.FindKeyNodeFull(PropertiesLabel, root.Content)
	if propsNode != nil {
		propertyMap := make(map[low.KeyReference[string]]low.ValueReference[*SchemaProxy])
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
			go buildProperty(currentProp, prop, bChan)
		}
		completedProps := 0
		for completedProps < totalProps {
			select {
			case res := <-bChan:
				completedProps++
				propertyMap[res.k] = res.v
			}
		}
		s.Properties = low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*SchemaProxy]]{
			Value:     propertyMap,
			KeyNode:   propLabel,
			ValueNode: propsNode,
		}
	}

	var allOf, anyOf, oneOf, not, items []low.ValueReference[*SchemaProxy]

	_, allOfLabel, allOfValue := utils.FindKeyNodeFull(AllOfLabel, root.Content)
	_, anyOfLabel, anyOfValue := utils.FindKeyNodeFull(AnyOfLabel, root.Content)
	_, oneOfLabel, oneOfValue := utils.FindKeyNodeFull(OneOfLabel, root.Content)
	_, notLabel, notValue := utils.FindKeyNodeFull(NotLabel, root.Content)
	_, itemsLabel, itemsValue := utils.FindKeyNodeFull(ItemsLabel, root.Content)

	errorChan := make(chan error)
	allOfChan := make(chan schemaProxyBuildResult)
	anyOfChan := make(chan schemaProxyBuildResult)
	oneOfChan := make(chan schemaProxyBuildResult)
	itemsChan := make(chan schemaProxyBuildResult)
	notChan := make(chan schemaProxyBuildResult)

	totalBuilds := countSubSchemaItems(allOfValue) +
		countSubSchemaItems(anyOfValue) +
		countSubSchemaItems(oneOfValue) +
		countSubSchemaItems(notValue) +
		countSubSchemaItems(itemsValue)

	if allOfValue != nil {
		go buildSchema(allOfChan, allOfLabel, allOfValue, errorChan, idx)
	}
	if anyOfValue != nil {
		go buildSchema(anyOfChan, anyOfLabel, anyOfValue, errorChan, idx)
	}
	if oneOfValue != nil {
		go buildSchema(oneOfChan, oneOfLabel, oneOfValue, errorChan, idx)
	}
	if itemsValue != nil {
		go buildSchema(itemsChan, itemsLabel, itemsValue, errorChan, idx)
	}
	if notValue != nil {
		go buildSchema(notChan, notLabel, notValue, errorChan, idx)
	}

	completeCount := 0
	for completeCount < totalBuilds {
		select {
		case e := <-errorChan:
			return e
		case r := <-allOfChan:
			completeCount++
			allOf = append(allOf, r.v)
		case r := <-anyOfChan:
			completeCount++
			anyOf = append(anyOf, r.v)
		case r := <-oneOfChan:
			completeCount++
			oneOf = append(oneOf, r.v)
		case r := <-itemsChan:
			completeCount++
			items = append(items, r.v)
		case r := <-notChan:
			completeCount++
			not = append(not, r.v)
		}
	}

	if len(anyOf) > 0 {
		s.AnyOf = low.NodeReference[[]low.ValueReference[*SchemaProxy]]{
			Value:     anyOf,
			KeyNode:   anyOfLabel,
			ValueNode: anyOfValue,
		}
	}
	if len(oneOf) > 0 {
		s.OneOf = low.NodeReference[[]low.ValueReference[*SchemaProxy]]{
			Value:     oneOf,
			KeyNode:   oneOfLabel,
			ValueNode: oneOfValue,
		}
	}
	if len(allOf) > 0 {
		s.AllOf = low.NodeReference[[]low.ValueReference[*SchemaProxy]]{
			Value:     allOf,
			KeyNode:   allOfLabel,
			ValueNode: allOfValue,
		}
	}
	if len(not) > 0 {
		s.Not = low.NodeReference[[]low.ValueReference[*SchemaProxy]]{
			Value:     not,
			KeyNode:   notLabel,
			ValueNode: notValue,
		}

	}
	if len(items) > 0 {
		s.Items = low.NodeReference[[]low.ValueReference[*SchemaProxy]]{
			Value:     items,
			KeyNode:   itemsLabel,
			ValueNode: itemsValue,
		}
	}
	return nil
}

func countSubSchemaItems(node *yaml.Node) int {
	if utils.IsNodeMap(node) {
		return 1
	}
	if utils.IsNodeArray(node) {
		return len(node.Content)
	}
	return 0
}

type schemaBuildResult struct {
	k low.KeyReference[string]
	v low.ValueReference[*Schema]
}

type schemaProxyBuildResult struct {
	k low.KeyReference[string]
	v low.ValueReference[*SchemaProxy]
}

func (s *Schema) extractExtensions(root *yaml.Node) {
	s.Extensions = low.ExtractExtensions(root)
}

func buildSchema(schemas chan schemaProxyBuildResult, labelNode, valueNode *yaml.Node, errors chan error, idx *index.SpecIndex) {

	if valueNode != nil {
		syncChan := make(chan *low.ValueReference[*SchemaProxy])
		errorChan := make(chan error)

		// build out a SchemaProxy for every sub-schema.
		build := func(kn *yaml.Node, vn *yaml.Node, c chan *low.ValueReference[*SchemaProxy], e chan error) {
			if h, _, _ := utils.IsNodeRefValue(vn); h {
				ref := low.LocateRefNode(vn, idx)
				if ref != nil {
					vn = ref
				} else {
					err := fmt.Errorf("build schema failed: reference cannot be found: %s, line %d, col %d",
						vn.Content[1].Value, vn.Content[1].Line, vn.Content[1].Column)
					e <- err
				}
			}

			// a proxy design works best here. polymorphism, pretty much guarantees that a sub-schema can
			// take on circular references through polymorphism. Like the resolver, if we try and follow these
			// journey's through hyperspace, we will end up creating endless amounts of threads, spinning off
			// chasing down circles, that in turn spin up endless threads.
			// In order to combat this, we need a schema proxy that will only resolve the schema when asked, and then
			// it will only do it one level at a time.
			sp := new(SchemaProxy)
			sp.kn = kn
			sp.vn = vn
			sp.idx = idx

			res := &low.ValueReference[*SchemaProxy]{
				Value:     sp,
				ValueNode: vn,
			}
			c <- res
		}

		if utils.IsNodeMap(valueNode) {
			if h, _, _ := utils.IsNodeRefValue(valueNode); h {
				ref := low.LocateRefNode(valueNode, idx)
				if ref != nil {
					valueNode = ref
				} else {
					errors <- fmt.Errorf("build schema failed: reference cannot be found: %s, line %d, col %d",
						valueNode.Content[1].Value, valueNode.Content[1].Line, valueNode.Content[1].Column)
					return
				}
			}

			// this only runs once, however to keep things consistent, it makes sense to use the same async method
			// that arrays will use.
			go build(labelNode, valueNode, syncChan, errorChan)
			select {
			case e := <-errorChan:
				errors <- e
				break
			case r := <-syncChan:
				schemas <- schemaProxyBuildResult{
					k: low.KeyReference[string]{
						KeyNode: labelNode,
						Value:   labelNode.Value,
					},
					v: *r,
				}
			}
		}
		if utils.IsNodeArray(valueNode) {
			refBuilds := 0
			for _, vn := range valueNode.Content {
				if h, _, _ := utils.IsNodeRefValue(vn); h {
					ref := low.LocateRefNode(vn, idx)
					if ref != nil {
						vn = ref
					} else {
						err := fmt.Errorf("build schema failed: reference cannot be found: %s, line %d, col %d",
							vn.Content[1].Value, vn.Content[1].Line, vn.Content[1].Column)
						errors <- err
						return
					}
				}
				refBuilds++
				go build(vn, vn, syncChan, errorChan)
			}
			completedBuilds := 0
			for completedBuilds < refBuilds {
				select {
				case res := <-syncChan:
					completedBuilds++
					schemas <- schemaProxyBuildResult{
						k: low.KeyReference[string]{
							KeyNode: labelNode,
							Value:   labelNode.Value,
						},
						v: *res,
					}
				}
			}
		}

	}
}

func ExtractSchema(root *yaml.Node, idx *index.SpecIndex) (*low.NodeReference[*SchemaProxy], error) {
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
		schema := &SchemaProxy{kn: schLabel, vn: schNode, idx: idx}
		return &low.NodeReference[*SchemaProxy]{Value: schema, KeyNode: schLabel, ValueNode: schNode}, nil
	}
	return nil, nil
}
