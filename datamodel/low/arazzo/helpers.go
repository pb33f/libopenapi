// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"fmt"
	"hash/maphash"
	"strconv"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"go.yaml.in/yaml/v4"
)

// arazzoBase bundles the common fields found in every Arazzo low-level struct
// so they can be initialized in a single helper call.
type arazzoBase struct {
	KeyNode    **yaml.Node
	RootNode   **yaml.Node
	Reference  **low.Reference
	NodeMap    *low.NodeMap
	Extensions **orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]]
	Index      **index.SpecIndex
	Context    *context.Context
}

// initBuild performs the common preamble shared by every Arazzo low-level Build method.
// It returns the resolved root node (after alias/merge processing) for further extraction.
func initBuild(b *arazzoBase, ctx context.Context, keyNode, root *yaml.Node, idx *index.SpecIndex) *yaml.Node {
	*b.KeyNode = keyNode
	root = utils.NodeAlias(root)
	*b.RootNode = root
	utils.CheckForMergeNodes(root)
	*b.Reference = new(low.Reference)
	b.NodeMap.Nodes = low.ExtractNodes(ctx, root)
	ext := low.ExtractExtensions(root)
	*b.Extensions = ext
	*b.Index = idx
	*b.Context = ctx
	low.ExtractExtensionNodes(ctx, ext, b.NodeMap.Nodes)
	return root
}

// findLabeledNode searches root's Content pairs for a key matching label.
// Returns the key node, value node, and whether the label was found.
func findLabeledNode(label string, root *yaml.Node) (key, value *yaml.Node, found bool) {
	for i := 0; i < len(root.Content); i += 2 {
		if i+1 >= len(root.Content) {
			break
		}
		if root.Content[i].Value == label {
			return root.Content[i], root.Content[i+1], true
		}
	}
	return nil, nil, false
}

// assignNodeReference centralizes the common "if err return; set field" pattern
// used by Build methods when extracting nested NodeReferences.
func assignNodeReference[T any](
	ref low.NodeReference[T],
	err error,
	assign func(low.NodeReference[T]),
) error {
	if err != nil {
		return err
	}
	assign(ref)
	return nil
}

// resolveAliasNode follows an alias chain without allowing malformed or cyclic
// aliases to hang model construction. The caller retains the authored node for
// source metadata and uses the returned node only for shape inspection/building.
func resolveAliasNode(node *yaml.Node) (*yaml.Node, error) {
	if node == nil {
		return nil, nil
	}
	if node.Kind != yaml.AliasNode {
		return node, nil
	}
	seen := make(map[*yaml.Node]struct{})
	current := node
	for current.Kind == yaml.AliasNode {
		if _, exists := seen[current]; exists {
			return nil, fmt.Errorf("cyclic YAML alias at line %d, column %d", node.Line, node.Column)
		}
		seen[current] = struct{}{}
		if current.Alias == nil {
			return nil, fmt.Errorf("empty YAML alias at line %d, column %d", current.Line, current.Column)
		}
		current = current.Alias
	}
	return current, nil
}

// extractScalarString preserves the authored YAML node while reading the scalar
// value from its resolved alias target.
func extractScalarString(label string, root *yaml.Node) (low.NodeReference[string], error) {
	var result low.NodeReference[string]
	key, value, found := findLabeledNode(label, root)
	if !found {
		return result, nil
	}
	result.KeyNode = key
	result.ValueNode = value
	resolved, err := resolveAliasNode(value)
	if err != nil {
		return result, fmt.Errorf("%s: %w", label, err)
	}
	if resolved.Tag == "!!null" {
		return result, nil
	}
	if resolved.Kind != yaml.ScalarNode {
		return result, fmt.Errorf("%s at line %d, column %d must be a scalar",
			label, value.Line, value.Column)
	}
	result.Value = resolved.Value
	return result, nil
}

