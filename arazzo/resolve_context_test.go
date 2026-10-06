// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	libopenapi "github.com/pb33f/libopenapi"
	high "github.com/pb33f/libopenapi/datamodel/high/arazzo"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

// stubOpenAPIFactory satisfies the factory contract without parsing a real document;
// these tests assert on which URL was requested, not on OpenAPI content.
func stubOpenAPIFactory(_ string, _ []byte) (*v3high.Document, error) {
	return &v3high.Document{}, nil
}

// ResolveSourcesWithContext must tolerate a nil context rather than panicking, so that
// callers converting from the context-free ResolveSources cannot regress by passing nil.
func TestResolveSourcesWithContext_NilContextTreatedAsBackground(t *testing.T) {
	doc := &high.Arazzo{}

	sources, err := ResolveSourcesWithContext(nil, doc, nil) //nolint:staticcheck // nil context is the case under test
	require.NoError(t, err)
	assert.Empty(t, sources)
}

// Cancellation must be observed before each source description is processed, so an
// already-cancelled context performs no retrieval at all.
func TestResolveSourcesWithContext_CancelledBeforeFirstSource(t *testing.T) {
	handlerCalls := 0
	doc := &high.Arazzo{
		SourceDescriptions: []*high.SourceDescription{{
			Name: "api",
			URL:  "https://example.com/openapi.yaml",
			Type: "openapi",
		}},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := ResolveSourcesWithContext(ctx, doc, &ResolveConfig{
		HTTPHandler: func(string) ([]byte, error) {
			handlerCalls++
			return []byte("openapi: 3.1.0"), nil
		},
	})

	require.ErrorIs(t, err, context.Canceled)
	assert.Zero(t, handlerCalls, "no source should be fetched after cancellation")
}

// The HTTPHandler path is a caller-supplied hook that bypasses the http.Client, so it
// needs its own cancellation check; otherwise a cancelled context would still invoke it.
func TestFetchHTTPSourceBytes_CancelledBeforeHandler(t *testing.T) {
	handlerCalls := 0
	config := &ResolveConfig{
		MaxBodySize: 1024,
		HTTPHandler: func(string) ([]byte, error) {
			handlerCalls++
			return []byte("content"), nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := fetchHTTPSourceBytes(ctx, "https://example.com/spec.yaml", config)
	require.ErrorIs(t, err, context.Canceled)
	assert.Zero(t, handlerCalls, "handler must not run once the context is cancelled")
}

func TestFetchHTTPSourceBytes_ContextHandlerCancelledInFlight(t *testing.T) {
	started := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	config := &ResolveConfig{
		MaxBodySize: 1024,
		Timeout:     30 * time.Second,
		HTTPHandlerWithContext: func(handlerCtx context.Context, _ string) ([]byte, error) {
			close(started)
			<-handlerCtx.Done()
			return nil, handlerCtx.Err()
		},
	}

	go func() {
		<-started
		cancel()
	}()

	_, err := fetchHTTPSourceBytes(ctx, "https://example.com/spec.yaml", config)
	require.ErrorIs(t, err, context.Canceled)
}

func TestFetchHTTPSourceBytes_ContextHandlerResults(t *testing.T) {
	handlerErr := errors.New("context handler failed")
	for _, test := range []struct {
		name       string
		body       []byte
		handlerErr error
		maxBody    int64
		wantErr    error
		wantText   string
	}{
		{name: "success", body: []byte("openapi: 3.1.0"), maxBody: 1024},
		{name: "handler error", handlerErr: handlerErr, maxBody: 1024, wantErr: handlerErr},
		{name: "oversized", body: []byte("too large"), maxBody: 3, wantText: "exceeds max size"},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := &ResolveConfig{
				MaxBodySize: test.maxBody,
				Timeout:     time.Second,
				HTTPHandlerWithContext: func(_ context.Context, _ string) ([]byte, error) {
					return test.body, test.handlerErr
				},
			}
			body, err := fetchHTTPSourceBytes(context.Background(), "https://example.com/spec.yaml", config)
			if test.wantErr != nil {
				require.ErrorIs(t, err, test.wantErr)
				return
			}
			if test.wantText != "" {
				require.ErrorContains(t, err, test.wantText)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.body, body)
		})
	}
}

// Cancelling mid-flight must abort a real in-progress HTTP fetch. Before the context was
// threaded through, the request used context.Background() and ran to completion.
func TestFetchHTTPSourceBytes_CancelDuringInFlightRequest(t *testing.T) {
	requestArrived := make(chan struct{})
	handlerDone := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(requestArrived)
		<-r.Context().Done() // hold the response open until the client goes away
		close(handlerDone)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Cancel only once the server has the request, so this exercises an in-flight
	// abort rather than a pre-flight rejection.
	go func() {
		<-requestArrived
		cancel()
	}()

	// Timeout must be non-zero: defaults are applied by ResolveSources, and calling
	// the fetch helper directly with a zero timeout would expire the context at once.
	_, err := fetchHTTPSourceBytes(ctx, server.URL, &ResolveConfig{
		MaxBodySize: 1024,
		Timeout:     30 * time.Second,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	<-handlerDone
}

// A document parsed with a retrieval URI carries an effective base URI. Relative source
// URLs must resolve against it without the caller restating the base in ResolveConfig.
func TestResolveSources_UsesEffectiveBaseURIFromDocumentOrigin(t *testing.T) {
	spec := []byte(`arazzo: 1.0.1
info:
  title: base uri test
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: ./nested/openapi.yaml
    type: openapi
workflows:
  - workflowId: wf
    steps:
      - stepId: s1
        operationId: op
`)

	doc, err := libopenapi.NewArazzoDocumentWithConfiguration(spec, &libopenapi.ArazzoDocumentConfiguration{
		RetrievalURI: "https://specs.example.com/workflows/main.arazzo.yaml",
	})
	require.NoError(t, err)

	var requested string
	sources, err := ResolveSources(doc, &ResolveConfig{
		HTTPHandler: func(url string) ([]byte, error) {
			requested = url
			return []byte("openapi: 3.1.0"), nil
		},
		OpenAPIFactory: stubOpenAPIFactory,
	})
	require.NoError(t, err)
	require.Len(t, sources, 1)

	assert.Equal(t, "https://specs.example.com/workflows/nested/openapi.yaml", requested,
		"relative source URL should resolve against the document's effective base URI")
}

// An explicit BaseURL is the caller's stated intent and must win over the document origin.
func TestResolveSources_ExplicitBaseURLOverridesDocumentOrigin(t *testing.T) {
	spec := []byte(`arazzo: 1.0.1
info:
  title: base uri precedence
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: ./openapi.yaml
    type: openapi
workflows:
  - workflowId: wf
    steps:
      - stepId: s1
        operationId: op
`)

	doc, err := libopenapi.NewArazzoDocumentWithConfiguration(spec, &libopenapi.ArazzoDocumentConfiguration{
		RetrievalURI: "https://origin.example.com/workflows/main.yaml",
	})
	require.NoError(t, err)

	var requested string
	_, err = ResolveSources(doc, &ResolveConfig{
		BaseURL: "https://override.example.com/elsewhere/",
		HTTPHandler: func(url string) ([]byte, error) {
			requested = url
			return []byte("openapi: 3.1.0"), nil
		},
		OpenAPIFactory: stubOpenAPIFactory,
	})
	require.NoError(t, err)

	assert.Equal(t, "https://override.example.com/elsewhere/openapi.yaml", requested)
}

// Resolving one document must not leave its base URI on a ResolveConfig that the caller
// reuses for another document.
func TestResolveSources_DoesNotWriteBaseURLBackIntoConfig(t *testing.T) {
	spec := []byte(`arazzo: 1.0.1
info:
  title: config reuse
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: ./openapi.yaml
    type: openapi
workflows:
  - workflowId: wf
    steps:
      - stepId: s1
        operationId: op
`)

	doc, err := libopenapi.NewArazzoDocumentWithConfiguration(spec, &libopenapi.ArazzoDocumentConfiguration{
		RetrievalURI: "https://first.example.com/workflows/main.yaml",
	})
	require.NoError(t, err)

	config := &ResolveConfig{
		HTTPHandler: func(string) ([]byte, error) {
			return []byte("openapi: 3.1.0"), nil
		},
		OpenAPIFactory: stubOpenAPIFactory,
	}

	_, err = ResolveSources(doc, config)
	require.NoError(t, err)

	assert.Empty(t, config.BaseURL,
		"document origin must not leak into a config the caller may reuse")
}

// Defaults and the derived base URI are resolution state, not caller state. Resolving
// must leave the caller's config untouched so it can be reused, or shared across
// concurrent resolves, without one document's settings leaking into another.
func TestResolveSources_LeavesCallerConfigUnmodified(t *testing.T) {
	spec := []byte(`arazzo: 1.0.1
info:
  title: config isolation
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: ./openapi.yaml
    type: openapi
workflows:
  - workflowId: wf
    steps:
      - stepId: s1
        operationId: op
`)

	doc, err := libopenapi.NewArazzoDocumentWithConfiguration(spec, &libopenapi.ArazzoDocumentConfiguration{
		RetrievalURI: "https://example.com/workflows/main.yaml",
	})
	require.NoError(t, err)

	config := &ResolveConfig{
		HTTPHandler: func(string) ([]byte, error) {
			return []byte("openapi: 3.1.0"), nil
		},
		OpenAPIFactory: stubOpenAPIFactory,
	}

	_, err = ResolveSources(doc, config)
	require.NoError(t, err)

	assert.Empty(t, config.BaseURL, "derived base URI must not be written back")
	assert.Zero(t, config.Timeout, "defaults must not be written back")
	assert.Zero(t, config.MaxBodySize)
	assert.Zero(t, config.MaxSources)
	assert.Empty(t, config.AllowedSchemes)
	assert.Nil(t, config.HTTPClient, "no client should be attached to the caller's config")
}

// Concurrent resolves of independent documents sharing one config must not race on the
// defaults block, which previously wrote Timeout, MaxBodySize, MaxSources,
// AllowedSchemes and HTTPClient straight through the caller's pointer.
//
// Each goroutine gets its own document on purpose. Resolving attaches OpenAPI source
// documents to the Arazzo model, so sharing a single document across concurrent resolves
// is a document-mutation race independent of the config, and is not what this covers.
func TestResolveSources_ConcurrentResolvesShareConfigSafely(t *testing.T) {
	spec := []byte(`arazzo: 1.0.1
info:
  title: concurrent
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
    type: openapi
workflows:
  - workflowId: wf
    steps:
      - stepId: s1
        operationId: op
`)

	config := &ResolveConfig{
		HTTPHandler: func(string) ([]byte, error) {
			return []byte("openapi: 3.1.0"), nil
		},
		OpenAPIFactory: stubOpenAPIFactory,
	}

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			doc, err := libopenapi.NewArazzoDocument(spec)
			if !assert.NoError(t, err) {
				return
			}
			_, resolveErr := ResolveSources(doc, config)
			assert.NoError(t, resolveErr)
		}()
	}
	wg.Wait()

	// The shared config must still be pristine after concurrent use.
	assert.Zero(t, config.Timeout)
	assert.Nil(t, config.HTTPClient)
	assert.Empty(t, config.AllowedSchemes)
}

// Source resolution attaches OpenAPI documents to the Arazzo model it was handed, so two
// goroutines resolving the same document mutate shared state. This reproduced a data race
// on Arazzo.openAPISourceDocs before that slice was guarded.
func TestResolveSources_ConcurrentResolvesOfSameDocumentAreSafe(t *testing.T) {
	spec := []byte(`arazzo: 1.0.1
info:
  title: shared document
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
    type: openapi
workflows:
  - workflowId: wf
    steps:
      - stepId: s1
        operationId: op
`)

	doc, err := libopenapi.NewArazzoDocument(spec)
	require.NoError(t, err)

	config := &ResolveConfig{
		HTTPHandler: func(string) ([]byte, error) {
			return []byte("openapi: 3.1.0"), nil
		},
		OpenAPIFactory: stubOpenAPIFactory,
	}

	const resolvers = 16
	var wg sync.WaitGroup
	for i := 0; i < resolvers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, resolveErr := ResolveSources(doc, config)
			assert.NoError(t, resolveErr)
		}()
	}
	wg.Wait()

	// Every resolve attaches exactly one source document; none may be lost to a
	// concurrent append.
	assert.Len(t, doc.GetOpenAPISourceDocuments(), resolvers)
}

// Attaching source documents must be safe concurrently with reading them, which is what
// validation does while another goroutine resolves.
func TestArazzo_AttachAndReadSourceDocumentsConcurrently(t *testing.T) {
	spec := []byte(`arazzo: 1.0.1
info:
  title: concurrent attach and read
  version: 1.0.0
workflows:
  - workflowId: wf
    steps:
      - stepId: s1
        operationId: op
`)

	doc, err := libopenapi.NewArazzoDocument(spec)
	require.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			doc.AddOpenAPISourceDocument(&v3high.Document{})
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			// The returned slice is a copy, so ranging it must stay valid even as
			// more documents are attached.
			for _, attached := range doc.GetOpenAPISourceDocuments() {
				_ = attached
			}
		}()
	}
	wg.Wait()

	assert.Len(t, doc.GetOpenAPISourceDocuments(), 16)
}
