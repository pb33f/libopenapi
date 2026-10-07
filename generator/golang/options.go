// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"reflect"

	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
)

// Option configures a Generator.
type Option func(*Generator)

// NameResolver maps OpenAPI names to Go identifiers. Returning an empty string
// falls back to the generator's default naming.
type NameResolver func(string) string

// ExternalRefResolver maps an external OpenAPI $ref to a Go type name.
// Returning an empty string falls back to deriving the type name from the
// reference tail.
type ExternalRefResolver func(ref string) string

// Diagnostic describes a notable generator decision.
type Diagnostic struct {
	Code    string
	Path    string
	Message string
}

const (
	DiagnosticComponentNameCollision     = "componentNameCollision"
	DiagnosticChildSchema                = "childSchema"
	DiagnosticAdditionalPropertiesFalse  = "additionalPropertiesFalse"
	DiagnosticArrayContains              = "arrayContains"
	DiagnosticBooleanItems               = "booleanItems"
	DiagnosticConstKeyword               = "constKeyword"
	DiagnosticContentSchema              = "contentSchema"
	DiagnosticDependentRequired          = "dependentRequired"
	DiagnosticDependentSchemas           = "dependentSchemas"
	DiagnosticDynamicReference           = "dynamicReference"
	DiagnosticExternalReference          = "externalReference"
	DiagnosticFieldNameCollision         = "fieldNameCollision"
	DiagnosticConditionalSchema          = "conditionalSchema"
	DiagnosticImplicitType               = "implicitType"
	DiagnosticMixedEnum                  = "mixedEnum"
	DiagnosticMultiTypeSchema            = "multiTypeSchema"
	DiagnosticNotSchema                  = "notSchema"
	DiagnosticNullEnum                   = "nullEnum"
	DiagnosticNullableKeyword            = "nullableKeyword"
	DiagnosticOptionalConstDiscriminator = "optionalConstDiscriminator"
	DiagnosticPatternProperties          = "patternProperties"
	DiagnosticPrefixItems                = "prefixItems"
	DiagnosticPropertyNames              = "propertyNames"
	DiagnosticSchemaMetadata             = "schemaMetadata"
	DiagnosticStringEncoded              = "stringEncoded"
	DiagnosticTypeNameCollision          = "typeNameCollision"
	DiagnosticUnevaluatedItems           = "unevaluatedItems"
	DiagnosticRootNameCollision          = "rootNameCollision"
	DiagnosticUnevaluatedProperties      = "unevaluatedProperties"
	DiagnosticValidationKeyword          = "validationKeyword"
)

// NameStyle selects how the generator names inline schemas and resolves name
// collisions.
type NameStyle int

const (
	// NameStyleQualified joins each inline schema's name to its parent's with
	// the nested type name delimiter (Parent_Child, Parent_Items_Item) and
	// resolves collisions with "__N" suffixes. It is the default.
	NameStyleQualified NameStyle = iota
	// NameStyleIdiomatic names inline schemas the way a Go author would. A
	// child is named after its nearest declared ancestor and its property
	// (AccountStatus), array elements take the singular property name
	// (OrganizationFundingEvent), and map values end in Value. Under a root
	// marked with WithInlineRoots, a child takes its bare property name
	// (Person) when it is free. Struct fields whose JSON names differ only by
	// a leading symbol are told apart with words (_id is UnderscoreID), and any
	// remaining collision takes a plain numeric suffix.
	NameStyleIdiomatic
)

type formatMapping struct {
	goType     string
	importPath string
}

type discriminatorRegistration struct {
	property string
	mapping  map[string]string
}

type fieldSchemaKey struct {
	owner reflect.Type
	name  string
}

// WithPackageName sets the generated Go package name.
func WithPackageName(name string) Option {
	return func(g *Generator) {
		g.packageName = name
	}
}

// WithOptionalFieldsAsPointers controls whether optional scalar fields render
// as pointers.
func WithOptionalFieldsAsPointers(enabled bool) Option {
	return func(g *Generator) {
		g.optionalFieldsAsPointers = enabled
	}
}

// WithOmitEmpty controls omitempty on optional generated tags.
func WithOmitEmpty(enabled bool) Option {
	return func(g *Generator) {
		g.omitEmpty = enabled
	}
}

// WithNullableAsPointer controls whether nullable scalar fields render as
// pointers.
func WithNullableAsPointer(enabled bool) Option {
	return func(g *Generator) {
		g.nullableAsPointer = enabled
	}
}

// WithOptionalNullableAsDoublePointer preserves all three JSON states for an
// optional nullable scalar: a nil outer pointer omits the field, a non-nil
// outer pointer containing nil writes null, and two non-nil pointers write the
// value. It only applies when optional fields and nullable values are both
// configured to use pointers.
func WithOptionalNullableAsDoublePointer(enabled bool) Option {
	return func(g *Generator) {
		g.optionalNullableAsDoublePointer = enabled
	}
}

