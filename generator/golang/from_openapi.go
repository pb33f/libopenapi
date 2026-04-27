// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
)

func (g *Generator) irFromOpenAPI(name string, proxy *highbase.SchemaProxy, path string) (*SchemaIR, error) {
	if proxy == nil {
		return nil, wrapPath(ErrNilSchema, path)
	}
	if cached := g.openapiCache[proxy]; cached != nil {
		return cached, nil
	}
	if proxy.IsReference() {
		ref := proxy.GetReference()
		ir := &SchemaIR{
			Name: g.publicName(refName(ref)),
			Ref:  ref,
			Kind: KindRef,
		}
		g.openapiCache[proxy] = ir
		return ir, nil
	}
	schema := proxy.Schema()
	if schema == nil {
		return nil, wrapPath(ErrNilSchema, path)
	}
	ir := g.irFromSchema(name, schema, path)
	g.openapiCache[proxy] = ir
	return ir, nil
}

func (g *Generator) irFromSchema(name string, schema *highbase.Schema, path string) *SchemaIR {
	ir := &SchemaIR{
		Name:        g.publicName(name),
		Format:      schema.Format,
		Description: schema.Description,
		Title:       schema.Title,
		Required:    make(map[string]struct{}),
		Properties:  nil,
		Enum:        schema.Enum,
		Const:       schema.Const,
		Extensions:  schema.Extensions,
	}
	if schema.Nullable != nil && *schema.Nullable {
		ir.Nullable = true
	}
	if schema.ReadOnly != nil && *schema.ReadOnly {
		ir.ReadOnly = true
		ir.Comments = append(ir.Comments, "readOnly")
	}
	if schema.WriteOnly != nil && *schema.WriteOnly {
		ir.WriteOnly = true
		ir.Comments = append(ir.Comments, "writeOnly")
	}
	if schema.Deprecated != nil && *schema.Deprecated {
		ir.Deprecated = true
		ir.Comments = append(ir.Comments, "Deprecated.")
	}
	if schema.Default != nil {
		ir.Comments = append(ir.Comments, "default value is defined in the OpenAPI schema")
	}
	if schema.Example != nil || len(schema.Examples) > 0 {
		ir.Comments = append(ir.Comments, "example value is defined in the OpenAPI schema")
	}
	g.collectShapeDiagnostics(path, schema)
	for _, t := range schema.Type {
		if t == "null" {
			ir.Nullable = true
		}
	}
	for _, required := range schema.Required {
		ir.Required[required] = struct{}{}
	}

	if len(schema.AllOf) > 0 {
		ir.Kind = KindAllOf
		for i, child := range schema.AllOf {
			childIR, err := g.irFromOpenAPI(name+"AllOf"+intString(i+1), child, path+".allOf")
			if err == nil {
				ir.AllOf = append(ir.AllOf, childIR)
			}
		}
		g.mergeAllOf(ir)
		return ir
	}

	if len(schema.OneOf) > 0 || len(schema.AnyOf) > 0 {
		g.populateUnion(ir, schema, path)
		return ir
	}

	if len(schema.Enum) > 0 {
		ir.Kind = KindEnum
		return ir
	}

	g.populateSchemaShape(ir, schema, path)
	return ir
}

func (g *Generator) populateSchemaShape(ir *SchemaIR, schema *highbase.Schema, path string) {
	switch primaryType(schema.Type) {
	case "string":
		ir.Kind = KindString
	case "integer":
		ir.Kind = KindInteger
	case "number":
		ir.Kind = KindNumber
	case "boolean":
		ir.Kind = KindBoolean
	case "array":
		ir.Kind = KindArray
		if schema.Items != nil && schema.Items.IsA() {
			item, err := g.irFromOpenAPI(ir.Name+"Item", schema.Items.A, path+".items")
			if err == nil {
				ir.Items = item
			}
		}
		for i, prefixItem := range schema.PrefixItems {
			child, err := g.irFromOpenAPI(ir.Name+"Tuple"+intString(i+1), prefixItem, path+".prefixItems")
			if err == nil {
				ir.PrefixItems = append(ir.PrefixItems, child)
			}
		}
		if len(ir.PrefixItems) > 0 {
			g.addDiagnostic(path, "prefixItems tuple shape rendered as []any")
		}
	case "object":
		g.populateObject(ir, schema, path)
	default:
		if schema.Properties != nil && schema.Properties.Len() > 0 {
			g.populateObject(ir, schema, path)
			return
		}
		if schema.AdditionalProperties != nil {
			g.populateObject(ir, schema, path)
			return
		}
		ir.Kind = KindAny
	}
}

