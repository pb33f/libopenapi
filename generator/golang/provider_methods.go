// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"strconv"
	"strings"

	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

func (g *Generator) recordSchemaMetadata(typeName string, schema *highbase.Schema) {
	if !g.schemaMetadataSidecar || typeName == "" || schema == nil {
		return
	}
	if _, ok := g.metadataSchemas[typeName]; ok {
		return
	}
	g.metadataSchemas[typeName] = schema
	g.metadataOrder = append(g.metadataOrder, typeName)
}

func (g *Generator) renderSchemaMetadataSidecarDecl() string {
	if !g.schemaMetadataSidecar || len(g.metadataOrder) == 0 {
		return ""
	}
	var b strings.Builder
	writeSchemaMetadataTypes(&b)
	b.WriteString("\nvar openAPISchemas = map[string]*openAPISchemaMetadata{\n")
	for _, typeName := range g.metadataOrder {
		b.WriteByte('\t')
		b.WriteString(strconv.Quote(typeName))
		b.WriteString(": ")
		b.WriteString(g.schemaMetadataLiteral(g.metadataSchemas[typeName], 1))
		b.WriteString(",\n")
	}
	b.WriteString("}\n")
	for _, typeName := range g.metadataOrder {
		b.WriteString("\nfunc (")
		b.WriteString(typeName)
		b.WriteString(") OpenAPISchemaMetadata() any {\n\treturn openAPISchemas[")
		b.WriteString(strconv.Quote(typeName))
		b.WriteString("]\n}\n")
	}
	return b.String()
}

func writeSchemaMetadataTypes(b *strings.Builder) {
	b.WriteString(`type openAPISchemaMetadata struct {
	Ref                   string
	SchemaTypeRef         string
	ExclusiveMaximum      *openAPIDynamicBoolNumber
	ExclusiveMinimum      *openAPIDynamicBoolNumber
	Type                  []string
	AllOf                 []*openAPISchemaMetadata
	OneOf                 []*openAPISchemaMetadata
	AnyOf                 []*openAPISchemaMetadata
	Discriminator         *openAPIDiscriminatorMetadata
	Examples              []*openAPIYAMLNode
	PrefixItems           []*openAPISchemaMetadata
	Contains              *openAPISchemaMetadata
	MinContains           *openAPIInt
	MaxContains           *openAPIInt
	If                    *openAPISchemaMetadata
	Else                  *openAPISchemaMetadata
	Then                  *openAPISchemaMetadata
	DependentSchemas      []openAPINamedSchemaMetadata
	DependentRequired     []openAPIStringList
	PatternProperties     []openAPINamedSchemaMetadata
	PropertyNames         *openAPISchemaMetadata
	UnevaluatedItems      *openAPISchemaMetadata
	UnevaluatedProperties *openAPIDynamicSchemaBool
	Items                 *openAPIDynamicSchemaBool
	ID                    string
	Anchor                string
	DynamicAnchor         string
	DynamicRef            string
	Comment               string
	ContentSchema         *openAPISchemaMetadata
	Vocabulary            []openAPIStringBool
	Not                   *openAPISchemaMetadata
	Properties            []openAPINamedSchemaMetadata
	Title                 string
	MultipleOf            *openAPIFloat
	Maximum               *openAPIFloat
	Minimum               *openAPIFloat
	MaxLength             *openAPIInt
	MinLength             *openAPIInt
	Pattern               string
	Format                string
	MaxItems              *openAPIInt
	MinItems              *openAPIInt
	UniqueItems           *openAPIBool
	MaxProperties         *openAPIInt
	MinProperties         *openAPIInt
	Required              []string
	Enum                  []*openAPIYAMLNode
	AdditionalProperties  *openAPIDynamicSchemaBool
	Description           string
	ContentEncoding       string
	ContentMediaType      string
	Default               *openAPIYAMLNode
	Const                 *openAPIYAMLNode
	Nullable              *openAPIBool
	ReadOnly              *openAPIBool
	WriteOnly             *openAPIBool
	Example               *openAPIYAMLNode
	Deprecated            *openAPIBool
	Extensions            []openAPINamedYAMLNode
}

type openAPIDynamicBoolNumber struct {
	Bool   *openAPIBool
	Number *openAPIFloat
}

type openAPIDynamicSchemaBool struct {
	Schema *openAPISchemaMetadata
	Bool   *openAPIBool
}

type openAPIDiscriminatorMetadata struct {
	PropertyName   string
	Mapping        []openAPIStringString
	DefaultMapping string
}

type openAPINamedSchemaMetadata struct {
	Name   string
	Schema *openAPISchemaMetadata
}

type openAPINamedYAMLNode struct {
	Name  string
	Value *openAPIYAMLNode
}

type openAPIStringBool struct {
	Name  string
	Value bool
}

type openAPIStringString struct {
	Name  string
	Value string
}

type openAPIStringList struct {
	Name   string
	Values []string
}

type openAPIYAMLNode struct {
	Kind    string
	Style   int
	Tag     string
	Value   string
	Anchor  string
	Content []*openAPIYAMLNode
	Alias   *openAPIYAMLNode
}

type openAPIFloat struct {
	Value float64
}

type openAPIInt struct {
	Value int64
}

type openAPIBool struct {
	Value bool
}
`)
}

