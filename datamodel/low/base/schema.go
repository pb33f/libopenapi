package base

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
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
func (s *SchemaDynamicValue[A, B]) IsA() bool {
	return s.N == 0
}

// IsB will return true if the 'B' or right value is set (OpenAPI 3.1)
func (s *SchemaDynamicValue[A, B]) IsB() bool {
	return s.N == 1
}

// Hash will generate a stable hash of the SchemaDynamicValue
func (s *SchemaDynamicValue[A, B]) Hash() [32]byte {
	var hash string
	if s.IsA() {
		hash = low.GenerateHashString(s.A)
	} else {
		hash = low.GenerateHashString(s.B)
	}
	return sha256.Sum256([]byte(hash))
}

// Schema represents a JSON Schema that support Swagger, OpenAPI 3 and OpenAPI 3.1
//
// Until 3.1 OpenAPI had a strange relationship with JSON Schema. It's been a super-set/sub-set
// mix, which has been confusing. So, instead of building a bunch of different models, we have compressed
// all variations into a single model that makes it easy to support multiple spec types.
//
//   - v2 schema: https://swagger.io/specification/v2/#schemaObject
//   - v3 schema: https://swagger.io/specification/#schema-object
//   - v3.1 schema: https://spec.openapis.org/oas/v3.1.0#schema-object
type Schema struct {
	// Reference to the '$schema' dialect setting (3.1 only)
	SchemaTypeRef low.NodeReference[string]

	// In versions 2 and 3.0, this ExclusiveMaximum can only be a boolean.
	ExclusiveMaximum low.NodeReference[*SchemaDynamicValue[bool, float64]]

	// In versions 2 and 3.0, this ExclusiveMinimum can only be a boolean.
	ExclusiveMinimum low.NodeReference[*SchemaDynamicValue[bool, float64]]

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
	// in 3.1 Contains is used by arrays and points to a Schema.
	Contains    low.NodeReference[*SchemaProxy]
	MinContains low.NodeReference[int64]
	MaxContains low.NodeReference[int64]

	// items can be a schema in 2.0, 3.0 and 3.1 or a bool in 3.1
	Items low.NodeReference[*SchemaDynamicValue[*SchemaProxy, bool]]

	// 3.1 only
	If                    low.NodeReference[*SchemaProxy]
	Else                  low.NodeReference[*SchemaProxy]
	Then                  low.NodeReference[*SchemaProxy]
	DependentSchemas      low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*SchemaProxy]]
	PatternProperties     low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*SchemaProxy]]
	PropertyNames         low.NodeReference[*SchemaProxy]
	UnevaluatedItems      low.NodeReference[*SchemaProxy]
	UnevaluatedProperties low.NodeReference[*SchemaDynamicValue[*SchemaProxy, *bool]]
	Anchor                low.NodeReference[string]

	// Compatible with all versions
	Title                low.NodeReference[string]
	MultipleOf           low.NodeReference[float64]
	Maximum              low.NodeReference[float64]
	Minimum              low.NodeReference[float64]
	MaxLength            low.NodeReference[int64]
	MinLength            low.NodeReference[int64]
	Pattern              low.NodeReference[string]
	Format               low.NodeReference[string]
	MaxItems             low.NodeReference[int64]
	MinItems             low.NodeReference[int64]
	UniqueItems          low.NodeReference[bool]
	MaxProperties        low.NodeReference[int64]
	MinProperties        low.NodeReference[int64]
	Required             low.NodeReference[[]low.ValueReference[string]]
	Enum                 low.NodeReference[[]low.ValueReference[any]]
	Not                  low.NodeReference[*SchemaProxy]
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

	// Parent Proxy refers back to the low level SchemaProxy that is proxying this schema.
	ParentProxy *SchemaProxy
	*low.Reference
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

		// check type of properties, if we have a low level map, we need to hash the values in a repeatable
		// order.
		to := reflect.TypeOf(s.AdditionalProperties.Value)
		vo := reflect.ValueOf(s.AdditionalProperties.Value)
		var values []string
		switch to.Kind() {
		case reflect.Slice:
			for i := 0; i < vo.Len(); i++ {
				vn := vo.Index(i).Interface()

				if jh, ok := vn.(low.HasValueUnTyped); ok {
					vn = jh.GetValueUntyped()
					fg := reflect.TypeOf(vn)
					gf := reflect.ValueOf(vn)

					if fg.Kind() == reflect.Map {
						for _, ky := range gf.MapKeys() {
							hu := ky.Interface()
							values = append(values, fmt.Sprintf("%s:%s", hu, low.GenerateHashString(gf.MapIndex(ky).Interface())))
						}
						continue
					}
					values = append(values, fmt.Sprintf("%d:%s", i, low.GenerateHashString(vn)))
				}
			}
			sort.Strings(values)
			d = append(d, strings.Join(values, "||"))

		case reflect.Map:
			for _, k := range vo.MapKeys() {
				var x string
				var l int
				var v any
				// extract key
				if o, ok := k.Interface().(low.HasKeyNode); ok {
					x = o.GetKeyNode().Value
					l = o.GetKeyNode().Line
					v = vo.MapIndex(k).Interface().(low.HasValueNodeUntyped).GetValueNode().Value
				}
				values = append(values, fmt.Sprintf("%d:%s:%s", l, x, low.GenerateHashString(v)))
			}
			sort.Strings(values)
			d = append(d, strings.Join(values, "||"))
		default:
			d = append(d, low.GenerateHashString(s.AdditionalProperties.Value))
		}
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
	propKeys := make([]string, len(s.Properties.Value))
	z := 0
	for i := range s.Properties.Value {
		propKeys[z] = i.Value
		z++
	}
	sort.Strings(propKeys)
	for k := range propKeys {
		d = append(d, low.GenerateHashString(s.FindProperty(propKeys[k]).Value))
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

	if !s.Not.IsEmpty() {
		d = append(d, low.GenerateHashString(s.Not.Value))
	}

	// check if items is a schema or a bool.
	if !s.Items.IsEmpty() && s.Items.Value.IsA() {
		d = append(d, low.GenerateHashString(s.Items.Value.A))
	}
	if !s.Items.IsEmpty() && s.Items.Value.IsB() {
		d = append(d, fmt.Sprint(s.Items.Value.B))
	}
	// 3.1 only props
	if !s.If.IsEmpty() {
		d = append(d, low.GenerateHashString(s.If.Value))
	}
	if !s.Else.IsEmpty() {
		d = append(d, low.GenerateHashString(s.Else.Value))
	}
	if !s.Then.IsEmpty() {
		d = append(d, low.GenerateHashString(s.Then.Value))
	}
	if !s.PropertyNames.IsEmpty() {
		d = append(d, low.GenerateHashString(s.PropertyNames.Value))
	}
	if !s.UnevaluatedProperties.IsEmpty() {
		d = append(d, low.GenerateHashString(s.UnevaluatedProperties.Value))
	}
	if !s.UnevaluatedItems.IsEmpty() {
		d = append(d, low.GenerateHashString(s.UnevaluatedItems.Value))
	}
	if !s.Anchor.IsEmpty() {
		d = append(d, fmt.Sprint(s.Anchor.Value))
	}

	depSchemasKeys := make([]string, len(s.DependentSchemas.Value))
	z = 0
	for i := range s.DependentSchemas.Value {
		depSchemasKeys[z] = i.Value
		z++
	}
	sort.Strings(depSchemasKeys)
	for k := range depSchemasKeys {
		d = append(d, low.GenerateHashString(s.FindDependentSchema(depSchemasKeys[k]).Value))
	}

	patternPropsKeys := make([]string, len(s.PatternProperties.Value))
	z = 0
	for i := range s.PatternProperties.Value {
		patternPropsKeys[z] = i.Value
		z++
	}
	sort.Strings(patternPropsKeys)
	for k := range patternPropsKeys {
		d = append(d, low.GenerateHashString(s.FindPatternProperty(patternPropsKeys[k]).Value))
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

	// contains
	if !s.Contains.IsEmpty() {
		d = append(d, low.GenerateHashString(s.Contains.Value))
	}
	if !s.MinContains.IsEmpty() {
		d = append(d, fmt.Sprint(s.MinContains.Value))
	}
	if !s.MaxContains.IsEmpty() {
		d = append(d, fmt.Sprint(s.MaxContains.Value))
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

// FindDependentSchema will return a ValueReference pointer containing a SchemaProxy pointer
// from a dependent schema key name. if found (3.1+ only)
func (s *Schema) FindDependentSchema(name string) *low.ValueReference[*SchemaProxy] {
	return low.FindItemInMap[*SchemaProxy](name, s.DependentSchemas.Value)
}

// FindPatternProperty will return a ValueReference pointer containing a SchemaProxy pointer
// from a pattern property key name. if found (3.1+ only)
func (s *Schema) FindPatternProperty(name string) *low.ValueReference[*SchemaProxy] {
	return low.FindItemInMap[*SchemaProxy](name, s.PatternProperties.Value)
}

// GetExtensions returns all extensions for Schema
func (s *Schema) GetExtensions() map[low.KeyReference[string]]low.ValueReference[any] {
	return s.Extensions
}

// Build will perform a number of operations.
// Extraction of the following happens in this method:
//   - Extensions
//   - Type
//   - ExclusiveMinimum and ExclusiveMaximum
//   - Examples
//   - AdditionalProperties
//   - Discriminator
//   - ExternalDocs
//   - XML
//   - Properties
//   - AllOf, OneOf, AnyOf
//   - Not
//   - Items
//   - PrefixItems
//   - If
//   - Else
//   - Then
//   - DependentSchemas
//   - PatternProperties
//   - PropertyNames
//   - UnevaluatedItems
//   - UnevaluatedProperties
//   - Anchor
func (s *Schema) Build(root *yaml.Node, idx *index.SpecIndex) error {
	root = utils.NodeAlias(root)
	utils.CheckForMergeNodes(root)
	s.Reference = new(low.Reference)
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

	// Build model using possibly dereferenced root
	if err := low.BuildModel(root, s); err != nil {
		return err
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
			s.ExclusiveMinimum = low.NodeReference[*SchemaDynamicValue[bool, float64]]{
				KeyNode:   exMinLabel,
				ValueNode: exMinValue,
				Value:     &SchemaDynamicValue[bool, float64]{N: 0, A: val},
			}
		}
		if utils.IsNodeIntValue(exMinValue) {
			val, _ := strconv.ParseFloat(exMinValue.Value, 64)
			s.ExclusiveMinimum = low.NodeReference[*SchemaDynamicValue[bool, float64]]{
				KeyNode:   exMinLabel,
				ValueNode: exMinValue,
				Value:     &SchemaDynamicValue[bool, float64]{N: 1, B: val},
			}
		}
	}

	// determine exclusive maximum type, bool (3.0) or int (3.1)
	_, exMaxLabel, exMaxValue := utils.FindKeyNodeFullTop(ExclusiveMaximumLabel, root.Content)
	if exMaxValue != nil {
		if utils.IsNodeBoolValue(exMaxValue) {
			val, _ := strconv.ParseBool(exMaxValue.Value)
			s.ExclusiveMaximum = low.NodeReference[*SchemaDynamicValue[bool, float64]]{
				KeyNode:   exMaxLabel,
				ValueNode: exMaxValue,
				Value:     &SchemaDynamicValue[bool, float64]{N: 0, A: val},
			}
		}
		if utils.IsNodeIntValue(exMaxValue) {
			val, _ := strconv.ParseFloat(exMaxValue.Value, 64)
			s.ExclusiveMaximum = low.NodeReference[*SchemaDynamicValue[bool, float64]]{
				KeyNode:   exMaxLabel,
				ValueNode: exMaxValue,
				Value:     &SchemaDynamicValue[bool, float64]{N: 1, B: val},
			}
		}
	}

	// handle schema reference type if set. (3.1)
	_, schemaRefLabel, schemaRefNode := utils.FindKeyNodeFullTop(SchemaTypeLabel, root.Content)
	if schemaRefNode != nil {
		s.SchemaTypeRef = low.NodeReference[string]{
			Value: schemaRefNode.Value, KeyNode: schemaRefLabel, ValueNode: schemaRefLabel,
		}
	}

	// handle anchor if set. (3.1)
	_, anchorLabel, anchorNode := utils.FindKeyNodeFullTop(AnchorLabel, root.Content)
	if anchorNode != nil {
		s.Anchor = low.NodeReference[string]{
			Value: anchorNode.Value, KeyNode: anchorLabel, ValueNode: anchorLabel,
		}
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
		if utils.IsNodeMap(addPNode) || utils.IsNodeArray(addPNode) {
			// check if this is a reference, or an inline schema.
			isRef, _, _ := utils.IsNodeRefValue(addPNode)
			var sp *SchemaProxy
			// now check if this object has a 'type' if so, it's a schema, if not... it's a random
			// object, and we should treat it as a raw map.
			if _, v := utils.FindKeyNodeTop(TypeLabel, addPNode.Content); v != nil {
				sp = &SchemaProxy{
					kn:  addPLabel,
					vn:  addPNode,
					idx: idx,
				}
			}
			if isRef {
				_, vn := utils.FindKeyNodeTop("$ref", addPNode.Content)
				sp = &SchemaProxy{
					kn:              addPLabel,
					vn:              addPNode,
					idx:             idx,
					isReference:     true,
					referenceLookup: vn.Value,
				}
			}

			// if this is a reference, or a schema, we're done.
			if sp != nil {
				s.AdditionalProperties = low.NodeReference[any]{Value: sp, KeyNode: addPLabel, ValueNode: addPNode}
			} else {

				// if this is a map, collect all the keys and values.
				if utils.IsNodeMap(addPNode) {

					addProps := make(map[low.KeyReference[string]]low.ValueReference[any])
					var label string
					for g := range addPNode.Content {
						if g%2 == 0 {
							label = addPNode.Content[g].Value
							continue
						} else {
							addProps[low.KeyReference[string]{Value: label, KeyNode: addPNode.Content[g-1]}] =
								low.ValueReference[any]{Value: addPNode.Content[g].Value, ValueNode: addPNode.Content[g]}
						}
					}
					s.AdditionalProperties = low.NodeReference[any]{Value: addProps, KeyNode: addPLabel, ValueNode: addPNode}
				}

				// if the node is an array, extract everything into a trackable structure
				if utils.IsNodeArray(addPNode) {
					var addProps []low.ValueReference[any]

					// if this is an array or maps, encode the map items correctly.
					for i := range addPNode.Content {
						if utils.IsNodeMap(addPNode.Content[i]) {
							var prop map[string]any
							_ = addPNode.Content[i].Decode(&prop)
							addProps = append(addProps,
								low.ValueReference[any]{Value: prop, ValueNode: addPNode.Content[i]})
						} else {
							addProps = append(addProps,
								low.ValueReference[any]{Value: addPNode.Content[i].Value, ValueNode: addPNode.Content[i]})
						}
					}

					s.AdditionalProperties =
						low.NodeReference[any]{Value: addProps, KeyNode: addPLabel, ValueNode: addPNode}
				}
			}
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
		_ = exDoc.Build(extDocLabel, extDocNode, idx) // throws no errors, can't check for one.
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

	// handle properties
	props, err := buildPropertyMap(root, idx, PropertiesLabel)
	if err != nil {
		return err
	}
	if props != nil {
		s.Properties = *props
	}

	// handle dependent schemas
	props, err = buildPropertyMap(root, idx, DependentSchemasLabel)
	if err != nil {
		return err
	}
	if props != nil {
		s.DependentSchemas = *props
	}

	// handle pattern properties
	props, err = buildPropertyMap(root, idx, PatternPropertiesLabel)
	if err != nil {
		return err
	}
	if props != nil {
		s.PatternProperties = *props
	}

	// check items type for schema or bool (3.1 only)
	itemsIsBool := false
	itemsBoolValue := false
	_, itemsLabel, itemsValue := utils.FindKeyNodeFullTop(ItemsLabel, root.Content)
	if itemsValue != nil {
		if utils.IsNodeBoolValue(itemsValue) {
			itemsIsBool = true
			itemsBoolValue, _ = strconv.ParseBool(itemsValue.Value)
		}
	}
	if itemsIsBool {
		s.Items = low.NodeReference[*SchemaDynamicValue[*SchemaProxy, bool]]{
			Value: &SchemaDynamicValue[*SchemaProxy, bool]{
				B: itemsBoolValue,
				N: 1,
			},
			KeyNode:   itemsLabel,
			ValueNode: itemsValue,
		}
	}

	// check unevaluatedProperties type for schema or bool (3.1 only)
	unevalIsBool := false
	unevalBoolValue := true
	_, unevalLabel, unevalValue := utils.FindKeyNodeFullTop(UnevaluatedPropertiesLabel, root.Content)
	if unevalValue != nil {
		if utils.IsNodeBoolValue(unevalValue) {
			unevalIsBool = true
			unevalBoolValue, _ = strconv.ParseBool(unevalValue.Value)
		}
	}
	if unevalIsBool {
		s.UnevaluatedProperties = low.NodeReference[*SchemaDynamicValue[*SchemaProxy, *bool]]{
			Value: &SchemaDynamicValue[*SchemaProxy, *bool]{
				B: &unevalBoolValue,
				N: 1,
			},
			KeyNode:   unevalLabel,
			ValueNode: unevalValue,
		}
	}

	var allOf, anyOf, oneOf, prefixItems []low.ValueReference[*SchemaProxy]
	var items, not, contains, sif, selse, sthen, propertyNames, unevalItems, unevalProperties low.ValueReference[*SchemaProxy]

	_, allOfLabel, allOfValue := utils.FindKeyNodeFullTop(AllOfLabel, root.Content)
	_, anyOfLabel, anyOfValue := utils.FindKeyNodeFullTop(AnyOfLabel, root.Content)
	_, oneOfLabel, oneOfValue := utils.FindKeyNodeFullTop(OneOfLabel, root.Content)
	_, notLabel, notValue := utils.FindKeyNodeFullTop(NotLabel, root.Content)
	_, prefixItemsLabel, prefixItemsValue := utils.FindKeyNodeFullTop(PrefixItemsLabel, root.Content)
	_, containsLabel, containsValue := utils.FindKeyNodeFullTop(ContainsLabel, root.Content)
	_, sifLabel, sifValue := utils.FindKeyNodeFullTop(IfLabel, root.Content)
	_, selseLabel, selseValue := utils.FindKeyNodeFullTop(ElseLabel, root.Content)
	_, sthenLabel, sthenValue := utils.FindKeyNodeFullTop(ThenLabel, root.Content)
	_, propNamesLabel, propNamesValue := utils.FindKeyNodeFullTop(PropertyNamesLabel, root.Content)
	_, unevalItemsLabel, unevalItemsValue := utils.FindKeyNodeFullTop(UnevaluatedItemsLabel, root.Content)
	_, unevalPropsLabel, unevalPropsValue := utils.FindKeyNodeFullTop(UnevaluatedPropertiesLabel, root.Content)

	errorChan := make(chan error)
	allOfChan := make(chan schemaProxyBuildResult)
	anyOfChan := make(chan schemaProxyBuildResult)
	oneOfChan := make(chan schemaProxyBuildResult)
	itemsChan := make(chan schemaProxyBuildResult)
	prefixItemsChan := make(chan schemaProxyBuildResult)
	notChan := make(chan schemaProxyBuildResult)
	containsChan := make(chan schemaProxyBuildResult)
	ifChan := make(chan schemaProxyBuildResult)
	elseChan := make(chan schemaProxyBuildResult)
	thenChan := make(chan schemaProxyBuildResult)
	propNamesChan := make(chan schemaProxyBuildResult)
	unevalItemsChan := make(chan schemaProxyBuildResult)
	unevalPropsChan := make(chan schemaProxyBuildResult)

	totalBuilds := countSubSchemaItems(allOfValue) +
		countSubSchemaItems(anyOfValue) +
		countSubSchemaItems(oneOfValue) +
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
	if prefixItemsValue != nil {
		go buildSchema(prefixItemsChan, prefixItemsLabel, prefixItemsValue, errorChan, idx)
	}
	if notValue != nil {
		totalBuilds++
		go buildSchema(notChan, notLabel, notValue, errorChan, idx)
	}
	if containsValue != nil {
		totalBuilds++
		go buildSchema(containsChan, containsLabel, containsValue, errorChan, idx)
	}
	if !itemsIsBool && itemsValue != nil {
		totalBuilds++
		go buildSchema(itemsChan, itemsLabel, itemsValue, errorChan, idx)
	}
	if sifValue != nil {
		totalBuilds++
		go buildSchema(ifChan, sifLabel, sifValue, errorChan, idx)
	}
	if selseValue != nil {
		totalBuilds++
		go buildSchema(elseChan, selseLabel, selseValue, errorChan, idx)
	}
	if sthenValue != nil {
		totalBuilds++
		go buildSchema(thenChan, sthenLabel, sthenValue, errorChan, idx)
	}
	if propNamesValue != nil {
		totalBuilds++
		go buildSchema(propNamesChan, propNamesLabel, propNamesValue, errorChan, idx)
	}
	if unevalItemsValue != nil {
		totalBuilds++
		go buildSchema(unevalItemsChan, unevalItemsLabel, unevalItemsValue, errorChan, idx)
	}
	if !unevalIsBool && unevalPropsValue != nil {
		totalBuilds++
		go buildSchema(unevalPropsChan, unevalPropsLabel, unevalPropsValue, errorChan, idx)
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
			items = r.v
		case r := <-prefixItemsChan:
			completeCount++
			prefixItems = append(prefixItems, r.v)
		case r := <-notChan:
			completeCount++
			not = r.v
		case r := <-containsChan:
			completeCount++
			contains = r.v
		case r := <-ifChan:
			completeCount++
			sif = r.v
		case r := <-elseChan:
			completeCount++
			selse = r.v
		case r := <-thenChan:
			completeCount++
			sthen = r.v
		case r := <-propNamesChan:
			completeCount++
			propertyNames = r.v
		case r := <-unevalItemsChan:
			completeCount++
			unevalItems = r.v
		case r := <-unevalPropsChan:
			completeCount++
			unevalProperties = r.v
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
	if !not.IsEmpty() {
		s.Not = low.NodeReference[*SchemaProxy]{
			Value:     not.Value,
			KeyNode:   notLabel,
			ValueNode: notValue,
		}
	}
	if !itemsIsBool && !items.IsEmpty() {
		s.Items = low.NodeReference[*SchemaDynamicValue[*SchemaProxy, bool]]{
			Value: &SchemaDynamicValue[*SchemaProxy, bool]{
				A: items.Value,
			},
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
	if !contains.IsEmpty() {
		s.Contains = low.NodeReference[*SchemaProxy]{
			Value:     contains.Value,
			KeyNode:   containsLabel,
			ValueNode: containsValue,
		}
	}
	if !sif.IsEmpty() {
		s.If = low.NodeReference[*SchemaProxy]{
			Value:     sif.Value,
			KeyNode:   sifLabel,
			ValueNode: sifValue,
		}
	}
	if !selse.IsEmpty() {
		s.Else = low.NodeReference[*SchemaProxy]{
			Value:     selse.Value,
			KeyNode:   selseLabel,
			ValueNode: selseValue,
		}
	}
	if !sthen.IsEmpty() {
		s.Then = low.NodeReference[*SchemaProxy]{
			Value:     sthen.Value,
			KeyNode:   sthenLabel,
			ValueNode: sthenValue,
		}
	}
	if !propertyNames.IsEmpty() {
		s.PropertyNames = low.NodeReference[*SchemaProxy]{
			Value:     propertyNames.Value,
			KeyNode:   propNamesLabel,
			ValueNode: propNamesValue,
		}
	}
	if !unevalItems.IsEmpty() {
		s.UnevaluatedItems = low.NodeReference[*SchemaProxy]{
			Value:     unevalItems.Value,
			KeyNode:   unevalItemsLabel,
			ValueNode: unevalItemsValue,
		}
	}
	if !unevalIsBool && !unevalProperties.IsEmpty() {
		s.UnevaluatedProperties = low.NodeReference[*SchemaDynamicValue[*SchemaProxy, *bool]]{
			Value: &SchemaDynamicValue[*SchemaProxy, *bool]{
				A: unevalProperties.Value,
			},
			KeyNode:   unevalPropsLabel,
			ValueNode: unevalPropsValue,
		}
	}
	return nil
}

func buildPropertyMap(root *yaml.Node, idx *index.SpecIndex, label string) (*low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*SchemaProxy]], error) {
	// for property, build in a new thread!
	bChan := make(chan schemaProxyBuildResult)

	buildProperty := func(label *yaml.Node, value *yaml.Node, c chan schemaProxyBuildResult, isRef bool,
		refString string,
	) {
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

	_, propLabel, propsNode := utils.FindKeyNodeFullTop(label, root.Content)
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
					return nil, fmt.Errorf("schema properties build failed: cannot find reference %s, line %d, col %d",
						prop.Content[1].Value, prop.Content[1].Line, prop.Content[1].Column)
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
		return &low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*SchemaProxy]]{
			Value:     propertyMap,
			KeyNode:   propLabel,
			ValueNode: propsNode,
		}, nil
	}
	return nil, nil
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
		type buildResult struct {
			res *low.ValueReference[*SchemaProxy]
			idx int
		}

		syncChan := make(chan buildResult)

		// build out a SchemaProxy for every sub-schema.
		build := func(kn *yaml.Node, vn *yaml.Node, schemaIdx int, c chan buildResult,
			isRef bool, refLocation string,
		) {
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
			c <- buildResult{
				res: res,
				idx: schemaIdx,
			}
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
			go build(labelNode, valueNode, -1, syncChan, isRef, refLocation)
			select {
			case r := <-syncChan:
				schemas <- schemaProxyBuildResult{
					k: low.KeyReference[string]{
						KeyNode: labelNode,
						Value:   labelNode.Value,
					},
					v: *r.res,
				}
			}
		}
		if utils.IsNodeArray(valueNode) {
			refBuilds := 0
			results := make([]*low.ValueReference[*SchemaProxy], len(valueNode.Content))

			for i, vn := range valueNode.Content {
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
				go build(vn, vn, i, syncChan, isRef, refLocation)
			}

			completedBuilds := 0
			for completedBuilds < refBuilds {
				select {
				case res := <-syncChan:
					completedBuilds++
					results[res.idx] = res.res
				}
			}

			for _, r := range results {
				schemas <- schemaProxyBuildResult{
					k: low.KeyReference[string]{
						KeyNode: labelNode,
						Value:   labelNode.Value,
					},
					v: *r,
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
		return &low.NodeReference[*SchemaProxy]{Value: schema, KeyNode: schLabel, ValueNode: schNode, ReferenceNode: isRef,
			Reference: refLocation}, nil
	}
	return nil, nil
}
