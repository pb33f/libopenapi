// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"fmt"
	"net/url"
	"strings"
)

// DocumentOrigin keeps authored identity separate from retrieval and effective base metadata.
// The values are runtime metadata and are never rendered into the document.
type DocumentOrigin struct {
	AuthoredSelf       string
	ResolvedIdentity   string
	RetrievalURI       string
	ApplicationBaseURI string
	EffectiveBaseURI   string
}

// ResolveDocumentOrigin applies Arazzo base-URI precedence without performing retrieval.
func ResolveDocumentOrigin(authoredSelf, retrievalURI, applicationBaseURI string) (*DocumentOrigin, error) {
	origin := &DocumentOrigin{
		AuthoredSelf:       authoredSelf,
		RetrievalURI:       retrievalURI,
		ApplicationBaseURI: applicationBaseURI,
	}
	retrieval, err := parseOptionalURI("retrieval URI", retrievalURI)
	if err != nil {
		return nil, err
	}
	applicationBase, err := parseOptionalURI("application base URI", applicationBaseURI)
	if err != nil {
		return nil, err
	}

	if authoredSelf != "" {
		self, parseErr := url.Parse(authoredSelf)
		// $self is authored document content. Preserve malformed values for the
		// validator instead of turning semantic document errors into build errors.
		if parseErr == nil {
			switch {
			case self.IsAbs():
				origin.ResolvedIdentity = self.String()
				origin.EffectiveBaseURI = self.String()
			case retrieval != nil:
				resolved := retrieval.ResolveReference(self)
				origin.ResolvedIdentity = resolved.String()
				origin.EffectiveBaseURI = resolved.String()
			case applicationBase != nil:
				resolved := applicationBase.ResolveReference(self)
				origin.ResolvedIdentity = resolved.String()
				origin.EffectiveBaseURI = resolved.String()
			default:
				origin.ResolvedIdentity = self.String()
				origin.EffectiveBaseURI = self.String()
			}
			return origin, nil
		}
	}

	switch {
	case retrieval != nil:
		origin.ResolvedIdentity = retrieval.String()
		origin.EffectiveBaseURI = retrieval.String()
	case applicationBase != nil:
		origin.ResolvedIdentity = applicationBase.String()
		origin.EffectiveBaseURI = applicationBase.String()
	}
	return origin, nil
}

func parseOptionalURI(name, value string) (*url.URL, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("invalid %s %q: %w", name, value, err)
	}
	return parsed, nil
}

// GetDocumentOrigin returns a copy of the document's immutable construction metadata.
func (a *Arazzo) GetDocumentOrigin() *DocumentOrigin {
	if a == nil || a.origin == nil {
		return nil
	}
	copy := *a.origin
	return &copy
}