func (g *Generator) schemaMetadataLiteral(schema *highbase.Schema, depth int) string {
	return g.schemaMetadataLiteralWithRef("", schema, depth)
}

func (g *Generator) schemaMetadataLiteralWithRef(ref string, schema *highbase.Schema, depth int) string {
	if schema == nil {
		return "nil"
	}
	var b strings.Builder
	b.WriteString("&openAPISchemaMetadata{\n")
	writeMetadataField(&b, depth+1, "Ref", metadataStringLiteral(ref))
	writeMetadataField(&b, depth+1, "SchemaTypeRef", metadataStringLiteral(schema.SchemaTypeRef))
	writeMetadataField(&b, depth+1, "ExclusiveMaximum", metadataDynamicBoolNumberLiteral(schema.ExclusiveMaximum))
	writeMetadataField(&b, depth+1, "ExclusiveMinimum", metadataDynamicBoolNumberLiteral(schema.ExclusiveMinimum))
	writeMetadataField(&b, depth+1, "Type", metadataStringSliceLiteral(schema.Type, depth+1))
	writeMetadataField(&b, depth+1, "AllOf", g.schemaSliceMetadataLiteral(schema.AllOf, depth+1))
	writeMetadataField(&b, depth+1, "OneOf", g.schemaSliceMetadataLiteral(schema.OneOf, depth+1))
	writeMetadataField(&b, depth+1, "AnyOf", g.schemaSliceMetadataLiteral(schema.AnyOf, depth+1))
	writeMetadataField(&b, depth+1, "Discriminator", metadataDiscriminatorLiteral(schema.Discriminator, depth+1))
	writeMetadataField(&b, depth+1, "Examples", metadataYAMLNodeSliceLiteral(schema.Examples, depth+1))
	writeMetadataField(&b, depth+1, "PrefixItems", g.schemaSliceMetadataLiteral(schema.PrefixItems, depth+1))
	writeMetadataField(&b, depth+1, "Contains", g.optionalSchemaProxyMetadataLiteral(schema.Contains, depth+1))
	writeMetadataField(&b, depth+1, "MinContains", metadataIntLiteral(schema.MinContains))
	writeMetadataField(&b, depth+1, "MaxContains", metadataIntLiteral(schema.MaxContains))
	writeMetadataField(&b, depth+1, "If", g.optionalSchemaProxyMetadataLiteral(schema.If, depth+1))
	writeMetadataField(&b, depth+1, "Else", g.optionalSchemaProxyMetadataLiteral(schema.Else, depth+1))
	writeMetadataField(&b, depth+1, "Then", g.optionalSchemaProxyMetadataLiteral(schema.Then, depth+1))
	writeMetadataField(&b, depth+1, "DependentSchemas", g.schemaMapMetadataLiteral(schema.DependentSchemas, depth+1))
	writeMetadataField(&b, depth+1, "DependentRequired", metadataStringListMapLiteral(schema.DependentRequired, depth+1))
	writeMetadataField(&b, depth+1, "PatternProperties", g.schemaMapMetadataLiteral(schema.PatternProperties, depth+1))
	writeMetadataField(&b, depth+1, "PropertyNames", g.optionalSchemaProxyMetadataLiteral(schema.PropertyNames, depth+1))
	writeMetadataField(&b, depth+1, "UnevaluatedItems", g.optionalSchemaProxyMetadataLiteral(schema.UnevaluatedItems, depth+1))
	writeMetadataField(&b, depth+1, "UnevaluatedProperties", g.metadataDynamicSchemaBoolLiteral(schema.UnevaluatedProperties, depth+1))
	writeMetadataField(&b, depth+1, "Items", g.metadataDynamicSchemaBoolLiteral(schema.Items, depth+1))
	writeMetadataField(&b, depth+1, "ID", metadataStringLiteral(schema.Id))
	writeMetadataField(&b, depth+1, "Anchor", metadataStringLiteral(schema.Anchor))
	writeMetadataField(&b, depth+1, "DynamicAnchor", metadataStringLiteral(schema.DynamicAnchor))
	writeMetadataField(&b, depth+1, "DynamicRef", metadataStringLiteral(schema.DynamicRef))
	writeMetadataField(&b, depth+1, "Comment", metadataStringLiteral(schema.Comment))
	writeMetadataField(&b, depth+1, "ContentSchema", g.optionalSchemaProxyMetadataLiteral(schema.ContentSchema, depth+1))
	writeMetadataField(&b, depth+1, "Vocabulary", metadataStringBoolMapLiteral(schema.Vocabulary, depth+1))
	writeMetadataField(&b, depth+1, "Not", g.optionalSchemaProxyMetadataLiteral(schema.Not, depth+1))
	writeMetadataField(&b, depth+1, "Properties", g.schemaMapMetadataLiteral(schema.Properties, depth+1))
	writeMetadataField(&b, depth+1, "Title", metadataStringLiteral(schema.Title))
	writeMetadataField(&b, depth+1, "MultipleOf", metadataFloatLiteral(schema.MultipleOf))
	writeMetadataField(&b, depth+1, "Maximum", metadataFloatLiteral(schema.Maximum))
	writeMetadataField(&b, depth+1, "Minimum", metadataFloatLiteral(schema.Minimum))
	writeMetadataField(&b, depth+1, "MaxLength", metadataIntLiteral(schema.MaxLength))
	writeMetadataField(&b, depth+1, "MinLength", metadataIntLiteral(schema.MinLength))
	writeMetadataField(&b, depth+1, "Pattern", metadataStringLiteral(schema.Pattern))
	writeMetadataField(&b, depth+1, "Format", metadataStringLiteral(schema.Format))
	writeMetadataField(&b, depth+1, "MaxItems", metadataIntLiteral(schema.MaxItems))
	writeMetadataField(&b, depth+1, "MinItems", metadataIntLiteral(schema.MinItems))
	writeMetadataField(&b, depth+1, "UniqueItems", metadataBoolLiteral(schema.UniqueItems))
	writeMetadataField(&b, depth+1, "MaxProperties", metadataIntLiteral(schema.MaxProperties))
	writeMetadataField(&b, depth+1, "MinProperties", metadataIntLiteral(schema.MinProperties))
	writeMetadataField(&b, depth+1, "Required", metadataStringSliceLiteral(schema.Required, depth+1))
	writeMetadataField(&b, depth+1, "Enum", metadataYAMLNodeSliceLiteral(schema.Enum, depth+1))
	writeMetadataField(&b, depth+1, "AdditionalProperties", g.metadataDynamicSchemaBoolLiteral(schema.AdditionalProperties, depth+1))
	writeMetadataField(&b, depth+1, "Description", metadataStringLiteral(schema.Description))
	writeMetadataField(&b, depth+1, "ContentEncoding", metadataStringLiteral(schema.ContentEncoding))
	writeMetadataField(&b, depth+1, "ContentMediaType", metadataStringLiteral(schema.ContentMediaType))
	writeMetadataField(&b, depth+1, "Default", optionalMetadataYAMLNodeLiteral(schema.Default, depth+1))
	writeMetadataField(&b, depth+1, "Const", optionalMetadataYAMLNodeLiteral(schema.Const, depth+1))
	writeMetadataField(&b, depth+1, "Nullable", metadataBoolLiteral(schema.Nullable))
	writeMetadataField(&b, depth+1, "ReadOnly", metadataBoolLiteral(schema.ReadOnly))
	writeMetadataField(&b, depth+1, "WriteOnly", metadataBoolLiteral(schema.WriteOnly))
	writeMetadataField(&b, depth+1, "Example", optionalMetadataYAMLNodeLiteral(schema.Example, depth+1))
	writeMetadataField(&b, depth+1, "Deprecated", metadataBoolLiteral(schema.Deprecated))
	writeMetadataField(&b, depth+1, "Extensions", metadataExtensionsLiteral(schema.Extensions, depth+1))
	b.WriteString(metadataIndent(depth))
	b.WriteByte('}')
	return b.String()
}

