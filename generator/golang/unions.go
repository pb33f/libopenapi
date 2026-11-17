// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"sort"
	"strconv"
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
		if strings.HasPrefix(target, "#") || strings.Contains(target, "/") || strings.Contains(target, ".") {
			typeName = g.refTypeName(target)
		}
		b.WriteString("\tcase ")
		b.WriteString(strconv.Quote(value))
		b.WriteString(":\n\t\tvar v ")
		b.WriteString(typeName)
		b.WriteString("\n\t\tif err := json.Unmarshal(data, &v); err != nil {\n\t\t\treturn err\n\t\t}\n\t\tu.Value = v\n")
	}
	if unknown := ir.Union.UnknownName; unknown != "" {
		b.WriteString("\tcase \"\":\n\t\treturn fmt.Errorf(\"missing ")
		b.WriteString(ir.Union.Discriminator.PropertyName)
		b.WriteString(" discriminator\")\n\tdefault:\n\t\tu.Value = ")
		b.WriteString(unknown)
		b.WriteString("{Discriminator: discriminator.Value, Raw: append(json.RawMessage(nil), data...)}\n\t}\n\treturn nil\n}\n")
		b.WriteString(unknownVariantDecl(ir.Name, unknown))
	} else {
		b.WriteString("\tdefault:\n\t\treturn fmt.Errorf(\"unknown ")
		b.WriteString(ir.Union.Discriminator.PropertyName)
		b.WriteString(" discriminator value %q\", discriminator.Value)\n\t}\n\treturn nil\n}\n")
	}
	g.decls = append(g.decls, b.String())
	g.recordSchemaMetadata(ir.Name+"Union", ir.SourceSchema)
}

// claimUnknownVariant reserves the name of the variant that keeps unmapped
// discriminator values, so it cannot collide with a document schema.
func (g *Generator) claimUnknownVariant(ir *SchemaIR) {
	if !g.unknownDiscriminatorVariants || ir.Union == nil || ir.Name == "" {
		return
	}
	preferred := ir.Name + "Unknown"
	if g.typeNames == nil {
		ir.Union.UnknownName = preferred
		return
	}
	ir.Union.UnknownName = g.typeNames.Claim(preferred, "")
	if ir.Union.UnknownName != preferred {
		g.addDiagnostic(DiagnosticTypeNameCollision, ir.Name, "unknown discriminator variant named "+ir.Union.UnknownName+" because "+preferred+" is taken")
	}
}

// unknownVariantDecl renders the variant that carries a discriminator value
// the union's mapping does not name. It marshals its raw JSON back unchanged.
func unknownVariantDecl(union, name string) string {
	var b strings.Builder
	b.WriteString("\n// ")
	b.WriteString(name)
	b.WriteString(" carries a ")
	b.WriteString(union)
	b.WriteString(" variant this client was not generated for.\ntype ")
	b.WriteString(name)
	b.WriteString(" struct {\n\t// Discriminator is the unmapped discriminator value.\n\tDiscriminator string\n\t// Raw is the variant's JSON exactly as received.\n\tRaw json.RawMessage\n}\n\nfunc (")
	b.WriteString(name)
	b.WriteString(") is")
	b.WriteString(union)
	b.WriteString("() {}\n\nfunc (u ")
	b.WriteString(name)
	b.WriteString(") MarshalJSON() ([]byte, error) {\n\tif len(u.Raw) == 0 {\n\t\treturn []byte(\"null\"), nil\n\t}\n\treturn u.Raw, nil\n}\n")
	return b.String()
}
