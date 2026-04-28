// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"reflect"
	"strings"
)

type fieldTag struct {
	name          string
	skip          bool
	omitempty     bool
	stringEncoded bool
	hasName       bool
	openapi       openAPIMetadata
}

func parseJSONTag(field reflect.StructField) fieldTag {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return fieldTag{skip: true}
	}
	if tag == "" {
		return fieldTag{name: field.Name, openapi: parseOpenAPITag(field.Tag.Get("openapi"))}
	}
	parts := strings.Split(tag, ",")
	name := parts[0]
	hasName := name != ""
	if name == "" {
		name = field.Name
	}
	ft := fieldTag{name: name, hasName: hasName, openapi: parseOpenAPITag(field.Tag.Get("openapi"))}
	for _, opt := range parts[1:] {
		if opt == "omitempty" || opt == "omitzero" {
			ft.omitempty = true
		}
		if opt == "string" {
			ft.stringEncoded = true
		}
	}
	return ft
}

func tagLiteral(name string, required bool, jsonTags, yamlTags, omitEmpty bool, openapiTag string) string {
	var tags []string
	value := name
	if !required && omitEmpty {
		value += ",omitempty"
	}
	if jsonTags {
		tags = append(tags, `json:"`+escapeStructTagValue(value)+`"`)
	}
	if yamlTags {
		tags = append(tags, `yaml:"`+escapeStructTagValue(value)+`"`)
	}
	if openapiTag != "" {
		tags = append(tags, `openapi:"`+escapeStructTagValue(openapiTag)+`"`)
	}
	if len(tags) == 0 {
		return ""
	}
	return "`" + strings.Join(tags, " ") + "`"
}

func escapeStructTagValue(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return value
}
