// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"

	high "github.com/pb33f/libopenapi/datamodel/high/arazzo"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
	"go.yaml.in/yaml/v4"
)

// CandidateDocumentProvider supplies application-owned source documents without I/O.
type CandidateDocumentProvider interface {
	Documents(context.Context) ([]CandidateDocument, error)
}

// SourceDocumentResolver resolves one source request. Implementations must honor cancellation.
type SourceDocumentResolver interface {
	Resolve(context.Context, SourceRequest) (*ResolvedSource, error)
}

// SourceDocumentAdapter identifies a source document without coupling libopenapi to a concrete library.
type SourceDocumentAdapter interface {
	SourceType() string
}

// OpenAPISourceAdapter exposes OpenAPI operation lookups to downstream validators.
type OpenAPISourceAdapter interface {
	SourceDocumentAdapter
	HasOperationID(string) bool
	HasOperationPath(string) bool
}

// AsyncAPISourceAdapter exposes AsyncAPI channel and operation lookups.
type AsyncAPISourceAdapter interface {
	SourceDocumentAdapter
	HasChannel(string) bool
	HasOperation(string) bool
}

// ArazzoSourceAdapter exposes Arazzo workflow lookups.
type ArazzoSourceAdapter interface {
	SourceDocumentAdapter
	HasWorkflow(string) bool
}

// CandidateDocument is a supplied source document and its portable/retrieval identities.
type CandidateDocument struct {
	Type             string
	ResolvedIdentity string
	RetrievalURI     string
	SourceBytes      []byte
	RootNode         *yaml.Node
	OpenAPIDocument  *v3high.Document
	ArazzoDocument   *high.Arazzo
	Adapter          SourceDocumentAdapter
}

// SourceRequest describes one sourceDescription lookup.
type SourceRequest struct {
	Name    string
	URL     string
	Type    string
	BaseURI string
}

// InMemoryResolverConfig configures candidate parsing, limits, and an optional explicit fallback.
type InMemoryResolverConfig struct {
	OpenAPIFactory OpenAPIDocumentFactory
	ArazzoFactory  ArazzoDocumentFactory
	Fallback       SourceDocumentResolver
	MaxSources     int // Maximum candidate documents to accept (default: 50).
}

// ErrDuplicateSourceIdentity indicates ambiguous supplied-document identity.
var ErrDuplicateSourceIdentity = errors.New("duplicate source document identity")

const defaultMaxCandidateSources = 50

type resolverLoadState uint8

const (
	resolverUnseen resolverLoadState = iota
	resolverLoading
	resolverLoaded
	resolverFailed
)

type resolverEntry struct {
	state   resolverLoadState
	source  *ResolvedSource
	err     error
	ready   chan struct{}
	waiters int
}

// InMemorySourceResolver resolves supplied documents before consulting an explicit fallback.
type InMemorySourceResolver struct {
	byIdentity  map[string]*ResolvedSource
	byRetrieval map[string]*ResolvedSource
	fallback    SourceDocumentResolver
	mu          sync.Mutex
	loads       map[string]*resolverEntry
}

// NewInMemorySourceResolver fully parses supplied candidates once and builds immutable lookup maps.
func NewInMemorySourceResolver(
	ctx context.Context,
	provider CandidateDocumentProvider,
	config *InMemoryResolverConfig,
) (*InMemorySourceResolver, error) {
	resolver := &InMemorySourceResolver{
		byIdentity:  make(map[string]*ResolvedSource),
		byRetrieval: make(map[string]*ResolvedSource),
		loads:       make(map[string]*resolverEntry),
	}
	resolverConfig := InMemoryResolverConfig{MaxSources: defaultMaxCandidateSources}
	if config != nil {
		resolverConfig = *config
		resolver.fallback = config.Fallback
	}
	if resolverConfig.MaxSources <= 0 {
		resolverConfig.MaxSources = defaultMaxCandidateSources
	}
	if provider == nil {
		return resolver, nil
	}
	candidates, err := provider.Documents(ctx)
	if err != nil {
		return nil, fmt.Errorf("load candidate documents: %w", err)
	}
	if len(candidates) > resolverConfig.MaxSources {
		return nil, fmt.Errorf(
			"too many candidate source documents: %d (max %d)",
			len(candidates),
			resolverConfig.MaxSources,
		)
	}
	for i := range candidates {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		source, buildErr := buildCandidateSource(candidates[i], &resolverConfig)
		if buildErr != nil {
			return nil, fmt.Errorf("candidate %d: %w", i, buildErr)
		}
		if registerErr := resolver.registerSourceLocked(source); registerErr != nil {
			return nil, registerErr
		}
	}
	return resolver, nil
}

