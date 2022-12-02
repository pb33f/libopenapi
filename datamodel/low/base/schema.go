package base

import (
	"crypto/sha256"
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"sort"
	"strconv"
	"strings"
)

// SchemaDynamicValue is used to hold multiple possible values for a schema property. There are two values, a left
// value (A) and a right value (B). The left value (A) is a 3.0 schema property value, the right value (B) is a 3.1
// schema value.
//
// OpenAPI 3.1 treats a Schema as a real JSON schema, which means some properties become incompatible, or others
// now support more than one primitive type or structure.
// The N value is a bit to make it each to know which value (A or B) is used, this prevents having to
// if/else on the value to determine which one is set.
type SchemaDynamicValue[A any, B any] struct {
	N int // 0 == A, 1 == B
	A A
	B B
}

// IsA will return true if the 'A' or left value is set. (OpenAPI 3)
func (s SchemaDynamicValue[A, B]) IsA() bool {
	return s.N == 0
}

// IsB will return true if the 'B' or right value is set (OpenAPI 3.1)
func (s SchemaDynamicValue[A, B]) IsB() bool {
	return s.N == 1
}

// Schema represents a JSON Schema that support Swagger, OpenAPI 3 and OpenAPI 3.1
//
// Until 3.1 OpenAPI had a strange relationship with JSON Schema. It's been a super-set/sub-set
// mix, which has been confusing. So, instead of building a bunch of different models, we have compressed
// all variations into a single model that makes it easy to support multiple spec types.
//
//  - v2 schema: https://swagger.io/specification/v2/#schemaObject
//  - v3 schema: https://swagger.io/specification/#schema-object
//  - v3.1 schema: https://spec.openapis.org/oas/v3.1.0#schema-object
type Schema struct {

	// Reference to the '$schema' dialect setting (3.1 only)
	SchemaTypeRef low.NodeReference[string]

	// In versions 2 and 3.0, this ExclusiveMaximum can only be a boolean.
	ExclusiveMaximum low.NodeReference[SchemaDynamicValue[bool, int64]]

	// In versions 2 and 3.0, this ExclusiveMinimum can only be a boolean.
	ExclusiveMinimum low.NodeReference[SchemaDynamicValue[bool, int64]]

	// In versions 2 and 3.0, this Type is a single value, so array will only ever have one value
	// in version 3.1, Type can be multiple values
	Type low.NodeReference[SchemaDynamicValue[string, []low.ValueReference[string]]]

	// Schemas are resolved on demand using a SchemaProxy
	AllOf low.NodeReference[[]low.ValueReference[*SchemaProxy]]

	// Polymorphic Schemas are only available in version 3+
	OneOf         low.NodeReference[[]low.ValueReference[*SchemaProxy]]
	AnyOf         low.NodeReference[[]low.ValueReference[*SchemaProxy]]
	Discriminator low.NodeReference[*Discriminator]

	// in 3.1 examples can be an array (which is recommended)
	Examples low.NodeReference[[]low.ValueReference[any]]
	// in 3.1 PrefixItems provides tuple validation using prefixItems.
	PrefixItems low.NodeReference[[]low.ValueReference[*SchemaProxy]]

	// Compatible with all versions
	Title                low.NodeReference[string]
	MultipleOf           low.NodeReference[int64]
	Maximum              low.NodeReference[int64]
	Minimum              low.NodeReference[int64]
	MaxLength            low.NodeReference[int64]
	MinLength            low.NodeReference[int64]
	Pattern              low.NodeReference[string]
	Format               low.NodeReference[string]
	MaxItems             low.NodeReference[int64]
	MinItems             low.NodeReference[int64]
	UniqueItems          low.NodeReference[int64]
	MaxProperties        low.NodeReference[int64]
	MinProperties        low.NodeReference[int64]
	Required             low.NodeReference[[]low.ValueReference[string]]
	Enum                 low.NodeReference[[]low.ValueReference[any]]
	Not                  low.NodeReference[[]low.ValueReference[*SchemaProxy]]
	Items                low.NodeReference[[]low.ValueReference[*SchemaProxy]]
	Properties           low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*SchemaProxy]]
	AdditionalProperties low.NodeReference[any]
	Description          low.NodeReference[string]
	ContentEncoding      low.NodeReference[string]
	ContentMediaType     low.NodeReference[string]
	Default              low.NodeReference[any]
	Nullable             low.NodeReference[bool]
	ReadOnly             low.NodeReference[bool]
	WriteOnly            low.NodeReference[bool]
	XML                  low.NodeReference[*XML]
	ExternalDocs         low.NodeReference[*ExternalDoc]
	Example              low.NodeReference[any]
	Deprecated           low.NodeReference[bool]
	Extensions           map[low.KeyReference[string]]low.ValueReference[any]
}