func (g *Generator) populateObject(ir *SchemaIR, schema *highbase.Schema, path string) {
	ir.Kind = KindObject
	ir.Properties = orderedProperties()
	if schema.Properties != nil {
		for propName, propSchema := range schema.Properties.FromOldest() {
			childName := ir.Name + g.publicName(propName)
			childIR, err := g.irFromOpenAPI(childName, propSchema, path+"."+propName)
			if err == nil {
				ir.Properties.Set(propName, childIR)
			}
		}
	}
	if schema.PatternProperties != nil && schema.PatternProperties.Len() > 0 {
		ir.PatternProperties = orderedProperties()
		for pattern, propSchema := range schema.PatternProperties.FromOldest() {
			childIR, err := g.irFromOpenAPI(ir.Name+"PatternProperty", propSchema, path+".patternProperties")
			if err == nil {
				ir.PatternProperties.Set(pattern, childIR)
			}
		}
		g.addDiagnostic(path, "patternProperties cannot be represented directly as Go struct fields")
	}
	if schema.AdditionalProperties != nil {
		switch {
		case schema.AdditionalProperties.IsA():
			childIR, err := g.irFromOpenAPI(ir.Name+"AdditionalProperty", schema.AdditionalProperties.A, path+".additionalProperties")
			if err == nil {
				ir.AdditionalProperties = childIR
			}
		case schema.AdditionalProperties.IsB():
			allowed := schema.AdditionalProperties.B
			ir.AdditionalAllowed = &allowed
		}
	}
}

func (g *Generator) collectShapeDiagnostics(path string, schema *highbase.Schema) {
	if schema == nil {
		return
	}
	if schema.PropertyNames != nil {
		g.addDiagnostic(path, "propertyNames is validation-only and was not rendered into Go model shape")
	}
	if schema.DependentSchemas != nil && schema.DependentSchemas.Len() > 0 {
		g.addDiagnostic(path, "dependentSchemas is validation-only and was not rendered into Go model shape")
	}
	if schema.DependentRequired != nil && schema.DependentRequired.Len() > 0 {
		g.addDiagnostic(path, "dependentRequired is validation-only and was not rendered into Go model shape")
	}
	if schema.If != nil || schema.Then != nil || schema.Else != nil {
		g.addDiagnostic(path, "if/then/else is validation-only and was not rendered into Go model shape")
	}
	if schema.Not != nil {
		g.addDiagnostic(path, "not is validation-only and was not rendered into Go model shape")
	}
	if schema.UnevaluatedProperties != nil && schema.UnevaluatedProperties.IsB() && !schema.UnevaluatedProperties.B {
		g.addDiagnostic(path, "unevaluatedProperties: false prevents extra JSON fields but has no direct Go field")
	}
}

