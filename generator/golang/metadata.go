// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"strconv"
	"strings"

	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"go.yaml.in/yaml/v4"
)

type openAPIMetadata struct {
	Present bool

	FormatSet      bool
	Format         string
	TitleSet       bool
	Title          string
	DescriptionSet bool
	Description    string
	NullableSet    bool
	Nullable       bool
	ReadOnlySet    bool
	ReadOnly       bool
	WriteOnlySet   bool
	WriteOnly      bool
	DeprecatedSet  bool
	Deprecated     bool

	MinimumSet          bool
	Minimum             float64
	MaximumSet          bool
	Maximum             float64
	ExclusiveMinimumSet bool
	ExclusiveMinimum    float64
	ExclusiveMaximumSet bool
	ExclusiveMaximum    float64
	MultipleOfSet       bool
	MultipleOf          float64
	MinLengthSet        bool
	MinLength           int64
	MaxLengthSet        bool
	MaxLength           int64
	PatternSet          bool
	Pattern             string
	MinItemsSet         bool
	MinItems            int64
	MaxItemsSet         bool
	MaxItems            int64
	UniqueItemsSet      bool
	UniqueItems         bool
	MinPropertiesSet    bool
	MinProperties       int64
	MaxPropertiesSet    bool
	MaxProperties       int64

	Enum  []*yaml.Node
	Const *yaml.Node
}

func parseOpenAPITag(raw string) openAPIMetadata {
	var meta openAPIMetadata
	if raw == "" {
		return meta
	}
	meta.Present = true
	for _, part := range splitEscaped(raw, ';') {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, value, hasValue := cutEscaped(part, '=')
		key = strings.TrimSpace(key)
		rawValue := strings.TrimSpace(value)
		value = unescapeOpenAPITagValue(rawValue)
		switch key {
		case "format":
			meta.FormatSet = hasValue
			meta.Format = value
		case "title":
			meta.TitleSet = hasValue
			meta.Title = value
		case "description", "desc":
			meta.DescriptionSet = hasValue
			meta.Description = value
		case "nullable":
			if parsed, ok := parseTagBool(value, hasValue); ok {
				meta.NullableSet = true
				meta.Nullable = parsed
			}
		case "readOnly":
			if parsed, ok := parseTagBool(value, hasValue); ok {
				meta.ReadOnlySet = true
				meta.ReadOnly = parsed
			}
		case "writeOnly":
			if parsed, ok := parseTagBool(value, hasValue); ok {
				meta.WriteOnlySet = true
				meta.WriteOnly = parsed
			}
		case "deprecated":
			if parsed, ok := parseTagBool(value, hasValue); ok {
				meta.DeprecatedSet = true
				meta.Deprecated = parsed
			}
		case "minimum":
			meta.Minimum, meta.MinimumSet = parseTagFloat(value, hasValue)
		case "maximum":
			meta.Maximum, meta.MaximumSet = parseTagFloat(value, hasValue)
		case "exclusiveMinimum":
			meta.ExclusiveMinimum, meta.ExclusiveMinimumSet = parseTagFloat(value, hasValue)
		case "exclusiveMaximum":
			meta.ExclusiveMaximum, meta.ExclusiveMaximumSet = parseTagFloat(value, hasValue)
		case "multipleOf":
			meta.MultipleOf, meta.MultipleOfSet = parseTagFloat(value, hasValue)
		case "minLength":
			meta.MinLength, meta.MinLengthSet = parseTagInt(value, hasValue)
		case "maxLength":
			meta.MaxLength, meta.MaxLengthSet = parseTagInt(value, hasValue)
		case "pattern":
			meta.PatternSet = hasValue
			meta.Pattern = value
		case "minItems":
			meta.MinItems, meta.MinItemsSet = parseTagInt(value, hasValue)
		case "maxItems":
			meta.MaxItems, meta.MaxItemsSet = parseTagInt(value, hasValue)
		case "uniqueItems":
			if parsed, ok := parseTagBool(value, hasValue); ok {
				meta.UniqueItemsSet = true
				meta.UniqueItems = parsed
			}
		case "minProperties":
			meta.MinProperties, meta.MinPropertiesSet = parseTagInt(value, hasValue)
		case "maxProperties":
			meta.MaxProperties, meta.MaxPropertiesSet = parseTagInt(value, hasValue)
		case "enum":
			if hasValue {
				meta.Enum = parseTagNodes(rawValue)
			}
		case "const":
			if hasValue {
				meta.Const = parseTagNode(rawValue)
			}
		}
	}
	return meta
}

