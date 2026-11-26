// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package jsonschema_test

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/jsonschema"
	"github.com/pb33f/testify/require"
)

type resourceFS func(string) (fs.File, error)

func (open resourceFS) Open(name string) (fs.File, error) { return open(name) }

type cachedResource struct {
	fs.File
	text   string
	closed bool
}

func (f *cachedResource) GetContent() string { return f.text }
func (f *cachedResource) Close() error       { f.closed = true; return nil }

func TestReferencePointersAndSourceDiagnostics(t *testing.T) {
	const prefix = "prefixItems: [true, false]\n"
	d, err := jsonschema.Load([]byte(prefix+"$ref: '#/prefixItems/1'"), "array.yaml", nil, true)
	require.NoError(t, err)
	require.Len(t, d.References, 1)
	require.Equal(t, "false", d.References[0].To.Value)
	d.Release()
	for _, tc := range []struct{ ref, want string }{
		{"#/prefixItems/01", "no schema at"}, {"#/prefixItems/3", "no schema at"},
		{"#/prefixItems/-1", "no schema at"}, {"#/prefixItems/not-index", "no schema at"},
		{"#/definitions/price", "(Draft 2020-12 names this keyword $defs)"},
		{"#absent", "unknown schema anchor"}, {"file://unrelated/share/a.json", "unsupported file URI host"},
		{"child.json", "file references are disabled"}, {"https://schemas.example.test/dep.json", "remote references are disabled"},
		{"urn:unknown", "no registered schema for"},
	} {
		t.Run(tc.ref, func(t *testing.T) {
			doc, err := jsonschema.Load([]byte(prefix+"properties:\n  value:\n    $ref: '"+tc.ref+"'"), "array.yaml", nil, true)
			require.Nil(t, doc)
			require.ErrorContains(t, err, tc.want)
			require.ErrorContains(t, err, "array.yaml:4")
		})
	}
}

func TestVirtualFilesUseTheirSnapshotPaths(t *testing.T) {
	snapshot := t.TempDir()
	cfg := datamodel.NewDocumentConfiguration()
	cfg.AllowFileReferences = true
	want := filepath.Join(snapshot, "currency.json")
	var opened []string
	cached := &cachedResource{text: `{"$anchor":"value","type":"string"}`}
	cfg.LocalFS = resourceFS(func(name string) (fs.File, error) {
		opened = append(opened, name)
		if name == want {
			return cached, nil
		}
		return nil, fs.ErrInvalid
	})
	doc, err := jsonschema.Load([]byte(`{"$ref":"currency.json#value"}`), filepath.Join(snapshot, "order.json"), cfg, true)
	require.NoError(t, err)
	defer doc.Release()
	require.Equal(t, []string{"currency.json", want}, opened)
	require.True(t, cached.closed)
	require.Same(t, doc.Sources["currency.json"].Node, doc.References[0].To)
	require.Equal(t, cached.text, string(doc.Sources["currency.json"].Data))
}

func TestRemoteVirtualFileErrors(t *testing.T) {
	cfg := datamodel.NewDocumentConfiguration()
	cfg.AllowRemoteReferences = true
	const target = "https://schemas.example.test/types/currency.json"
	var location string
	load := func(open resourceFS) error {
		cfg.RemoteFS = resourceFS(func(name string) (fs.File, error) { location = name; return open(name) })
		doc, err := jsonschema.Load([]byte(`{"$ref":"`+target+`"}`), "order.json", cfg, true)
		require.Equal(t, target, location)
		if doc != nil {
			doc.Release()
		}
		return err
	}
	require.ErrorIs(t, load(func(string) (fs.File, error) { return nil, fs.ErrPermission }), fs.ErrPermission)
	require.ErrorContains(t, load(func(string) (fs.File, error) { return nil, nil }), "missing file")
	require.NoError(t, load(func(string) (fs.File, error) {
		return fstest.MapFS{"currency.json": {Data: []byte("true")}}.Open("currency.json")
	}))
}

