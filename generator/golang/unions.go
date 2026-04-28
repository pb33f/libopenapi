// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"sort"
	"strings"
)

func (g *Generator) renderUnionDecl(ir *SchemaIR) {
	if ir == nil || ir.Union == nil {
		return
	}
	if ir.Union.Strategy == UnionDiscriminator {
		g.renderDiscriminatedUnion(ir)
		return
	}
	g.renderRawUnion(ir)
}

func (g *Generator) renderRawUnion(ir *SchemaIR) {
	name := ir.Name + "Union"
	if !g.rememberDecl(name) {
		return
	}
	g.addImport("encoding/json")
	var b strings.Builder
	b.WriteString("type ")
	b.WriteString(name)
	b.WriteString(" struct {\n\tRaw json.RawMessage\n}\n\n")
	b.WriteString("func (u *")
	b.WriteString(name)
	b.WriteString(") UnmarshalJSON(data []byte) error {\n\tu.Raw = append(u.Raw[:0], data...)\n\treturn nil\n}\n\n")
	b.WriteString("func (u ")
	b.WriteString(name)
	b.WriteString(") MarshalJSON() ([]byte, error) {\n\tif len(u.Raw) == 0 {\n\t\treturn []byte(\"null\"), nil\n\t}\n\treturn u.Raw, nil\n}\n")
	b.WriteString("\nfunc (u ")
	b.WriteString(name)
	b.WriteString(") IsZero() bool {\n\treturn len(u.Raw) == 0\n}\n")
	b.WriteString("\nfunc (u ")
	b.WriteString(name)
	b.WriteString(") Bytes() []byte {\n\treturn append([]byte(nil), u.Raw...)\n}\n")
	g.decls = append(g.decls, b.String())
	g.recordSchemaMetadata(name, ir.SourceSchema)
}

func (g *Generator) renderDiscriminatedUnion(ir *SchemaIR) {
	if ir.Union == nil || ir.Union.Discriminator == nil {
		g.renderRawUnion(ir)
		return
	}
	for _, variant := range ir.Union.Variants {
		g.renderNested(variant)
	}
	if !g.rememberDecl(ir.Name + "Union") {
		return
	}
	g.addImport("encoding/json")
	g.addImport("fmt")
	var b strings.Builder
	b.WriteString("type ")
	b.WriteString(ir.Name)
	b.WriteString(" interface {\n\tis")
	b.WriteString(ir.Name)
	b.WriteString("()\n}\n\n")
	for _, variant := range ir.Union.Variants {
		if variant == nil || variant.Name == "" {
			continue
		}
		b.WriteString("func (")
		b.WriteString(variant.Name)
		b.WriteString(") is")
		b.WriteString(ir.Name)
		b.WriteString("() {}\n\n")
	}
	b.WriteString("type ")
	b.WriteString(ir.Name)
	b.WriteString("Union struct {\n\tValue ")
	b.WriteString(ir.Name)
	b.WriteString("\n}\n\n")
	b.WriteString("func (u ")
	b.WriteString(ir.Name)
	b.WriteString("Union) MarshalJSON() ([]byte, error) {\n\tif u.Value == nil {\n\t\treturn []byte(\"null\"), nil\n\t}\n\treturn json.Marshal(u.Value)\n}\n\n")
	b.WriteString("func (u ")
	b.WriteString(ir.Name)
	b.WriteString("Union) IsZero() bool {\n\treturn u.Value == nil\n}\n\n")
	b.WriteString("func (u *")
	b.WriteString(ir.Name)
	b.WriteString("Union) UnmarshalJSON(data []byte) error {\n\tvar discriminator struct {\n\t\tValue string `json:\"")
	b.WriteString(ir.Union.Discriminator.PropertyName)
	b.WriteString("\"`\n\t}\n\tif err := json.Unmarshal(data, &discriminator); err != nil {\n\t\treturn err\n\t}\n\tswitch discriminator.Value {\n")
	values := make([]string, 0, len(ir.Union.Discriminator.Mapping))
	for value := range ir.Union.Discriminator.Mapping {
		values = append(values, value)
	}
	sort.Strings(values)
	for _, value := range values {
		target := ir.Union.Discriminator.Mapping[value]
		typeName := target
		if strings.HasPrefix(target, "#") || strings.Contains(target, "/") {
			typeName = g.refTypeName(target)
		}
		b.WriteString("\tcase ")
		b.WriteString(strconvQuote(value))
		b.WriteString(":\n\t\tvar v ")
		b.WriteString(typeName)
		b.WriteString("\n\t\tif err := json.Unmarshal(data, &v); err != nil {\n\t\t\treturn err\n\t\t}\n\t\tu.Value = v\n")
	}
	b.WriteString("\tdefault:\n\t\treturn fmt.Errorf(\"unknown ")
	b.WriteString(ir.Union.Discriminator.PropertyName)
	b.WriteString(" discriminator value %q\", discriminator.Value)\n\t}\n\treturn nil\n}\n")
	g.decls = append(g.decls, b.String())
	g.recordSchemaMetadata(ir.Name+"Union", ir.SourceSchema)
}