func (g *Generator) schemaProxyMetadataLiteral(proxy *highbase.SchemaProxy, depth int) string {
	if proxy == nil {
		return "nil"
	}
	if proxy.IsReference() {
		if schema := referenceSiblingMetadataSchema(proxy); schema != nil {
			return g.schemaMetadataLiteralWithRef(proxy.GetReference(), schema, depth)
		}
		return "&openAPISchemaMetadata{Ref: " + strconv.Quote(proxy.GetReference()) + "}"
	}
	schema, _ := proxy.BuildSchema()
	return g.schemaMetadataLiteral(schema, depth)
}

func referenceSiblingMetadataSchema(proxy *highbase.SchemaProxy) *highbase.Schema {
	if proxy == nil || !proxy.IsReference() {
		return nil
	}
	if proxy.GoLow() == nil {
		return proxy.Schema()
	}
	if refNode := proxy.GetReferenceNode(); refNode != nil && len(refNode.Content) > 2 {
		return schemaFromReferenceSiblingNode(refNode)
	}
	return nil
}

func schemaFromReferenceSiblingNode(refNode *yaml.Node) *highbase.Schema {
	siblingNode := &yaml.Node{Kind: yaml.MappingNode}
	if refNode != nil {
		for i := 0; i < len(refNode.Content)-1; i += 2 {
			if refNode.Content[i].Value != "$ref" {
				siblingNode.Content = append(siblingNode.Content, refNode.Content[i], refNode.Content[i+1])
			}
		}
	}
	if len(siblingNode.Content) == 0 {
		return nil
	}
	raw, _ := yaml.Marshal(siblingNode)
	proxy, _ := schemaProxyFromProviderYAML("RefSibling", string(raw))
	return proxy.Schema()
}