// extractArray extracts a YAML sequence node into a slice of ValueReferences for the given label.
func extractArray[N any, T interface {
	*N
	Build(context.Context, *yaml.Node, *yaml.Node, *index.SpecIndex) error
}](
	ctx context.Context, label string, root *yaml.Node, idx *index.SpecIndex,
) (low.NodeReference[[]low.ValueReference[T]], error) {
	var result low.NodeReference[[]low.ValueReference[T]]
	key, value, found := findLabeledNode(label, root)
	if !found {
		return result, nil
	}
	result.KeyNode = key
	result.ValueNode = value
	resolvedValue, err := resolveAliasNode(value)
	if err != nil {
		return result, fmt.Errorf("%s: %w", label, err)
	}
	if resolvedValue.Tag == "!!null" {
		return result, nil
	}
	if resolvedValue.Kind != yaml.SequenceNode {
		return result, fmt.Errorf("%s at line %d, column %d must be a sequence of objects",
			label, value.Line, value.Column)
	}
	items := make([]low.ValueReference[T], 0, len(resolvedValue.Content))
	for _, itemNode := range resolvedValue.Content {
		resolvedItem, resolveErr := resolveAliasNode(itemNode)
		if resolveErr != nil {
			return result, fmt.Errorf("%s item: %w", label, resolveErr)
		}
		if resolvedItem.Kind != yaml.MappingNode {
			return result, fmt.Errorf("%s item at line %d, column %d must be a mapping",
				label, itemNode.Line, itemNode.Column)
		}
		obj := T(new(N))
		if err := low.BuildModel(resolvedItem, obj); err != nil {
			return result, err
		}
		if err := obj.Build(ctx, nil, resolvedItem, idx); err != nil {
			return result, err
		}
		items = append(items, low.ValueReference[T]{
			Value:     obj,
			ValueNode: itemNode,
		})
	}
	result.Value = items
	return result, nil
}

// extractObjectMap extracts a YAML mapping node into an ordered map of string keys to built objects.
func extractObjectMap[N any, T interface {
	*N
	Build(context.Context, *yaml.Node, *yaml.Node, *index.SpecIndex) error
}](
	ctx context.Context, label string, root *yaml.Node, idx *index.SpecIndex,
) (low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[T]]], error) {
	var result low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[T]]]
	key, value, found := findLabeledNode(label, root)
	if !found {
		return result, nil
	}
	result.KeyNode = key
	result.ValueNode = value
	resolvedValue, err := resolveAliasNode(value)
	if err != nil {
		return result, fmt.Errorf("%s: %w", label, err)
	}
	if resolvedValue.Tag == "!!null" {
		return result, nil
	}
	if resolvedValue.Kind != yaml.MappingNode {
		return result, fmt.Errorf("%s at line %d, column %d must be a mapping",
			label, value.Line, value.Column)
	}
	m := orderedmap.New[low.KeyReference[string], low.ValueReference[T]]()
	for j := 0; j < len(resolvedValue.Content); j += 2 {
		if j+1 >= len(resolvedValue.Content) {
			break
		}
		mapKey := resolvedValue.Content[j]
		mapVal := resolvedValue.Content[j+1]
		resolvedMapVal, resolveErr := resolveAliasNode(mapVal)
		if resolveErr != nil {
			return result, fmt.Errorf("%s member %q: %w", label, mapKey.Value, resolveErr)
		}
		if resolvedMapVal.Kind != yaml.MappingNode {
			return result, fmt.Errorf("%s member %q at line %d, column %d must be a mapping",
				label, mapKey.Value, mapVal.Line, mapVal.Column)
		}
		obj := T(new(N))
		if err := low.BuildModel(resolvedMapVal, obj); err != nil {
			return result, err
		}
		if err := obj.Build(ctx, mapKey, resolvedMapVal, idx); err != nil {
			return result, err
		}
		m.Set(low.KeyReference[string]{
			Value:   mapKey.Value,
			KeyNode: mapKey,
		}, low.ValueReference[T]{
			Value:     obj,
			ValueNode: mapVal,
		})
	}
	result.Value = m
	return result, nil
}

// extractStringArray extracts a YAML sequence of scalar strings into a NodeReference.
func extractStringArray(label string, root *yaml.Node) (low.NodeReference[[]low.ValueReference[string]], error) {
	var result low.NodeReference[[]low.ValueReference[string]]
	key, value, found := findLabeledNode(label, root)
	if !found {
		return result, nil
	}
	result.KeyNode = key
	result.ValueNode = value
	resolvedValue, err := resolveAliasNode(value)
	if err != nil {
		return result, fmt.Errorf("%s: %w", label, err)
	}
	if resolvedValue.Tag == "!!null" {
		return result, nil
	}
	if resolvedValue.Kind != yaml.SequenceNode {
		return result, fmt.Errorf("%s at line %d, column %d must be a sequence of scalar strings",
			label, value.Line, value.Column)
	}
	items := make([]low.ValueReference[string], 0, len(resolvedValue.Content))
	for _, itemNode := range resolvedValue.Content {
		resolvedItem, resolveErr := resolveAliasNode(itemNode)
		if resolveErr != nil {
			return result, fmt.Errorf("%s item: %w", label, resolveErr)
		}
		if resolvedItem.Kind != yaml.ScalarNode || resolvedItem.Tag == "!!null" {
			return result, fmt.Errorf("%s item at line %d, column %d must be a scalar step identifier",
				label, itemNode.Line, itemNode.Column)
		}
		items = append(items, low.ValueReference[string]{
			Value:     resolvedItem.Value,
			ValueNode: itemNode,
		})
	}
	result.Value = items
	return result, nil
}