func (g *Generator) applyOpenAPIMetadata(ir *SchemaIR, meta openAPIMetadata) {
	if ir == nil || !meta.Present {
		return
	}
	ir.FieldMetadata = true
	schema := ir.SourceSchema
	if schema == nil {
		schema = &highbase.Schema{}
		ir.SourceSchema = schema
	}
	if meta.FormatSet {
		ir.Format = meta.Format
		schema.Format = meta.Format
	}
	if meta.TitleSet {
		ir.Title = meta.Title
		schema.Title = meta.Title
	}
	if meta.DescriptionSet {
		ir.Description = meta.Description
		schema.Description = meta.Description
	}
	if meta.NullableSet {
		ir.Nullable = meta.Nullable
		schema.Nullable = nil
	}
	if meta.ReadOnlySet {
		ir.ReadOnly = meta.ReadOnly
		schema.ReadOnly = boolPtr(meta.ReadOnly)
	}
	if meta.WriteOnlySet {
		ir.WriteOnly = meta.WriteOnly
		schema.WriteOnly = boolPtr(meta.WriteOnly)
	}
	if meta.DeprecatedSet {
		ir.Deprecated = meta.Deprecated
		schema.Deprecated = boolPtr(meta.Deprecated)
	}
	if meta.MinimumSet {
		schema.Minimum = &meta.Minimum
	}
	if meta.MaximumSet {
		schema.Maximum = &meta.Maximum
	}
	if meta.ExclusiveMinimumSet {
		schema.ExclusiveMinimum = &highbase.DynamicValue[bool, float64]{N: 1, B: meta.ExclusiveMinimum}
	}
	if meta.ExclusiveMaximumSet {
		schema.ExclusiveMaximum = &highbase.DynamicValue[bool, float64]{N: 1, B: meta.ExclusiveMaximum}
	}
	if meta.MultipleOfSet {
		schema.MultipleOf = &meta.MultipleOf
	}
	if meta.MinLengthSet {
		schema.MinLength = &meta.MinLength
	}
	if meta.MaxLengthSet {
		schema.MaxLength = &meta.MaxLength
	}
	if meta.PatternSet {
		schema.Pattern = meta.Pattern
	}
	if meta.MinItemsSet {
		schema.MinItems = &meta.MinItems
	}
	if meta.MaxItemsSet {
		schema.MaxItems = &meta.MaxItems
	}
	if meta.UniqueItemsSet {
		schema.UniqueItems = &meta.UniqueItems
	}
	if meta.MinPropertiesSet {
		schema.MinProperties = &meta.MinProperties
	}
	if meta.MaxPropertiesSet {
		schema.MaxProperties = &meta.MaxProperties
	}
	if len(meta.Enum) > 0 {
		ir.Enum = meta.Enum
		schema.Enum = meta.Enum
	}
	if meta.Const != nil {
		ir.Const = meta.Const
		schema.Const = meta.Const
	}
}

func (g *Generator) openAPITagLiteral(ir *SchemaIR, fieldType string) string {
	if !g.openapiTags || ir == nil {
		return ""
	}
	var parts []string
	add := func(key, value string) {
		if value == "" || strings.Contains(value, "`") {
			return
		}
		parts = append(parts, key+"="+escapeOpenAPITagValue(value))
	}
	addBool := func(key string) {
		parts = append(parts, key)
	}
	addInt := func(key string, value *int64) {
		if value != nil {
			parts = append(parts, key+"="+strconv.FormatInt(*value, 10))
		}
	}
	addFloat := func(key string, value *float64) {
		if value != nil {
			parts = append(parts, key+"="+strconv.FormatFloat(*value, 'g', -1, 64))
		}
	}
	if strings.HasPrefix(fieldType, "*") || ir.Nullable {
		parts = append(parts, "nullable="+strconv.FormatBool(ir.Nullable))
	}
	add("format", ir.Format)
	add("title", ir.Title)
	add("description", ir.Description)
	if ir.ReadOnly {
		addBool("readOnly")
	}
	if ir.WriteOnly {
		addBool("writeOnly")
	}
	if ir.Deprecated {
		addBool("deprecated")
	}
	if len(ir.Enum) > 0 {
		encoded := encodeTagNodes(ir.Enum)
		if encoded != "" {
			parts = append(parts, "enum="+encoded)
		}
	}
	if ir.Const != nil {
		if encoded := encodeTagNode(ir.Const); encoded != "" {
			parts = append(parts, "const="+encoded)
		}
	}
	if schema := ir.SourceSchema; schema != nil {
		addFloat("minimum", schema.Minimum)
		addFloat("maximum", schema.Maximum)
		if schema.ExclusiveMinimum != nil && schema.ExclusiveMinimum.IsB() {
			value := schema.ExclusiveMinimum.B
			addFloat("exclusiveMinimum", &value)
		}
		if schema.ExclusiveMaximum != nil && schema.ExclusiveMaximum.IsB() {
			value := schema.ExclusiveMaximum.B
			addFloat("exclusiveMaximum", &value)
		}
		addFloat("multipleOf", schema.MultipleOf)
		addInt("minLength", schema.MinLength)
		addInt("maxLength", schema.MaxLength)
		add("pattern", schema.Pattern)
		addInt("minItems", schema.MinItems)
		addInt("maxItems", schema.MaxItems)
		if schema.UniqueItems != nil {
			parts = append(parts, "uniqueItems="+strconv.FormatBool(*schema.UniqueItems))
		}
		addInt("minProperties", schema.MinProperties)
		addInt("maxProperties", schema.MaxProperties)
	}
	return strings.Join(parts, ";")
}

