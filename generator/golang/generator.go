// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"reflect"
	"sort"

	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
)

// Generator holds immutable configuration for code generation. Each public
// entry point runs against a fresh copy of this configuration (see run), so a
// configured Generator carries no per-invocation state and is safe to reuse for
// many documents and to share across goroutines.
type Generator struct {
	packageName string

	optionalFieldsAsPointers         bool
	omitEmpty                        bool
	nullableAsPointer                bool
	jsonTags                         bool
	yamlTags                         bool
	enumConstants                    bool
	optionalConstDiscriminatorUnions bool
	additionalPropertiesMethods      bool
	generatedComment                 bool
	openapiTags                      bool
	schemaMetadataSidecar            bool
	nestedTypeNameDelimiter          string

	nameResolver          NameResolver
	typeNameResolver      NameResolver
	fieldNameResolver     NameResolver
	enumValueNameResolver NameResolver
	externalRefResolver   ExternalRefResolver
	headerComment         string
	packageComment        string

	formatMappings map[string]formatMapping
	typeSchemas    map[reflect.Type]*highbase.SchemaProxy
	fieldSchemas   map[fieldSchemaKey]*highbase.SchemaProxy
	jsonSchemas    map[fieldSchemaKey]*highbase.SchemaProxy

	diagnostics     []Diagnostic
	imports         map[string]struct{}
	decls           []string
	seenDecls       map[string]struct{}
	metadataSchemas map[string]*highbase.Schema
	metadataOrder   []string

	openapiCache map[*highbase.SchemaProxy]*SchemaIR
	reflectCache map[reflect.Type]*SchemaIR
	reflectStack map[reflect.Type]bool
	typeNames    *nameRegistry

	componentNames     map[string]struct{}
	componentTypeNames map[string]string
	componentKinds     map[string]Kind
	currentComponent   string

	oneOfRegistrations         map[reflect.Type][]reflect.Type
	discriminatorRegistrations map[reflect.Type]discriminatorRegistration
}

// SchemaSet contains OpenAPI schemas generated from one or more Go types.
type SchemaSet struct {
	// Root is the first generated root schema, kept as a convenience for
	// single-root callers.
	Root *highbase.SchemaProxy
	// Roots contains every requested root schema keyed by generated type name.
	Roots *orderedmap.Map[string, *highbase.SchemaProxy]
	// Components contains reusable schemas discovered while walking the root
	// graph.
	Components *orderedmap.Map[string, *highbase.SchemaProxy]
	// Diagnostics reports schema features that required a lossy or notable
	// model-generation decision.
	Diagnostics []Diagnostic
}

const SchemaMetadataFileName = "schema_metadata.go"

// GeneratedFile contains Go source generated from OpenAPI schemas.
type GeneratedFile struct {
	PackageName    string
	Source         []byte
	SchemaMetadata *GeneratedSourceFile
	Types          []*GeneratedType
	Diagnostics    []Diagnostic
}

// GeneratedSourceFile contains a named generated source file.
type GeneratedSourceFile struct {
	Name   string
	Source []byte
}

// GeneratedType describes one top-level generated Go type.
type GeneratedType struct {
	Name string
	Kind Kind
}

// NewGenerator creates a Go model generator.
func NewGenerator(opts ...Option) *Generator {
	g := &Generator{
		packageName:                 "models",
		optionalFieldsAsPointers:    true,
		omitEmpty:                   true,
		nullableAsPointer:           true,
		additionalPropertiesMethods: true,
		nestedTypeNameDelimiter:     "_",
		jsonTags:                    true,
		formatMappings:              make(map[string]formatMapping),
		typeSchemas:                 make(map[reflect.Type]*highbase.SchemaProxy),
		fieldSchemas:                make(map[fieldSchemaKey]*highbase.SchemaProxy),
		jsonSchemas:                 make(map[fieldSchemaKey]*highbase.SchemaProxy),
		imports:                     make(map[string]struct{}),
		seenDecls:                   make(map[string]struct{}),
		metadataSchemas:             make(map[string]*highbase.Schema),
		openapiCache:                make(map[*highbase.SchemaProxy]*SchemaIR),
		reflectCache:                make(map[reflect.Type]*SchemaIR),
		reflectStack:                make(map[reflect.Type]bool),
		oneOfRegistrations:          make(map[reflect.Type][]reflect.Type),
		discriminatorRegistrations:  make(map[reflect.Type]discriminatorRegistration),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(g)
		}
	}
	return g
}

