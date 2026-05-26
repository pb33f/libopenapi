// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"

	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
)

func stringSet(names ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(names))
	for _, name := range names {
		set[name] = struct{}{}
	}
	return set
}

// fillSentinel sets a non-zero value on every kind of field present on
// highbase.Schema so a round trip through applySchemaFidelity reveals which
// fields are propagated.
func fillSentinel(v reflect.Value) {
	switch v.Kind() {
	case reflect.String:
		v.SetString("sentinel")
	case reflect.Pointer:
		v.Set(reflect.New(v.Type().Elem()))
	case reflect.Slice:
		v.Set(reflect.MakeSlice(v.Type(), 1, 1))
	}
}

// TestApplySchemaFidelityCoversSchemaFields is a tripwire for the Schema hash
// contract applied to fidelity: every exported highbase.Schema field must
// either be propagated verbatim by applySchemaFidelity or be consciously
// listed as shape-derived (set from IR structure in openapiFromIR) or as a
// non-content navigation field. When libopenapi adds a Schema field this test
// fails until the new field is handled, instead of silently losing fidelity.
func TestApplySchemaFidelityCoversSchemaFields(t *testing.T) {
	shapeOrIgnored := stringSet(
		// Shape-derived: openapiFromIR sets these from the IR structure.
		"Type", "AllOf", "AnyOf", "OneOf", "Discriminator", "Properties",
		"PatternProperties", "PrefixItems", "AdditionalProperties", "Required",
		"Enum", "Const", "Nullable", "Items",
		// Navigation only, no schema content.
		"ParentProxy",
	)

	src := &highbase.Schema{}
	sv := reflect.ValueOf(src).Elem()
	for i := 0; i < sv.NumField(); i++ {
		if sv.Type().Field(i).PkgPath == "" {
			fillSentinel(sv.Field(i))
		}
	}

	target := &highbase.Schema{}
	applySchemaFidelity(target, &SchemaIR{SourceSchema: src})

	tv := reflect.ValueOf(target).Elem()
	for i := 0; i < tv.NumField(); i++ {
		field := tv.Type().Field(i)
		if field.PkgPath != "" {
			continue
		}
		_, exempt := shapeOrIgnored[field.Name]
		copied := !tv.Field(i).IsZero()
		switch {
		case !copied && !exempt:
			t.Fatalf("applySchemaFidelity does not propagate schema field %q; copy it in applySchemaFidelity or record it as shape-derived", field.Name)
		case copied && exempt:
			t.Fatalf("schema field %q is propagated by applySchemaFidelity but listed as shape-derived; update the allowlist", field.Name)
		}
	}
}

// emittedStructFields parses the sidecar struct definitions and returns each
// emitted struct's field-name to field-type mapping.
func emittedStructFields(t *testing.T) map[string]map[string]string {
	t.Helper()
	var b strings.Builder
	writeSchemaMetadataTypes(&b)
	file, err := parser.ParseFile(token.NewFileSet(), "", "package golang\n"+b.String(), 0)
	if err != nil {
		t.Fatalf("emitted metadata struct definitions do not parse: %v", err)
	}
	out := make(map[string]map[string]string)
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			ts := spec.(*ast.TypeSpec)
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			fields := make(map[string]string)
			for _, field := range st.Fields.List {
				for _, name := range field.Names {
					fields[name.Name] = exprString(field.Type)
				}
			}
			out[ts.Name.Name] = fields
		}
	}
	return out
}

func exprString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + exprString(e.X)
	case *ast.ArrayType:
		return "[]" + exprString(e.Elt)
	default:
		return "?"
	}
}

// providerTypeToEmitted maps a reflect type string for a read-side metadata
// type onto the name the sidecar emits for it (provider -> openAPI prefix, no
// package qualifier).
func providerTypeToEmitted(reflectType string) string {
	return strings.ReplaceAll(reflectType, "golang.provider", "openAPI")
}

func collectProviderTypes(t reflect.Type, seen map[string]reflect.Type) {
	for t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || !strings.HasPrefix(t.Name(), "provider") {
		return
	}
	if _, ok := seen[t.Name()]; ok {
		return
	}
	seen[t.Name()] = t
	for i := 0; i < t.NumField(); i++ {
		collectProviderTypes(t.Field(i).Type, seen)
	}
}

// TestSchemaMetadataMirrorParity guards the two hand-maintained metadata
// hierarchies against drift: the read-side providerSchemaMetadata structs
// (schema_metadata.go) and the openAPISchemaMetadata structs emitted into the
// generated sidecar (provider_methods.go) must define identical type and field
// sets, or a generated round trip silently loses metadata.
func TestSchemaMetadataMirrorParity(t *testing.T) {
	emitted := emittedStructFields(t)

	providerTypes := make(map[string]reflect.Type)
	collectProviderTypes(reflect.TypeOf(providerSchemaMetadata{}), providerTypes)

	if len(providerTypes) != len(emitted) {
		t.Fatalf("metadata type count mismatch: read=%d emitted=%d", len(providerTypes), len(emitted))
	}

	for name, rt := range providerTypes {
		emittedName := "openAPI" + strings.TrimPrefix(name, "provider")
		emittedFields, ok := emitted[emittedName]
		if !ok {
			t.Fatalf("emitted sidecar is missing type %q for read type %q", emittedName, name)
		}
		if rt.NumField() != len(emittedFields) {
			t.Fatalf("type %q field count mismatch: read=%d emitted=%d", name, rt.NumField(), len(emittedFields))
		}
		for i := 0; i < rt.NumField(); i++ {
			field := rt.Field(i)
			emittedType, ok := emittedFields[field.Name]
			if !ok {
				t.Fatalf("emitted type %q is missing field %q", emittedName, field.Name)
			}
			if want := providerTypeToEmitted(field.Type.String()); want != emittedType {
				t.Fatalf("type %q field %q type mismatch: read=%s emitted=%s", name, field.Name, want, emittedType)
			}
		}
	}
}
