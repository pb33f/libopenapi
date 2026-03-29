// Copyright 2024-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package renderer

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"regexp"
	"unicode"

	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"go.yaml.in/yaml/v4"
)

// xmlNameRegex matches characters that are NOT valid in XML names.
var xmlNameRegex = regexp.MustCompile(`[^a-zA-Z0-9._\-:]`)

// sanitizeXMLName makes a string safe for use as an XML element or attribute name.
// Invalid characters are replaced with '_'. Names starting with a digit get a '_' prefix.
func sanitizeXMLName(name string) string {
	if name == "" {
		return "_"
	}
	s := xmlNameRegex.ReplaceAllString(name, "_")
	if len(s) > 0 && (unicode.IsDigit(rune(s[0])) || s[0] == '-' || s[0] == '.') {
		s = "_" + s
	}
	return s
}

// resolveNodeType determines the effective nodeType for a property schema, considering
// both the OpenAPI 3.2+ nodeType field and the deprecated attribute/wrapped fields.
func resolveNodeType(propSchema *highbase.Schema) string {
	if propSchema == nil || propSchema.XML == nil {
		return "element"
	}
	x := propSchema.XML
	if x.NodeType != "" {
		return x.NodeType
	}
	// Legacy backward compat
	if x.Attribute {
		return "attribute"
	}
	return "element"
}

// resolveElementName determines the XML element name for a property, using
// the XML name override if available, otherwise sanitizing the map key.
func resolveElementName(key string, propSchema *highbase.Schema) string {
	if propSchema != nil && propSchema.XML != nil && propSchema.XML.Name != "" {
		return propSchema.XML.Name
	}
	return sanitizeXMLName(key)
}

// getPropertySchema looks up the schema for a specific property name.
func getPropertySchema(parentSchema *highbase.Schema, key string) *highbase.Schema {
	if parentSchema == nil || parentSchema.Properties == nil {
		return nil
	}
	if proxy, ok := parentSchema.Properties.Get(key); ok && proxy != nil {
		return proxy.Schema()
	}
	return nil
}

// isWrappedArray returns true if an array schema should use a wrapper element.
// In OpenAPI 3.2+ this is nodeType "element"; legacy uses wrapped: true.
func isWrappedArray(schema *highbase.Schema) bool {
	if schema == nil || schema.XML == nil {
		return false
	}
	x := schema.XML
	if x.NodeType == "element" {
		return true
	}
	if x.NodeType == "" && x.Wrapped {
		return true
	}
	return false
}

// buildStartElement creates an xml.StartElement with optional namespace prefix handling.
func buildStartElement(name string, schema *highbase.Schema) xml.StartElement {
	local := name
	var attrs []xml.Attr

	if schema != nil && schema.XML != nil {
		x := schema.XML
		if x.Prefix != "" && x.Namespace != "" {
			local = x.Prefix + ":" + name
			attrs = appendNamespaceAttr(attrs, x.Prefix, x.Namespace)
		} else if x.Namespace != "" {
			attrs = appendNamespaceAttr(attrs, "", x.Namespace)
		}
	}

	return xml.StartElement{
		Name: xml.Name{Local: local},
		Attr: attrs,
	}
}

func appendNamespaceAttr(attrs []xml.Attr, prefix, namespace string) []xml.Attr {
	if namespace == "" {
		return attrs
	}
	attrName := "xmlns"
	if prefix != "" {
		attrName = "xmlns:" + prefix
	}
	for _, attr := range attrs {
		if attr.Name.Local == attrName {
			return attrs
		}
	}
	return append(attrs, xml.Attr{
		Name:  xml.Name{Local: attrName},
		Value: namespace,
	})
}

// RenderXML renders a value as XML. If schema is provided, uses its XML metadata
// (xml.name, xml.attribute, xml.namespace, xml.prefix, xml.wrapped) for correct output.
// If schema is nil, falls back to basic element-based XML (map keys → element names).
//
// Note: nodeType "cdata" is treated as "text" in this version — Go's xml.Encoder has
// no first-class CDATA token support.
func (mg *MockGenerator) RenderXML(value any, schema *highbase.Schema) []byte {
	if value == nil {
		return nil
	}

	// Decode *yaml.Node to native Go types
	if y, ok := value.(*yaml.Node); ok {
		var decoded any
		if err := y.Decode(&decoded); err != nil {
			return nil
		}
		value = decoded
	}

	var buf bytes.Buffer
	buf.Grow(512)
	enc := xml.NewEncoder(&buf)
	if mg.pretty {
		enc.Indent("", "  ")
	}

	// XML declaration
	_ = enc.EncodeToken(xml.ProcInst{Target: "xml", Inst: []byte(`version="1.0" encoding="UTF-8"`)})
	if mg.pretty {
		_ = enc.EncodeToken(xml.CharData("\n"))
	}

	// Root element name
	rootName := "root"
	if schema != nil && schema.XML != nil && schema.XML.Name != "" {
		rootName = schema.XML.Name
	}

	start := buildStartElement(rootName, schema)

	mg.renderXMLValue(enc, start, value, schema)

	_ = enc.Flush()
	return buf.Bytes()
}