func parseTagBool(value string, hasValue bool) (bool, bool) {
	if !hasValue {
		return true, true
	}
	parsed, err := strconv.ParseBool(value)
	return parsed, err == nil
}

func parseTagFloat(value string, hasValue bool) (float64, bool) {
	if !hasValue {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(value, 64)
	return parsed, err == nil
}

func parseTagInt(value string, hasValue bool) (int64, bool) {
	if !hasValue {
		return 0, false
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	return parsed, err == nil
}

func parseTagNodes(value string) []*yaml.Node {
	tokens := splitEscaped(value, '|')
	nodes := make([]*yaml.Node, 0, len(tokens))
	for _, token := range tokens {
		if node := parseTagNode(token); node != nil {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func parseTagNode(value string) *yaml.Node {
	kind, raw, ok := strings.Cut(value, ":")
	if !ok {
		return stringNode(unescapeOpenAPITagValue(value))
	}
	raw = unescapeOpenAPITagValue(raw)
	switch kind {
	case "str":
		return stringNode(raw)
	case "int":
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: raw}
	case "float":
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!float", Value: raw}
	case "bool":
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: raw}
	case "null":
		return nullNode()
	default:
		return stringNode(unescapeOpenAPITagValue(value))
	}
}

func encodeTagNodes(nodes []*yaml.Node) string {
	values := make([]string, 0, len(nodes))
	for _, node := range nodes {
		if encoded := encodeTagNode(node); encoded != "" {
			values = append(values, encoded)
		}
	}
	return strings.Join(values, "|")
}

func encodeTagNode(node *yaml.Node) string {
	if node == nil {
		return ""
	}
	if nodeIsNull(node) {
		return "null:"
	}
	if strings.Contains(node.Value, "`") {
		return ""
	}
	prefix := "str:"
	switch node.Tag {
	case "!!int":
		prefix = "int:"
	case "!!float":
		prefix = "float:"
	case "!!bool":
		prefix = "bool:"
	}
	return prefix + escapeOpenAPITagValue(node.Value)
}

func splitEscaped(value string, sep rune) []string {
	var out []string
	var b strings.Builder
	escaped := false
	for _, r := range value {
		if escaped {
			b.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			b.WriteRune(r)
			continue
		}
		if r == sep {
			out = append(out, b.String())
			b.Reset()
			continue
		}
		b.WriteRune(r)
	}
	out = append(out, b.String())
	return out
}

func cutEscaped(value string, sep rune) (string, string, bool) {
	escaped := false
	for i, r := range value {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if r == sep {
			return value[:i], value[i+len(string(r)):], true
		}
	}
	return value, "", false
}

func escapeOpenAPITagValue(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `;`, `\;`)
	value = strings.ReplaceAll(value, `=`, `\=`)
	value = strings.ReplaceAll(value, `|`, `\|`)
	return value
}

func unescapeOpenAPITagValue(value string) string {
	var b strings.Builder
	escaped := false
	for _, r := range value {
		if escaped {
			b.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		b.WriteRune(r)
	}
	if escaped {
		b.WriteByte('\\')
	}
	return b.String()
}

func boolPtr(value bool) *bool {
	return &value
}