// WithGenerateJSONTags controls generated json tags.
func WithGenerateJSONTags(enabled bool) Option {
	return func(g *Generator) {
		g.jsonTags = enabled
	}
}

// WithGenerateYAMLTags controls generated yaml tags.
func WithGenerateYAMLTags(enabled bool) Option {
	return func(g *Generator) {
		g.yamlTags = enabled
	}
}

// WithEnumConstants controls whether enum values generate Go constants.
func WithEnumConstants(enabled bool) Option {
	return func(g *Generator) {
		g.enumConstants = enabled
	}
}

// WithHeaderComment writes a file header comment before the package clause.
func WithHeaderComment(text string) Option {
	return func(g *Generator) {
		g.headerComment = text
	}
}

// WithPackageComment writes a package doc comment before the package clause.
func WithPackageComment(text string) Option {
	return func(g *Generator) {
		g.packageComment = text
	}
}

// WithGeneratedComment writes a standard generated-code comment.
func WithGeneratedComment(enabled bool) Option {
	return func(g *Generator) {
		g.generatedComment = enabled
	}
}

// WithOpenAPITags controls whether generated struct fields include compact
// openapi tags for metadata that cannot be recovered from Go reflection alone.
func WithOpenAPITags(enabled bool) Option {
	return func(g *Generator) {
		g.openapiTags = enabled
	}
}

// WithSchemaMetadataSidecar controls whether generated named types include a
// typed OpenAPISchemaMetadata sidecar. Enabling the sidecar preserves original
// OpenAPI schema fidelity for Go reflection round trips. Disabling it keeps the
// generated model code leaner, but OpenAPI -> Go -> OpenAPI reconstruction is
// intentionally lossy and falls back to Go type shape plus tags.
func WithSchemaMetadataSidecar(enabled bool) Option {
	return func(g *Generator) {
		g.schemaMetadataSidecar = enabled
	}
}

// WithFormatMapping maps an OpenAPI string format to a Go type and optional
// import path.
func WithFormatMapping(format, goType, importPath string) Option {
	return func(g *Generator) {
		if g.formatMappings == nil {
			g.formatMappings = make(map[string]formatMapping)
		}
		g.formatMappings[format] = formatMapping{goType: goType, importPath: importPath}
	}
}

// WithNameResolver sets a broad fallback resolver for generated Go names.
func WithNameResolver(resolver NameResolver) Option {
	return func(g *Generator) {
		g.nameResolver = resolver
	}
}

// WithTypeNameResolver sets a resolver for generated Go type names.
func WithTypeNameResolver(resolver NameResolver) Option {
	return func(g *Generator) {
		g.typeNameResolver = resolver
	}
}

// WithFieldNameResolver sets a resolver for generated Go struct field names.
func WithFieldNameResolver(resolver NameResolver) Option {
	return func(g *Generator) {
		g.fieldNameResolver = resolver
	}
}

// WithEnumValueNameResolver sets a resolver for generated enum constant suffixes.
func WithEnumValueNameResolver(resolver NameResolver) Option {
	return func(g *Generator) {
		g.enumValueNameResolver = resolver
	}
}

// WithOptionalConstDiscriminatorUnions allows optional shared const
// discriminator properties to produce typed oneOf unions.
func WithOptionalConstDiscriminatorUnions(enabled bool) Option {
	return func(g *Generator) {
		g.optionalConstDiscriminatorUnions = enabled
	}
}

// WithAdditionalPropertiesMethods controls whether schema-valued
// additionalProperties generates JSON marshal/unmarshal methods that round-trip
// unknown fields through the AdditionalProperties map.
func WithAdditionalPropertiesMethods(enabled bool) Option {
	return func(g *Generator) {
		g.additionalPropertiesMethods = enabled
	}
}

// WithNestedTypeNameDelimiter sets the separator inserted between generated
// parent and child type names for inline schemas. The default is "_"; passing
// an empty delimiter restores compact names like ParentChild.
func WithNestedTypeNameDelimiter(delimiter string) Option {
	return func(g *Generator) {
		g.nestedTypeNameDelimiter = delimiter
	}
}

// WithNameStyle sets how inline schemas are named and collisions resolved.
func WithNameStyle(style NameStyle) Option {
	return func(g *Generator) {
		g.nameStyle = style
	}
}

// WithInlineRoots marks top-level schemas, by their key in the rendered map,
// that a document declared inline, such as an operation's response body,
// rather than as named components. The document gives the parts of such a
// schema no names except their property names, so with NameStyleIdiomatic
// their nested types are named for the property alone when that name is free.
func WithInlineRoots(keys ...string) Option {
	return func(g *Generator) {
		if g.inlineRoots == nil {
			g.inlineRoots = make(map[string]struct{}, len(keys))
		}
		for _, key := range keys {
			g.inlineRoots[key] = struct{}{}
		}
	}
}