// registerSourceLocked atomically validates and records a source's canonical and
// retrieval identities. The caller must hold r.mu after the resolver is published.
func (r *InMemorySourceResolver) registerSourceLocked(source *ResolvedSource, retrievalAliases ...string) error {
	retrievals := make(map[string]struct{}, len(retrievalAliases)+2)
	addRetrieval := func(key string) {
		if key != "" {
			retrievals[key] = struct{}{}
		}
	}
	for _, key := range retrievalAliases {
		addRetrieval(key)
	}
	addRetrieval(source.RetrievalURI)
	addRetrieval(source.URL)
	check := func(key string) error {
		if existing := r.byIdentity[key]; existing != nil && existing != source {
			return fmt.Errorf("%w %q", ErrDuplicateSourceIdentity, key)
		}
		if existing := r.byRetrieval[key]; existing != nil && existing != source {
			return fmt.Errorf("%w %q", ErrDuplicateSourceIdentity, key)
		}
		return nil
	}
	if source.Identity != "" {
		if err := check(source.Identity); err != nil {
			return err
		}
	}
	for key := range retrievals {
		if err := check(key); err != nil {
			return err
		}
	}
	if source.Identity != "" {
		r.byIdentity[source.Identity] = source
	}
	for key := range retrievals {
		r.byRetrieval[key] = source
	}
	return nil
}

func buildCandidateSource(candidate CandidateDocument, config *InMemoryResolverConfig) (*ResolvedSource, error) {
	sourceBytes := candidate.SourceBytes
	if len(sourceBytes) == 0 && candidate.RootNode != nil {
		rendered, err := yaml.Marshal(candidate.RootNode)
		if err != nil {
			return nil, fmt.Errorf("render candidate root node: %w", err)
		}
		sourceBytes = rendered
	}
	sourceType := strings.ToLower(strings.TrimSpace(candidate.Type))
	if sourceType == "" {
		switch {
		case candidate.OpenAPIDocument != nil:
			sourceType = "openapi"
		case candidate.ArazzoDocument != nil:
			sourceType = "arazzo"
		case candidate.Adapter != nil:
			sourceType = strings.ToLower(candidate.Adapter.SourceType())
		}
	}

	switch sourceType {
	case "openapi":
		if candidate.OpenAPIDocument == nil {
			if config.OpenAPIFactory == nil {
				return nil, fmt.Errorf("no OpenAPIFactory configured")
			}
			document, err := config.OpenAPIFactory(candidate.RetrievalURI, sourceBytes)
			if err != nil {
				return nil, fmt.Errorf("parse OpenAPI candidate: %w", err)
			}
			candidate.OpenAPIDocument = document
		}
	case "arazzo":
		if candidate.ArazzoDocument == nil {
			if config.ArazzoFactory == nil {
				return nil, fmt.Errorf("no ArazzoFactory configured")
			}
			document, err := config.ArazzoFactory(candidate.RetrievalURI, sourceBytes)
			if err != nil {
				return nil, fmt.Errorf("parse Arazzo candidate: %w", err)
			}
			candidate.ArazzoDocument = document
		}
	default:
		if candidate.Adapter == nil {
			return nil, fmt.Errorf("unsupported source type %q requires an adapter", sourceType)
		}
	}

	identity := candidate.ResolvedIdentity
	if identity == "" && candidate.ArazzoDocument != nil {
		if origin := candidate.ArazzoDocument.GetDocumentOrigin(); origin != nil {
			identity = origin.ResolvedIdentity
		}
		if identity == "" {
			identity = candidate.ArazzoDocument.Self
		}
	}
	if identity == "" {
		identity = candidate.RetrievalURI
	}
	return &ResolvedSource{
		URL:             candidate.RetrievalURI,
		RetrievalURI:    candidate.RetrievalURI,
		Identity:        identity,
		Type:            sourceType,
		SourceBytes:     sourceBytes,
		RootNode:        candidate.RootNode,
		OpenAPIDocument: candidate.OpenAPIDocument,
		ArazzoDocument:  candidate.ArazzoDocument,
		Adapter:         candidate.Adapter,
	}, nil
}

type resolverContextKey struct{}

func withResolverLoad(ctx context.Context, key string) context.Context {
	current, _ := ctx.Value(resolverContextKey{}).(map[string]struct{})
	next := make(map[string]struct{}, len(current)+1)
	for item := range current {
		next[item] = struct{}{}
	}
	next[key] = struct{}{}
	return context.WithValue(ctx, resolverContextKey{}, next)
}

func resolverLoadActive(ctx context.Context, key string) bool {
	current, _ := ctx.Value(resolverContextKey{}).(map[string]struct{})
	_, ok := current[key]
	return ok
}

