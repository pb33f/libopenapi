// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/pb33f/libopenapi"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
)

type SchemaProvider interface {
	OpenAPISchema() *highbase.SchemaProxy
}

type SchemaYAMLProvider interface {
	OpenAPISchemaYAML() string
}

var schemaProviderType = reflect.TypeOf((*SchemaProvider)(nil)).Elem()
var schemaMetadataProviderType = reflect.TypeOf((*SchemaMetadataProvider)(nil)).Elem()
var schemaYAMLProviderType = reflect.TypeOf((*SchemaYAMLProvider)(nil)).Elem()
var rawMessageType = reflect.TypeOf(json.RawMessage{})

func (g *Generator) irFromReflect(t reflect.Type, name, path string) (*SchemaIR, error) {
	return g.irFromReflectName(t, name, false, path)
}

func (g *Generator) irFromReflectName(t reflect.Type, name string, nameResolved bool, path string) (*SchemaIR, error) {
	if t == nil {
		return nil, wrapPath(ErrNilType, path)
	}
	resolvedName := name
	if !nameResolved {
		resolvedName = g.publicName(name)
	}
	nullable := false
	for t.Kind() == reflect.Pointer {
		nullable = true
		t = t.Elem()
	}
	if schema := g.typeSchemas[t]; schema != nil {
		return g.irFromTypeSchema(t, resolvedName, path, schema, nullable)
	}
	if t == rawMessageType {
		return &SchemaIR{Name: resolvedName, Kind: KindAny, Nullable: nullable}, nil
	}
	if g.reflectStack[t] {
		return &SchemaIR{
			Name:     g.publicName(typeName(t)),
			Ref:      "#/components/schemas/" + g.publicName(typeName(t)),
			Kind:     KindRef,
			Nullable: nullable,
		}, nil
	}
	if cached := g.reflectCache[t]; cached != nil {
		cp := *cached
		cp.Nullable = cp.Nullable || nullable
		return &cp, nil
	}
	if t.Kind() != reflect.Interface && implementsOrPointerImplements(t, schemaMetadataProviderType) {
		return g.irFromSchemaMetadataProvider(t, resolvedName, path, nullable)
	}
	if t.Kind() != reflect.Interface && implementsOrPointerImplements(t, schemaProviderType) {
		return g.irFromSchemaProvider(t, resolvedName, path, nullable)
	}
	if t.Kind() != reflect.Interface && implementsOrPointerImplements(t, schemaYAMLProviderType) {
		return g.irFromSchemaYAMLProvider(t, resolvedName, path, nullable)
	}

	var ir *SchemaIR
	var err error
	switch t.Kind() {
	case reflect.String:
		ir = &SchemaIR{Name: resolvedName, Kind: KindString}
	case reflect.Bool:
		ir = &SchemaIR{Name: resolvedName, Kind: KindBoolean}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		ir = &SchemaIR{Name: resolvedName, Kind: KindInteger}
		if t.Kind() == reflect.Int32 {
			ir.Format = "int32"
		}
		if t.Kind() == reflect.Int64 {
			ir.Format = "int64"
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		ir = &SchemaIR{Name: resolvedName, Kind: KindInteger}
	case reflect.Float32:
		ir = &SchemaIR{Name: resolvedName, Kind: KindNumber, Format: "float"}
	case reflect.Float64:
		ir = &SchemaIR{Name: resolvedName, Kind: KindNumber, Format: "double"}
	case reflect.Slice, reflect.Array:
		ir, err = g.irFromReflectArray(t, resolvedName, path, false)
	case reflect.Map:
		ir, err = g.irFromReflectMap(t, resolvedName, path, false)
	case reflect.Struct:
		ir, err = g.irFromReflectStruct(t, resolvedName, path, false)
	case reflect.Interface:
		ir, err = g.irFromReflectInterface(t, resolvedName, path, false)
	default:
		err = wrapPath(ErrUnsupportedType, path)
	}
	if err != nil {
		return nil, err
	}
	g.reflectCache[t] = ir
	if nullable {
		cp := *ir
		cp.Nullable = true
		return &cp, nil
	}
	return ir, nil
}

func (g *Generator) irFromTypeSchema(t reflect.Type, name, path string, schema *highbase.SchemaProxy, nullable bool) (*SchemaIR, error) {
	schemaName := name
	if t.Name() != "" {
		schemaName = typeName(t)
	}
	ir, err := g.irFromOpenAPI(schemaName, schema, path)
	if err != nil {
		return nil, err
	}
	base := *ir
	base.ExactSource = true
	g.reflectCache[t] = &base
	if nullable {
		cp := base
		cp.Nullable = true
		return &cp, nil
	}
	return &base, nil
}

func (g *Generator) irFromSchemaProvider(t reflect.Type, name, path string, nullable bool) (*SchemaIR, error) {
	provider := providerValue(t).(SchemaProvider)
	schemaName := name
	if t.Name() != "" {
		schemaName = g.publicName(typeName(t))
	}
	ir, err := g.irFromOpenAPI(schemaName, provider.OpenAPISchema(), path)
	if err != nil {
		return nil, err
	}
	base := *ir
	base.ExactSource = true
	g.reflectCache[t] = &base
	if nullable {
		cp := base
		cp.Nullable = true
		return &cp, nil
	}
	return &base, nil
}

func (g *Generator) irFromSchemaMetadataProvider(t reflect.Type, name, path string, nullable bool) (*SchemaIR, error) {
	provider := providerValue(t).(SchemaMetadataProvider)
	proxy, err := schemaProxyFromProviderMetadata(provider.OpenAPISchemaMetadata())
	if err != nil {
		return nil, wrapPath(err, path)
	}
	schemaName := name
	if t.Name() != "" {
		schemaName = g.publicName(typeName(t))
	}
	ir, _ := g.irFromOpenAPI(schemaName, proxy, path)
	base := *ir
	base.ExactSource = true
	g.reflectCache[t] = &base
	if nullable {
		cp := base
		cp.Nullable = true
		return &cp, nil
	}
	return &base, nil
}

func (g *Generator) irFromSchemaYAMLProvider(t reflect.Type, name, path string, nullable bool) (*SchemaIR, error) {
	provider := providerValue(t).(SchemaYAMLProvider)
	schemaName := name
	if t.Name() != "" {
		schemaName = g.publicName(typeName(t))
	}
	proxy, err := schemaProxyFromProviderYAML(schemaName, provider.OpenAPISchemaYAML())
	if err != nil {
		return nil, wrapPath(err, path)
	}
	ir, _ := g.irFromOpenAPI(schemaName, proxy, path)
	base := *ir
	base.ExactSource = true
	g.reflectCache[t] = &base
	if nullable {
		cp := base
		cp.Nullable = true
		return &cp, nil
	}
	return &base, nil
}

func (g *Generator) irFromReflectArray(t reflect.Type, name, path string, nullable bool) (*SchemaIR, error) {
	if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8 {
		return &SchemaIR{Name: name, Kind: KindString, Format: "byte", Nullable: nullable}, nil
	}
	item, err := g.irFromReflectName(t.Elem(), g.nestedTypeName(name, "Item"), true, path+"[]")
	if err != nil {
		return nil, err
	}
	return &SchemaIR{Name: name, Kind: KindArray, Items: item, Nullable: nullable}, nil
}

func (g *Generator) irFromReflectMap(t reflect.Type, name, path string, nullable bool) (*SchemaIR, error) {
	if t.Key().Kind() != reflect.String {
		return nil, wrapPath(ErrUnsupportedMapKey, path)
	}
	value, err := g.irFromReflectName(t.Elem(), g.nestedTypeName(name, "Value"), true, path+"{}")
	if err != nil {
		return nil, err
	}
	return &SchemaIR{Name: name, Kind: KindObject, AdditionalProperties: value, Nullable: nullable}, nil
}

func (g *Generator) irFromReflectStruct(t reflect.Type, name, path string, nullable bool) (*SchemaIR, error) {
	if t.PkgPath() == "time" && t.Name() == "Time" {
		return &SchemaIR{Name: name, Kind: KindString, Format: "date-time", Nullable: nullable}, nil
	}
	structName := name
	if t.Name() != "" {
		structName = g.publicName(typeName(t))
	}
	ir := newObjectIR(structName)
	ir.Nullable = nullable
	g.reflectCache[t] = ir
	g.reflectStack[t] = true
	defer delete(g.reflectStack, t)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" && !field.Anonymous {
			continue
		}
		tag := parseJSONTag(field)
		if tag.skip {
			continue
		}
		child, err := g.irFromReflectField(t, field, tag, g.nestedTypeName(ir.Name, tag.name), path+"."+field.Name)
		if err != nil {
			return nil, err
		}
		if tag.stringEncoded {
			child = g.stringEncodedIR(child, path+"."+field.Name)
		}
		if tag.openapi.Present {
			child = cloneIR(child)
		}
		g.applyOpenAPIMetadata(child, tag.openapi)
		if field.Anonymous && !tag.hasName && child.Kind == KindObject && child.Properties != nil {
			for propName, prop := range child.Properties.FromOldest() {
				ir.Properties.Set(propName, prop)
			}
			if !tag.omitempty && field.Type.Kind() != reflect.Pointer {
				for req := range child.Required {
					ir.Required[req] = struct{}{}
				}
			}
			continue
		}
		ir.Properties.Set(tag.name, child)
		if !tag.omitempty {
			ir.Required[tag.name] = struct{}{}
		}
	}
	return ir, nil
}

