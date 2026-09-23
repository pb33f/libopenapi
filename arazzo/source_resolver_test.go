// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	high "github.com/pb33f/libopenapi/datamodel/high/arazzo"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
	low "github.com/pb33f/libopenapi/datamodel/low/arazzo"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

type staticCandidateProvider struct {
	documents []CandidateDocument
	err       error
	calls     int
}

func (p *staticCandidateProvider) Documents(context.Context) ([]CandidateDocument, error) {
	p.calls++
	return p.documents, p.err
}

type testSourceAdapter struct {
	sourceType string
}

func (a *testSourceAdapter) SourceType() string {
	return a.sourceType
}

type resolverFunc func(context.Context, SourceRequest) (*ResolvedSource, error)

func (f resolverFunc) Resolve(ctx context.Context, request SourceRequest) (*ResolvedSource, error) {
	return f(ctx, request)
}

func TestNewInMemorySourceResolver_EmptyAndProviderErrors(t *testing.T) {
	resolver, err := NewInMemorySourceResolver(context.Background(), nil, nil)
	require.NoError(t, err)
	require.NotNil(t, resolver)

	sentinel := errors.New("provider failed")
	provider := &staticCandidateProvider{err: sentinel}
	resolver, err = NewInMemorySourceResolver(context.Background(), provider, nil)
	assert.Nil(t, resolver)
	require.Error(t, err)
	assert.ErrorIs(t, err, sentinel)
	assert.Equal(t, 1, provider.calls)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	provider = &staticCandidateProvider{documents: []CandidateDocument{{Type: "openapi"}}}
	resolver, err = NewInMemorySourceResolver(ctx, provider, nil)
	assert.Nil(t, resolver)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestNewInMemorySourceResolver_MaxSources(t *testing.T) {
	documents := []CandidateDocument{
		{
			ResolvedIdentity: "https://example.com/one.yaml",
			Adapter:          &testSourceAdapter{sourceType: "asyncapi"},
		},
		{
			ResolvedIdentity: "https://example.com/two.yaml",
			Adapter:          &testSourceAdapter{sourceType: "asyncapi"},
		},
	}
	provider := &staticCandidateProvider{documents: documents}

	resolver, err := NewInMemorySourceResolver(
		context.Background(),
		provider,
		&InMemoryResolverConfig{MaxSources: 1},
	)
	assert.Nil(t, resolver)
	assert.EqualError(t, err, "too many candidate source documents: 2 (max 1)")

	resolver, err = NewInMemorySourceResolver(
		context.Background(),
		provider,
		&InMemoryResolverConfig{MaxSources: len(documents)},
	)
	require.NoError(t, err)
	require.NotNil(t, resolver)

	resolver, err = NewInMemorySourceResolver(
		context.Background(),
		&staticCandidateProvider{documents: make([]CandidateDocument, defaultMaxCandidateSources+1)},
		&InMemoryResolverConfig{MaxSources: -1},
	)
	assert.Nil(t, resolver)
	assert.EqualError(t, err, "too many candidate source documents: 51 (max 50)")
}

func TestNewInMemorySourceResolver_BuildsAllCandidateShapes(t *testing.T) {
	var openAPICalls, arazzoCalls int
	arazzoDocument := high.NewArazzoWithOrigin(&low.Arazzo{}, &high.DocumentOrigin{
		ResolvedIdentity: "https://identity.example/arazzo.yaml",
	})
	provider := &staticCandidateProvider{documents: []CandidateDocument{
		{
			RetrievalURI:    "https://retrieval.example/openapi.yaml",
			OpenAPIDocument: &v3high.Document{},
		},
		{
			RetrievalURI:   "https://retrieval.example/arazzo.yaml",
			ArazzoDocument: arazzoDocument,
		},
		{
			RetrievalURI: "https://retrieval.example/raw-openapi.yaml",
			Type:         "openapi",
			SourceBytes:  []byte("openapi: 3.1.0"),
		},
		{
			RetrievalURI: "https://retrieval.example/raw-arazzo.yaml",
			Type:         "arazzo",
			RootNode:     &yaml.Node{Kind: yaml.MappingNode},
		},
		{
			ResolvedIdentity: "https://identity.example/asyncapi.yaml",
			RetrievalURI:     "https://retrieval.example/asyncapi.yaml",
			Adapter:          &testSourceAdapter{sourceType: "asyncapi"},
		},
	}}
	resolver, err := NewInMemorySourceResolver(context.Background(), provider, &InMemoryResolverConfig{
		OpenAPIFactory: func(sourceURL string, bytes []byte) (*v3high.Document, error) {
			openAPICalls++
			assert.Equal(t, "https://retrieval.example/raw-openapi.yaml", sourceURL)
			assert.NotEmpty(t, bytes)
			return &v3high.Document{}, nil
		},
		ArazzoFactory: func(sourceURL string, bytes []byte) (*high.Arazzo, error) {
			arazzoCalls++
			assert.Equal(t, "https://retrieval.example/raw-arazzo.yaml", sourceURL)
			assert.NotEmpty(t, bytes)
			return &high.Arazzo{Self: "https://identity.example/raw-arazzo.yaml"}, nil
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, openAPICalls)
	assert.Equal(t, 1, arazzoCalls)

	source, err := resolver.Resolve(context.Background(), SourceRequest{
		Name: "portable",
		URL:  "https://identity.example/arazzo.yaml",
		Type: "Arazzo",
	})
	require.NoError(t, err)
	assert.Equal(t, "portable", source.Name)
	assert.Equal(t, "arazzo", source.Type)
	assert.Same(t, arazzoDocument, source.ArazzoDocument)

	source, err = resolver.Resolve(context.Background(), SourceRequest{
		URL: "https://retrieval.example/openapi.yaml",
	})
	require.NoError(t, err)
	assert.NotNil(t, source.OpenAPIDocument)

	source, err = resolver.Resolve(context.Background(), SourceRequest{
		URL: "https://identity.example/asyncapi.yaml",
	})
	require.NoError(t, err)
	assert.Equal(t, "asyncapi", source.Type)
	assert.NotNil(t, source.Adapter)
}

func TestNewInMemorySourceResolver_CandidateFailures(t *testing.T) {
	sentinel := errors.New("parse failed")
	tests := []struct {
		name      string
		candidate CandidateDocument
		config    *InMemoryResolverConfig
	}{
		{
			name:      "missing OpenAPI factory",
			candidate: CandidateDocument{Type: "openapi"},
		},
		{
			name:      "OpenAPI factory error",
			candidate: CandidateDocument{Type: "openapi"},
			config: &InMemoryResolverConfig{OpenAPIFactory: func(string, []byte) (*v3high.Document, error) {
				return nil, sentinel
			}},
		},
		{
			name:      "missing Arazzo factory",
			candidate: CandidateDocument{Type: "arazzo"},
		},
		{
			name:      "Arazzo factory error",
			candidate: CandidateDocument{Type: "arazzo"},
			config: &InMemoryResolverConfig{ArazzoFactory: func(string, []byte) (*high.Arazzo, error) {
				return nil, sentinel
			}},
		},
		{
			name:      "unsupported without adapter",
			candidate: CandidateDocument{Type: "asyncapi"},
		},
		{
			name: "root node render error",
			candidate: CandidateDocument{
				Type:     "openapi",
				RootNode: &yaml.Node{Kind: yaml.Kind(99)},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolver, err := NewInMemorySourceResolver(
				context.Background(),
				&staticCandidateProvider{documents: []CandidateDocument{test.candidate}},
				test.config,
			)
			assert.Nil(t, resolver)
			require.Error(t, err)
		})
	}
}

func TestNewInMemorySourceResolver_IdentityFallbacksAndDuplicates(t *testing.T) {
	selfDocument := &high.Arazzo{Self: "https://identity.example/self.yaml"}
	resolver, err := NewInMemorySourceResolver(
		context.Background(),
		&staticCandidateProvider{documents: []CandidateDocument{{
			Type:           "arazzo",
			RetrievalURI:   "https://retrieval.example/self.yaml",
			ArazzoDocument: selfDocument,
		}}},
		nil,
	)
	require.NoError(t, err)
	source, err := resolver.Resolve(context.Background(), SourceRequest{URL: selfDocument.Self})
	require.NoError(t, err)
	assert.Equal(t, selfDocument.Self, source.Identity)

	for _, documents := range [][]CandidateDocument{
		{
			{ResolvedIdentity: "duplicate", OpenAPIDocument: &v3high.Document{}},
			{ResolvedIdentity: "duplicate", OpenAPIDocument: &v3high.Document{}},
		},
		{
			{ResolvedIdentity: "one", RetrievalURI: "duplicate", OpenAPIDocument: &v3high.Document{}},
			{ResolvedIdentity: "two", RetrievalURI: "duplicate", OpenAPIDocument: &v3high.Document{}},
		},
		{
			{ResolvedIdentity: "shared", RetrievalURI: "one", OpenAPIDocument: &v3high.Document{}},
			{ResolvedIdentity: "two", RetrievalURI: "shared", OpenAPIDocument: &v3high.Document{}},
		},
		{
			{ResolvedIdentity: "one", RetrievalURI: "shared", OpenAPIDocument: &v3high.Document{}},
			{ResolvedIdentity: "shared", RetrievalURI: "two", OpenAPIDocument: &v3high.Document{}},
		},
	} {
		resolver, err = NewInMemorySourceResolver(
			context.Background(),
			&staticCandidateProvider{documents: documents},
			nil,
		)
		assert.Nil(t, resolver)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrDuplicateSourceIdentity)
	}
}

func TestInMemorySourceResolver_RelativeLookupAndErrors(t *testing.T) {
	resolver, err := NewInMemorySourceResolver(
		context.Background(),
		&staticCandidateProvider{documents: []CandidateDocument{{
			ResolvedIdentity: "https://example.com/specs/api.yaml",
			RetrievalURI:     "https://retrieval.example/api.yaml",
			OpenAPIDocument:  &v3high.Document{},
		}}},
		nil,
	)
	require.NoError(t, err)
	source, err := resolver.Resolve(context.Background(), SourceRequest{
		Name:    "api",
		URL:     "../specs/api.yaml",
		BaseURI: "https://example.com/workflows/root.yaml",
	})
	require.NoError(t, err)
	assert.Equal(t, "api", source.Name)
	source, err = resolver.Resolve(context.Background(), SourceRequest{URL: "https://retrieval.example/api.yaml"})
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/specs/api.yaml", source.Identity)

	_, err = resolver.Resolve(context.Background(), SourceRequest{URL: "https://example.com/%zz"})
	require.Error(t, err)
	_, err = resolver.Resolve(context.Background(), SourceRequest{URL: "api.yaml", BaseURI: "https://example.com/%zz"})
	require.Error(t, err)
	_, err = resolver.Resolve(context.Background(), SourceRequest{URL: "missing.yaml"})
	assert.ErrorIs(t, err, ErrUnresolvedSourceDesc)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = resolver.Resolve(ctx, SourceRequest{URL: "anything"})
	assert.ErrorIs(t, err, context.Canceled)

	var nilResolver *InMemorySourceResolver
	_, err = nilResolver.Resolve(context.Background(), SourceRequest{})
	require.Error(t, err)
}

func TestInMemorySourceResolver_FallbackDeduplicationAndStates(t *testing.T) {
	var calls atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	fallback := resolverFunc(func(_ context.Context, request SourceRequest) (*ResolvedSource, error) {
		if calls.Add(1) == 1 {
			close(started)
		}
		<-release
		return &ResolvedSource{URL: request.URL, Identity: request.URL, Type: "openapi"}, nil
	})
	resolver, err := NewInMemorySourceResolver(context.Background(), nil, &InMemoryResolverConfig{Fallback: fallback})
	require.NoError(t, err)

	var wait sync.WaitGroup
	wait.Add(2)
	results := make(chan *ResolvedSource, 2)
	errs := make(chan error, 2)
	for range 2 {
		go func() {
			defer wait.Done()
			source, resolveErr := resolver.Resolve(context.Background(), SourceRequest{
				Name: "api",
				URL:  "https://example.com/api.yaml",
			})
			results <- source
			errs <- resolveErr
		}()
	}
	<-started
	close(release)
	wait.Wait()
	close(results)
	close(errs)
	for resolveErr := range errs {
		require.NoError(t, resolveErr)
	}
	for source := range results {
		require.NotNil(t, source)
		assert.Equal(t, "api", source.Name)
	}
	assert.EqualValues(t, 1, calls.Load())

	source, err := resolver.Resolve(context.Background(), SourceRequest{URL: "https://example.com/api.yaml"})
	require.NoError(t, err)
	assert.NotNil(t, source)
	assert.EqualValues(t, 1, calls.Load())

	// A completed cycle placeholder is intentionally held only in the load table,
	// so exercise the loaded-entry lookup independently of the permanent maps.
	resolver.loads["loaded-cycle"] = &resolverEntry{
		state:  resolverLoaded,
		source: &ResolvedSource{Identity: "loaded-cycle", URL: "loaded-cycle"},
	}
	source, err = resolver.Resolve(context.Background(), SourceRequest{Name: "cycle", URL: "loaded-cycle"})
	require.NoError(t, err)
	assert.Equal(t, "cycle", source.Name)

	resolver.loads["unseen"] = &resolverEntry{state: resolverUnseen}
	_, err = resolver.Resolve(context.Background(), SourceRequest{URL: "unseen"})
	require.NoError(t, err)
}

func TestInMemorySourceResolver_IndexesFallbackCanonicalIdentity(t *testing.T) {
	var calls atomic.Int32
	fallback := resolverFunc(func(_ context.Context, request SourceRequest) (*ResolvedSource, error) {
		calls.Add(1)
		return &ResolvedSource{
			URL:          request.URL,
			RetrievalURI: "https://retrieval.example/workflow.yaml",
			Identity:     "https://identity.example/workflow.yaml",
			Type:         "arazzo",
		}, nil
	})
	resolver, err := NewInMemorySourceResolver(
		context.Background(), nil, &InMemoryResolverConfig{Fallback: fallback},
	)
	require.NoError(t, err)

	for _, sourceURL := range []string{
		"https://retrieval.example/workflow.yaml",
		"https://identity.example/workflow.yaml",
	} {
		source, resolveErr := resolver.Resolve(context.Background(), SourceRequest{URL: sourceURL})
		require.NoError(t, resolveErr)
		assert.Equal(t, "https://identity.example/workflow.yaml", source.Identity)
	}
	assert.EqualValues(t, 1, calls.Load(), "identity lookup must reuse the retrieval load")
}

func TestInMemorySourceResolver_ContextFailuresAreRetryable(t *testing.T) {
	for _, transientErr := range []error{context.Canceled, context.DeadlineExceeded} {
		t.Run(transientErr.Error(), func(t *testing.T) {
			var calls atomic.Int32
			fallback := resolverFunc(func(_ context.Context, request SourceRequest) (*ResolvedSource, error) {
				if calls.Add(1) == 1 {
					return nil, fmt.Errorf("first caller ended: %w", transientErr)
				}
				return &ResolvedSource{URL: request.URL, Identity: request.URL, Type: "openapi"}, nil
			})
			resolver, err := NewInMemorySourceResolver(
				context.Background(), nil, &InMemoryResolverConfig{Fallback: fallback},
			)
			require.NoError(t, err)

			request := SourceRequest{URL: "https://example.com/retry.yaml"}
			_, err = resolver.Resolve(context.Background(), request)
			require.ErrorIs(t, err, transientErr)

			source, err := resolver.Resolve(context.Background(), request)
			require.NoError(t, err)
			require.NotNil(t, source)
			assert.Equal(t, request.URL, source.Identity)
			assert.EqualValues(t, 2, calls.Load())
		})
	}
}

func TestInMemorySourceResolver_LoadingCyclesCancellationAndFailures(t *testing.T) {
	resolver, err := NewInMemorySourceResolver(context.Background(), nil, &InMemoryResolverConfig{
		Fallback: resolverFunc(func(context.Context, SourceRequest) (*ResolvedSource, error) {
			return nil, errors.New("unexpected fallback")
		}),
	})
	require.NoError(t, err)
	loading := &resolverEntry{
		state:  resolverLoading,
		source: &ResolvedSource{Identity: "cycle", URL: "cycle"},
		ready:  make(chan struct{}),
	}
	resolver.loads["cycle"] = loading
	cycleContext := withResolverLoad(withResolverLoad(context.Background(), "parent"), "cycle")
	source, err := resolver.Resolve(cycleContext, SourceRequest{Name: "self", URL: "cycle"})
	require.NoError(t, err)
	assert.Equal(t, "self", source.Name)
	assert.True(t, resolverLoadActive(cycleContext, "parent"))

	waitingContext, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, resolveErr := resolver.Resolve(waitingContext, SourceRequest{URL: "cycle"})
		result <- resolveErr
	}()
	require.Eventually(t, func() bool {
		resolver.mu.Lock()
		defer resolver.mu.Unlock()
		return loading.waiters > 0
	}, time.Second, time.Millisecond)
	cancel()
	assert.ErrorIs(t, <-result, context.Canceled)

	readyEntry := &resolverEntry{
		state: resolverLoading,
		ready: make(chan struct{}),
	}
	resolver.loads["ready"] = readyEntry
	readyResult := make(chan *ResolvedSource, 1)
	readyError := make(chan error, 1)
	go func() {
		resolved, resolveErr := resolver.Resolve(context.Background(), SourceRequest{Name: "ready", URL: "ready"})
		readyResult <- resolved
		readyError <- resolveErr
	}()
	require.Eventually(t, func() bool {
		resolver.mu.Lock()
		defer resolver.mu.Unlock()
		return readyEntry.waiters > 0
	}, time.Second, time.Millisecond)
	resolver.mu.Lock()
	readyEntry.source = &ResolvedSource{Identity: "ready"}
	readyEntry.state = resolverLoaded
	close(readyEntry.ready)
	resolver.mu.Unlock()
	require.NoError(t, <-readyError)
	assert.Equal(t, "ready", (<-readyResult).Name)

	sentinel := errors.New("fallback failed")
	failedResolver, err := NewInMemorySourceResolver(context.Background(), nil, &InMemoryResolverConfig{
		Fallback: resolverFunc(func(context.Context, SourceRequest) (*ResolvedSource, error) {
			return nil, sentinel
		}),
	})
	require.NoError(t, err)
	for range 2 {
		_, err = failedResolver.Resolve(context.Background(), SourceRequest{URL: "failed"})
		assert.ErrorIs(t, err, sentinel)
	}

	nilResolver, err := NewInMemorySourceResolver(context.Background(), nil, &InMemoryResolverConfig{
		Fallback: resolverFunc(func(context.Context, SourceRequest) (*ResolvedSource, error) {
			return nil, nil
		}),
	})
	require.NoError(t, err)
	_, err = nilResolver.Resolve(context.Background(), SourceRequest{URL: "nil"})
	assert.ErrorIs(t, err, ErrUnresolvedSourceDesc)

	_, err = resolver.loadedResult("missing", SourceRequest{})
	assert.ErrorIs(t, err, ErrUnresolvedSourceDesc)
	resolver.loads["failed"] = &resolverEntry{state: resolverFailed, err: sentinel}
	_, err = resolver.loadedResult("failed", SourceRequest{})
	assert.ErrorIs(t, err, sentinel)
}

func TestInMemorySourceResolver_RecursiveFallbackCycle(t *testing.T) {
	var resolver *InMemorySourceResolver
	var calls int
	fallback := resolverFunc(func(ctx context.Context, request SourceRequest) (*ResolvedSource, error) {
		calls++
		return resolver.Resolve(ctx, request)
	})
	var err error
	resolver, err = NewInMemorySourceResolver(context.Background(), nil, &InMemoryResolverConfig{Fallback: fallback})
	require.NoError(t, err)
	source, err := resolver.Resolve(context.Background(), SourceRequest{URL: "https://example.com/self"})
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/self", source.Identity)
	assert.Equal(t, 1, calls)
}

func TestInMemorySourceResolver_CrossDocumentFallbackCycle(t *testing.T) {
	var resolver *InMemorySourceResolver
	calls := make(map[string]int)
	fallback := resolverFunc(func(ctx context.Context, request SourceRequest) (*ResolvedSource, error) {
		calls[request.URL]++
		switch request.URL {
		case "https://example.com/a":
			return resolver.Resolve(ctx, SourceRequest{URL: "https://example.com/b"})
		case "https://example.com/b":
			return resolver.Resolve(ctx, SourceRequest{URL: "https://example.com/a"})
		default:
			return nil, errors.New("unexpected source")
		}
	})
	var err error
	resolver, err = NewInMemorySourceResolver(context.Background(), nil, &InMemoryResolverConfig{Fallback: fallback})
	require.NoError(t, err)
	source, err := resolver.Resolve(context.Background(), SourceRequest{URL: "https://example.com/a"})
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/a", source.Identity)
	assert.Equal(t, 1, calls["https://example.com/a"])
	assert.Equal(t, 1, calls["https://example.com/b"])
}

func TestInMemorySourceResolver_DiamondGraphLoadsSharedSourceOnce(t *testing.T) {
	var resolver *InMemorySourceResolver
	calls := make(map[string]int)
	fallback := resolverFunc(func(ctx context.Context, request SourceRequest) (*ResolvedSource, error) {
		calls[request.URL]++
		resolve := func(sourceURL string) error {
			_, resolveErr := resolver.Resolve(ctx, SourceRequest{URL: sourceURL})
			return resolveErr
		}
		switch request.URL {
		case "https://example.com/a":
			require.NoError(t, resolve("https://example.com/b"))
			require.NoError(t, resolve("https://example.com/c"))
		case "https://example.com/b", "https://example.com/c":
			require.NoError(t, resolve("https://example.com/d"))
		}
		return &ResolvedSource{
			URL:      request.URL,
			Identity: request.URL,
			Type:     "openapi",
		}, nil
	})
	var err error
	resolver, err = NewInMemorySourceResolver(
		context.Background(),
		nil,
		&InMemoryResolverConfig{Fallback: fallback},
	)
	require.NoError(t, err)

	source, err := resolver.Resolve(
		context.Background(),
		SourceRequest{URL: "https://example.com/a"},
	)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/a", source.Identity)
	for _, sourceURL := range []string{
		"https://example.com/a",
		"https://example.com/b",
		"https://example.com/c",
		"https://example.com/d",
	} {
		assert.Equal(t, 1, calls[sourceURL], sourceURL)
	}
}

func TestLegacySourceResolver(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	legacy := NewLegacySourceResolver(nil)
	_, err := legacy.Resolve(ctx, SourceRequest{})
	assert.ErrorIs(t, err, context.Canceled)

	legacy = NewLegacySourceResolver(&ResolveConfig{
		HTTPHandler: func(string) ([]byte, error) {
			return []byte("openapi: 3.1.0"), nil
		},
		OpenAPIFactory: func(string, []byte) (*v3high.Document, error) {
			return &v3high.Document{}, nil
		},
	})
	source, err := legacy.Resolve(context.Background(), SourceRequest{
		Name:    "api",
		URL:     "api.yaml",
		Type:    "openapi",
		BaseURI: "https://example.com/specs/",
	})
	require.NoError(t, err)
	assert.Equal(t, "api", source.Name)
	assert.Equal(t, "https://example.com/specs/api.yaml", source.URL)
	assert.Equal(t, source.URL, source.RetrievalURI)
	assert.Equal(t, source.URL, source.Identity)
	assert.NotEmpty(t, source.SourceBytes)

	legacy = NewLegacySourceResolver(&ResolveConfig{
		HTTPHandler: func(string) ([]byte, error) { return nil, errors.New("fetch failed") },
	})
	_, err = legacy.Resolve(context.Background(), SourceRequest{URL: "https://example.com/api.yaml"})
	require.Error(t, err)
}

func BenchmarkInMemorySourceResolverManyDocuments(b *testing.B) {
	const documentCount = 1_000
	documents := make([]CandidateDocument, documentCount)
	for index := range documents {
		sourceURL := fmt.Sprintf("https://example.com/source-%d.yaml", index)
		documents[index] = CandidateDocument{
			ResolvedIdentity: sourceURL,
			RetrievalURI:     sourceURL,
			Adapter:          &testSourceAdapter{sourceType: "asyncapi"},
		}
	}
	provider := &staticCandidateProvider{documents: documents}
	config := &InMemoryResolverConfig{MaxSources: documentCount}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		resolver, err := NewInMemorySourceResolver(context.Background(), provider, config)
		if err != nil {
			b.Fatal(err)
		}
		for index := range documents {
			sourceURL := fmt.Sprintf("https://example.com/source-%d.yaml", index)
			if _, err = resolver.Resolve(
				context.Background(),
				SourceRequest{URL: sourceURL},
			); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func TestResolvedSource_WithRequestNilAndDefaults(t *testing.T) {
	var source *ResolvedSource
	assert.Nil(t, source.withRequest(SourceRequest{}))

	source = &ResolvedSource{Name: "original", Type: "openapi"}
	copy := source.withRequest(SourceRequest{})
	assert.Equal(t, "original", copy.Name)
	assert.Equal(t, "openapi", copy.Type)
	assert.NotSame(t, source, copy)

	copy = source.withRequest(SourceRequest{Name: "requested", Type: "arazzo"})
	assert.Equal(t, "requested", copy.Name)
	assert.Equal(t, "openapi", copy.Type)

	source = &ResolvedSource{}
	copy = source.withRequest(SourceRequest{Type: "AsyncAPI"})
	assert.Equal(t, "asyncapi", copy.Type)
}

func TestResolveSources_PopulatesArazzoIdentityMetadata(t *testing.T) {
	tests := []struct {
		name     string
		document *high.Arazzo
		identity string
	}{
		{
			name: "resolved origin",
			document: high.NewArazzoWithOrigin(&low.Arazzo{}, &high.DocumentOrigin{
				ResolvedIdentity: "https://identity.example/origin.yaml",
			}),
			identity: "https://identity.example/origin.yaml",
		},
		{
			name:     "authored self fallback",
			document: &high.Arazzo{Self: "https://identity.example/self.yaml"},
			identity: "https://identity.example/self.yaml",
		},
		{
			name:     "retrieval fallback",
			document: nil,
			identity: "https://retrieval.example/source.yaml",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document := &high.Arazzo{
				SourceDescriptions: []*high.SourceDescription{{
					Name: "source",
					URL:  "https://retrieval.example/source.yaml",
					Type: "arazzo",
				}},
			}
			resolved, err := ResolveSources(document, &ResolveConfig{
				HTTPHandler: func(string) ([]byte, error) {
					return []byte("arazzo: 1.1.0"), nil
				},
				ArazzoFactory: func(string, []byte) (*high.Arazzo, error) {
					return test.document, nil
				},
			})
			require.NoError(t, err)
			require.Len(t, resolved, 1)
			assert.Equal(t, test.identity, resolved[0].Identity)
			assert.Equal(t, "https://retrieval.example/source.yaml", resolved[0].RetrievalURI)
			assert.Equal(t, []byte("arazzo: 1.1.0"), resolved[0].SourceBytes)
		})
	}
}
