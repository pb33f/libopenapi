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
		return highbase.CreateSchemaProxyRef(ir.Ref)
	}
	if ir.Name != "" && ir.Name != g.currentComponent && isComponentKind(ir.Kind) {
		if _, ok := g.componentNames[ir.Name]; ok {
			return highbase.CreateSchemaProxyRef("#/components/schemas/" + ir.Name)
		}
	}
	schema := &highbase.Schema{
		Description: ir.Description,
		Format:      ir.Format,
		Enum:        ir.Enum,
		Const:       ir.Const,
		Extensions:  ir.Extensions,
	}
	if ir.Nullable {
		nullable := true
		schema.Nullable = &nullable
	}
	switch ir.Kind {
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
		schema.Type = []string{"string"}
		schema.Enum = ir.Enum
	default:
		schema.Type = []string{"object"}
	}
	return highbase.CreateSchemaProxy(schema)
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

func stringNode(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}