func TestRemoteResponsesAndTransportErrors(t *testing.T) {
	const root = "https://name:password@schemas.example.test/root.json"
	const input = `{"$ref":"currency.json"}`
	cfg := datamodel.NewDocumentConfiguration()
	cfg.AllowRemoteReferences = true
	cause := errors.New("upstream unavailable")
	for _, tc := range []struct {
		name string
		get  func(string) (*http.Response, error)
		want string
		wrap bool
	}{
		{"status", func(string) (*http.Response, error) {
			return &http.Response{StatusCode: 503, Body: io.NopCloser(strings.NewReader(""))}, nil
		}, "HTTP 503", false},
		{"plain error", func(string) (*http.Response, error) { return nil, cause }, cause.Error(), true},
		{"credentialed transport", func(location string) (*http.Response, error) {
			return nil, &url.Error{Op: "Get", URL: location, Err: cause}
		}, cause.Error(), true},
		{"invalid sibling", func(string) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("type: not-a-type"))}, nil
		}, "currency.json", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg.RemoteURLHandler = tc.get
			doc, err := jsonschema.Load([]byte(input), root, cfg, true)
			require.Nil(t, doc)
			require.ErrorContains(t, err, tc.want)
			require.NotContains(t, err.Error(), "password")
			if tc.wrap {
				require.ErrorIs(t, err, cause)
			}
		})
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/currency.json" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, `{"$anchor":"iso","type":"string"}`)
	}))
	defer server.Close()
	cfg.RemoteURLHandler = nil
	doc, err := jsonschema.Load([]byte(`{"$ref":"currency.json#iso"}`), server.URL+"/root.json", cfg, true)
	require.NoError(t, err)
	defer doc.Release()
	require.Same(t, doc.Sources[server.URL+"/currency.json"].Node, doc.References[0].To)
}

