package index

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestResolveReferenceValue_FromIndex(t *testing.T) {
	spec := []byte(`openapi: 3.0.0
components:
  schemas:
    Label:
      type: string
`)

	var root yaml.Node
	err := yaml.Unmarshal(spec, &root)
	assert.NoError(t, err)

	specIndex := NewSpecIndexWithConfig(&root, CreateOpenAPIIndexConfig())
	resolved := ResolveReferenceValue("#/components/schemas/Label", specIndex, nil)
	asMap, ok := resolved.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "string", asMap["type"])
}

func TestResolveReferenceValue_DoesNotLoadDocumentDataForNonLocalRefs(t *testing.T) {
	loadCount := 0
	resolved := ResolveReferenceValue("https://example.com/openapi.yaml#/components/schemas/Foo", nil,
		func() map[string]interface{} {
			loadCount++
			return nil
		})
	assert.Nil(t, resolved)
	assert.Equal(t, 0, loadCount)
}

func TestResolveReferenceValue_DoesNotLoadDocumentDataForUnsupportedLocalAnchorRefs(t *testing.T) {
	loadCount := 0
	resolved := ResolveReferenceValue("#anchor-name", nil, func() map[string]interface{} {
		loadCount++
		return nil
	})
	assert.Nil(t, resolved)
	assert.Equal(t, 0, loadCount)
}

func TestResolveReferenceValue_LoadsDocumentDataWhenIndexMissing(t *testing.T) {
	loadCount := 0
	resolved := ResolveReferenceValue("#/components/responses/BadRequest", nil, func() map[string]interface{} {
		loadCount++
		return map[string]interface{}{
			"components": map[string]interface{}{
				"responses": map[string]interface{}{
					"BadRequest": map[string]interface{}{
						"description": "bad request",
					},
				},
			},
		}
	})

	assert.Equal(t, 1, loadCount)
	asMap, ok := resolved.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "bad request", asMap["description"])
}

func TestResolveReferenceValue_LocalPointerFallbackRootPointer(t *testing.T) {
	resolved := ResolveReferenceValue("#", nil, func() map[string]interface{} {
		return map[string]interface{}{
			"openapi": "3.1.0",
		}
	})
	asMap, ok := resolved.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "3.1.0", asMap["openapi"])
}

func TestResolveReferenceValue_LocalPointerFallbackHandlesEscapesAndArrays(t *testing.T) {
	resolved := ResolveReferenceValue("#/a~1b/c~0d/1/name", nil, func() map[string]interface{} {
		return map[string]interface{}{
			"a/b": map[string]interface{}{
				"c~d": []interface{}{
					"zero",
					map[string]interface{}{"name": "ok"},
				},
			},
		}
	})
	assert.Equal(t, "ok", resolved)
}

func TestResolveReferenceValue_LocalPointerFallbackHandlesURLEncodedSegments(t *testing.T) {
	resolved := ResolveReferenceValue("#/paths/~1v1~1pets~1%7Bid%7D/get", nil, func() map[string]interface{} {
		return map[string]interface{}{
			"paths": map[string]interface{}{
				"/v1/pets/{id}": map[string]interface{}{
					"get": map[string]interface{}{
						"operationId": "getPet",
					},
				},
			},
		}
	})
	asMap, ok := resolved.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "getPet", asMap["operationId"])
}

func TestResolveReferenceValue_LocalPointerFallbackRequiresDocProvider(t *testing.T) {
	assert.Nil(t, ResolveReferenceValue("#/components/schemas/Foo", nil, nil))
}

func TestResolveReferenceValue_LocalPointerFallbackRequiresDocData(t *testing.T) {
	assert.Nil(t, ResolveReferenceValue("#/components/schemas/Foo", nil, func() map[string]interface{} {
		return nil
	}))
}

func TestResolveReferenceValue_EmptyRefReturnsNil(t *testing.T) {
	assert.Nil(t, ResolveReferenceValue("", nil, nil))
}

func TestResolveLocalJSONPointer_InvalidInputs(t *testing.T) {
	assert.Nil(t, resolveLocalJSONPointer(nil, ""))
	assert.Nil(t, resolveLocalJSONPointer(map[string]interface{}{}, "components/schemas/Foo"))
}

func TestResolveLocalJSONPointer_RootPointerReturnsDocument(t *testing.T) {
	doc := map[string]interface{}{"a": "b"}
	assert.Equal(t, doc, resolveLocalJSONPointer(doc, "#"))
}

func TestResolveLocalJSONPointer_MissingMapKeyReturnsNil(t *testing.T) {
	doc := map[string]interface{}{
		"components": map[string]interface{}{},
	}
	assert.Nil(t, resolveLocalJSONPointer(doc, "#/components/schemas/Foo"))
}

func TestResolveLocalJSONPointer_InvalidArrayIndexesReturnNil(t *testing.T) {
	doc := map[string]interface{}{
		"items": []interface{}{"a"},
	}
	assert.Nil(t, resolveLocalJSONPointer(doc, "#/items/not-an-int"))
	assert.Nil(t, resolveLocalJSONPointer(doc, "#/items/2"))
	assert.Nil(t, resolveLocalJSONPointer(doc, "#/items/-1"))
}

func TestResolveLocalJSONPointer_UnsupportedIntermediateTypeReturnsNil(t *testing.T) {
	doc := map[string]interface{}{
		"a": "scalar",
	}
	assert.Nil(t, resolveLocalJSONPointer(doc, "#/a/b"))
}
