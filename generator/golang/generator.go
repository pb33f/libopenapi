// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"reflect"
	"sort"

	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
)

type Generator struct {
	packageName string

	optionalFieldsAsPointers         bool
	omitEmpty                        bool
	nullableAsPointer                bool
	jsonTags                         bool
	yamlTags                         bool
	enumConstants                    bool
	optionalConstDiscriminatorUnions bool
	generatedComment                 bool

	nameResolver   NameResolver
	headerComment  string
	packageComment string

	formatMappings map[string]formatMapping

	diagnostics []Diagnostic
	imports     map[string]struct{}
	decls       []string
	seenDecls   map[string]struct{}
	usedNames   map[string]struct{}

	openapiCache map[*highbase.SchemaProxy]*SchemaIR
	reflectCache map[reflect.Type]*SchemaIR
	reflectStack map[reflect.Type]bool

	componentNames   map[string]struct{}
	currentComponent string

	oneOfRegistrations         map[reflect.Type][]reflect.Type
	discriminatorRegistrations map[reflect.Type]discriminatorRegistration
}

type SchemaSet struct {
	Root        *highbase.SchemaProxy
	Components  *orderedmap.Map[string, *highbase.SchemaProxy]
	Diagnostics []Diagnostic
}

type File struct {
	PackageName string
	Source      []byte
	Types       []*Type
	Diagnostics []Diagnostic
}

type Type struct {
	Name string
	Kind Kind
}

type RenderResult = File

func NewGenerator(opts ...Option) *Generator {
	g := &Generator{
		packageName:                "models",
		optionalFieldsAsPointers:   true,
		omitEmpty:                  true,
		nullableAsPointer:          true,
		jsonTags:                   true,
		formatMappings:             make(map[string]formatMapping),
		imports:                    make(map[string]struct{}),
		seenDecls:                  make(map[string]struct{}),
		usedNames:                  make(map[string]struct{}),
		openapiCache:               make(map[*highbase.SchemaProxy]*SchemaIR),
		reflectCache:               make(map[reflect.Type]*SchemaIR),
		reflectStack:               make(map[reflect.Type]bool),
		oneOfRegistrations:         make(map[reflect.Type][]reflect.Type),
		discriminatorRegistrations: make(map[reflect.Type]discriminatorRegistration),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(g)
		}
	}
	return g
}

func RenderSchema(name string, schema *highbase.SchemaProxy, opts ...Option) ([]byte, error) {
	return NewGenerator(opts...).RenderSchema(name, schema)
}

func SchemaFromValue(value any, opts ...Option) (*highbase.SchemaProxy, error) {
	return NewGenerator(opts...).SchemaFromValue(value)
}

func SchemaFromType(t reflect.Type, opts ...Option) (*highbase.SchemaProxy, error) {
	return NewGenerator(opts...).SchemaFromType(t)
}

func SchemasFromValues(values ...any) (*SchemaSet, error) {
	return NewGenerator().SchemasFromValues(values...)
}

func SchemasFromTypes(types ...reflect.Type) (*SchemaSet, error) {
	return NewGenerator().SchemasFromTypes(types...)
}

func (g *Generator) RenderSchema(name string, schema *highbase.SchemaProxy) ([]byte, error) {
	if schema == nil {
		return nil, wrapPath(ErrNilSchema, name)
	}
	g.diagnostics = nil
	ir, err := g.irFromOpenAPI(name, schema, name)
	if err != nil {
		return nil, err
	}
	file, err := g.renderFile([]*SchemaIR{ir})
	if err != nil {
		return nil, err
	}
	return file.Source, nil
}

func (g *Generator) RenderSchemas(schemas *orderedmap.Map[string, *highbase.SchemaProxy]) (*File, error) {
	if err := validatePackageName(g.packageName); err != nil {
		return nil, err
	}
	g.diagnostics = nil
	if schemas == nil {
		return g.renderFile(nil)
	}
	irs := make([]*SchemaIR, 0, schemas.Len())
	for name, schema := range schemas.FromOldest() {
		ir, err := g.irFromOpenAPI(name, schema, name)
		if err != nil {
			return nil, err
		}
		irs = append(irs, ir)
	}
	return g.renderFile(irs)
}

func (g *Generator) SchemaFromValue(value any) (*highbase.SchemaProxy, error) {
	if value == nil {
		return nil, wrapPath(ErrNilType, "")
	}
	return g.SchemaFromType(reflect.TypeOf(value))
}

func (g *Generator) SchemaFromType(t reflect.Type) (*highbase.SchemaProxy, error) {
	if t == nil {
		return nil, wrapPath(ErrNilType, "")
	}
	ir, err := g.irFromReflect(derefType(t), typeName(derefType(t)), typeName(derefType(t)))
	if err != nil {
		return nil, err
	}
	return g.openapiFromIR(ir), nil
}

func (g *Generator) SchemasFromValues(values ...any) (*SchemaSet, error) {
	types := make([]reflect.Type, 0, len(values))
	for _, value := range values {
		if value == nil {
			return nil, wrapPath(ErrNilType, "")
		}
		types = append(types, reflect.TypeOf(value))
	}
	return g.SchemasFromTypes(types...)
}

func (g *Generator) SchemasFromTypes(types ...reflect.Type) (*SchemaSet, error) {
	g.diagnostics = nil
	g.reflectCache = make(map[reflect.Type]*SchemaIR)
	g.reflectStack = make(map[reflect.Type]bool)
	components := orderedmap.New[string, *highbase.SchemaProxy]()
	var root *highbase.SchemaProxy
	for i, t := range types {
		if t == nil {
			return nil, wrapPath(ErrNilType, "")
		}
		t = derefType(t)
		ir, err := g.irFromReflect(t, typeName(t), typeName(t))
		if err != nil {
			return nil, err
		}
		if i == 0 {
			root = g.openapiFromIR(ir)
			if ir.Name != "" {
				root = highbase.CreateSchemaProxyRef("#/components/schemas/" + ir.Name)
			}
		}
	}
	irs := make([]*SchemaIR, 0, len(g.reflectCache))
	for _, ir := range g.reflectCache {
		if ir != nil && ir.Name != "" && isComponentKind(ir.Kind) {
			irs = append(irs, ir)
		}
	}
	sortIRsByName(irs)
	componentNames := make(map[string]struct{}, len(irs))
	for _, ir := range irs {
		componentNames[ir.Name] = struct{}{}
	}
	previousNames := g.componentNames
	previousComponent := g.currentComponent
	g.componentNames = componentNames
	defer func() {
		g.componentNames = previousNames
		g.currentComponent = previousComponent
	}()
	for _, ir := range irs {
		if _, exists := components.Get(ir.Name); exists {
			g.addDiagnostic(ir.Name, "component name collision resolved by keeping first schema")
			continue
		}
		g.currentComponent = ir.Name
		components.Set(ir.Name, g.openapiFromIR(ir))
	}
	return &SchemaSet{
		Root:        root,
		Components:  components,
		Diagnostics: append([]Diagnostic(nil), g.diagnostics...),
	}, nil
}

func (g *Generator) addDiagnostic(path, message string) {
	g.diagnostics = append(g.diagnostics, Diagnostic{Path: path, Message: message})
}

func (g *Generator) addImport(path string) {
	if path != "" {
		g.imports[path] = struct{}{}
	}
}

func isComponentKind(kind Kind) bool {
	return kind == KindObject || kind == KindAllOf || kind == KindEnum || kind == KindUnion
}

func sortIRsByName(irs []*SchemaIR) {
	sort.SliceStable(irs, func(i, j int) bool {
		return irs[i].Name < irs[j].Name
	})
}
