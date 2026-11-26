// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package jsonschema_test

import (
	"errors"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"testing/fstest"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/jsonschema"
	"github.com/pb33f/testify/require"
)

func TestResourceLabelsOwnershipAndRelease(t *testing.T) {
	schema := []byte(`title: Customer
$vocabulary:
  https://json-schema.org/draft/2020-12/vocab/validation: true
  https://schemas.example.test/optional: false
properties:
  kind: false
$defs:
  data: {$id: 'types/data?version=2'}
examples:
  - {type: string}
`)
	doc, err := jsonschema.Load(schema, "", nil, true)
	require.NoError(t, err)
	require.Len(t, doc.Resources, 2)
	require.Equal(t, "schema.yaml", doc.Root.ID)
	require.Equal(t, "Customer", doc.Root.Label())
	resource := doc.Resources[1]
	require.Equal(t, "types/data?version=2", resource.ID)
	require.Equal(t, "data?version=2", resource.Label())
	require.Equal(t, "/$defs/data", resource.Pointer)
	require.Same(t, doc.Root.Source, resource.Source)
	registered := resource.Source.Index.GetAllSchemaIds()
	require.Len(t, registered, 1)
	for id, entry := range registered {
		require.Same(t, resource.Node, entry.SchemaNode)
		require.Same(t, entry, doc.Rolodex.LookupSchemaById(id))
	}
	root := doc.Root.Node
	require.True(t, doc.IsSchema(root))
	require.True(t, doc.IsSchema(resource.Node))
	require.False(t, doc.IsSchema(root.Content[0]))                              // the title key
	require.False(t, doc.IsSchema(root.Content[len(root.Content)-1].Content[0])) // an example
	doc.Release()
	require.False(t, doc.IsSchema(root))
	require.Nil(t, doc.Root)
	require.Nil(t, doc.Sources)
	require.Nil(t, doc.Rolodex)
	doc.Release()
	var unloaded *jsonschema.Document
	require.False(t, unloaded.IsSchema(root))
	unloaded.Release()
}

func TestSchemaLabelWithoutBasename(t *testing.T) {
	doc, err := jsonschema.Load([]byte("$id: ."), "root.yml", nil, true)
	require.NoError(t, err)
	defer doc.Release()
	require.Equal(t, "Schema", doc.Root.Label())
}

func TestConfiguredRetrievalBaseAndFilename(t *testing.T) {
	base, err := url.Parse("https://schemas.example.test/releases")
	require.NoError(t, err)
	type sourceFile struct{ source, filename string }
	files := []sourceFile{
		{"local/overridden.json", "overridden.json"},
		{filepath.Join(t.TempDir(), "overridden.json"), "overridden.json"},
		{"https://old.example.test/files/overridden.json?signature=old", "overridden.json"},
	}
	if runtime.GOOS != "windows" {
		files = append(files, sourceFile{"catalog:v2.json", "catalog:v2.json"})
	}
	for _, file := range files {
		cfg := datamodel.NewDocumentConfiguration()
		cfg.SpecFilePath, cfg.BaseURL = file.source, base
		doc, err := jsonschema.Load([]byte(`{"type":"string"}`), "order.yml", cfg, true)
		require.NoError(t, err)
		require.Equal(t, file.filename, doc.Root.ID)
		require.Equal(t, "https://schemas.example.test/releases/"+file.filename, doc.Root.Source.Name)
		require.Equal(t, doc.Root.Source.Name, doc.Root.Source.Index.GetSpecAbsolutePath())
		doc.Release()
	}
}

func TestRejectAmbiguousOrInvalidSchemaResources(t *testing.T) {
	for _, tc := range []struct{ name, input, want string }{
		{"JSON extension", "type: string", "invalid JSON in invalid.json"},
		{"non-object vocabulary", `{"$vocabulary":true}`, "$vocabulary must be an object"},
		{"vocabulary boolean", `{"$vocabulary":{"https://schemas.example.test/optional":"no"}}`, "vocabulary requirement must be boolean"},
		{"unknown vocabulary shorthand", `{"$vocabulary":{"core":true}}`, "unsupported required vocabulary"},
		{"ID fragment", `{"$id":"urn:example#part"}`, "invalid schema $id"},
		{"empty ID", `{"$id":""}`, "$id cannot be empty"},
		{"anchor syntax", `{"$anchor":"1value"}`, "invalid $anchor"},
		{"ambiguous anchor", `{"$defs":{"a":{"$anchor":"town"},"b":{"$anchor":"town"}}}`, "duplicate schema address"},
		{"invalid pointer URI", `{"$ref":"bad%XY"}`, "invalid $ref"},
		{"single schema", `{"not":17}`, "schema must be an object or boolean"},
		{"schema map", `{"properties":true}`, "properties requires an object"},
		{"non-string map key", "properties: {17: {}}", "JSON object keys must be strings"},
		{"schema list", `{"allOf":{}}`, "allOf requires an array"},
		{"schema list operand", `{"allOf":[false,17]}`, "schema must be an object or boolean"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := jsonschema.Load([]byte(tc.input), "invalid.json", nil, true)
			require.ErrorContains(t, err, tc.want)
			require.Nil(t, doc)
		})
	}
	// A declared ID can take a later member's retrieval URI.
	doc, err := jsonschema.LoadCollection(map[string][]byte{
		"a.json": []byte(`{"$id":"b.json"}`), "b.json": []byte(`{"$id":"urn:different"}`),
	}, t.TempDir(), nil)
	require.ErrorContains(t, err, "duplicate schema address")
	require.ErrorContains(t, err, "a.json:1")
	require.ErrorContains(t, err, "b.json:1")
	require.Nil(t, doc)
}

func TestRelativeLoadingAfterWorkingDirectoryWasRemoved(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows retains the current directory while a process uses it")
	}
	cwd := t.TempDir()
	root := filepath.Join(cwd, "root.json")
	t.Chdir(cwd)
	t.Setenv("PWD", "/")
	require.NoError(t, os.Chmod(cwd, 0))
	require.NoError(t, os.Remove(cwd))
	got, cwdErr := os.Getwd()
	if cwdErr == nil {
		t.Skipf("host can still resolve the removed working directory as %q", got)
	}
	wantErr := fs.ErrNotExist
	if errors.Is(cwdErr, fs.ErrPermission) {
		wantErr = fs.ErrPermission
	}
	_, err := jsonschema.Load([]byte("true"), "root.json", nil, true)
	require.ErrorIs(t, err, wantErr)
	_, err = jsonschema.LoadCollection(nil, ".", nil)
	require.ErrorIs(t, err, wantErr)

	cfg := datamodel.NewDocumentConfiguration()
	cfg.BasePath = "schemas"
	cfg.LocalFS = fstest.MapFS{"child.json": {Data: []byte("true")}}
	_, err = jsonschema.Load([]byte(`{"$ref":"child.json"}`), root, cfg, true)
	require.ErrorIs(t, err, wantErr)
}
