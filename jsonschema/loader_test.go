// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package jsonschema

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/testify/require"
)

func TestLoadInputBoundaries(t *testing.T) {
	for _, tc := range []struct {
		input    string
		explicit bool
		want     string
	}{
		{`{"$vocabulary":{"urn:unsupported":true}}`, true, "unsupported required vocabulary"},
		{"{}", false, "--schema"}, {`{"$schema":"https://example.org/app"}`, false, "unsupported"},
		{"null", true, "object or boolean"}, {"type: object\ntype: string", true, "already defined"},
		{`{"openapi":"3.1.0","$schema":"https://json-schema.org/draft/2020-12/schema"}`, true, "conflicting"},
		{`{"$defs":{"a":{"$id":"urn:money"},"b":{"$id":"urn:money"}}}`, true, "duplicate"},
		{`{"$dynamicAnchor":"scope","$dynamicRef":"#scope"}`, true, "dynamic scope"}, {`{"$ref":"#/missing"}`, true, "no schema"},
		{"true\n---\nfalse", true, "one schema"},
	} {
		_, err := Load([]byte(tc.input), "schema.yaml", nil, tc.explicit)
		require.ErrorContains(t, err, tc.want)
	}
	d, err := Load([]byte(`{"$schema":"https://json-schema.org/draft/2020-12/schema","examples":[{"$ref":"missing","$id":"urn:x"},{"$id":"urn:x"}]}`), "schema.json", nil, false)
	require.NoError(t, err)
	d.Release()
}

func TestInvalidConstraintsFailBeforeComparison(t *testing.T) {
	for _, input := range []string{`{"type":42}`, `{"required":42}`, "minLength: -2", `{"properties":{"amount":{"multipleOf":0}}}`} {
		_, err := Load([]byte(input), "bad.yaml", nil, true)
		require.ErrorContains(t, err, "bad.yaml")
	}
	_, err := LoadCollection(map[string][]byte{"bad.json": []byte(`{"type":"invalid"}`)}, ".", nil)
	require.ErrorContains(t, err, "bad.json")
}

func TestMetaValidationChecksDistinctPropertySchemasAndPreservesLiterals(t *testing.T) {
	for _, kw := range []string{"properties", "$defs", "dependentSchemas"} {
		_, err := Load([]byte(`{"`+kw+`":{"a":{"type":"string"},"b":{"type":"string"},"c":{"type":41}}}`), "bad.json", nil, true)
		require.ErrorContains(t, err, "/"+kw+"/c/type")
		_, err = Load([]byte(`{"`+kw+`":{"a":{"type":41},"b":{"type":41},"c":{"type":41}}}`), "bad.json", nil, true)
		for _, name := range []string{"a", "b", "c"} {
			require.ErrorContains(t, err, "/"+kw+"/"+name+"/type")
		}
	}
	doc, err := Load([]byte(`{"const":{"properties":{"a":true,"b":true}},"properties":{"a":true,"b":true}}`), "good.json", nil, true)
	require.NoError(t, err)
	defer doc.Release()
	var skipped bool
	meta, err := doc.metaValue(doc.Root.Node, false, &skipped)
	require.NoError(t, err)
	require.True(t, skipped)
	value := meta.(map[string]any)
	require.Len(t, value["properties"], 1)
	require.Len(t, value["const"].(map[string]any)["properties"], 2)
	require.Len(t, member(doc.Root.Node, "properties").Content, 4)
}

func BenchmarkLoadRepeatedPropertySchemas(b *testing.B) {
	var input strings.Builder
	input.WriteString(`{"$schema":"` + Dialect + `","properties":{`)
	for i := range 8000 {
		if i > 0 {
			input.WriteByte(',')
		}
		fmt.Fprintf(&input, `"field%d":{"type":"string","maxLength":32}`, i)
	}
	input.WriteString("}}")
	data := []byte(input.String())
	b.ReportAllocs()
	for b.Loop() {
		d, err := Load(data, "wide.json", nil, false)
		if err != nil {
			b.Fatal(err)
		}
		d.Release()
	}
}

func TestSchemaNumberValidationKeepsYAMLPrecision(t *testing.T) {
	for _, tc := range []struct {
		schema string
		valid  bool
	}{
		{"multipleOf: 1.0e-400", true},
		{"minimum: 1.01e+400", true},
		{"minLength: 0.9999999999999999999", false},
		{"minLength: 1.0000000000000000001", false},
	} {
		d, err := Load([]byte(tc.schema), "value.yaml", nil, true)
		if tc.valid {
			require.NoError(t, err)
			d.Release()
		} else {
			require.Error(t, err)
		}
	}
}

func TestYAMLNumbersInSchemaAnnotations(t *testing.T) {
	input := []byte("examples: [null, +0x20, 1_024, .25, -.75, 2., 1.e+2]")
	doc, err := Load(input, "numbers.yaml", nil, true)
	require.NoError(t, err)
	defer doc.Release()

	projected, err := doc.metaValue(doc.Root.Node, false, nil)
	require.NoError(t, err)
	require.Equal(t, []any{
		nil, json.Number("32"), json.Number("1024"), json.Number("0.25"),
		json.Number("-0.75"), json.Number("2.0"), json.Number("1.0e+2"),
	}, projected.(map[string]any)["examples"])
	require.Equal(t, input, doc.Root.Source.Data)
}

func TestNonFiniteYAMLNumberReportsAuthoredLine(t *testing.T) {
	_, err := Load([]byte("examples:\n  - 12\n  - .inf\n"), "measurements.yaml", nil, true)
	require.ErrorContains(t, err, `measurements.yaml: line 3: invalid JSON number ".inf"`)
}

func TestNativeFilePathsResolveThroughFileURIs(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "my root.json")
	want := filepath.Join(dir, "amount.json")
	require.NoError(t, os.WriteFile(want, []byte("true"), 0o600))
	root := fileURL(name)
	require.Equal(t, name, filePath(root))
	require.Equal(t, want, filePath(root.ResolveReference(&url.URL{Path: "amount.json"})))
	config := datamodel.NewDocumentConfiguration()
	config.AllowFileReferences = true
	doc, err := Load([]byte(`{"$ref":"amount.json"}`), name, config, true)
	require.NoError(t, err)
	defer doc.Release()
	require.Len(t, doc.Sources, 2)
	require.Same(t, doc.Sources["amount.json"].Node, doc.References[0].To)
}

func TestRelativeIDOmitsRetrievalCredentials(t *testing.T) {
	d, err := Load([]byte(`{"$id":"customer?version=2","$ref":"#"}`), "https://name:private@schemas.example.test/draft/root.json?token=secret", nil, true)
	require.NoError(t, err)
	defer d.Release()
	require.Equal(t, "https://schemas.example.test/draft/customer?version=2", d.Root.ID)
	require.NotContains(t, d.Root.Source.Name, "private")
	require.NotContains(t, d.Root.Source.Name, "secret")
	require.Same(t, d.Root.Node, d.References[0].To)
}

func TestDynamicRefWithStaticTarget(t *testing.T) {
	d, err := Load([]byte(`{"$defs":{"a":{"$anchor":"static"}},"$dynamicRef":"#static","$ref":"#static"}`), "a.json", nil, true)
	require.NoError(t, err)
	defer d.Release()
	require.Len(t, d.References, 2)
	require.ElementsMatch(t, []string{"$ref", "$dynamicRef"}, []string{d.References[0].Keyword, d.References[1].Keyword})
	require.Same(t, d.References[0].To, d.References[1].To)
	require.NotNil(t, d.References[0].To)
}