// extractRawNode extracts a raw *yaml.Node for a given label without further processing.
func extractRawNode(label string, root *yaml.Node) low.NodeReference[*yaml.Node] {
	var result low.NodeReference[*yaml.Node]
	key, value, found := findLabeledNode(label, root)
	if !found {
		return result
	}
	result.KeyNode = key
	result.ValueNode = value
	result.Value = value
	return result
}

// extractExpressionsMap extracts a YAML mapping node into an ordered map of string keys to string values.
func extractExpressionsMap(label string, root *yaml.Node) low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[string]]] {
	var result low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[string]]]
	key, value, found := findLabeledNode(label, root)
	if !found {
		return result
	}
	result.KeyNode = key
	result.ValueNode = value
	if value.Kind != yaml.MappingNode {
		return result
	}
	m := orderedmap.New[low.KeyReference[string], low.ValueReference[string]]()
	for j := 0; j < len(value.Content); j += 2 {
		if j+1 >= len(value.Content) {
			break
		}
		mapKey := value.Content[j]
		mapVal := value.Content[j+1]
		m.Set(low.KeyReference[string]{
			Value:   mapKey.Value,
			KeyNode: mapKey,
		}, low.ValueReference[string]{
			Value:     mapVal.Value,
			ValueNode: mapVal,
		})
	}
	result.Value = m
	return result
}

// extractOutputValuesMap extracts a map whose values are runtime-expression scalars or Selector Objects.
func extractOutputValuesMap(
	ctx context.Context,
	label string,
	root *yaml.Node,
	idx *index.SpecIndex,
) (low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*OutputValue]]], error) {
	var result low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*OutputValue]]]
	key, value, found := findLabeledNode(label, root)
	if !found {
		return result, nil
	}
	result.KeyNode = key
	result.ValueNode = value
	resolvedValue, err := resolveAliasNode(value)
	if err != nil {
		return result, fmt.Errorf("%s: %w", label, err)
	}
	if resolvedValue.Tag == "!!null" {
		return result, nil
	}
	if resolvedValue.Kind != yaml.MappingNode {
		return result, fmt.Errorf("%s at line %d, column %d must be a mapping",
			label, value.Line, value.Column)
	}
	outputs := orderedmap.New[low.KeyReference[string], low.ValueReference[*OutputValue]]()
	for i := 0; i < len(resolvedValue.Content); i += 2 {
		if i+1 >= len(resolvedValue.Content) {
			break
		}
		outputKey := resolvedValue.Content[i]
		outputNode := resolvedValue.Content[i+1]
		output := new(OutputValue)
		if err := output.Build(ctx, outputKey, outputNode, idx); err != nil {
			return result, err
		}
		outputs.Set(low.KeyReference[string]{
			Value:   outputKey.Value,
			KeyNode: outputKey,
		}, low.ValueReference[*OutputValue]{
			Value:     output,
			ValueNode: outputNode,
		})
	}
	result.Value = outputs
	return result, nil
}

// extractRawNodeMap extracts a YAML mapping node into an ordered map of string keys to raw *yaml.Node values.
func extractRawNodeMap(label string, root *yaml.Node) (low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]]], error) {
	var result low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]]]
	key, value, found := findLabeledNode(label, root)
	if !found {
		return result, nil
	}
	result.KeyNode = key
	result.ValueNode = value
	resolvedValue, err := resolveAliasNode(value)
	if err != nil {
		return result, fmt.Errorf("%s: %w", label, err)
	}
	if resolvedValue.Tag == "!!null" {
		return result, nil
	}
	if resolvedValue.Kind != yaml.MappingNode {
		return result, fmt.Errorf("%s at line %d, column %d must be a mapping",
			label, value.Line, value.Column)
	}
	m := orderedmap.New[low.KeyReference[string], low.ValueReference[*yaml.Node]]()
	for j := 0; j < len(resolvedValue.Content); j += 2 {
		if j+1 >= len(resolvedValue.Content) {
			break
		}
		mapKey := resolvedValue.Content[j]
		mapVal := resolvedValue.Content[j+1]
		m.Set(low.KeyReference[string]{
			Value:   mapKey.Value,
			KeyNode: mapKey,
		}, low.ValueReference[*yaml.Node]{
			Value:     mapVal,
			ValueNode: mapVal,
		})
	}
	result.Value = m
	return result, nil
}

// extractComponentRef extracts a string field from root.Content by label, returning it as a NodeReference.
// Used for the 'reference' field which is renamed to ComponentRef in structs to avoid collision
// with the embedded *low.Reference.
func extractComponentRef(label string, root *yaml.Node) low.NodeReference[string] {
	key, value, found := findLabeledNode(label, root)
	if !found {
		return low.NodeReference[string]{}
	}
	return low.NodeReference[string]{
		Value:     value.Value,
		KeyNode:   key,
		ValueNode: value,
	}
}