func (g *Generator) optionalSchemaProxyMetadataLiteral(proxy *highbase.SchemaProxy, depth int) string {
	if proxy == nil {
		return ""
	}
	return g.schemaProxyMetadataLiteral(proxy, depth)
}

func (g *Generator) schemaSliceMetadataLiteral(schemas []*highbase.SchemaProxy, depth int) string {
	if len(schemas) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("[]*openAPISchemaMetadata{\n")
	for _, schema := range schemas {
		b.WriteString(metadataIndent(depth + 1))
		if schema == nil {
			b.WriteString("nil")
		} else {
			b.WriteString(g.schemaProxyMetadataLiteral(schema, depth+1))
		}
		b.WriteString(",\n")
	}
	b.WriteString(metadataIndent(depth))
	b.WriteByte('}')
	return b.String()
}

func (g *Generator) schemaMapMetadataLiteral(schemas *orderedmap.Map[string, *highbase.SchemaProxy], depth int) string {
	if schemas == nil || schemas.Len() == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("[]openAPINamedSchemaMetadata{\n")
	for name, schema := range schemas.FromOldest() {
		b.WriteString(metadataIndent(depth + 1))
		b.WriteString("{Name: ")
		b.WriteString(strconv.Quote(name))
		b.WriteString(", Schema: ")
		if schema == nil {
			b.WriteString("nil")
		} else {
			b.WriteString(g.schemaProxyMetadataLiteral(schema, depth+1))
		}
		b.WriteString("},\n")
	}
	b.WriteString(metadataIndent(depth))
	b.WriteByte('}')
	return b.String()
}