// run returns a generator carrying fresh per-invocation state. Configuration is
// shared with the receiver and treated as read-only during generation, so a
// configured Generator is safe to reuse across calls and across goroutines.
// renderFile owns the rendering output buffers (imports, decls, metadata), so
// they are reset there rather than duplicated here.
func (g *Generator) run() *Generator {
	r := *g
	r.diagnostics = nil
	r.openapiCache = make(map[*highbase.SchemaProxy]*SchemaIR)
	r.reflectCache = make(map[reflect.Type]*SchemaIR)
	r.reflectStack = make(map[reflect.Type]bool)
	r.typeNames = nil
	r.componentNames = nil
	r.componentTypeNames = nil
	r.componentKinds = nil
	r.currentComponent = ""
	return &r
}

// RenderSchema renders a single OpenAPI schema as Go source.
func RenderSchema(name string, schema *highbase.SchemaProxy, opts ...Option) ([]byte, error) {
	return NewGenerator(opts...).RenderSchema(name, schema)
}

// SchemaFromValue generates an OpenAPI schema for the runtime type of value.
func SchemaFromValue(value any, opts ...Option) (*highbase.SchemaProxy, error) {
	return NewGenerator(opts...).SchemaFromValue(value)
}

// SchemaFromType generates an OpenAPI schema for a Go reflection type.
func SchemaFromType(t reflect.Type, opts ...Option) (*highbase.SchemaProxy, error) {
	return NewGenerator(opts...).SchemaFromType(t)
}

// SchemasFromValues generates an OpenAPI component graph for runtime values.
func SchemasFromValues(values ...any) (*SchemaSet, error) {
	return NewGenerator().SchemasFromValues(values...)
}

// SchemasFromValuesWithOptions generates an OpenAPI component graph for runtime
// values using generator options.
func SchemasFromValuesWithOptions(values []any, opts ...Option) (*SchemaSet, error) {
	return NewGenerator(opts...).SchemasFromValues(values...)
}

// SchemasFromTypes generates an OpenAPI component graph for Go reflection
// types.
func SchemasFromTypes(types ...reflect.Type) (*SchemaSet, error) {
	return NewGenerator().SchemasFromTypes(types...)
}

// SchemasFromTypesWithOptions generates an OpenAPI component graph for Go
// reflection types using generator options.
func SchemasFromTypesWithOptions(types []reflect.Type, opts ...Option) (*SchemaSet, error) {
	return NewGenerator(opts...).SchemasFromTypes(types...)
}

// RenderSchema renders a single OpenAPI schema as Go source using this
// generator.
func (g *Generator) RenderSchema(name string, schema *highbase.SchemaProxy) ([]byte, error) {
	if schema == nil {
		return nil, wrapPath(ErrNilSchema, name)
	}
	r := g.run()
	r.typeNames = newNameRegistry()
	ir, err := r.irFromOpenAPI(name, schema, name)
	if err != nil {
		return nil, err
	}
	file, err := r.renderFile([]*SchemaIR{ir})
	if err != nil {
		return nil, err
	}
	return file.Source, nil
}

// RenderSchemas renders an ordered map of OpenAPI schemas as one Go source
// file.
func (g *Generator) RenderSchemas(schemas *orderedmap.Map[string, *highbase.SchemaProxy]) (*GeneratedFile, error) {
	if err := validatePackageName(g.packageName); err != nil {
		return nil, err
	}
	r := g.run()
	if schemas == nil {
		return r.renderFile(nil)
	}
	r.typeNames = newNameRegistry()
	r.componentTypeNames = r.resolveComponentTypeNames(schemas)
	irs := make([]*SchemaIR, 0, schemas.Len())
	for name, schema := range schemas.FromOldest() {
		ir, err := r.irFromOpenAPI(name, schema, name)
		if err != nil {
			return nil, err
		}
		irs = append(irs, ir)
	}
	r.componentKinds = make(map[string]Kind, len(irs))
	for _, ir := range irs {
		if ir != nil && ir.Name != "" {
			r.componentKinds[ir.Name] = ir.Kind
		}
	}
	return r.renderFile(irs)
}

