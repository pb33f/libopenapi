// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"hash/maphash"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

// Step represents a low-level Arazzo Step Object.
// https://spec.openapis.org/arazzo/v1.1.0#step-object
type Step struct {
	StepId          low.NodeReference[string]
	Description     low.NodeReference[string]
	OperationId     low.NodeReference[string]
	OperationPath   low.NodeReference[string]
	ChannelPath     low.NodeReference[string]
	Action          low.NodeReference[string]
	WorkflowId      low.NodeReference[string]
	CorrelationId   low.NodeReference[string]
	Timeout         low.NodeReference[int64]
	DependsOn       low.NodeReference[[]low.ValueReference[string]]
	Parameters      low.NodeReference[[]low.ValueReference[*Parameter]]
	RequestBody     low.NodeReference[*RequestBody]
	SuccessCriteria low.NodeReference[[]low.ValueReference[*Criterion]]
	OnSuccess       low.NodeReference[[]low.ValueReference[*SuccessAction]]
	OnFailure       low.NodeReference[[]low.ValueReference[*FailureAction]]
	Outputs         low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*OutputValue]]]
	Extensions      *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]]
	KeyNode         *yaml.Node
	RootNode        *yaml.Node
	index           *index.SpecIndex
	context         context.Context
	*low.Reference
	low.NodeMap
}

var extractStepParameters = extractArray[Parameter]
var extractStepSuccessCriteria = extractArray[Criterion]
var extractStepOnSuccess = extractArray[SuccessAction]

// GetIndex returns the index.SpecIndex instance attached to the Step object.
// For Arazzo low models this is typically nil, because Arazzo parsing does not build a SpecIndex.
// The index parameter is still required to satisfy the shared low.Buildable interface and generic extractors.
func (s *Step) GetIndex() *index.SpecIndex {
	return s.index
}

// GetContext returns the context.Context instance used when building the Step object.
func (s *Step) GetContext() context.Context {
	return s.context
}

// FindExtension returns a ValueReference containing the extension value, if found.
func (s *Step) FindExtension(ext string) *low.ValueReference[*yaml.Node] {
	return low.FindItemInOrderedMap(ext, s.Extensions)
}

// GetRootNode returns the root yaml node of the Step object.
func (s *Step) GetRootNode() *yaml.Node {
	return s.RootNode
}

// GetKeyNode returns the key yaml node of the Step object.
func (s *Step) GetKeyNode() *yaml.Node {
	return s.KeyNode
}

// Build will extract all properties of the Step object.
func (s *Step) Build(ctx context.Context, keyNode, root *yaml.Node, idx *index.SpecIndex) error {
	root = initBuild(&arazzoBase{
		KeyNode:    &s.KeyNode,
		RootNode:   &s.RootNode,
		Reference:  &s.Reference,
		NodeMap:    &s.NodeMap,
		Extensions: &s.Extensions,
		Index:      &s.index,
		Context:    &s.context,
	}, ctx, keyNode, root, idx)

	// Reject a wrong node kind on the Arazzo 1.1 additions rather than silently
	// dropping the value; see requireNodeKind.
	if err := requireScalar(ChannelPathLabel, "a scalar channel reference", root); err != nil {
		return err
	}
	if err := requireScalar(ActionLabel, "a scalar action name", root); err != nil {
		return err
	}
	if err := requireScalar(CorrelationIdLabel, "a scalar string", root); err != nil {
		return err
	}
	if err := requireInteger(TimeoutLabel, "an integer number of milliseconds", root); err != nil {
		return err
	}
	if err := requireSequence(DependsOnLabel, "a sequence of step identifiers", root); err != nil {
		return err
	}

	dependsOn, err := extractStringArray(DependsOnLabel, root)
	if err != nil {
		return err
	}
	s.DependsOn = dependsOn

	params, err := extractStepParameters(ctx, ParametersLabel, root, idx)
	if err != nil {
		return err
	}
	s.Parameters = params

	reqBody, err := low.ExtractObject[*RequestBody](ctx, RequestBodyLabel, root, idx)
	if err != nil {
		return err
	}
	s.RequestBody = reqBody

	criteria, err := extractStepSuccessCriteria(ctx, SuccessCriteriaLabel, root, idx)
	if err != nil {
		return err
	}
	s.SuccessCriteria = criteria

	onSuccess, err := extractStepOnSuccess(ctx, OnSuccessLabel, root, idx)
	if err != nil {
		return err
	}
	s.OnSuccess = onSuccess

	onFailure, err := extractArray[FailureAction](ctx, OnFailureLabel, root, idx)
	if err != nil {
		return err
	}
	s.OnFailure = onFailure

	outputs, err := extractOutputValuesMap(ctx, OutputsLabel, root, idx)
	if err != nil {
		return err
	}
	s.Outputs = outputs

	return nil
}

