// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"strings"

	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
)

func (g *Generator) irFromOpenAPI(name string, proxy *highbase.SchemaProxy, path string) (*SchemaIR, error) {
	return g.irFromOpenAPIName(name, false, proxy, path)
}

func (g *Generator) irFromOpenAPIName(name string, nameResolved bool, proxy *highbase.SchemaProxy, path string) (*SchemaIR, error) {
	if proxy == nil {
		return nil, wrapPath(ErrNilSchema, path)
	}
	if cached := g.openapiCache[proxy]; cached != nil {
		return cached, nil
	}
	if proxy.IsReference() {
		ref := proxy.GetReference()
		typeName := g.refTypeName(ref)
		if !strings.HasPrefix(ref, "#/") {
			g.addDiagnostic(DiagnosticExternalReference, path, "external reference rendered as Go type "+typeName)
		}
		ir := &SchemaIR{
			Name: typeName,
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
	ir := g.irFromSchema(name, nameResolved, schema, path)
	g.openapiCache[proxy] = ir
	return ir, nil
}

func (g *Generator) irFromSchema(name string, nameResolved bool, schema *highbase.Schema, path string) *SchemaIR {
	g.collectShapeDiagnostics(path, schema)
	if schema.DynamicRef != "" && schemaHasOnlyDynamicRefShape(schema) {
		nullable := schema.Nullable != nil && *schema.Nullable
		return &SchemaIR{
			Name:         g.refTypeName(schema.DynamicRef),
			Ref:          schema.DynamicRef,
			Kind:         KindRef,
			DynamicRef:   true,
			Nullable:     nullable,
			Format:       schema.Format,
			Description:  schema.Description,
			Title:        schema.Title,
			Extensions:   schema.Extensions,
			SourceSchema: schema,
		}
	}
	typeName := g.openapiSchemaTypeName(name, nameResolved, schema, path)
	ir := &SchemaIR{
		Name:         typeName,
		Format:       schema.Format,
		Description:  schema.Description,
		Title:        schema.Title,
		Required:     make(map[string]struct{}),
		Properties:   nil,
		Enum:         schema.Enum,
		Const:        schema.Const,
		Extensions:   schema.Extensions,
		SourceSchema: schema,
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
	for _, t := range schema.Type {
		if t == "null" {
			ir.Nullable = true
		}
	}
	if schema.Const != nil && nodeIsNull(schema.Const) {
		ir.Nullable = true
	}
	for _, required := range schema.Required {
		ir.Required[required] = struct{}{}
	}

	if len(schema.AllOf) > 0 {
		ir.Kind = KindAllOf
		for i, child := range schema.AllOf {
			childIR, err := g.irFromOpenAPIName(g.nestedTypeName(ir.Name, "AllOf"+intString(i+1)), true, child, path+".allOf")
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
		if enumHasNull(schema.Enum) {
			ir.Nullable = true
			g.addDiagnostic(DiagnosticNullEnum, path, "enum contains null; generated model uses nullable Go shape for non-null enum values")
		}
		if enumIsMixed(schema.Enum) {
			g.addDiagnostic(DiagnosticMixedEnum, path, "mixed-type enum rendered as any because Go constants require one scalar base type")
		}
		ir.Kind = KindEnum
		return ir
	}

	if nonNull := nonNullTypes(schema.Type); len(nonNull) > 1 {
		g.populateMultiTypeUnion(ir, nonNull, path)
		return ir
	}

	g.populateSchemaShape(ir, schema, path)
	return ir
}

func (g *Generator) openapiSchemaTypeName(name string, nameResolved bool, schema *highbase.Schema, path string) string {
	if !nameResolved && g.componentTypeNames != nil {
		return g.componentTypeName(name)
	}
	candidate := name
	if !nameResolved {
		candidate = g.publicName(name)
	}
	if !nameResolved || schemaDeclaresType(schema) {
		return g.resolveTypeName(path, candidate, path)
	}
	return candidate
}

func schemaDeclaresType(schema *highbase.Schema) bool {
	if schema == nil {
		return false
	}
	if len(schema.AllOf) > 0 || len(schema.OneOf) > 0 || len(schema.AnyOf) > 0 || len(schema.Enum) > 0 {
		return true
	}
	if len(nonNullTypes(schema.Type)) > 1 {
		return true
	}
	typ, _, _ := primaryTypeForSchema(schema)
	if typ != "object" {
		return false
	}
	return (schema.Properties != nil && schema.Properties.Len() > 0) ||
		schema.AdditionalProperties != nil ||
		(schema.PatternProperties != nil && schema.PatternProperties.Len() > 0)
}

func (g *Generator) populateSchemaShape(ir *SchemaIR, schema *highbase.Schema, path string) {
	typ, implicit, ambiguous := primaryTypeForSchema(schema)
	if implicit {
		g.addDiagnostic(DiagnosticImplicitType, path, "schema type inferred from JSON Schema keywords")
	}
	if ambiguous {
		g.addDiagnostic(DiagnosticImplicitType, path, "schema has validation keywords for multiple JSON types; generated model uses any")
	}
	switch typ {
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
			item, err := g.irFromOpenAPIName(g.nestedTypeName(ir.Name, "Item"), true, schema.Items.A, path+".items")
			if err == nil {
				ir.Items = item
			}
		} else if schema.Items != nil && schema.Items.IsB() && !schema.Items.B {
			g.addDiagnostic(DiagnosticBooleanItems, path, "items: false constrains array length but generated Go model uses []any")
		}
		for i, prefixItem := range schema.PrefixItems {
			child, err := g.irFromOpenAPIName(g.nestedTypeName(ir.Name, "Tuple"+intString(i+1)), true, prefixItem, path+".prefixItems")
			if err == nil {
				ir.PrefixItems = append(ir.PrefixItems, child)
			}
		}
		if len(ir.PrefixItems) > 0 {
			g.addDiagnostic(DiagnosticPrefixItems, path, "prefixItems tuple shape rendered as []any")
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

func (g *Generator) populateMultiTypeUnion(ir *SchemaIR, types []string, path string) {
	g.addDiagnostic(DiagnosticMultiTypeSchema, path, "multi-type JSON Schema rendered as json.RawMessage union")
	ir.Kind = KindUnion
	ir.Union = &UnionIR{Kind: UnionAnyOf, Strategy: UnionRawMessage, FromMultiType: true}
	for _, typ := range types {
		ir.Union.Variants = append(ir.Union.Variants, &SchemaIR{
			Name: g.nestedTypeName(ir.Name, typ),
			Kind: kindForJSONType(typ),
		})
	}
}

func (g *Generator) populateObject(ir *SchemaIR, schema *highbase.Schema, path string) {
	ir.Kind = KindObject
	ir.Properties = orderedProperties()
	if schema.Properties != nil {
		for propName, propSchema := range schema.Properties.FromOldest() {
			childIR, err := g.irFromOpenAPIName(g.nestedTypeName(ir.Name, propName), true, propSchema, path+"."+propName)
			if err == nil {
				ir.Properties.Set(propName, childIR)
			}
		}
	}
	if schema.PatternProperties != nil && schema.PatternProperties.Len() > 0 {
		ir.PatternProperties = orderedProperties()
		for pattern, propSchema := range schema.PatternProperties.FromOldest() {
			childIR, err := g.irFromOpenAPIName(g.nestedTypeName(ir.Name, "PatternProperty"), true, propSchema, path+".patternProperties")
			if err == nil {
				ir.PatternProperties.Set(pattern, childIR)
			}
		}
		g.addDiagnostic(DiagnosticPatternProperties, path, "patternProperties cannot be represented directly as Go struct fields")
	}
	if schema.AdditionalProperties != nil {
		switch {
		case schema.AdditionalProperties.IsA():
			childIR, err := g.irFromOpenAPIName(g.nestedTypeName(ir.Name, "AdditionalProperty"), true, schema.AdditionalProperties.A, path+".additionalProperties")
			if err == nil {
				ir.AdditionalProperties = childIR
			}
		case schema.AdditionalProperties.IsB():
			allowed := schema.AdditionalProperties.B
			ir.AdditionalAllowed = &allowed
			if !allowed {
				g.addDiagnostic(DiagnosticAdditionalPropertiesFalse, path, "additionalProperties: false prevents extra JSON fields but generated Go models do not reject unknown fields")
			}
		}
	}
}

func (g *Generator) collectShapeDiagnostics(path string, schema *highbase.Schema) {
	if schema == nil {
		return
	}
	if schema.PropertyNames != nil {
		g.addDiagnostic(DiagnosticPropertyNames, path, "propertyNames is validation-only and was not rendered into Go model shape")
	}
	if hasSchemaMetadata(schema) {
		g.addDiagnostic(DiagnosticSchemaMetadata, path, "JSON Schema metadata keywords are preserved in the source schema but do not change generated Go model shape")
	}
	if schema.DynamicRef != "" {
		g.addDiagnostic(DiagnosticDynamicReference, path, "$dynamicRef rendered as a Go reference name without dynamic resolution behavior")
	}
	if schema.ContentSchema != nil {
		g.addDiagnostic(DiagnosticContentSchema, path, "contentSchema describes decoded string content and was not rendered into Go model shape")
	}
	if schema.Contains != nil || schema.MinContains != nil || schema.MaxContains != nil {
		g.addDiagnostic(DiagnosticArrayContains, path, "contains/minContains/maxContains are validation-only and were not rendered into Go model shape")
	}
	if schema.UnevaluatedItems != nil {
		g.addDiagnostic(DiagnosticUnevaluatedItems, path, "unevaluatedItems is validation-only and was not rendered into Go model shape")
	}
	if schema.DependentSchemas != nil && schema.DependentSchemas.Len() > 0 {
		g.addDiagnostic(DiagnosticDependentSchemas, path, "dependentSchemas is validation-only and was not rendered into Go model shape")
	}
	if schema.DependentRequired != nil && schema.DependentRequired.Len() > 0 {
		g.addDiagnostic(DiagnosticDependentRequired, path, "dependentRequired is validation-only and was not rendered into Go model shape")
	}
	if schema.If != nil || schema.Then != nil || schema.Else != nil {
		g.addDiagnostic(DiagnosticConditionalSchema, path, "if/then/else is validation-only and was not rendered into Go model shape")
	}
	if schema.Not != nil {
		g.addDiagnostic(DiagnosticNotSchema, path, "not is validation-only and was not rendered into Go model shape")
	}
	if schema.Const != nil {
		g.addDiagnostic(DiagnosticConstKeyword, path, "const is validation-only and was not enforced by the generated Go model")
	}
	if hasValidationKeyword(schema) {
		g.addDiagnostic(DiagnosticValidationKeyword, path, "JSON Schema validation keywords are not enforced by generated Go models")
	}
	if schema.UnevaluatedProperties != nil {
		g.addDiagnostic(DiagnosticUnevaluatedProperties, path, "unevaluatedProperties is validation-only and was not rendered into Go model shape")
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
		variantName := g.nestedTypeName(ir.Name, "Variant"+intString(i+1))
		if built, err := child.BuildSchema(); err == nil && built != nil && built.Title != "" {
			variantName = g.nestedTypeName(ir.Name, built.Title)
		}
		childIR, err := g.irFromOpenAPIName(variantName, true, child, path+".union")
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
		g.addDiagnostic(DiagnosticOptionalConstDiscriminator, path, "oneOf has a shared const discriminator property, but it is optional; using json.RawMessage")
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

func nonNullTypes(types []string) []string {
	out := make([]string, 0, len(types))
	for _, t := range types {
		if t != "null" {
			out = append(out, t)
		}
	}
	return out
}

func primaryTypeForSchema(schema *highbase.Schema) (string, bool, bool) {
	types := nonNullTypes(schema.Type)
	if len(types) > 0 {
		return types[0], false, false
	}
	var inferred []string
	if hasStringKeyword(schema) {
		inferred = append(inferred, "string")
	}
	if hasNumberKeyword(schema) {
		inferred = append(inferred, "number")
	}
	if hasArrayKeyword(schema) {
		inferred = append(inferred, "array")
	}
	if hasObjectKeyword(schema) {
		inferred = append(inferred, "object")
	}
	if len(inferred) == 1 {
		return inferred[0], true, false
	}
	if len(inferred) > 1 {
		return "", false, true
	}
	return "", false, false
}

func kindForJSONType(typ string) Kind {
	switch typ {
	case "string":
		return KindString
	case "integer":
		return KindInteger
	case "number":
		return KindNumber
	case "boolean":
		return KindBoolean
	case "array":
		return KindArray
	case "object":
		return KindObject
	default:
		return KindAny
	}
}

func schemaHasOnlyDynamicRefShape(schema *highbase.Schema) bool {
	return schema.DynamicRef != "" &&
		len(schema.Type) == 0 &&
		len(schema.AllOf) == 0 &&
		len(schema.OneOf) == 0 &&
		len(schema.AnyOf) == 0 &&
		len(schema.Enum) == 0 &&
		schema.Const == nil &&
		schema.Not == nil &&
		schema.Properties == nil &&
		schema.Items == nil &&
		len(schema.PrefixItems) == 0 &&
		schema.AdditionalProperties == nil
}

func hasSchemaMetadata(schema *highbase.Schema) bool {
	return schema.SchemaTypeRef != "" ||
		schema.Id != "" ||
		schema.Anchor != "" ||
		schema.DynamicAnchor != "" ||
		schema.Comment != "" ||
		(schema.Vocabulary != nil && schema.Vocabulary.Len() > 0)
}

func hasValidationKeyword(schema *highbase.Schema) bool {
	return schema.MultipleOf != nil ||
		schema.Maximum != nil ||
		schema.Minimum != nil ||
		schema.ExclusiveMaximum != nil ||
		schema.ExclusiveMinimum != nil ||
		schema.MaxLength != nil ||
		schema.MinLength != nil ||
		schema.Pattern != "" ||
		schema.MaxItems != nil ||
		schema.MinItems != nil ||
		schema.UniqueItems != nil ||
		schema.MaxProperties != nil ||
		schema.MinProperties != nil ||
		schema.ContentEncoding != "" ||
		schema.ContentMediaType != ""
}

func hasStringKeyword(schema *highbase.Schema) bool {
	return schema.MaxLength != nil ||
		schema.MinLength != nil ||
		schema.Pattern != "" ||
		schema.ContentEncoding != "" ||
		schema.ContentMediaType != "" ||
		schema.ContentSchema != nil
}

func hasNumberKeyword(schema *highbase.Schema) bool {
	return schema.MultipleOf != nil ||
		schema.Maximum != nil ||
		schema.Minimum != nil ||
		schema.ExclusiveMaximum != nil ||
		schema.ExclusiveMinimum != nil
}

func hasArrayKeyword(schema *highbase.Schema) bool {
	return schema.Items != nil ||
		len(schema.PrefixItems) > 0 ||
		schema.Contains != nil ||
		schema.MinContains != nil ||
		schema.MaxContains != nil ||
		schema.MaxItems != nil ||
		schema.MinItems != nil ||
		schema.UniqueItems != nil ||
		schema.UnevaluatedItems != nil
}

func hasObjectKeyword(schema *highbase.Schema) bool {
	return (schema.Properties != nil && schema.Properties.Len() > 0) ||
		schema.AdditionalProperties != nil ||
		(schema.PatternProperties != nil && schema.PatternProperties.Len() > 0) ||
		schema.PropertyNames != nil ||
		schema.MaxProperties != nil ||
		schema.MinProperties != nil ||
		len(schema.Required) > 0 ||
		(schema.DependentSchemas != nil && schema.DependentSchemas.Len() > 0) ||
		(schema.DependentRequired != nil && schema.DependentRequired.Len() > 0) ||
		schema.UnevaluatedProperties != nil
}

func nonNullVariants(variants []*SchemaIR) []*SchemaIR {
	var out []*SchemaIR
	for _, variant := range variants {
		if variant == nil {
			continue
		}
		if isNullOnlyIR(variant) {
			continue
		}
		out = append(out, variant)
	}
	return out
}

func isNullOnlyIR(ir *SchemaIR) bool {
	if ir == nil {
		return false
	}
	if ir.SourceSchema != nil {
		return schemaOnlyAllowsNull(ir.SourceSchema)
	}
	if ir.Const != nil && nodeIsNull(ir.Const) {
		return true
	}
	if len(ir.Enum) > 0 {
		shape := enumShapeFor(ir.Enum)
		return shape.nullable && shape.nonNullValues == 0
	}
	return false
}

func schemaOnlyAllowsNull(schema *highbase.Schema) bool {
	if schema == nil {
		return false
	}
	if schema.Const != nil && nodeIsNull(schema.Const) {
		return true
	}
	if len(schema.Enum) > 0 {
		shape := enumShapeFor(schema.Enum)
		return shape.nullable && shape.nonNullValues == 0
	}
	return len(schema.Type) > 0 && len(nonNullTypes(schema.Type)) == 0
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
