// Copyright 2022-2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package libopenapi

import (
	"os"
	"sync"
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	v2high "github.com/pb33f/libopenapi/datamodel/high/v2"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

const concurrentCallers = 16

// runConcurrently starts every call at the same moment and waits for all of them.
func runConcurrently(calls ...func()) {
	var start, done sync.WaitGroup
	start.Add(1)
	for _, call := range calls {
		done.Add(1)
		go func() {
			defer done.Done()
			start.Wait()
			call()
		}()
	}
	start.Done()
	done.Wait()
}

func TestDocument_BuildV3Model_ConcurrentCallsShareOneModel(t *testing.T) {
	spec, err := os.ReadFile("test_specs/burgershop.openapi.yaml")
	require.NoError(t, err)
	doc, err := NewDocument(spec)
	require.NoError(t, err)

	models := make([]*DocumentModel[v3high.Document], concurrentCallers)
	errs := make([]error, concurrentCallers)
	var calls []func()
	for i := range concurrentCallers {
		calls = append(calls, func() { models[i], errs[i] = doc.BuildV3Model() })
	}
	// readers run alongside the builds
	calls = append(calls,
		func() { _ = doc.GetRolodex() },
		func() { _ = doc.GetSpecInfo() },
		func() { _ = doc.GetConfiguration() },
		func() { _, _ = doc.Render() },
		func() { _, _ = doc.Serialize() },
	)
	runConcurrently(calls...)

	for i := range concurrentCallers {
		assert.NoError(t, errs[i])
		require.NotNil(t, models[i])
		assert.Same(t, models[0], models[i], "every caller gets the one cached model")
	}
	assert.NotNil(t, doc.GetRolodex())
	assert.NotNil(t, doc.GetConfiguration())

	rendered, renderErr := doc.Render()
	assert.NoError(t, renderErr)
	assert.NotEmpty(t, rendered)
}

func TestDocument_BuildV2Model_ConcurrentCallsShareOneModel(t *testing.T) {
	spec, err := os.ReadFile("test_specs/petstorev2.json")
	require.NoError(t, err)
	doc, err := NewDocument(spec)
	require.NoError(t, err)

	models := make([]*DocumentModel[v2high.Swagger], concurrentCallers)
	errs := make([]error, concurrentCallers)
	var calls []func()
	for i := range concurrentCallers {
		calls = append(calls, func() { models[i], errs[i] = doc.BuildV2Model() })
	}
	calls = append(calls,
		func() { _ = doc.GetRolodex() },
		func() { _ = doc.GetConfiguration() },
	)
	runConcurrently(calls...)

	for i := range concurrentCallers {
		assert.NoError(t, errs[i])
		require.NotNil(t, models[i])
		assert.Same(t, models[0], models[i], "every caller gets the one cached model")
	}
}

func TestDocument_ConfigurationAndReleaseAreSafeDuringBuilds(t *testing.T) {
	spec, err := os.ReadFile("test_specs/burgershop.openapi.yaml")
	require.NoError(t, err)
	doc, err := NewDocument(spec)
	require.NoError(t, err)

	var calls []func()
	for range concurrentCallers {
		calls = append(calls,
			func() { _, _ = doc.BuildV3Model() },
			func() { doc.SetConfiguration(datamodel.NewDocumentConfiguration()) },
		)
	}
	calls = append(calls, doc.Release)
	runConcurrently(calls...)

	// Release cleared the document, whichever order the calls ran in
	doc.Release()
	model, buildErr := doc.BuildV3Model()
	assert.Nil(t, model)
	assert.Error(t, buildErr)
	assert.Nil(t, doc.GetSpecInfo())
	assert.Nil(t, doc.GetRolodex())
}
