// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package bundler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

// https://github.com/pb33f/libopenapi/issues/733
func TestBundleDocumentComposed_PreservesRootOrderOfReferencedItems(t *testing.T) {
	dir := t.TempDir()
	spec := []byte(`openapi: 3.0.3
info: {title: t, version: '1'}
paths:
  /c:
    get:
      responses:
        '200': {description: ok}
      callbacks:
        onEvent:
          '{$request.body#/c}':
            post:
              responses:
                '200': {description: ok}
          '{$request.body#/b}':
            $ref: './refs.yaml#/items/b'
          '{$request.body#/a}':
            $ref: './refs.yaml#/items/a'
  /b:
    $ref: './refs.yaml#/items/b'
  /a:
    $ref: './refs.yaml#/items/a'
  /d:
    get:
      responses:
        '200': {description: ok}
`)
	refs := []byte(`items:
  a:
    post:
      responses:
        '200': {description: ok}
  b:
    post:
      responses:
        '200': {description: ok}
`)
	specPath := filepath.Join(dir, "openapi.yaml")
	require.NoError(t, os.WriteFile(specPath, spec, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "refs.yaml"), refs, 0o600))

	doc, err := libopenapi.NewDocumentWithConfiguration(spec, &datamodel.DocumentConfiguration{
		SpecFilePath:        specPath,
		BasePath:            dir,
		AllowFileReferences: true,
	})
	require.NoError(t, err)
	model, err := doc.BuildV3Model()
	require.NoError(t, err)

	paths := model.Model.Paths
	callback := paths.PathItems.GetOrZero("/c").Get.Callbacks.GetOrZero("onEvent")

	wantPaths := []string{"/c", "/b", "/a", "/d"}
	wantExpressions := []string{"{$request.body#/c}", "{$request.body#/b}", "{$request.body#/a}"}

	renderedKeys := func(v interface{ MarshalYAML() (interface{}, error) }, inline bool) []string {
		var out interface{}
		var err error
		if inline {
			out, err = v.(interface{ MarshalYAMLInline() (interface{}, error) }).MarshalYAMLInline()
		} else {
			out, err = v.MarshalYAML()
		}
		require.NoError(t, err)
		var keys []string
		content := out.(*yaml.Node).Content
		for i := 0; i < len(content); i += 2 {
			keys = append(keys, content[i].Value)
		}
		return keys
	}

	for _, inline := range []bool{false, true} {
		assert.Equal(t, wantPaths, renderedKeys(paths, inline), "paths, inline=%t", inline)
		assert.Equal(t, wantExpressions, renderedKeys(callback, inline), "callback, inline=%t", inline)
	}

	bundled, err := BundleDocumentComposed(&model.Model, nil)
	require.NoError(t, err)

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal(bundled, &root))
	var got []string
	top := root.Content[0].Content
	for i := 0; i < len(top); i += 2 {
		if top[i].Value == "paths" {
			for j := 0; j < len(top[i+1].Content); j += 2 {
				got = append(got, top[i+1].Content[j].Value)
			}
		}
	}
	assert.Equal(t, wantPaths, got)
}