func (g *Generator) populateUnion(ir *SchemaIR, schema *highbase.Schema, path string) {
	kind := UnionOneOf
	children := schema.OneOf
	if len(children) == 0 {
		kind = UnionAnyOf
		children = schema.AnyOf
	}
	variants := make([]*SchemaIR, 0, len(children))
	for i, child := range children {
		variantName := ir.Name + "Variant" + intString(i+1)
		if built, err := child.BuildSchema(); err == nil && built != nil && built.Title != "" {
			variantName = ir.Name + built.Title
		}
		childIR, err := g.irFromOpenAPI(variantName, child, path+".union")
		if err == nil {
			variants = append(variants, childIR)
		}
	}
	nonNull := nonNullVariants(variants)
	if len(nonNull) == 1 && len(nonNull) != len(variants) {
		*ir = *nonNull[0]
		ir.Nullable = true
		return
	}
	ir.Kind = KindUnion
	ir.Union = &UnionIR{Kind: kind, Variants: variants, Strategy: UnionRawMessage}
	if kind != UnionOneOf {
		return
	}
	if schema.Discriminator != nil && schema.Discriminator.PropertyName != "" {
		ir.Union.Discriminator = discriminatorFromSchema(schema)
		ir.Union.Strategy = UnionDiscriminator
		return
	}
	if disc := inferConstDiscriminator(variants); disc != nil {
		ir.Union.Discriminator = disc
		if !disc.Optional || g.optionalConstDiscriminatorUnions {
			ir.Union.Strategy = UnionDiscriminator
			return
		}
		g.addDiagnostic(path, "oneOf has a shared const discriminator property, but it is optional; using json.RawMessage")
	}
}

func (g *Generator) mergeAllOf(ir *SchemaIR) {
	merged := newObjectIR(ir.Name)
	merged.Description = ir.Description
	for _, child := range ir.AllOf {
		if child == nil {
			continue
		}
		if child.Kind == KindRef {
			merged.AllOf = append(merged.AllOf, child)
			continue
		}
		if child.Kind == KindObject && child.Properties != nil {
			for name, prop := range child.Properties.FromOldest() {
				merged.Properties.Set(name, prop)
			}
			for req := range child.Required {
				merged.Required[req] = struct{}{}
			}
			continue
		}
		merged.AllOf = append(merged.AllOf, child)
	}
	*ir = *merged
}

func orderedProperties() *orderedmap.Map[string, *SchemaIR] {
	return orderedmap.New[string, *SchemaIR]()
}

func primaryType(types []string) string {
	for _, t := range types {
		if t != "null" {
			return t
		}
	}
	return ""
}

func nonNullVariants(variants []*SchemaIR) []*SchemaIR {
	var out []*SchemaIR
	for _, variant := range variants {
		if variant == nil {
			continue
		}
		if variant.Kind == KindAny && variant.Nullable {
			continue
		}
		out = append(out, variant)
	}
	return out
}

func discriminatorFromSchema(schema *highbase.Schema) *Discriminator {
	disc := &Discriminator{
		PropertyName: schema.Discriminator.PropertyName,
		Mapping:      make(map[string]string),
	}
	if schema.Discriminator.Mapping != nil {
		for k, v := range schema.Discriminator.Mapping.FromOldest() {
			disc.Mapping[k] = v
		}
	}
	return disc
}

func inferConstDiscriminator(variants []*SchemaIR) *Discriminator {
	if len(variants) == 0 {
		return nil
	}
	type candidate struct {
		values   map[string]string
		optional bool
	}
	candidates := make(map[string]*candidate)
	for i, variant := range variants {
		if variant == nil || variant.Properties == nil {
			return nil
		}
		seen := make(map[string]struct{})
		for propName, prop := range variant.Properties.FromOldest() {
			if prop == nil || prop.Const == nil {
				continue
			}
			var value string
			if err := prop.Const.Decode(&value); err != nil || value == "" {
				continue
			}
			seen[propName] = struct{}{}
			c := candidates[propName]
			if c == nil {
				c = &candidate{values: make(map[string]string)}
				candidates[propName] = c
			}
			if _, exists := c.values[value]; exists {
				return nil
			}
			c.values[value] = variant.Name
			if !isRequired(variant, propName) {
				c.optional = true
			}
		}
		for propName := range candidates {
			if _, ok := seen[propName]; !ok && i > 0 {
				delete(candidates, propName)
			}
		}
	}
	for propName, c := range candidates {
		if len(c.values) == len(variants) {
			return &Discriminator{PropertyName: propName, Mapping: c.values, Optional: c.optional}
		}
	}
	return nil
}