// Resolve returns an identity match first, then a retrieval-location match, then an explicit fallback.
func (r *InMemorySourceResolver) Resolve(ctx context.Context, request SourceRequest) (*ResolvedSource, error) {
	if r == nil {
		return nil, fmt.Errorf("nil source resolver")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	resolvedURI, err := resolveSourceRequestURI(request.URL, request.BaseURI)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	if source := r.byIdentity[resolvedURI]; source != nil {
		r.mu.Unlock()
		return source.withRequest(request), nil
	}
	if source := r.byRetrieval[resolvedURI]; source != nil {
		r.mu.Unlock()
		return source.withRequest(request), nil
	}
	if r.fallback == nil {
		r.mu.Unlock()
		return nil, fmt.Errorf("%w: %s", ErrUnresolvedSourceDesc, resolvedURI)
	}

	if entry := r.loads[resolvedURI]; entry != nil {
		switch entry.state {
		case resolverLoading:
			if resolverLoadActive(ctx, resolvedURI) {
				source := entry.source
				r.mu.Unlock()
				return source.withRequest(request), nil
			}
			entry.waiters++
			ready := entry.ready
			r.mu.Unlock()
			select {
			case <-ready:
				return entry.result(request)
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		case resolverLoaded:
			source := entry.source
			r.mu.Unlock()
			return source.withRequest(request), nil
		case resolverFailed:
			loadErr := entry.err
			if errors.Is(loadErr, context.Canceled) || errors.Is(loadErr, context.DeadlineExceeded) {
				delete(r.loads, resolvedURI)
				break
			}
			r.mu.Unlock()
			return nil, loadErr
		default:
		}
	}
	entry := &resolverEntry{
		state: resolverLoading,
		source: &ResolvedSource{
			URL:          resolvedURI,
			RetrievalURI: resolvedURI,
			Identity:     resolvedURI,
			Type:         strings.ToLower(request.Type),
		},
		ready: make(chan struct{}),
	}
	r.loads[resolvedURI] = entry
	r.mu.Unlock()

	fallbackRequest := request
	fallbackRequest.URL = resolvedURI
	fallbackRequest.BaseURI = ""
	source, loadErr := r.fallback.Resolve(withResolverLoad(ctx, resolvedURI), fallbackRequest)
	if loadErr == nil && source == nil {
		loadErr = fmt.Errorf("%w: fallback returned no source for %s", ErrUnresolvedSourceDesc, resolvedURI)
	}

	r.mu.Lock()
	// A recursive source cycle returns the ancestor's in-flight placeholder to
	// terminate traversal. It is useful to the current load but is not a fully
	// loaded result and must not be promoted into the permanent identity maps.
	cyclePlaceholder := loadErr == nil && (resolverLoadActive(ctx, source.Identity) ||
		resolverLoadActive(ctx, source.RetrievalURI) ||
		resolverLoadActive(ctx, source.URL))
	if loadErr == nil && !cyclePlaceholder {
		loadErr = r.registerSourceLocked(source, resolvedURI)
	}
	if loadErr != nil {
		entry.state = resolverFailed
		entry.err = loadErr
	} else {
		entry.state = resolverLoaded
		entry.source = source
	}
	close(entry.ready)
	r.mu.Unlock()
	if loadErr != nil {
		return nil, loadErr
	}
	return source.withRequest(request), nil
}

func (r *InMemorySourceResolver) loadedResult(key string, request SourceRequest) (*ResolvedSource, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry := r.loads[key]
	if entry == nil {
		return nil, fmt.Errorf("%w: %s", ErrUnresolvedSourceDesc, key)
	}
	return entry.result(request)
}

func (e *resolverEntry) result(request SourceRequest) (*ResolvedSource, error) {
	if e.err != nil {
		return nil, e.err
	}
	return e.source.withRequest(request), nil
}

func resolveSourceRequestURI(reference, base string) (string, error) {
	parsed, err := url.Parse(reference)
	if err != nil {
		return "", fmt.Errorf("invalid source URI %q: %w", reference, err)
	}
	if parsed.IsAbs() || base == "" {
		return parsed.String(), nil
	}
	baseURI, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid source base URI %q: %w", base, err)
	}
	return baseURI.ResolveReference(parsed).String(), nil
}

func (s *ResolvedSource) withRequest(request SourceRequest) *ResolvedSource {
	if s == nil {
		return nil
	}
	copy := *s
	if request.Name != "" {
		copy.Name = request.Name
	}
	if copy.Type == "" && request.Type != "" {
		copy.Type = strings.ToLower(request.Type)
	}
	return &copy
}

// LegacySourceResolver adapts the existing opt-in file/HTTP resolver to SourceDocumentResolver.
type LegacySourceResolver struct {
	config ResolveConfig
}

// NewLegacySourceResolver creates an explicit compatibility adapter around ResolveSources.
func NewLegacySourceResolver(config *ResolveConfig) *LegacySourceResolver {
	resolver := new(LegacySourceResolver)
	if config != nil {
		resolver.config = *config
	}
	return resolver
}

// Resolve delegates one request to the existing resolver without enabling implicit retrieval elsewhere.
func (r *LegacySourceResolver) Resolve(ctx context.Context, request SourceRequest) (*ResolvedSource, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	document := &high.Arazzo{
		SourceDescriptions: []*high.SourceDescription{{
			Name: request.Name,
			URL:  request.URL,
			Type: request.Type,
		}},
	}
	config := r.config
	config.BaseURL = request.BaseURI
	sources, err := ResolveSourcesWithContext(ctx, document, &config)
	if err != nil {
		return nil, err
	}
	return sources[0], nil
}
