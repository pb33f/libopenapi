// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"reflect"
)

type Option func(*Generator)

type NameResolver func(string) string

type Diagnostic struct {
	Path    string
	Message string
}

type formatMapping struct {
	goType     string
	importPath string
}

type discriminatorRegistration struct {
	property string
	mapping  map[string]string
}

func WithPackageName(name string) Option {
	return func(g *Generator) {
		g.packageName = name
	}
}

func WithOptionalFieldsAsPointers(enabled bool) Option {
	return func(g *Generator) {
		g.optionalFieldsAsPointers = enabled
	}
}

func WithOmitEmpty(enabled bool) Option {
	return func(g *Generator) {
		g.omitEmpty = enabled
	}
}

func WithNullableAsPointer(enabled bool) Option {
	return func(g *Generator) {
		g.nullableAsPointer = enabled
	}
}

func WithGenerateJSONTags(enabled bool) Option {
	return func(g *Generator) {
		g.jsonTags = enabled
	}
}

func WithGenerateYAMLTags(enabled bool) Option {
	return func(g *Generator) {
		g.yamlTags = enabled
	}
}

func WithEnumConstants(enabled bool) Option {
	return func(g *Generator) {
		g.enumConstants = enabled
	}
}

func WithHeaderComment(text string) Option {
	return func(g *Generator) {
		g.headerComment = text
	}
}

func WithPackageComment(text string) Option {
	return func(g *Generator) {
		g.packageComment = text
	}
}

func WithGeneratedComment(enabled bool) Option {
	return func(g *Generator) {
		g.generatedComment = enabled
	}
}

func WithFormatMapping(format, goType, importPath string) Option {
	return func(g *Generator) {
		if g.formatMappings == nil {
			g.formatMappings = make(map[string]formatMapping)
		}
		g.formatMappings[format] = formatMapping{goType: goType, importPath: importPath}
	}
}

func WithNameResolver(resolver NameResolver) Option {
	return func(g *Generator) {
		g.nameResolver = resolver
	}
}

func WithOptionalConstDiscriminatorUnions(enabled bool) Option {
	return func(g *Generator) {
		g.optionalConstDiscriminatorUnions = enabled
	}
}

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