func (g *Generator) metadataDynamicSchemaBoolLiteral(value *highbase.DynamicValue[*highbase.SchemaProxy, bool], depth int) string {
	if value == nil {
		return ""
	}
	if value.IsB() {
		return "&openAPIDynamicSchemaBool{Bool: " + metadataBoolValueLiteral(value.B) + "}"
	}
	return "&openAPIDynamicSchemaBool{Schema: " + g.schemaProxyMetadataLiteral(value.A, depth) + "}"
}

func metadataDynamicBoolNumberLiteral(value *highbase.DynamicValue[bool, float64]) string {
	if value == nil {
		return ""
	}
	if value.IsB() {
		return "&openAPIDynamicBoolNumber{Number: " + metadataFloatValueLiteral(value.B) + "}"
	}
	return "&openAPIDynamicBoolNumber{Bool: " + metadataBoolValueLiteral(value.A) + "}"
}

func metadataDiscriminatorLiteral(discriminator *highbase.Discriminator, depth int) string {
	if discriminator == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("&openAPIDiscriminatorMetadata{\n")
	writeMetadataField(&b, depth+1, "PropertyName", metadataStringLiteral(discriminator.PropertyName))
	writeMetadataField(&b, depth+1, "Mapping", metadataStringStringMapLiteral(discriminator.Mapping, depth+1))
	writeMetadataField(&b, depth+1, "DefaultMapping", metadataStringLiteral(discriminator.DefaultMapping))
	b.WriteString(metadataIndent(depth))
	b.WriteByte('}')
	return b.String()
}

func metadataStringStringMapLiteral(values *orderedmap.Map[string, string], depth int) string {
	if values == nil || values.Len() == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("[]openAPIStringString{\n")
	for name, value := range values.FromOldest() {
		b.WriteString(metadataIndent(depth + 1))
		b.WriteString("{Name: ")
		b.WriteString(strconv.Quote(name))
		b.WriteString(", Value: ")
		b.WriteString(strconv.Quote(value))
		b.WriteString("},\n")
	}
	b.WriteString(metadataIndent(depth))
	b.WriteByte('}')
	return b.String()
}

func metadataStringBoolMapLiteral(values *orderedmap.Map[string, bool], depth int) string {
	if values == nil || values.Len() == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("[]openAPIStringBool{\n")
	for name, value := range values.FromOldest() {
		b.WriteString(metadataIndent(depth + 1))
		b.WriteString("{Name: ")
		b.WriteString(strconv.Quote(name))
		b.WriteString(", Value: ")
		b.WriteString(strconv.FormatBool(value))
		b.WriteString("},\n")
	}
	b.WriteString(metadataIndent(depth))
	b.WriteByte('}')
	return b.String()
}

func metadataStringListMapLiteral(values *orderedmap.Map[string, []string], depth int) string {
	if values == nil || values.Len() == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("[]openAPIStringList{\n")
	for name, list := range values.FromOldest() {
		b.WriteString(metadataIndent(depth + 1))
		b.WriteString("{Name: ")
		b.WriteString(strconv.Quote(name))
		b.WriteString(", Values: ")
		b.WriteString(metadataStringSliceLiteral(list, depth+1))
		b.WriteString("},\n")
	}
	b.WriteString(metadataIndent(depth))
	b.WriteByte('}')
	return b.String()
}

func metadataExtensionsLiteral(values *orderedmap.Map[string, *yaml.Node], depth int) string {
	if values == nil || values.Len() == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("[]openAPINamedYAMLNode{\n")
	for name, value := range values.FromOldest() {
		b.WriteString(metadataIndent(depth + 1))
		b.WriteString("{Name: ")
		b.WriteString(strconv.Quote(name))
		b.WriteString(", Value: ")
		b.WriteString(metadataYAMLNodeLiteral(value, depth+1))
		b.WriteString("},\n")
	}
	b.WriteString(metadataIndent(depth))
	b.WriteByte('}')
	return b.String()
}