func cloneIR(ir *SchemaIR) *SchemaIR {
	if ir == nil {
		return nil
	}
	cp := *ir
	if ir.SourceSchema != nil {
		schema := *ir.SourceSchema
		cp.SourceSchema = &schema
	}
	return &cp
}

func (g *Generator) irFromReflectField(owner reflect.Type, field reflect.StructField, tag fieldTag, name, path string) (*SchemaIR, error) {
	if schema := g.fieldSchema(owner, field, tag.name); schema != nil {
		return g.irFromFieldSchema(field.Type, name, path, schema)
	}
	return g.irFromReflectName(field.Type, name, true, path)
}

func (g *Generator) fieldSchema(owner reflect.Type, field reflect.StructField, jsonName string) *highbase.SchemaProxy {
	owner = derefType(owner)
	if g.fieldSchemas != nil {
		if schema := g.fieldSchemas[fieldSchemaKey{owner: owner, name: field.Name}]; schema != nil {
			return schema
		}
	}
	if g.jsonSchemas != nil {
		return g.jsonSchemas[fieldSchemaKey{owner: owner, name: jsonName}]
	}
	return nil
}

func (g *Generator) irFromFieldSchema(fieldType reflect.Type, name, path string, schema *highbase.SchemaProxy) (*SchemaIR, error) {
	nullable := false
	for fieldType.Kind() == reflect.Pointer {
		nullable = true
		fieldType = fieldType.Elem()
	}
	ir, err := g.irFromOpenAPIName(name, true, schema, path)
	if err != nil {
		return nil, err
	}
	cp := *ir
	cp.Nullable = cp.Nullable || nullable
	return &cp, nil
}

