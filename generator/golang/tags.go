// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"reflect"
	"strings"
)

type fieldTag struct {
	name      string
	skip      bool
	omitempty bool
}

func parseJSONTag(field reflect.StructField) fieldTag {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return fieldTag{skip: true}
	}
	if tag == "" {
		return fieldTag{name: field.Name}
	}
	parts := strings.Split(tag, ",")
	name := parts[0]
	if name == "" {
		name = field.Name
	}
	ft := fieldTag{name: name}
	for _, opt := range parts[1:] {
		if opt == "omitempty" {
			ft.omitempty = true
		}
	}
	return ft
}

func tagLiteral(name string, required bool, jsonTags, yamlTags, omitEmpty bool) string {
	var tags []string
	value := name
	if !required && omitEmpty {
		value += ",omitempty"
	}
	if jsonTags {
		tags = append(tags, `json:"`+value+`"`)
	}
	if yamlTags {
		tags = append(tags, `yaml:"`+value+`"`)
	}
	if len(tags) == 0 {
		return ""
	}
	return "`" + strings.Join(tags, " ") + "`"
}
