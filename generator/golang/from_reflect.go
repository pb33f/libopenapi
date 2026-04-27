// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"reflect"

	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
)

type SchemaProvider interface {
	OpenAPISchema() *highbase.SchemaProxy
}

var schemaProviderType = reflect.TypeOf((*SchemaProvider)(nil)).Elem()

func (g *Generator) irFromReflect(t reflect.Type, name, path string) (*SchemaIR, error) {
	if t == nil {
		return nil, wrapPath(ErrNilType, path)
	}
	nullable := false
	for t.Kind() == reflect.Pointer {
		nullable = true
		t = t.Elem()
	}
	if t.Implements(schemaProviderType) || reflect.PointerTo(t).Implements(schemaProviderType) {
		return g.irFromSchemaProvider(t, name, path)
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

	var ir *SchemaIR
	var err error
	switch t.Kind() {
	case reflect.String:
		ir = &SchemaIR{Name: g.publicName(name), Kind: KindString, Nullable: nullable}
	case reflect.Bool:
		ir = &SchemaIR{Name: g.publicName(name), Kind: KindBoolean, Nullable: nullable}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		ir = &SchemaIR{Name: g.publicName(name), Kind: KindInteger, Nullable: nullable}
		if t.Kind() == reflect.Int32 {
			ir.Format = "int32"
		}
		if t.Kind() == reflect.Int64 {
			ir.Format = "int64"
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		ir = &SchemaIR{Name: g.publicName(name), Kind: KindInteger, Nullable: nullable}
	case reflect.Float32:
		ir = &SchemaIR{Name: g.publicName(name), Kind: KindNumber, Format: "float", Nullable: nullable}
	case reflect.Float64:
		ir = &SchemaIR{Name: g.publicName(name), Kind: KindNumber, Format: "double", Nullable: nullable}
	case reflect.Slice, reflect.Array:
		ir, err = g.irFromReflectArray(t, name, path, nullable)
	case reflect.Map:
		ir, err = g.irFromReflectMap(t, name, path, nullable)
	case reflect.Struct:
		ir, err = g.irFromReflectStruct(t, name, path, nullable)
	case reflect.Interface:
		ir, err = g.irFromReflectInterface(t, name, path, nullable)
	default:
		err = wrapPath(ErrUnsupportedType, path)
	}
	if err != nil {
		return nil, err
	}
	g.reflectCache[t] = ir
	return ir, nil
}

func (g *Generator) irFromSchemaProvider(t reflect.Type, name, path string) (*SchemaIR, error) {
	var provider SchemaProvider
	provider = reflect.New(t).Interface().(SchemaProvider)
	return g.irFromOpenAPI(name, provider.OpenAPISchema(), path)
}

func (g *Generator) irFromReflectArray(t reflect.Type, name, path string, nullable bool) (*SchemaIR, error) {
	item, err := g.irFromReflect(t.Elem(), name+"Item", path+"[]")
	if err != nil {
		return nil, err
	}
	return &SchemaIR{Name: g.publicName(name), Kind: KindArray, Items: item, Nullable: nullable}, nil
}

func (g *Generator) irFromReflectMap(t reflect.Type, name, path string, nullable bool) (*SchemaIR, error) {
	if t.Key().Kind() != reflect.String {
		return nil, wrapPath(ErrUnsupportedMapKey, path)
	}
	value, err := g.irFromReflect(t.Elem(), name+"Value", path+"{}")
	if err != nil {
		return nil, err
	}
	return &SchemaIR{Name: g.publicName(name), Kind: KindObject, AdditionalProperties: value, Nullable: nullable}, nil
}

func (g *Generator) irFromReflectStruct(t reflect.Type, name, path string, nullable bool) (*SchemaIR, error) {
	if t.PkgPath() == "time" && t.Name() == "Time" {
		return &SchemaIR{Name: g.publicName(name), Kind: KindString, Format: "date-time", Nullable: nullable}, nil
	}
	ir := newObjectIR(g.publicName(typeName(t)))
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
		child, err := g.irFromReflect(field.Type, ir.Name+g.publicName(tag.name), path+"."+field.Name)
		if err != nil {
			return nil, err
		}
		if field.Anonymous && field.Tag.Get("json") == "" && child.Kind == KindObject && child.Properties != nil {
			for propName, prop := range child.Properties.FromOldest() {
				ir.Properties.Set(propName, prop)
			}
			for req := range child.Required {
				ir.Required[req] = struct{}{}
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

func (g *Generator) irFromReflectInterface(t reflect.Type, name, path string, nullable bool) (*SchemaIR, error) {
	variants, ok := g.oneOfRegistrations[t]
	if !ok {
		return nil, wrapPath(ErrUnsupportedType, path)
	}
	ir := &SchemaIR{
		Name:     g.publicName(name),
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
