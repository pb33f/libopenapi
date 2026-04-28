// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"encoding/json"
	"reflect"
	"testing"
)

type ReflectionParityEmbedded struct {
	TraceID string `json:"trace_id"`
}

type ReflectionParityModel struct {
	*ReflectionParityEmbedded `json:",omitempty"`
	Count                     int             `json:"count,string,omitempty"`
	Limit                     int             `json:"limit,omitzero"`
	Data                      []byte          `json:"data,omitempty"`
	Raw                       json.RawMessage `json:"raw,omitempty"`
}

func TestReflectionParityJSONEncodingSemantics(t *testing.T) {
	set, err := SchemasFromTypes(reflect.TypeOf(ReflectionParityModel{}))
	if err != nil {
		t.Fatal(err)
	}
	model := componentSchema(t, set, "ReflectionParityModel")
	if _, ok := model.Properties.Get("ReflectionParityEmbedded"); ok {
		t.Fatal("anonymous embedded field should be flattened when tag has no explicit name")
	}
	trace, ok := model.Properties.Get("trace_id")
	if !ok {
		t.Fatal("missing promoted trace_id property")
	}
	if trace.Schema().Type[0] != "string" {
		t.Fatalf("trace_id should be string, got %#v", trace.Schema())
	}
	if containsString(model.Required, "trace_id") {
		t.Fatalf("omitempty anonymous pointer fields should not promote required children: %#v", model.Required)
	}
	if containsString(model.Required, "limit") {
		t.Fatalf("omitzero should make limit optional: %#v", model.Required)
	}

	count := model.Properties.GetOrZero("count").Schema()
	if count.Type[0] != "string" {
		t.Fatalf("json ,string field should render as string, got %#v", count)
	}
	data := model.Properties.GetOrZero("data").Schema()
	if data.Type[0] != "string" || data.Format != "byte" {
		t.Fatalf("[]byte should render as string byte, got %#v", data)
	}
	raw := model.Properties.GetOrZero("raw").Schema()
	if len(raw.Type) != 0 {
		t.Fatalf("json.RawMessage should render as unconstrained schema, got %#v", raw)
	}
	if !hasDiagnosticCode(set.Diagnostics, DiagnosticStringEncoded) {
		t.Fatalf("expected string encoded diagnostic, got %#v", set.Diagnostics)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
