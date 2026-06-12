// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package datamodel

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const lazyJSONYAML = `openapi: 3.0.1
info:
  title: lazy
  version: 1.0.0
paths: {}`

const lazyJSONJSON = `{"openapi":"3.0.1","info":{"title":"lazy","version":"1.0.0"},"paths":{}}`

func TestSpecInfo_LazyJSON_YAMLInput(t *testing.T) {
	r, err := ExtractSpecInfo([]byte(lazyJSONYAML))
	require.NoError(t, err)

	// nothing is built eagerly.
	assert.Nil(t, r.SpecJSON)
	assert.Nil(t, r.SpecJSONBytes)

	j := r.GetSpecJSON()
	require.NotNil(t, j)
	assert.Equal(t, "3.0.1", (*j)["openapi"])
	assert.NoError(t, r.GetSpecJSONError())

	b := r.GetSpecJSONBytes()
	require.NotNil(t, b)
	assert.Greater(t, len(*b), 0)

	// accessors populate the public fields for mixed readers.
	assert.Equal(t, j, r.SpecJSON)
	assert.Equal(t, b, r.SpecJSONBytes)
}

func TestSpecInfo_LazyJSON_JSONInput(t *testing.T) {
	r, err := ExtractSpecInfo([]byte(lazyJSONJSON))
	require.NoError(t, err)

	assert.Nil(t, r.SpecJSON)

	j := r.GetSpecJSON()
	require.NotNil(t, j)
	assert.Equal(t, "3.0.1", (*j)["openapi"])

	// JSON input: the bytes are the original document, not a copy.
	b := r.GetSpecJSONBytes()
	require.NotNil(t, b)
	assert.Equal(t, r.SpecBytes, b)
}

func TestSpecInfo_LazyJSON_Concurrent(t *testing.T) {
	r, err := ExtractSpecInfo([]byte(lazyJSONYAML))
	require.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			assert.NotNil(t, r.GetSpecJSON())
			assert.NotNil(t, r.GetSpecJSONBytes())
		}()
	}
	wg.Wait()
}

func TestSpecInfo_LazyJSON_SkipJSONConversion(t *testing.T) {
	r, err := ExtractSpecInfoWithConfig([]byte(lazyJSONYAML), &DocumentConfiguration{
		SkipJSONConversion: true,
	})
	require.NoError(t, err)
	// the flag's contract holds for accessors too: no JSON representation, ever.
	// consumers (e.g. vacuum turbo mode) rely on nil as the "conversion disabled" signal.
	assert.Nil(t, r.SpecJSON)
	assert.Nil(t, r.GetSpecJSON())
	assert.Nil(t, r.GetSpecJSONBytes())
	assert.NoError(t, r.GetSpecJSONError())
}

func TestSpecInfo_LazyJSON_AfterRelease(t *testing.T) {
	r, err := ExtractSpecInfo([]byte(lazyJSONYAML))
	require.NoError(t, err)

	r.Release()

	// released: inputs are gone, accessors return nil without panicking and
	// nothing is resurrected. no error either - nothing was built, nothing failed.
	assert.Nil(t, r.GetSpecJSON())
	assert.Nil(t, r.GetSpecJSONBytes())
	assert.NoError(t, r.GetSpecJSONError())
	assert.Nil(t, r.SpecJSON)
	assert.Nil(t, r.SpecJSONBytes)

	// same nil-safety for JSON input, which builds from the original bytes.
	rj, err := ExtractSpecInfo([]byte(lazyJSONJSON))
	require.NoError(t, err)
	rj.Release()
	assert.Nil(t, rj.GetSpecJSON())
	assert.Nil(t, rj.GetSpecJSONBytes())
}

func TestSpecInfo_LazyJSON_YAMLDecodeError(t *testing.T) {
	// a tagged scalar that cannot decode to its tag passes extraction (only duplicate
	// keys are validated eagerly) but fails the lazy build.
	spec := "openapi: 3.0.1\ninfo:\n  title: lazy\n  version: !!int notanint\npaths: {}"
	r, err := ExtractSpecInfo([]byte(spec))
	require.NoError(t, err)

	assert.Nil(t, r.GetSpecJSON())
	assert.Nil(t, r.GetSpecJSONBytes())
	assert.ErrorContains(t, r.GetSpecJSONError(), "failed to decode YAML to JSON")
}

func TestSpecInfo_LazyJSON_JSONUnmarshalError(t *testing.T) {
	// bypass lets structurally invalid JSON through extraction; the lazy build
	// surfaces the decode failure instead.
	bad := `{"openapi": }`
	r, err := ExtractSpecInfoWithDocumentCheck([]byte(bad), true)
	require.NoError(t, err)
	r.SpecFileType = JSONFileType

	// GetSpecJSONError builds on first call, so it works standalone too.
	assert.ErrorContains(t, r.GetSpecJSONError(), "failed to unmarshal JSON")
	assert.Nil(t, r.GetSpecJSON())
}