// hashYAMLNode writes a yaml.Node tree directly into a maphash.Hash for efficient hashing.
func hashYAMLNode(h *maphash.Hash, node *yaml.Node) {
	hashYAMLNodePath(h, node, make(map[*yaml.Node]struct{}))
}

func hashYAMLNodePath(h *maphash.Hash, node *yaml.Node, active map[*yaml.Node]struct{}) {
	if node == nil {
		return
	}
	if _, exists := active[node]; exists {
		return
	}
	active[node] = struct{}{}
	defer delete(active, node)
	switch node.Kind {
	case yaml.ScalarNode:
		h.WriteString(node.Value)
		h.WriteByte(low.HASH_PIPE)
	case yaml.MappingNode, yaml.SequenceNode:
		// The kind is part of the hash: without it a mapping and a sequence holding the same
		// scalars are indistinguishable, so `{a: b}` and `[a, b]` would compare as equal.
		h.WriteByte(byte(node.Kind))
		h.WriteByte(low.HASH_PIPE)
		for _, child := range node.Content {
			hashYAMLNodePath(h, child, active)
		}
	case yaml.DocumentNode:
		for _, child := range node.Content {
			hashYAMLNodePath(h, child, active)
		}
	case yaml.AliasNode:
		if node.Alias != nil {
			hashYAMLNodePath(h, node.Alias, active)
		}
	}
}

// hashExtensionsInto writes extension hashes directly into the hasher without intermediate allocations.
func hashExtensionsInto(h *maphash.Hash, ext *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]]) {
	if ext == nil {
		return
	}
	for pair := ext.First(); pair != nil; pair = pair.Next() {
		h.WriteString(pair.Key().Value)
		h.WriteByte(low.HASH_PIPE)
		hashYAMLNode(h, pair.Value().Value)
	}
}

// requireNodeKind verifies that a named field, when present, uses one of the expected
// YAML node kinds.
//
// The reflection-driven model builder silently ignores a value whose node kind does not
// match the target Go field, which turns an authoring mistake into quiet data loss: a
// mapping-valued timeout, or a scalar dependsOn, simply disappears from the model. The
// composite Arazzo 1.1 objects (Selector, OutputValue, targetSelectorType) already reject
// an unexpected node kind with a source-positioned error, so this brings the scalar and
// sequence fields into line with them.
//
// An absent field is not an error; requiredness is the validator's concern, not the
// parser's. Likewise a well-formed value that is semantically wrong is left alone.
func requireNodeKind(label, description string, root *yaml.Node, kinds ...yaml.Kind) error {
	_, value, found := findLabeledNode(label, root)
	if !found || value == nil {
		return nil
	}
	resolved, err := resolveAliasNode(value)
	if err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	// An explicit or aliased YAML null is treated as absent rather than as a kind violation.
	if resolved.Tag == "!!null" {
		return nil
	}
	for _, kind := range kinds {
		if resolved.Kind == kind {
			return nil
		}
	}
	return fmt.Errorf("%s at line %d, column %d must be %s",
		label, value.Line, value.Column, description)
}

// requireScalar is the common case: a field that must be a single scalar value.
func requireScalar(label, description string, root *yaml.Node) error {
	return requireNodeKind(label, description, root, yaml.ScalarNode)
}

// requireSequence is the common case: a field that must be a YAML sequence.
func requireSequence(label, description string, root *yaml.Node) error {
	return requireNodeKind(label, description, root, yaml.SequenceNode)
}

// requireInteger verifies that a named field, when present, is an integer the model
// builder will actually accept.
//
// A node-kind check alone is not enough. The builder gates integer fields on the YAML
// tag via utils.IsNodeIntValue, so a quoted scalar such as "5000" is a well-formed
// scalar that parses as an integer yet is still silently dropped. Gating on the same
// predicate the builder uses closes that gap; the subsequent base-10 parse then rejects
// values the builder would coerce to zero, such as 0x1F.
func requireInteger(label, description string, root *yaml.Node) error {
	_, value, found := findLabeledNode(label, root)
	if !found || value == nil {
		return nil
	}
	resolved, err := resolveAliasNode(value)
	if err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	if resolved.Tag == "!!null" {
		return nil
	}
	if !utils.IsNodeIntValue(resolved) {
		return fmt.Errorf("%s at line %d, column %d must be %s, got %q",
			label, value.Line, value.Column, description, resolved.Value)
	}
	if _, err := strconv.ParseInt(resolved.Value, 10, 64); err != nil {
		return fmt.Errorf("%s at line %d, column %d must be %s, got %q",
			label, value.Line, value.Column, description, resolved.Value)
	}
	return nil
}