func metadataYAMLNodeSliceLiteral(nodes []*yaml.Node, depth int) string {
	if len(nodes) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("[]*openAPIYAMLNode{\n")
	for _, node := range nodes {
		b.WriteString(metadataIndent(depth + 1))
		b.WriteString(metadataYAMLNodeLiteral(node, depth+1))
		b.WriteString(",\n")
	}
	b.WriteString(metadataIndent(depth))
	b.WriteByte('}')
	return b.String()
}

func metadataYAMLNodeLiteral(node *yaml.Node, depth int) string {
	if node == nil {
		return "nil"
	}
	var b strings.Builder
	b.WriteString("&openAPIYAMLNode{\n")
	writeMetadataField(&b, depth+1, "Kind", strconv.Quote(metadataYAMLKind(node.Kind)))
	writeMetadataField(&b, depth+1, "Style", metadataPlainIntLiteral(int64(node.Style)))
	writeMetadataField(&b, depth+1, "Tag", metadataStringLiteral(node.Tag))
	writeMetadataField(&b, depth+1, "Value", metadataStringLiteral(node.Value))
	writeMetadataField(&b, depth+1, "Anchor", metadataStringLiteral(node.Anchor))
	writeMetadataField(&b, depth+1, "Content", metadataYAMLNodeContentLiteral(node.Content, depth+1))
	writeMetadataField(&b, depth+1, "Alias", optionalMetadataYAMLNodeLiteral(node.Alias, depth+1))
	b.WriteString(metadataIndent(depth))
	b.WriteByte('}')
	return b.String()
}

func optionalMetadataYAMLNodeLiteral(node *yaml.Node, depth int) string {
	if node == nil {
		return ""
	}
	return metadataYAMLNodeLiteral(node, depth)
}

func metadataYAMLNodeContentLiteral(nodes []*yaml.Node, depth int) string {
	if len(nodes) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("[]*openAPIYAMLNode{\n")
	for _, node := range nodes {
		b.WriteString(metadataIndent(depth + 1))
		b.WriteString(metadataYAMLNodeLiteral(node, depth+1))
		b.WriteString(",\n")
	}
	b.WriteString(metadataIndent(depth))
	b.WriteByte('}')
	return b.String()
}

func metadataYAMLKind(kind yaml.Kind) string {
	switch kind {
	case yaml.DocumentNode:
		return "document"
	case yaml.SequenceNode:
		return "sequence"
	case yaml.MappingNode:
		return "mapping"
	case yaml.AliasNode:
		return "alias"
	default:
		return "scalar"
	}
}

func metadataStringSliceLiteral(values []string, depth int) string {
	if len(values) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("[]string{")
	for i, value := range values {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(strconv.Quote(value))
	}
	b.WriteByte('}')
	return b.String()
}

func metadataStringLiteral(value string) string {
	if value == "" {
		return ""
	}
	return strconv.Quote(value)
}

func metadataFloatLiteral(value *float64) string {
	if value == nil {
		return ""
	}
	return metadataFloatValueLiteral(*value)
}

func metadataFloatValueLiteral(value float64) string {
	return "&openAPIFloat{Value: " + strconv.FormatFloat(value, 'g', -1, 64) + "}"
}

func metadataIntLiteral(value *int64) string {
	if value == nil {
		return ""
	}
	return metadataIntValueLiteral(*value)
}

func metadataIntValueLiteral(value int64) string {
	return "&openAPIInt{Value: " + strconv.FormatInt(value, 10) + "}"
}

func metadataPlainIntLiteral(value int64) string {
	if value == 0 {
		return ""
	}
	return strconv.FormatInt(value, 10)
}

func metadataBoolLiteral(value *bool) string {
	if value == nil {
		return ""
	}
	return metadataBoolValueLiteral(*value)
}

func metadataBoolValueLiteral(value bool) string {
	return "&openAPIBool{Value: " + strconv.FormatBool(value) + "}"
}

func writeMetadataField(b *strings.Builder, depth int, name, value string) {
	if value == "" {
		return
	}
	b.WriteString(metadataIndent(depth))
	b.WriteString(name)
	b.WriteString(": ")
	b.WriteString(value)
	b.WriteString(",\n")
}

func metadataIndent(depth int) string {
	if depth <= 0 {
		return ""
	}
	return strings.Repeat("\t", depth)
}