// Hash will calculate a SHA256 hash from the values of the schema, This allows equality checking against
// Schemas defined inside an OpenAPI document. The only way to know if a schema has changed, is to hash it.
func (s *Schema) Hash() [32]byte {
	// calculate a hash from every property in the schema.
	var d []string
	if !s.SchemaTypeRef.IsEmpty() {
		d = append(d, fmt.Sprint(s.SchemaTypeRef.Value))
	}
	if !s.Title.IsEmpty() {
		d = append(d, fmt.Sprint(s.Title.Value))
	}
	if !s.MultipleOf.IsEmpty() {
		d = append(d, fmt.Sprint(s.MultipleOf.Value))
	}
	if !s.Maximum.IsEmpty() {
		d = append(d, fmt.Sprint(s.Maximum.Value))
	}
	if !s.Minimum.IsEmpty() {
		d = append(d, fmt.Sprint(s.Minimum.Value))
	}
	if !s.MaxLength.IsEmpty() {
		d = append(d, fmt.Sprint(s.MaxLength.Value))
	}
	if !s.MinLength.IsEmpty() {
		d = append(d, fmt.Sprint(s.MinLength.Value))
	}
	if !s.Pattern.IsEmpty() {
		d = append(d, fmt.Sprint(s.Pattern.Value))
	}
	if !s.Format.IsEmpty() {
		d = append(d, fmt.Sprint(s.Format.Value))
	}
	if !s.MaxItems.IsEmpty() {
		d = append(d, fmt.Sprint(s.MaxItems.Value))
	}
	if !s.MinItems.IsEmpty() {
		d = append(d, fmt.Sprint(s.MinItems.Value))
	}
	if !s.UniqueItems.IsEmpty() {
		d = append(d, fmt.Sprint(s.UniqueItems.Value))
	}
	if !s.MaxProperties.IsEmpty() {
		d = append(d, fmt.Sprint(s.MaxProperties.Value))
	}
	if !s.MinProperties.IsEmpty() {
		d = append(d, fmt.Sprint(s.MinProperties.Value))
	}
	if !s.AdditionalProperties.IsEmpty() {
		d = append(d, low.GenerateHashString(s.AdditionalProperties.Value))
	}
	if !s.Description.IsEmpty() {
		d = append(d, fmt.Sprint(s.Description.Value))
	}
	if !s.ContentEncoding.IsEmpty() {
		d = append(d, fmt.Sprint(s.ContentEncoding.Value))
	}
	if !s.ContentMediaType.IsEmpty() {
		d = append(d, fmt.Sprint(s.ContentMediaType.Value))
	}
	if !s.Default.IsEmpty() {
		d = append(d, low.GenerateHashString(s.Default.Value))
	}
	if !s.Nullable.IsEmpty() {
		d = append(d, fmt.Sprint(s.Nullable.Value))
	}
	if !s.ReadOnly.IsEmpty() {
		d = append(d, fmt.Sprint(s.ReadOnly.Value))
	}
	if !s.WriteOnly.IsEmpty() {
		d = append(d, fmt.Sprint(s.WriteOnly.Value))
	}
	if !s.Deprecated.IsEmpty() {
		d = append(d, fmt.Sprint(s.Deprecated.Value))
	}
	if !s.ExclusiveMaximum.IsEmpty() && s.ExclusiveMaximum.Value.IsA() {
		d = append(d, fmt.Sprint(s.ExclusiveMaximum.Value.A))
	}
	if !s.ExclusiveMaximum.IsEmpty() && s.ExclusiveMaximum.Value.IsB() {
		d = append(d, fmt.Sprint(s.ExclusiveMaximum.Value.B))
	}
	if !s.ExclusiveMinimum.IsEmpty() && s.ExclusiveMinimum.Value.IsA() {
		d = append(d, fmt.Sprint(s.ExclusiveMinimum.Value.A))
	}
	if !s.ExclusiveMinimum.IsEmpty() && s.ExclusiveMinimum.Value.IsB() {
		d = append(d, fmt.Sprint(s.ExclusiveMinimum.Value.B))
	}
	if !s.Type.IsEmpty() && s.Type.Value.IsA() {
		d = append(d, fmt.Sprint(s.Type.Value.A))
	}
	if !s.Type.IsEmpty() && s.Type.Value.IsB() {
		j := make([]string, len(s.Type.Value.B))
		for h := range s.Type.Value.B {
			j[h] = s.Type.Value.B[h].Value
		}
		sort.Strings(j)
		d = append(d, strings.Join(j, "|"))
	}

	keys := make([]string, len(s.Required.Value))
	for i := range s.Required.Value {
		keys[i] = s.Required.Value[i].Value
	}
	sort.Strings(keys)
	d = append(d, keys...)

	keys = make([]string, len(s.Enum.Value))
	for i := range s.Enum.Value {
		keys[i] = fmt.Sprint(s.Enum.Value[i].Value)
	}
	sort.Strings(keys)
	d = append(d, keys...)

	for i := range s.Enum.Value {
		d = append(d, fmt.Sprint(s.Enum.Value[i].Value))
	}
	propertyKeys := make([]string, len(s.Properties.Value))
	z := 0
	for i := range s.Properties.Value {
		propertyKeys[z] = i.Value
		z++
	}
	sort.Strings(propertyKeys)
	for k := range propertyKeys {
		prop := s.FindProperty(propertyKeys[k]).Value
		if !prop.IsSchemaReference() {
			d = append(d, low.GenerateHashString(prop.Schema()))
		}
	}
	if s.XML.Value != nil {
		d = append(d, low.GenerateHashString(s.XML.Value))
	}
	if s.ExternalDocs.Value != nil {
		d = append(d, low.GenerateHashString(s.ExternalDocs.Value))
	}
	if s.Discriminator.Value != nil {
		d = append(d, low.GenerateHashString(s.Discriminator.Value))
	}

	// hash polymorphic data
	if len(s.OneOf.Value) > 0 {
		oneOfKeys := make([]string, len(s.OneOf.Value))
		oneOfEntities := make(map[string]*SchemaProxy)
		z = 0
		for i := range s.OneOf.Value {
			g := s.OneOf.Value[i].Value
			r := low.GenerateHashString(g)
			oneOfEntities[r] = g
			oneOfKeys[z] = r
			z++

		}
		sort.Strings(oneOfKeys)
		for k := range oneOfKeys {
			d = append(d, low.GenerateHashString(oneOfEntities[oneOfKeys[k]]))
		}
	}

	if len(s.AllOf.Value) > 0 {
		allOfKeys := make([]string, len(s.AllOf.Value))
		allOfEntities := make(map[string]*SchemaProxy)
		z = 0
		for i := range s.AllOf.Value {
			g := s.AllOf.Value[i].Value
			r := low.GenerateHashString(g)
			allOfEntities[r] = g
			allOfKeys[z] = r
			z++

		}
		sort.Strings(allOfKeys)
		for k := range allOfKeys {
			d = append(d, low.GenerateHashString(allOfEntities[allOfKeys[k]]))
		}
	}

	if len(s.AnyOf.Value) > 0 {
		anyOfKeys := make([]string, len(s.AnyOf.Value))
		anyOfEntities := make(map[string]*SchemaProxy)
		z = 0
		for i := range s.AnyOf.Value {
			g := s.AnyOf.Value[i].Value
			r := low.GenerateHashString(g)
			anyOfEntities[r] = g
			anyOfKeys[z] = r
			z++

		}
		sort.Strings(anyOfKeys)
		for k := range anyOfKeys {
			d = append(d, low.GenerateHashString(anyOfEntities[anyOfKeys[k]]))
		}
	}

	if len(s.Not.Value) > 0 {
		notKeys := make([]string, len(s.Not.Value))
		notEntities := make(map[string]*SchemaProxy)
		z = 0
		for i := range s.Not.Value {
			g := s.Not.Value[i].Value
			r := low.GenerateHashString(g)
			notEntities[r] = g
			notKeys[z] = r
			z++

		}
		sort.Strings(notKeys)
		for k := range notKeys {
			d = append(d, low.GenerateHashString(notEntities[notKeys[k]]))
		}
	}

	if len(s.Items.Value) > 0 {
		itemsKeys := make([]string, len(s.Items.Value))
		itemsEntities := make(map[string]*SchemaProxy)
		z = 0
		for i := range s.Items.Value {
			g := s.Items.Value[i].Value
			r := low.GenerateHashString(g)
			itemsEntities[r] = g
			itemsKeys[z] = r
			z++
		}
		sort.Strings(itemsKeys)
		for k := range itemsKeys {
			d = append(d, low.GenerateHashString(itemsEntities[itemsKeys[k]]))
		}
	}

	if len(s.PrefixItems.Value) > 0 {
		itemsKeys := make([]string, len(s.PrefixItems.Value))
		itemsEntities := make(map[string]*SchemaProxy)
		z = 0
		for i := range s.PrefixItems.Value {
			g := s.PrefixItems.Value[i].Value
			r := low.GenerateHashString(g)
			itemsEntities[r] = g
			itemsKeys[z] = r
			z++
		}
		sort.Strings(itemsKeys)
		for k := range itemsKeys {
			d = append(d, low.GenerateHashString(itemsEntities[itemsKeys[k]]))
		}
	}

	// add extensions to hash
	keys = make([]string, len(s.Extensions))
	z = 0
	for k := range s.Extensions {
		keys[z] = fmt.Sprintf("%s-%x", k.Value, sha256.Sum256([]byte(fmt.Sprint(s.Extensions[k].Value))))
		z++
	}
	sort.Strings(keys)
	d = append(d, keys...)
	if s.Example.Value != nil {
		d = append(d, low.GenerateHashString(s.Example.Value))
	}
	if !s.Examples.IsEmpty() {
		var xph []string
		for w := range s.Examples.Value {
			xph = append(xph, low.GenerateHashString(s.Examples.Value[w].Value))
		}
		sort.Strings(xph)
		d = append(d, strings.Join(xph, "|"))
	}
	return sha256.Sum256([]byte(strings.Join(d, "|")))
}