func (g *Generator) stringEncodedIR(ir *SchemaIR, path string) *SchemaIR {
	if ir == nil {
		return nil
	}
	if ir.Kind != KindString && ir.Kind != KindInteger && ir.Kind != KindNumber && ir.Kind != KindBoolean {
		g.addDiagnostic(DiagnosticStringEncoded, path, "json string option is only modeled for scalar fields")
		return ir
	}
	cp := *ir
	cp.Kind = KindString
	cp.Format = ""
	cp.Comments = append(cp.Comments, "encoded as a JSON string")
	g.addDiagnostic(DiagnosticStringEncoded, path, "json string option rendered as OpenAPI string schema")
	return &cp
}

func (g *Generator) irFromReflectInterface(t reflect.Type, name, path string, nullable bool) (*SchemaIR, error) {
	variants, ok := g.oneOfRegistrations[t]
	if !ok {
		return nil, wrapPath(ErrUnsupportedType, path)
	}
	ir := &SchemaIR{
		Name:     name,
		Kind:     KindUnion,
		Nullable: nullable,
		Union:    &UnionIR{Kind: UnionOneOf, Strategy: UnionRawMessage},
	}
	for _, variantType := range variants {
		variantIR, err := g.irFromReflect(variantType, typeName(variantType), path+"."+typeName(variantType))
		if err != nil {
			return nil, err
		}
		ir.Union.Variants = append(ir.Union.Variants, &SchemaIR{
			Name: variantIR.Name,
			Ref:  "#/components/schemas/" + variantIR.Name,
			Kind: KindRef,
		})
	}
	if reg, ok := g.discriminatorRegistrations[t]; ok {
		ir.Union.Strategy = UnionDiscriminator
		ir.Union.Discriminator = &Discriminator{
			PropertyName: reg.property,
			Mapping:      reg.mapping,
		}
	}
	return ir, nil
}

func implementsOrPointerImplements(t reflect.Type, iface reflect.Type) bool {
	return t.Implements(iface) || reflect.PointerTo(t).Implements(iface)
}

func providerValue(t reflect.Type) any {
	return reflect.New(t).Interface()
}

func schemaProxyFromProviderYAML(name, schemaYAML string) (*highbase.SchemaProxy, error) {
	if name == "" {
		name = "Schema"
	}
	var b strings.Builder
	b.WriteString("openapi: 3.1.0\n")
	b.WriteString("info:\n")
	b.WriteString("  title: Generated Schema Provider\n")
	b.WriteString("  version: 1.0.0\n")
	b.WriteString("paths: {}\n")
	b.WriteString("components:\n")
	b.WriteString("  schemas:\n")
	b.WriteString("    ")
	b.WriteString(name)
	b.WriteString(":\n")
	b.WriteString(indentSchemaYAML(schemaYAML, "      "))
	doc, err := libopenapi.NewDocument([]byte(b.String()))
	if err != nil {
		return nil, err
	}
	model, _ := doc.BuildV3Model()
	schema, _ := model.Model.Components.Schemas.Get(name)
	return schema, nil
}

func indentSchemaYAML(in, prefix string) string {
	lines := strings.Split(strings.TrimSuffix(in, "\n"), "\n")
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(prefix)
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}