func TestReferenceDiagnosticsPreserveLocationAndRedactCredentials(t *testing.T) {
	const root = "https://schemas.example.test/order.yaml"
	cause := errors.New("upstream unavailable")
	for _, tc := range []struct{ keyword, ref, label string }{
		{"$ref", "https://reader:private@types.example.test/money.json?signature=secret#currency", "https://types.example.test/money.json#currency"},
		{"$ref", "types/money.json?signature=secret#currency", "types/money.json#currency"},
		{"$ref", "../types/money.json#currency", "../types/money.json#currency"},
		{"$dynamicRef", "https://reader:private@types.example.test/money.json?signature=secret#currency", "https://types.example.test/money.json#currency"},
	} {
		t.Run(tc.keyword+tc.label, func(t *testing.T) {
			cfg := datamodel.NewDocumentConfiguration()
			cfg.AllowRemoteReferences = true
			cfg.RemoteURLHandler = func(location string) (*http.Response, error) {
				if tc.keyword == "$dynamicRef" {
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"$dynamicAnchor":"currency"}`))}, nil
				}
				return nil, &url.Error{Op: "Get", URL: location, Err: cause}
			}
			data := fmt.Sprintf("properties:\n  amount:\n    title: Amount\n    %s: %q\n", tc.keyword, tc.ref)
			_, err := jsonschema.Load([]byte(data), root, cfg, true)
			require.ErrorContains(t, err, root+":4:")
			require.ErrorContains(t, err, fmt.Sprintf("%s %q", tc.keyword, tc.label))
			require.NotContains(t, err.Error(), "reader")
			require.NotContains(t, err.Error(), "private")
			require.NotContains(t, err.Error(), "signature")
			require.NotContains(t, err.Error(), "secret")
			if tc.keyword == "$ref" {
				require.ErrorIs(t, err, cause)
			} else {
				require.ErrorContains(t, err, "dynamic scope")
			}
		})
	}
}

func TestCollectionResolvesExplicitAndOutsideAddresses(t *testing.T) {
	dir := t.TempDir()
	cfg := datamodel.NewDocumentConfiguration()
	cfg.BasePath = dir
	cfg.LocalFS = resourceFS(func(name string) (fs.File, error) {
		switch name {
		case "types/currency.json", "../shared.json":
			return fstest.MapFS{"spec.json": {Data: []byte("true")}}.Open("spec.json")
		default:
			return nil, fs.ErrNotExist
		}
	})
	members := map[string][]byte{"order.json": []byte(`{"allOf":[{"$ref":"types/currency.json"},{"$ref":"../shared.json"}]}`)}
	doc, err := jsonschema.LoadCollection(members, dir, cfg)
	require.NoError(t, err)
	require.Len(t, doc.Sources, 3)
	require.Same(t, doc.Sources["../shared.json"].Node, doc.References[1].To)
	doc.Release()
	for _, tc := range []struct{ ref, want string }{
		{"file://unrelated/some.json", "unsupported file URI host"}, {"absent.json", "file does not exist"},
		{"file:relative.json", "Rel:"},
	} {
		members["order.json"] = []byte(fmt.Sprintf(`{"$ref":%q}`, tc.ref))
		doc, err = jsonschema.LoadCollection(members, dir, cfg)
		require.Nil(t, doc)
		require.ErrorContains(t, err, tc.want)
	}
}

func TestUnusableRemoteBaseCannotResolveSibling(t *testing.T) {
	cfg := datamodel.NewDocumentConfiguration()
	cfg.BaseURL = &url.URL{Scheme: "https", Host: "schemas.example.test:invalid"}
	cfg.AllowRemoteReferences = true
	doc, err := jsonschema.Load([]byte(`{"$ref":"other.json"}`), "root.json", cfg, true)
	require.Nil(t, doc)
	require.ErrorContains(t, err, "invalid port")
}

func TestCollectionRetrievalAliasCannotHideAnEmbeddedID(t *testing.T) {
	base := t.TempDir()
	cfg := datamodel.NewDocumentConfiguration()
	cfg.BasePath = filepath.Join(base, "dependencies")
	// The dependency claims the missing snapshot address for a nested
	// resource. Registering its root at that address would resolve the
	// original reference to a different node depending on lookup order.
	cfg.LocalFS = fstest.MapFS{"currency.json": {Data: []byte(`{"$defs":{"shadow":{"$id":"../snapshot/currency.json"}}}`)}}
	doc, err := jsonschema.LoadCollection(map[string][]byte{"order.json": []byte(`{"$ref":"currency.json"}`)}, filepath.Join(base, "snapshot"), cfg)
	require.ErrorContains(t, err, "duplicate schema address")
	require.ErrorContains(t, err, "currency.json")
	require.Nil(t, doc)
}

func TestDependencyDiagnosticsNameTheirOwnFile(t *testing.T) {
	cfg := datamodel.NewDocumentConfiguration()
	cfg.AllowFileReferences, cfg.BasePath = true, "."
	cfg.LocalFS = fstest.MapFS{
		"shape.json":  {Data: []byte("{\n\n\n\"properties\":[]}")},
		"keys.yaml":   {Data: []byte("properties:\n  200: {}\n")},
		"legacy.json": {Data: []byte("{\n\"definitions\":{}}")},
	}
	for file, want := range map[string]string{
		"shape.json":  `order.json:1: resolve $ref "shape.json": shape.json: line 4: properties requires an object`,
		"keys.yaml":   `order.json:1: resolve $ref "keys.yaml": keys.yaml: line 2: JSON object keys must be strings`,
		"legacy.json": `order.json:1: resolve $ref "legacy.json": legacy.json:2: definitions belongs to an earlier draft`,
	} {
		doc, err := jsonschema.Load([]byte(`{"$ref":"`+file+`"}`), "order.json", cfg, true)
		require.ErrorContains(t, err, want)
		require.Nil(t, doc)
	}
	// A declared dialect keeps keywords from earlier drafts as annotations.
	doc, err := jsonschema.Load([]byte(`{"$schema":"`+jsonschema.Dialect+`","definitions":{"price":{"type":"number"}},"dependencies":{}}`), "declared.json", nil, false)
	require.NoError(t, err)
	doc.Release()
}