// renderXMLValue recursively renders a value as XML tokens.
func (mg *MockGenerator) renderXMLValue(enc *xml.Encoder, start xml.StartElement, value any, schema *highbase.Schema) {
	if value == nil {
		return
	}

	switch v := value.(type) {
	case map[string]any:
		mg.renderXMLMap(enc, start, v, schema)
	case []any:
		mg.renderXMLSlice(enc, start, v, schema)
	default:
		// Scalar value
		_ = enc.EncodeToken(start)
		_ = enc.EncodeToken(xml.CharData(fmt.Sprint(v)))
		_ = enc.EncodeToken(start.End())
	}
}

// renderXMLMap renders a map as an XML element with child elements, attributes, and text content.
func (mg *MockGenerator) renderXMLMap(enc *xml.Encoder, start xml.StartElement, m map[string]any, schema *highbase.Schema) {
	// Three-pass rendering:
	// 1. Collect attributes → add to start element
	// 2. Collect text/cdata nodes
	// 3. Emit child elements

	type childEntry struct {
		key    string
		value  any
		schema *highbase.Schema
	}

	var textValues []any
	var children []childEntry

	for key, val := range m {
		propSchema := getPropertySchema(schema, key)
		nodeType := resolveNodeType(propSchema)

		switch nodeType {
		case "attribute":
			attrName := resolveElementName(key, propSchema)
			// Apply prefix for attributes too
			if propSchema != nil && propSchema.XML != nil && propSchema.XML.Prefix != "" {
				attrName = propSchema.XML.Prefix + ":" + attrName
				start.Attr = appendNamespaceAttr(start.Attr, propSchema.XML.Prefix, propSchema.XML.Namespace)
			}
			start.Attr = append(start.Attr, xml.Attr{
				Name:  xml.Name{Local: attrName},
				Value: fmt.Sprint(val),
			})
		case "text", "cdata":
			textValues = append(textValues, val)
		case "none":
			// Skip the node itself, include sub-properties directly
			if subMap, ok := val.(map[string]any); ok {
				for sk, sv := range subMap {
					children = append(children, childEntry{key: sk, value: sv, schema: getPropertySchema(propSchema, sk)})
				}
			} else {
				children = append(children, childEntry{key: key, value: val, schema: propSchema})
			}
		default: // "element"
			children = append(children, childEntry{key: key, value: val, schema: propSchema})
		}
	}

	_ = enc.EncodeToken(start)

	// Emit text content
	for _, tv := range textValues {
		_ = enc.EncodeToken(xml.CharData(fmt.Sprint(tv)))
	}

	// Emit child elements
	for _, child := range children {
		elemName := resolveElementName(child.key, child.schema)
		childStart := buildStartElement(elemName, child.schema)

		// Handle arrays
		if arr, ok := child.value.([]any); ok {
			mg.renderXMLArray(enc, childStart, arr, child.schema, child.key)
		} else {
			mg.renderXMLValue(enc, childStart, child.value, child.schema)
		}
	}

	_ = enc.EncodeToken(start.End())
}

// renderXMLSlice renders a top-level slice (when the root value is an array).
func (mg *MockGenerator) renderXMLSlice(enc *xml.Encoder, start xml.StartElement, arr []any, schema *highbase.Schema) {
	_ = enc.EncodeToken(start)
	itemSchema := mg.getItemsSchema(schema)
	itemName := "item"
	if itemSchema != nil && itemSchema.XML != nil && itemSchema.XML.Name != "" {
		itemName = itemSchema.XML.Name
	}
	for _, item := range arr {
		itemStart := buildStartElement(itemName, itemSchema)
		mg.renderXMLValue(enc, itemStart, item, itemSchema)
	}
	_ = enc.EncodeToken(start.End())
}

// renderXMLArray renders an array property, handling wrapped vs unwrapped.
func (mg *MockGenerator) renderXMLArray(enc *xml.Encoder, elemStart xml.StartElement, arr []any, propSchema *highbase.Schema, key string) {
	itemSchema := mg.getItemsSchema(propSchema)
	itemName := "item"
	if itemSchema != nil && itemSchema.XML != nil && itemSchema.XML.Name != "" {
		itemName = itemSchema.XML.Name
	}

	if isWrappedArray(propSchema) {
		// Wrapped: <wrapper><item/><item/>...</wrapper>
		_ = enc.EncodeToken(elemStart)
		for _, item := range arr {
			itemStart := buildStartElement(itemName, itemSchema)
			mg.renderXMLValue(enc, itemStart, item, itemSchema)
		}
		_ = enc.EncodeToken(elemStart.End())
	} else {
		// Unwrapped: repeated elements directly under parent
		for _, item := range arr {
			itemStart := buildStartElement(itemName, itemSchema)
			mg.renderXMLValue(enc, itemStart, item, itemSchema)
		}
	}
}

// getItemsSchema extracts the items schema from an array schema.
func (mg *MockGenerator) getItemsSchema(schema *highbase.Schema) *highbase.Schema {
	if schema == nil || schema.Items == nil || !schema.Items.IsA() {
		return nil
	}
	return schema.Items.A.Schema()
}