// FindProperty will return a ValueReference pointer containing a SchemaProxy pointer
// from a property key name. if found
func (s *Schema) FindProperty(name string) *low.ValueReference[*SchemaProxy] {
	return low.FindItemInMap[*SchemaProxy](name, s.Properties.Value)
}

// GetExtensions returns all extensions for Schema
func (s *Schema) GetExtensions() map[low.KeyReference[string]]low.ValueReference[any] {
	return s.Extensions
}

// Build will perform a number of operations.
// Extraction of the following happens in this method:
//  - Extensions
//  - Type
//  - ExclusiveMinimum and ExclusiveMaximum
//  - Examples
//  - AdditionalProperties
//  - Discriminator
//  - ExternalDocs
//  - XML
//  - Properties
//  - AllOf, OneOf, AnyOf
//  - Not
//  - Items
//  - PrefixItems
func (s *Schema) Build(root *yaml.Node, idx *index.SpecIndex) error {
	if h, _, _ := utils.IsNodeRefValue(root); h {
		ref, err := low.LocateRefNode(root, idx)
		if ref != nil {
			root = ref
			if err != nil {
				if !idx.AllowCircularReferenceResolving() {
					return fmt.Errorf("build schema failed: %s", err.Error())
				}
			}
		} else {
			return fmt.Errorf("build schema failed: reference cannot be found: '%s', line %d, col %d",
				root.Content[1].Value, root.Content[1].Line, root.Content[1].Column)
		}
	}

	s.extractExtensions(root)

	// determine schema type, singular (3.0) or multiple (3.1), use a variable value
	_, typeLabel, typeValue := utils.FindKeyNodeFullTop(TypeLabel, root.Content)
	if typeValue != nil {
		if utils.IsNodeStringValue(typeValue) {
			s.Type = low.NodeReference[SchemaDynamicValue[string, []low.ValueReference[string]]]{
				KeyNode:   typeLabel,
				ValueNode: typeValue,
				Value:     SchemaDynamicValue[string, []low.ValueReference[string]]{N: 0, A: typeValue.Value},
			}
		}
		if utils.IsNodeArray(typeValue) {

			var refs []low.ValueReference[string]
			for r := range typeValue.Content {
				refs = append(refs, low.ValueReference[string]{
					Value:     typeValue.Content[r].Value,
					ValueNode: typeValue.Content[r],
				})
			}
			s.Type = low.NodeReference[SchemaDynamicValue[string, []low.ValueReference[string]]]{
				KeyNode:   typeLabel,
				ValueNode: typeValue,
				Value:     SchemaDynamicValue[string, []low.ValueReference[string]]{N: 1, B: refs},
			}
		}
	}

	// determine exclusive minimum type, bool (3.0) or int (3.1)
	_, exMinLabel, exMinValue := utils.FindKeyNodeFullTop(ExclusiveMinimumLabel, root.Content)
	if exMinValue != nil {
		if utils.IsNodeBoolValue(exMinValue) {
			val, _ := strconv.ParseBool(exMinValue.Value)
			s.ExclusiveMinimum = low.NodeReference[SchemaDynamicValue[bool, int64]]{
				KeyNode:   exMinLabel,
				ValueNode: exMinValue,
				Value:     SchemaDynamicValue[bool, int64]{N: 0, A: val},
			}
		}
		if utils.IsNodeIntValue(exMinValue) {
			val, _ := strconv.ParseInt(exMinValue.Value, 10, 64)
			s.ExclusiveMinimum = low.NodeReference[SchemaDynamicValue[bool, int64]]{
				KeyNode:   exMinLabel,
				ValueNode: exMinValue,
				Value:     SchemaDynamicValue[bool, int64]{N: 1, B: val},
			}
		}
	}

	// determine exclusive maximum type, bool (3.0) or int (3.1)
	_, exMaxLabel, exMaxValue := utils.FindKeyNodeFullTop(ExclusiveMaximumLabel, root.Content)
	if exMaxValue != nil {
		if utils.IsNodeBoolValue(exMaxValue) {
			val, _ := strconv.ParseBool(exMaxValue.Value)
			s.ExclusiveMaximum = low.NodeReference[SchemaDynamicValue[bool, int64]]{
				KeyNode:   exMaxLabel,
				ValueNode: exMaxValue,
				Value:     SchemaDynamicValue[bool, int64]{N: 0, A: val},
			}
		}
		if utils.IsNodeIntValue(exMaxValue) {
			val, _ := strconv.ParseInt(exMaxValue.Value, 10, 64)
			s.ExclusiveMaximum = low.NodeReference[SchemaDynamicValue[bool, int64]]{
				KeyNode:   exMaxLabel,
				ValueNode: exMaxValue,
				Value:     SchemaDynamicValue[bool, int64]{N: 1, B: val},
			}
		}
	}

	// handle schema reference type if set. (3.1)
	_, schemaRefLabel, schemaRefNode := utils.FindKeyNodeFullTop(SchemaTypeLabel, root.Content)
	if schemaRefNode != nil {
		s.SchemaTypeRef = low.NodeReference[string]{
			Value: schemaRefNode.Value, KeyNode: schemaRefLabel, ValueNode: schemaRefLabel}
	}

	// handle example if set. (3.0)
	_, expLabel, expNode := utils.FindKeyNodeFull(ExampleLabel, root.Content)
	if expNode != nil {
		s.Example = low.NodeReference[any]{Value: ExtractExampleValue(expNode), KeyNode: expLabel, ValueNode: expNode}
	}

	// handle examples if set.(3.1)
	_, expArrLabel, expArrNode := utils.FindKeyNodeFullTop(ExamplesLabel, root.Content)
	if expArrNode != nil {
		if utils.IsNodeArray(expArrNode) {
			var examples []low.ValueReference[any]
			for i := range expArrNode.Content {
				examples = append(examples, low.ValueReference[any]{Value: ExtractExampleValue(expArrNode.Content[i]), ValueNode: expArrNode.Content[i]})
			}
			s.Examples = low.NodeReference[[]low.ValueReference[any]]{
				Value:     examples,
				ValueNode: expArrNode,
				KeyNode:   expArrLabel,
			}
		}
	}

	_, addPLabel, addPNode := utils.FindKeyNodeFullTop(AdditionalPropertiesLabel, root.Content)
	if addPNode != nil {
		if utils.IsNodeMap(addPNode) {
			// check if this is a reference, or an inline schema.
			isRef, _, _ := utils.IsNodeRefValue(addPNode)
			sp := &SchemaProxy{
				kn:  addPLabel,
				vn:  addPNode,
				idx: idx,
			}
			if isRef {
				sp.isReference = true
				_, vn := utils.FindKeyNodeTop("$ref", addPNode.Content)
				sp.referenceLookup = vn.Value
			}
			s.AdditionalProperties = low.NodeReference[any]{Value: sp, KeyNode: addPLabel, ValueNode: addPNode}
		}
		if utils.IsNodeBoolValue(addPNode) {
			b, _ := strconv.ParseBool(addPNode.Value)
			s.AdditionalProperties = low.NodeReference[any]{Value: b, KeyNode: addPLabel, ValueNode: addPNode}
		}
	}

	// handle discriminator if set.
	_, discLabel, discNode := utils.FindKeyNodeFullTop(DiscriminatorLabel, root.Content)
	if discNode != nil {
		var discriminator Discriminator
		_ = low.BuildModel(discNode, &discriminator)
		s.Discriminator = low.NodeReference[*Discriminator]{Value: &discriminator, KeyNode: discLabel, ValueNode: discNode}
	}

	// handle externalDocs if set.
	_, extDocLabel, extDocNode := utils.FindKeyNodeFullTop(ExternalDocsLabel, root.Content)
	if extDocNode != nil {
		var exDoc ExternalDoc
		_ = low.BuildModel(extDocNode, &exDoc)
		_ = exDoc.Build(extDocNode, idx) // throws no errors, can't check for one.
		s.ExternalDocs = low.NodeReference[*ExternalDoc]{Value: &exDoc, KeyNode: extDocLabel, ValueNode: extDocNode}
	}

	// handle xml if set.
	_, xmlLabel, xmlNode := utils.FindKeyNodeFullTop(XMLLabel, root.Content)
	if xmlNode != nil {
		var xml XML
		_ = low.BuildModel(xmlNode, &xml)
		// extract extensions if set.
		_ = xml.Build(xmlNode, idx) // returns no errors, can't check for one.
		s.XML = low.NodeReference[*XML]{Value: &xml, KeyNode: xmlLabel, ValueNode: xmlNode}
	}

	// for property, build in a new thread!
	bChan := make(chan schemaProxyBuildResult)

	var buildProperty = func(label *yaml.Node, value *yaml.Node, c chan schemaProxyBuildResult, isRef bool,
		refString string) {
		c <- schemaProxyBuildResult{
			k: low.KeyReference[string]{
				KeyNode: label,
				Value:   label.Value,
			},
			v: low.ValueReference[*SchemaProxy]{
				Value:     &SchemaProxy{kn: label, vn: value, idx: idx, isReference: isRef, referenceLookup: refString},
				ValueNode: value,
			},
		}
	}

	// handle properties
	_, propLabel, propsNode := utils.FindKeyNodeFullTop(PropertiesLabel, root.Content)
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
			isRef := false
			refString := ""
			if h, _, l := utils.IsNodeRefValue(prop); h {
				ref, _ := low.LocateRefNode(prop, idx)
				if ref != nil {
					isRef = true
					prop = ref
					refString = l
				} else {
					return fmt.Errorf("schema properties build failed: cannot find reference %s, line %d, col %d",
						prop.Content[1].Value, prop.Content[1].Column, prop.Content[1].Line)
				}
			}
			totalProps++
			go buildProperty(currentProp, prop, bChan, isRef, refString)
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

	var allOf, anyOf, oneOf, not, items, prefixItems []low.ValueReference[*SchemaProxy]

	_, allOfLabel, allOfValue := utils.FindKeyNodeFullTop(AllOfLabel, root.Content)
	_, anyOfLabel, anyOfValue := utils.FindKeyNodeFullTop(AnyOfLabel, root.Content)
	_, oneOfLabel, oneOfValue := utils.FindKeyNodeFullTop(OneOfLabel, root.Content)
	_, notLabel, notValue := utils.FindKeyNodeFullTop(NotLabel, root.Content)
	_, itemsLabel, itemsValue := utils.FindKeyNodeFullTop(ItemsLabel, root.Content)
	_, prefixItemsLabel, prefixItemsValue := utils.FindKeyNodeFullTop(PrefixItemsLabel, root.Content)

	errorChan := make(chan error)
	allOfChan := make(chan schemaProxyBuildResult)
	anyOfChan := make(chan schemaProxyBuildResult)
	oneOfChan := make(chan schemaProxyBuildResult)
	itemsChan := make(chan schemaProxyBuildResult)
	prefixItemsChan := make(chan schemaProxyBuildResult)
	notChan := make(chan schemaProxyBuildResult)

	totalBuilds := countSubSchemaItems(allOfValue) +
		countSubSchemaItems(anyOfValue) +
		countSubSchemaItems(oneOfValue) +
		countSubSchemaItems(notValue) +
		countSubSchemaItems(itemsValue) +
		countSubSchemaItems(prefixItemsValue)

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
	if prefixItemsValue != nil {
		go buildSchema(prefixItemsChan, prefixItemsLabel, prefixItemsValue, errorChan, idx)
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
		case r := <-prefixItemsChan:
			completeCount++
			prefixItems = append(prefixItems, r.v)
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
	if len(prefixItems) > 0 {
		s.PrefixItems = low.NodeReference[[]low.ValueReference[*SchemaProxy]]{
			Value:     prefixItems,
			KeyNode:   prefixItemsLabel,
			ValueNode: prefixItemsValue,
		}
	}
	return nil
}

// count the number of sub-schemas in a node.
func countSubSchemaItems(node *yaml.Node) int {
	if utils.IsNodeMap(node) {
		return 1
	}
	if utils.IsNodeArray(node) {
		return len(node.Content)
	}
	return 0
}

// schema build result container used for async building.
type schemaProxyBuildResult struct {
	k low.KeyReference[string]
	v low.ValueReference[*SchemaProxy]
}

// extract extensions from schema
func (s *Schema) extractExtensions(root *yaml.Node) {
	s.Extensions = low.ExtractExtensions(root)
}

// build out a child schema for parent schema.
func buildSchema(schemas chan schemaProxyBuildResult, labelNode, valueNode *yaml.Node, errors chan error, idx *index.SpecIndex) {

	if valueNode != nil {
		syncChan := make(chan *low.ValueReference[*SchemaProxy])

		// build out a SchemaProxy for every sub-schema.
		build := func(kn *yaml.Node, vn *yaml.Node, c chan *low.ValueReference[*SchemaProxy],
			isRef bool, refLocation string) {
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
			if isRef {
				sp.referenceLookup = refLocation
				sp.isReference = true
			}
			res := &low.ValueReference[*SchemaProxy]{
				Value:     sp,
				ValueNode: vn,
			}
			c <- res
		}

		isRef := false
		refLocation := ""
		if utils.IsNodeMap(valueNode) {
			h := false
			if h, _, refLocation = utils.IsNodeRefValue(valueNode); h {
				isRef = true
				ref, _ := low.LocateRefNode(valueNode, idx)
				if ref != nil {
					valueNode = ref
				} else {
					errors <- fmt.Errorf("build schema failed: reference cannot be found: %s, line %d, col %d",
						valueNode.Content[1].Value, valueNode.Content[1].Line, valueNode.Content[1].Column)
				}
			}

			// this only runs once, however to keep things consistent, it makes sense to use the same async method
			// that arrays will use.
			go build(labelNode, valueNode, syncChan, isRef, refLocation)
			select {
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
				isRef = false
				h := false
				if h, _, refLocation = utils.IsNodeRefValue(vn); h {
					isRef = true
					ref, _ := low.LocateRefNode(vn, idx)
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
				go build(vn, vn, syncChan, isRef, refLocation)
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

// ExtractSchema will return a pointer to a NodeReference that contains a *SchemaProxy if successful. The function
// will specifically look for a key node named 'schema' and extract the value mapped to that key. If the operation
// fails then no NodeReference is returned and an error is returned instead.
func ExtractSchema(root *yaml.Node, idx *index.SpecIndex) (*low.NodeReference[*SchemaProxy], error) {
	var schLabel, schNode *yaml.Node
	errStr := "schema build failed: reference '%s' cannot be found at line %d, col %d"

	isRef := false
	refLocation := ""
	if rf, rl, _ := utils.IsNodeRefValue(root); rf {
		// locate reference in index.
		isRef = true
		ref, _ := low.LocateRefNode(root, idx)
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
			h := false
			if h, _, refLocation = utils.IsNodeRefValue(schNode); h {
				isRef = true
				ref, _ := low.LocateRefNode(schNode, idx)
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
		schema := &SchemaProxy{kn: schLabel, vn: schNode, idx: idx, isReference: isRef, referenceLookup: refLocation}
		return &low.NodeReference[*SchemaProxy]{Value: schema, KeyNode: schLabel, ValueNode: schNode}, nil
	}
	return nil, nil
}