func (g *Generator) resolveComponentTypeNames(schemas *orderedmap.Map[string, *highbase.SchemaProxy]) map[string]string {
	names := make(map[string]string)
	if schemas == nil {
		return names
	}
	registry := g.typeNames
	if registry == nil {
		registry = newNameRegistry()
	}
	for name := range schemas.FromOldest() {
		resolved, collision := registry.resolve(name, g.publicName(name))
		names[name] = resolved
		if collision {
			g.addDiagnostic(DiagnosticComponentNameCollision, name, "component name collision resolved as "+resolved)
		}
	}
	return names
}

func (g *Generator) resolveTypeName(original, candidate, path string) string {
	if g.typeNames == nil {
		return candidate
	}
	resolved, collision := g.typeNames.resolve(original, candidate)
	if collision {
		g.addDiagnostic(DiagnosticTypeNameCollision, path, "type name collision resolved as "+resolved)
	}
	return resolved
}

// SchemaFromValue generates an OpenAPI schema for the runtime type of value
// using this generator.
func (g *Generator) SchemaFromValue(value any) (*highbase.SchemaProxy, error) {
	if value == nil {
		return nil, wrapPath(ErrNilType, "")
	}
	return g.SchemaFromType(reflect.TypeOf(value))
}

// SchemaFromType generates an OpenAPI schema for a Go reflection type using
// this generator.
func (g *Generator) SchemaFromType(t reflect.Type) (*highbase.SchemaProxy, error) {
	if t == nil {
		return nil, wrapPath(ErrNilType, "")
	}
	r := g.run()
	nameType := derefType(t)
	ir, err := r.irFromReflect(t, typeName(nameType), typeName(nameType))
	if err != nil {
		return nil, err
	}
	return r.openapiFromIR(ir), nil
}

// SchemasFromValues generates an OpenAPI component graph for runtime values
// using this generator.
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

// SchemasFromTypes generates an OpenAPI component graph for Go reflection types
// using this generator.
func (g *Generator) SchemasFromTypes(types ...reflect.Type) (*SchemaSet, error) {
	r := g.run()
	roots := orderedmap.New[string, *highbase.SchemaProxy]()
	components := orderedmap.New[string, *highbase.SchemaProxy]()
	var root *highbase.SchemaProxy
	for i, t := range types {
		if t == nil {
			return nil, wrapPath(ErrNilType, "")
		}
		nameType := derefType(t)
		ir, err := r.irFromReflect(t, typeName(nameType), typeName(nameType))
		if err != nil {
			return nil, err
		}
		rootName := ir.Name
		rootProxy := r.rootProxy(ir)
		if i == 0 {
			root = rootProxy
		}
		if _, exists := roots.Get(rootName); exists {
			r.addDiagnostic(DiagnosticRootNameCollision, rootName, "root name collision resolved by keeping first schema")
			continue
		}
		roots.Set(rootName, rootProxy)
	}
	irs := make([]*SchemaIR, 0, len(r.reflectCache))
	for _, ir := range r.reflectCache {
		if ir != nil && ir.Name != "" && isComponentKind(ir.Kind) {
			irs = append(irs, ir)
		}
	}
	sortIRsByName(irs)
	componentNames := make(map[string]struct{}, len(irs))
	for _, ir := range irs {
		componentNames[ir.Name] = struct{}{}
	}
	r.componentNames = componentNames
	for _, ir := range irs {
		if _, exists := components.Get(ir.Name); exists {
			r.addDiagnostic(DiagnosticComponentNameCollision, ir.Name, "component name collision resolved by keeping first schema")
			continue
		}
		r.currentComponent = ir.Name
		components.Set(ir.Name, r.openapiFromIR(ir))
	}
	return &SchemaSet{
		Root:        root,
		Roots:       roots,
		Components:  components,
		Diagnostics: append([]Diagnostic(nil), r.diagnostics...),
	}, nil
}

func (g *Generator) rootProxy(ir *SchemaIR) *highbase.SchemaProxy {
	if ir != nil && ir.Name != "" && isComponentKind(ir.Kind) {
		ref := "#/components/schemas/" + ir.Name
		if ir.Nullable {
			return nullableReferenceProxy(ref, false, ir)
		}
		return highbase.CreateSchemaProxyRef(ref)
	}
	return g.openapiFromIR(ir)
}

func (g *Generator) addDiagnostic(code, path, message string) {
	g.diagnostics = append(g.diagnostics, Diagnostic{Code: code, Path: path, Message: message})
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
