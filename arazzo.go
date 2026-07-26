// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package libopenapi

import (
	gocontext "context"
	"fmt"
	"log/slog"

	high "github.com/pb33f/libopenapi/datamodel/high/arazzo"
	"github.com/pb33f/libopenapi/datamodel/low"
	lowArazzo "github.com/pb33f/libopenapi/datamodel/low/arazzo"
	"go.yaml.in/yaml/v4"
)

// ArazzoDocumentConfiguration supplies immutable parsing and origin context.
type ArazzoDocumentConfiguration struct {
	Context            gocontext.Context
	RetrievalURI       string
	ApplicationBaseURI string

	// Logger receives debug detail about how the document identity and effective
	// base URI were derived. Origin resolution weighs $self, the retrieval URI and
	// the application base URI against each other, so the outcome is worth tracing
	// when a relative source URL resolves somewhere unexpected. Nil disables logging.
	Logger *slog.Logger
}

// NewArazzoDocument parses raw bytes into a high-level Arazzo document.
func NewArazzoDocument(arazzoBytes []byte) (*high.Arazzo, error) {
	return NewArazzoDocumentWithConfiguration(arazzoBytes, nil)
}

// NewArazzoDocumentWithConfiguration parses an Arazzo document with retrieval and base-URI context.
func NewArazzoDocumentWithConfiguration(
	arazzoBytes []byte,
	configuration *ArazzoDocumentConfiguration,
) (*high.Arazzo, error) {
	config := ArazzoDocumentConfiguration{}
	if configuration != nil {
		config = *configuration
	}
	var rootNode yaml.Node
	if err := yaml.Unmarshal(arazzoBytes, &rootNode); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if rootNode.Kind != yaml.DocumentNode || len(rootNode.Content) == 0 {
		return nil, fmt.Errorf("invalid YAML document structure")
	}

	mappingNode := rootNode.Content[0]
	if mappingNode.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected YAML mapping, got %v", mappingNode.Kind)
	}

	// Build the low-level model
	lowDoc := &lowArazzo.Arazzo{}
	if err := low.BuildModel(mappingNode, lowDoc); err != nil {
		return nil, fmt.Errorf("failed to build low-level model: %w", err)
	}

	ctx := config.Context
	if ctx == nil {
		ctx = gocontext.Background()
	}
	if err := lowDoc.Build(ctx, nil, mappingNode, nil); err != nil {
		return nil, fmt.Errorf("failed to build arazzo document: %w", err)
	}

	origin, err := high.ResolveDocumentOrigin(
		lowDoc.Self.Value,
		config.RetrievalURI,
		config.ApplicationBaseURI,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve arazzo document origin: %w", err)
	}

	// ResolveDocumentOrigin returns a non-nil origin on every non-error path, so only
	// the logger itself needs guarding.
	if config.Logger != nil {
		config.Logger.DebugContext(ctx, "resolved arazzo document origin",
			"authoredSelf", origin.AuthoredSelf,
			"resolvedIdentity", origin.ResolvedIdentity,
			"retrievalURI", origin.RetrievalURI,
			"applicationBaseURI", origin.ApplicationBaseURI,
			"effectiveBaseURI", origin.EffectiveBaseURI)
	}

	// Build the high-level model
	highDoc := high.NewArazzoWithOrigin(lowDoc, origin)
	return highDoc, nil
}
