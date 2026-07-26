// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"sync"

	"github.com/pb33f/libopenapi/datamodel/high"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	low "github.com/pb33f/libopenapi/datamodel/low/arazzo"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

// Arazzo represents a high-level Arazzo document.
// https://spec.openapis.org/arazzo/v1.1.0
type Arazzo struct {
	Arazzo             string                              `json:"arazzo,omitempty" yaml:"arazzo,omitempty"`
	Self               string                              `json:"$self,omitempty" yaml:"$self,omitempty"`
	Info               *Info                               `json:"info,omitempty" yaml:"info,omitempty"`
	SourceDescriptions []*SourceDescription                `json:"sourceDescriptions,omitempty" yaml:"sourceDescriptions,omitempty"`
	Workflows          []*Workflow                         `json:"workflows,omitempty" yaml:"workflows,omitempty"`
	Components         *Components                         `json:"components,omitempty" yaml:"components,omitempty"`
	Extensions         *orderedmap.Map[string, *yaml.Node] `json:"-" yaml:"-"`

	// openAPISourceDocs is attached after construction by source resolution, unlike
	// every other field here, which is populated once while the model is still
	// private to its builder. ResolveSources attaches to the document it was handed,
	// so two goroutines resolving the same document, or one resolving while another
	// validates, both touch this slice concurrently. It is guarded accordingly.
	sourceDocsMu      sync.RWMutex
	openAPISourceDocs []*v3.Document

	origin *DocumentOrigin
	low    *low.Arazzo
}

// NewArazzo creates a new high-level Arazzo instance from a low-level one.
func NewArazzo(a *low.Arazzo) *Arazzo {
	return NewArazzoWithOrigin(a, nil)
}

// NewArazzoWithOrigin creates a high-level Arazzo instance with immutable origin metadata.
func NewArazzoWithOrigin(a *low.Arazzo, origin *DocumentOrigin) *Arazzo {
	h := new(Arazzo)
	h.low = a
	if origin != nil {
		originCopy := *origin
		h.origin = &originCopy
	}
	if !a.Arazzo.IsEmpty() {
		h.Arazzo = a.Arazzo.Value
	}
	if !a.Self.IsEmpty() {
		h.Self = a.Self.Value
	}
	if !a.Info.IsEmpty() {
		h.Info = NewInfo(a.Info.Value)
	}
	if !a.SourceDescriptions.IsEmpty() {
		h.SourceDescriptions = buildSlice(a.SourceDescriptions.Value, NewSourceDescription)
	}
	if !a.Workflows.IsEmpty() {
		h.Workflows = buildSlice(a.Workflows.Value, NewWorkflow)
	}
	if !a.Components.IsEmpty() {
		h.Components = NewComponents(a.Components.Value)
	}
	h.Extensions = high.ExtractExtensions(a.Extensions)
	return h
}

// GoLow returns the low-level Arazzo instance used to create the high-level one.
func (a *Arazzo) GoLow() *low.Arazzo {
	return a.low
}

// GoLowUntyped returns the low-level Arazzo instance with no type.
func (a *Arazzo) GoLowUntyped() any {
	return a.low
}

// AddOpenAPISourceDocument attaches one or more OpenAPI source documents to this Arazzo model.
// Attached documents are runtime metadata and are not rendered or serialized.
// It is safe to call concurrently, and concurrently with GetOpenAPISourceDocuments.
func (a *Arazzo) AddOpenAPISourceDocument(docs ...*v3.Document) {
	if a == nil || len(docs) == 0 {
		return
	}
	a.sourceDocsMu.Lock()
	defer a.sourceDocsMu.Unlock()
	for _, doc := range docs {
		if doc != nil {
			a.openAPISourceDocs = append(a.openAPISourceDocs, doc)
		}
	}
}

// GetOpenAPISourceDocuments returns attached OpenAPI source documents.
// The returned slice is a copy, so callers may retain it while more are attached.
func (a *Arazzo) GetOpenAPISourceDocuments() []*v3.Document {
	if a == nil {
		return nil
	}
	a.sourceDocsMu.RLock()
	defer a.sourceDocsMu.RUnlock()
	if len(a.openAPISourceDocs) == 0 {
		return nil
	}
	docs := make([]*v3.Document, len(a.openAPISourceDocs))
	copy(docs, a.openAPISourceDocs)
	return docs
}

// Render returns a YAML representation of the Arazzo object as a byte slice.
func (a *Arazzo) Render() ([]byte, error) {
	return yaml.Marshal(a)
}

// MarshalYAML creates a ready to render YAML representation of the Arazzo object.
func (a *Arazzo) MarshalYAML() (any, error) {
	m := orderedmap.New[string, any]()
	setKnownField := func(label string) bool {
		switch label {
		case low.ArazzoLabel:
			if a.Arazzo != "" {
				m.Set(label, a.Arazzo)
			}
		case low.SelfLabel:
			if a.Self != "" {
				m.Set(label, a.Self)
			}
		case low.InfoLabel:
			if a.Info != nil {
				m.Set(label, a.Info)
			}
		case low.SourceDescriptionsLabel:
			if len(a.SourceDescriptions) > 0 {
				m.Set(label, a.SourceDescriptions)
			}
		case low.WorkflowsLabel:
			if len(a.Workflows) > 0 {
				m.Set(label, a.Workflows)
			}
		case low.ComponentsLabel:
			if a.Components != nil {
				m.Set(label, a.Components)
			}
		default:
			return false
		}
		return true
	}
	if a.low != nil && a.low.RootNode != nil {
		for i := 0; i+1 < len(a.low.RootNode.Content); i += 2 {
			label := a.low.RootNode.Content[i].Value
			if setKnownField(label) {
				continue
			}
			if a.Extensions != nil {
				if extension, ok := a.Extensions.Get(label); ok {
					m.Set(label, extension)
				}
			}
		}
	}
	for _, label := range []string{
		low.ArazzoLabel,
		low.SelfLabel,
		low.InfoLabel,
		low.SourceDescriptionsLabel,
		low.WorkflowsLabel,
		low.ComponentsLabel,
	} {
		if _, exists := m.Get(label); !exists {
			setKnownField(label)
		}
	}
	marshalExtensions(m, a.Extensions)
	return m, nil
}