// WithDecodeOnlyRoots marks top-level schemas, by their key in the rendered
// map, that are only ever decoded, such as response bodies. Decoding cannot
// tell a JSON null from an absent field, so optional nullable fields under
// these schemas keep a single pointer even with
// WithOptionalNullableAsDoublePointer.
func WithDecodeOnlyRoots(keys ...string) Option {
	return func(g *Generator) {
		if g.decodeOnlyRoots == nil {
			g.decodeOnlyRoots = make(map[string]struct{}, len(keys))
		}
		for _, key := range keys {
			g.decodeOnlyRoots[key] = struct{}{}
		}
	}
}

// WithFallbackDescriptions documents top-level schemas, by their key in the
// rendered map, that carry no description or title of their own.
func WithFallbackDescriptions(descriptions map[string]string) Option {
	return func(g *Generator) {
		g.fallbackDescriptions = descriptions
	}
}

// WithReservedTypeNames keeps generated type names clear of names that other
// code in the same package declares.
func WithReservedTypeNames(names ...string) Option {
	return func(g *Generator) {
		g.reservedTypeNames = append(g.reservedTypeNames, names...)
	}
}

// WithUntypedAsRawMessage renders schemas that do not describe their JSON
// shape as json.RawMessage instead of any or map[string]any. This covers
// empty schemas, objects without declared properties, and arrays without
// items (rendered as []json.RawMessage). The raw bytes are kept exactly as
// received, and callers decode them into types they own.
func WithUntypedAsRawMessage(enabled bool) Option {
	return func(g *Generator) {
		g.untypedAsRawMessage = enabled
	}
}

// WithExternalRefTypeResolver sets a resolver for external OpenAPI $ref values
// when rendering Go type names. The resolver is not used for local component
// references.
func WithExternalRefTypeResolver(resolver ExternalRefResolver) Option {
	return func(g *Generator) {
		g.externalRefResolver = resolver
	}
}

// WithTypeSchema overrides reflected schema generation for a specific Go type.
// This is useful for project scalar aliases that need a custom OpenAPI format,
// enum, or extension without implementing SchemaProvider on the type.
func WithTypeSchema(t reflect.Type, schema *highbase.SchemaProxy) Option {
	return func(g *Generator) {
		if t == nil || schema == nil {
			return
		}
		if g.typeSchemas == nil {
			g.typeSchemas = make(map[reflect.Type]*highbase.SchemaProxy)
		}
		g.typeSchemas[derefType(t)] = schema
	}
}

// WithFieldSchema overrides reflected schema generation for a specific Go
// struct field name while keeping the surrounding model reflected normally.
func WithFieldSchema(t reflect.Type, fieldName string, schema *highbase.SchemaProxy) Option {
	return func(g *Generator) {
		if t == nil || fieldName == "" || schema == nil {
			return
		}
		if g.fieldSchemas == nil {
			g.fieldSchemas = make(map[fieldSchemaKey]*highbase.SchemaProxy)
		}
		g.fieldSchemas[fieldSchemaKey{owner: derefType(t), name: fieldName}] = schema
	}
}

// WithFieldSchemaByJSONName overrides reflected schema generation for a
// specific JSON field name while keeping the surrounding model reflected
// normally.
func WithFieldSchemaByJSONName(t reflect.Type, jsonName string, schema *highbase.SchemaProxy) Option {
	return func(g *Generator) {
		if t == nil || jsonName == "" || schema == nil {
			return
		}
		if g.jsonSchemas == nil {
			g.jsonSchemas = make(map[fieldSchemaKey]*highbase.SchemaProxy)
		}
		g.jsonSchemas[fieldSchemaKey{owner: derefType(t), name: jsonName}] = schema
	}
}

// WithOneOfTypes registers concrete variants for a Go interface when producing
// OpenAPI oneOf schemas from reflection.
func WithOneOfTypes(target any, variants ...any) Option {
	return func(g *Generator) {
		key := interfaceKey(target)
		if key == nil {
			return
		}
		types := make([]reflect.Type, 0, len(variants))
		for _, variant := range variants {
			if t := reflect.TypeOf(variant); t != nil {
				types = append(types, derefType(t))
			}
		}
		g.oneOfRegistrations[key] = types
	}
}

// WithDiscriminatorMapping registers discriminator metadata for a reflected
// interface union.
func WithDiscriminatorMapping(target any, property string, mapping map[string]string) Option {
	return func(g *Generator) {
		key := interfaceKey(target)
		if key == nil {
			return
		}
		cp := make(map[string]string, len(mapping))
		for k, v := range mapping {
			cp[k] = v
		}
		g.discriminatorRegistrations[key] = discriminatorRegistration{
			property: property,
			mapping:  cp,
		}
	}
}