// GetExtensions returns all Step extensions and satisfies the low.HasExtensions interface.
func (s *Step) GetExtensions() *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]] {
	return s.Extensions
}

// Hash will return a consistent hash of the Step object.
func (s *Step) Hash() uint64 {
	return low.WithHasher(func(h *maphash.Hash) uint64 {
		if !s.StepId.IsEmpty() {
			h.WriteString(s.StepId.Value)
			h.WriteByte(low.HASH_PIPE)
		}
		if !s.Description.IsEmpty() {
			h.WriteString(s.Description.Value)
			h.WriteByte(low.HASH_PIPE)
		}
		if !s.OperationId.IsEmpty() {
			h.WriteString(s.OperationId.Value)
			h.WriteByte(low.HASH_PIPE)
		}
		if !s.OperationPath.IsEmpty() {
			h.WriteString(s.OperationPath.Value)
			h.WriteByte(low.HASH_PIPE)
		}
		if !s.ChannelPath.IsEmpty() {
			h.WriteString(s.ChannelPath.Value)
			h.WriteByte(low.HASH_PIPE)
		}
		if !s.Action.IsEmpty() {
			h.WriteString(s.Action.Value)
			h.WriteByte(low.HASH_PIPE)
		}
		if !s.WorkflowId.IsEmpty() {
			h.WriteString(s.WorkflowId.Value)
			h.WriteByte(low.HASH_PIPE)
		}
		if !s.CorrelationId.IsEmpty() {
			h.WriteString(s.CorrelationId.Value)
			h.WriteByte(low.HASH_PIPE)
		}
		if !s.Timeout.IsEmpty() {
			low.HashInt64(h, s.Timeout.Value)
		}
		if !s.DependsOn.IsEmpty() {
			for _, dependency := range s.DependsOn.Value {
				h.WriteString(dependency.Value)
				h.WriteByte(low.HASH_PIPE)
			}
		}
		if !s.Parameters.IsEmpty() {
			for _, p := range s.Parameters.Value {
				low.HashUint64(h, p.Value.Hash())
			}
		}
		if !s.RequestBody.IsEmpty() {
			low.HashUint64(h, s.RequestBody.Value.Hash())
		}
		if !s.SuccessCriteria.IsEmpty() {
			for _, c := range s.SuccessCriteria.Value {
				low.HashUint64(h, c.Value.Hash())
			}
		}
		if !s.OnSuccess.IsEmpty() {
			for _, a := range s.OnSuccess.Value {
				low.HashUint64(h, a.Value.Hash())
			}
		}
		if !s.OnFailure.IsEmpty() {
			for _, a := range s.OnFailure.Value {
				low.HashUint64(h, a.Value.Hash())
			}
		}
		if !s.Outputs.IsEmpty() {
			for pair := s.Outputs.Value.First(); pair != nil; pair = pair.Next() {
				h.WriteString(pair.Key().Value)
				h.WriteByte(low.HASH_PIPE)
				low.HashUint64(h, pair.Value().Value.Hash())
			}
		}
		hashExtensionsInto(h, s.Extensions)
		return h.Sum64()
	})
}
