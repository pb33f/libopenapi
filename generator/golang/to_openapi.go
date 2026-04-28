// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"sort"

	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

func (g *Generator) openapiFromIR(ir *SchemaIR) *highbase.SchemaProxy {
	if ir == nil {
		return highbase.CreateSchemaProxy(&highbase.Schema{})
	}
	if ir.Kind == KindRef {
		if ir.Nullable {
			return g.nullableRefProxy(ir)
		}
		if ir.DynamicRef {
			schema := &highbase.Schema{DynamicRef: ir.Ref}
			applySchemaFidelity(schema, ir)
			return highbase.CreateSchemaProxy(schema)
		}
		return highbase.CreateSchemaProxyRef(ir.Ref)
	}
	if ir.Name != "" && ir.Name != g.currentComponent && isComponentKind(ir.Kind) {
		if _, ok := g.componentNames[ir.Name]; ok {
			ref := "#/components/schemas/" + ir.Name
			if ir.Nullable {
				return nullableReferenceProxy(ref, false, ir)
			}
			return referenceProxy(ref, ir)
		}
	}
	if ir.ExactSource && ir.SourceSchema != nil && !ir.Nullable {
		return highbase.CreateSchemaProxy(ir.SourceSchema)
	}
	if ir.ExactSource && ir.SourceSchema != nil && ir.Nullable {
		schema := *ir.SourceSchema
		applyNativeNullability(&schema, ir)
		return highbase.CreateSchemaProxy(&schema)
	}
	schema := &highbase.Schema{
		Description: ir.Description,
		Title:       ir.Title,
		Format:      ir.Format,
		Enum:        ir.Enum,
		Const:       ir.Const,
		Extensions:  ir.Extensions,
	}
	applySchemaFidelity(schema, ir)
	switch ir.Kind {
	case KindAny:
	case KindString:
		schema.Type = []string{"string"}
	case KindInteger:
		schema.Type = []string{"integer"}
	case KindNumber:
		schema.Type = []string{"number"}
	case KindBoolean:
		schema.Type = []string{"boolean"}
	case KindArray:
		schema.Type = []string{"array"}
		if ir.Items != nil {
			schema.Items = &highbase.DynamicValue[*highbase.SchemaProxy, bool]{
				A: g.openapiFromIR(ir.Items),
			}
		}
		for _, item := range ir.PrefixItems {
			schema.PrefixItems = append(schema.PrefixItems, g.openapiFromIR(item))
		}
	case KindObject, KindAllOf:
		g.populateOpenAPIObject(schema, ir)
	case KindUnion:
		g.populateOpenAPIUnion(schema, ir)
	case KindEnum:
		switch enumShapeFor(ir.Enum).goType {
		case "int":
			schema.Type = []string{"integer"}
		case "float64":
			schema.Type = []string{"number"}
		case "bool":
			schema.Type = []string{"boolean"}
		case "any":
			schema.Type = nil
		default:
			schema.Type = []string{"string"}
		}
		schema.Enum = ir.Enum
	default:
		schema.Type = []string{"object"}
	}
	applyIRBooleans(schema, ir)
	applyNativeNullability(schema, ir)
	return highbase.CreateSchemaProxy(schema)
}

func applyIRBooleans(schema *highbase.Schema, ir *SchemaIR) {
	if schema == nil || ir == nil {
		return
	}
	if ir.ReadOnly {
		schema.ReadOnly = boolPtr(true)
	}
	if ir.WriteOnly {
		schema.WriteOnly = boolPtr(true)
	}
	if ir.Deprecated {
		schema.Deprecated = boolPtr(true)
	}
}

func (g *Generator) nullableRefProxy(ir *SchemaIR) *highbase.SchemaProxy {
	return nullableReferenceProxy(ir.Ref, ir.DynamicRef, ir)
}

func nullableReferenceProxy(target string, dynamic bool, ir *SchemaIR) *highbase.SchemaProxy {
	var ref *highbase.SchemaProxy
	if dynamic {
		refSchema := &highbase.Schema{DynamicRef: target}
		applySchemaFidelity(refSchema, ir)
		ref = highbase.CreateSchemaProxy(refSchema)
	} else {
		ref = referenceProxy(target, ir)
	}
	return highbase.CreateSchemaProxy(&highbase.Schema{
		AnyOf: []*highbase.SchemaProxy{
			ref,
			highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"null"}}),
		},
	})
}

func referenceProxy(target string, ir *SchemaIR) *highbase.SchemaProxy {
	if ir != nil && ir.FieldMetadata {
		return highbase.CreateSchemaProxyRefWithSchema(target, refSiblingSchema(ir))
	}
	return highbase.CreateSchemaProxyRef(target)
}

func refSiblingSchema(ir *SchemaIR) *highbase.Schema {
	schema := &highbase.Schema{
		Description: ir.Description,
		Title:       ir.Title,
		Format:      ir.Format,
		Enum:        ir.Enum,
		Const:       ir.Const,
		Extensions:  ir.Extensions,
	}
	applySchemaFidelity(schema, ir)
	applyIRBooleans(schema, ir)
	schema.Nullable = nil
	return schema
}

func (g *Generator) populateOpenAPIObject(schema *highbase.Schema, ir *SchemaIR) {
	schema.Type = []string{"object"}
	if ir.Properties != nil && ir.Properties.Len() > 0 {
		props := orderedmap.New[string, *highbase.SchemaProxy]()
		for name, prop := range ir.Properties.FromOldest() {
			props.Set(name, g.openapiFromIR(prop))
		}
		schema.Properties = props
	}
	if ir.PatternProperties != nil && ir.PatternProperties.Len() > 0 {
		patternProps := orderedmap.New[string, *highbase.SchemaProxy]()
		for name, prop := range ir.PatternProperties.FromOldest() {
			patternProps.Set(name, g.openapiFromIR(prop))
		}
		schema.PatternProperties = patternProps
	}
	if len(ir.Required) > 0 {
		required := make([]string, 0, len(ir.Required))
		for name := range ir.Required {
			required = append(required, name)
		}
		sort.Strings(required)
		schema.Required = required
	}
	if ir.AdditionalProperties != nil {
		schema.AdditionalProperties = &highbase.DynamicValue[*highbase.SchemaProxy, bool]{
			A: g.openapiFromIR(ir.AdditionalProperties),
		}
	} else if ir.AdditionalAllowed != nil {
		schema.AdditionalProperties = &highbase.DynamicValue[*highbase.SchemaProxy, bool]{
			N: 1,
			B: *ir.AdditionalAllowed,
		}
	}
}

func (g *Generator) populateOpenAPIUnion(schema *highbase.Schema, ir *SchemaIR) {
	if ir.Union == nil {
		return
	}
	if ir.Union.FromMultiType && ir.SourceSchema != nil && len(ir.SourceSchema.Type) > 0 {
		schema.Type = append([]string(nil), ir.SourceSchema.Type...)
		return
	}
	variants := make([]*highbase.SchemaProxy, 0, len(ir.Union.Variants))
	for _, variant := range ir.Union.Variants {
		variants = append(variants, g.openapiFromIR(variant))
	}
	if ir.Union.Kind == UnionAnyOf {
		schema.AnyOf = variants
	} else {
		schema.OneOf = variants
	}
	if ir.Union.Discriminator != nil {
		mapping := orderedmap.New[string, string]()
		keys := make([]string, 0, len(ir.Union.Discriminator.Mapping))
		for k := range ir.Union.Discriminator.Mapping {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			mapping.Set(k, ir.Union.Discriminator.Mapping[k])
		}
		schema.Discriminator = &highbase.Discriminator{
			PropertyName: ir.Union.Discriminator.PropertyName,
			Mapping:      mapping,
		}
	}
}

func applySchemaFidelity(schema *highbase.Schema, ir *SchemaIR) {
	if schema == nil || ir == nil || ir.SourceSchema == nil {
		return
	}
	src := ir.SourceSchema
	schema.Description = src.Description
	schema.Title = src.Title
	schema.Format = src.Format
	schema.Extensions = src.Extensions
	schema.SchemaTypeRef = src.SchemaTypeRef
	schema.ExclusiveMaximum = src.ExclusiveMaximum
	schema.ExclusiveMinimum = src.ExclusiveMinimum
	schema.Examples = src.Examples
	schema.Contains = src.Contains
	schema.MinContains = src.MinContains
	schema.MaxContains = src.MaxContains
	schema.If = src.If
	schema.Else = src.Else
	schema.Then = src.Then
	schema.DependentSchemas = src.DependentSchemas
	schema.DependentRequired = src.DependentRequired
	schema.PropertyNames = src.PropertyNames
	schema.UnevaluatedItems = src.UnevaluatedItems
	schema.UnevaluatedProperties = src.UnevaluatedProperties
	if src.Items != nil && src.Items.IsB() {
		schema.Items = src.Items
	}
	schema.Id = src.Id
	schema.Anchor = src.Anchor
	schema.DynamicAnchor = src.DynamicAnchor
	if src.DynamicRef != "" && !ir.DynamicRef {
		schema.DynamicRef = src.DynamicRef
	}
	schema.Comment = src.Comment
	schema.ContentSchema = src.ContentSchema
	schema.Vocabulary = src.Vocabulary
	schema.Not = src.Not
	schema.MultipleOf = src.MultipleOf
	schema.Maximum = src.Maximum
	schema.Minimum = src.Minimum
	schema.MaxLength = src.MaxLength
	schema.MinLength = src.MinLength
	schema.Pattern = src.Pattern
	schema.MaxItems = src.MaxItems
	schema.MinItems = src.MinItems
	schema.UniqueItems = src.UniqueItems
	schema.MaxProperties = src.MaxProperties
	schema.MinProperties = src.MinProperties
	schema.ContentEncoding = src.ContentEncoding
	schema.ContentMediaType = src.ContentMediaType
	schema.Default = src.Default
	schema.Example = src.Example
	schema.ReadOnly = src.ReadOnly
	schema.WriteOnly = src.WriteOnly
	schema.Deprecated = src.Deprecated
	schema.XML = src.XML
	schema.ExternalDocs = src.ExternalDocs
}

func applyNativeNullability(schema *highbase.Schema, ir *SchemaIR) {
	if schema == nil || ir == nil || !ir.Nullable {
		return
	}
	schema.Nullable = nil
	if schemaNeedsNullAlternative(schema) {
		original := *schema
		original.Nullable = nil
		*schema = highbase.Schema{
			AnyOf: []*highbase.SchemaProxy{
				highbase.CreateSchemaProxy(&original),
				highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"null"}}),
			},
		}
		return
	}
	if len(schema.Type) > 0 && !schemaTypeContains(schema.Type, "null") {
		schema.Type = append(append([]string(nil), schema.Type...), "null")
	}
	if len(schema.Enum) > 0 && !enumHasNull(schema.Enum) {
		schema.Enum = append(append([]*yaml.Node(nil), schema.Enum...), nullNode())
	}
}

func schemaNeedsNullAlternative(schema *highbase.Schema) bool {
	if schema == nil {
		return false
	}
	return len(schema.OneOf) > 0 ||
		len(schema.AnyOf) > 0 ||
		len(schema.AllOf) > 0 ||
		schema.Not != nil ||
		schema.Const != nil ||
		schema.DynamicRef != ""
}

func stringNode(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}

func nullNode() *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}
}

func schemaTypeContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
