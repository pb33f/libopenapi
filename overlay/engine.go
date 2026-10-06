// Copyright 2022-2025 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package overlay

import (
	"fmt"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/jsonpath/pkg/jsonpath"
	"github.com/pb33f/jsonpath/pkg/jsonpath/config"
	highoverlay "github.com/pb33f/libopenapi/datamodel/high/overlay"
	"github.com/pb33f/libopenapi/internal/jsonnode"
	"github.com/pb33f/libopenapi/utils"
)

// Apply applies the given overlay to the target document bytes.
// It returns the modified document bytes and any warnings encountered.
func Apply(targetBytes []byte, overlay *highoverlay.Overlay) (*Result, error) {
	if overlay == nil {
		return nil, ErrInvalidOverlay
	}

	if err := validateOverlay(overlay); err != nil {
		return nil, err
	}

	var rootNode yaml.Node
	if err := jsonnode.Unmarshal(targetBytes, &rootNode); err != nil {
		return nil, err
	}

	// Parent index is built lazily and rebuilt after updates/copies to ensure
	// remove actions can target nodes created by earlier update/copy actions.
	var parentIdx parentIndex
	parentIdxStale := true

	var warnings []*Warning
	for _, original := range overlay.Actions {
		action, err := resolveAction(overlay, original)
		if err != nil {
			return nil, &OverlayError{Action: original, Cause: err}
		}
		if action.Remove && parentIdxStale {
			parentIdx = newParentIndex(&rootNode)
			parentIdxStale = false
		}

		actionWarnings, err := applyAction(&rootNode, action, parentIdx)
		if err != nil {
			return nil, &OverlayError{Action: action, Cause: err}
		}
		warnings = append(warnings, actionWarnings...)

		// Mark parent index as stale after update or copy operations
		// (both can add new nodes that subsequent remove actions may target)
		if action.Update != nil || action.Copy != "" {
			parentIdxStale = true
		}
	}

	// JSON input retains flow style. Render the root as block YAML so the
	// public document parser does not mistake YAML flow syntax for JSON.
	if len(rootNode.Content) > 0 {
		rootNode.Content[0].Style &^= yaml.FlowStyle
	}
	resultBytes, err := yaml.Marshal(&rootNode)
	if err != nil {
		return nil, err
	}

	return &Result{
		Bytes:    resultBytes,
		Warnings: warnings,
	}, nil
}

func applyAction(root *yaml.Node, action *highoverlay.Action, parentIdx parentIndex) ([]*Warning, error) {
	var warnings []*Warning

	path, err := jsonpath.NewPath(action.Target, config.WithPropertyNameExtension(), config.WithLazyContextTracking())
	if err != nil {
		return nil, ErrInvalidJSONPath
	}

	nodes := path.Query(root)

	if len(nodes) == 0 {
		warnings = append(warnings, &Warning{
			Action:  action,
			Target:  action.Target,
			Message: "target matched zero nodes",
		})
		return warnings, nil
	}

	// Removal takes precedence. When copy and update coexist, neither applies.
	if action.Remove {
		applyRemoveAction(parentIdx, nodes)
		return warnings, nil
	}
	if action.Update != nil && action.Copy != "" {
		return warnings, nil
	}
	if action.Copy != "" {
		return applyCopyAction(root, nodes, action.Copy)
	}
	if action.Update != nil && !action.Update.IsZero() {
		if err := validateTargets(nodes); err != nil {
			return nil, err
		}
		if err := applyUpdateAction(nodes, action.Update); err != nil {
			return nil, err
		}
	}

	return warnings, nil
}

func applyCopyAction(root *yaml.Node, targetNodes []*yaml.Node, copyPath string) ([]*Warning, error) {
	var warnings []*Warning

	path, err := jsonpath.NewPath(copyPath, config.WithPropertyNameExtension(), config.WithLazyContextTracking())
	if err != nil {
		return nil, ErrInvalidJSONPath
	}

	sourceNodes := path.Query(root)

	// Single-node constraint per spec: copy source must select exactly one node
	if len(sourceNodes) == 0 {
		return nil, ErrCopySourceNotFound
	}
	if len(sourceNodes) > 1 {
		return nil, ErrCopySourceMultiple
	}

	if err := validateTargets(targetNodes); err != nil {
		return nil, err
	}
	// Freeze the source before any target changes; the source may also be a target.
	sourceNode := utils.CloneYAMLNode(sourceNodes[0])
	if err := applyUpdateAction(targetNodes, sourceNode); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCopyTypeMismatch, err)
	}

	return warnings, nil
}

func applyRemoveAction(idx parentIndex, nodes []*yaml.Node) {
	for _, node := range nodes {
		removeNode(idx, node)
	}
}

func applyUpdateAction(nodes []*yaml.Node, update *yaml.Node) error {
	for _, node := range nodes {
		if node.Kind == yaml.SequenceNode && update.Kind != yaml.SequenceNode {
			node.Content = append(node.Content, utils.CloneYAMLNode(update))
			continue
		}
		if err := mergeNode(node, update); err != nil {
			return err
		}
	}
	return nil
}

type parentIndex map[*yaml.Node]*yaml.Node

func newParentIndex(root *yaml.Node) parentIndex {
	index := parentIndex{}
	index.indexNodeRecursively(root)
	return index
}

func (index parentIndex) indexNodeRecursively(parent *yaml.Node) {
	for _, child := range parent.Content {
		index[child] = parent
		index.indexNodeRecursively(child)
	}
}

func (index parentIndex) getParent(child *yaml.Node) *yaml.Node {
	return index[child]
}

func removeNode(idx parentIndex, node *yaml.Node) {
	parent := idx.getParent(node)
	if parent == nil {
		return
	}

	for i, child := range parent.Content {
		if child == node {
			switch parent.Kind {
			case yaml.MappingNode:
				// JSONPath returns value nodes (odd indices), so remove both key and value
				parent.Content = append(parent.Content[:i-1], parent.Content[i+1:]...)
				return
			case yaml.SequenceNode:
				parent.Content = append(parent.Content[:i], parent.Content[i+1:]...)
				return
			}
		}
	}
}

// mergeNode enforces recursive property compatibility. Array-item appends apply
// only at action targets, not to properties inside an object merge.
func mergeNode(node *yaml.Node, merge *yaml.Node) error {
	if node.Kind != merge.Kind {
		return ErrIncompatibleUpdate
	}
	switch node.Kind {
	case yaml.MappingNode:
		return mergeMappingNode(node, merge)
	case yaml.SequenceNode:
		mergeSequenceNode(node, merge)
	case yaml.ScalarNode:
		// The tag and style must follow the value (for example, string to boolean).
		node.Value, node.Tag, node.Style = merge.Value, merge.Tag, merge.Style
	default:
		return ErrIncompatibleUpdate
	}
	return nil
}

func mergeMappingNode(node *yaml.Node, merge *yaml.Node) error {
	// Small objects and single-property updates are faster without a map.
	// Index larger merges once to avoid quadratic scans across wide objects.
	var properties map[string]*yaml.Node
	const smallObjectProperties = 32
	const smallUpdateProperties = 4
	if (len(node.Content)/2 > smallObjectProperties || len(merge.Content)/2 > smallObjectProperties) && len(merge.Content)/2 > smallUpdateProperties {
		capacity := max(len(node.Content), len(merge.Content)) / 2
		properties = make(map[string]*yaml.Node, capacity)
		for j := 0; j+1 < len(node.Content); j += 2 {
			properties[node.Content[j].Value] = node.Content[j+1]
		}
	}
	for i := 0; i+1 < len(merge.Content); i += 2 {
		key, value := merge.Content[i], merge.Content[i+1]
		var target *yaml.Node
		if properties != nil {
			target = properties[key.Value]
		} else {
			for j := 0; j+1 < len(node.Content); j += 2 {
				if node.Content[j].Value == key.Value {
					target = node.Content[j+1]
					break
				}
			}
		}
		if target != nil {
			if err := mergeNode(target, value); err != nil {
				return fmt.Errorf("property %q: %w", key.Value, err)
			}
		} else {
			cloned := utils.CloneYAMLNode(value)
			node.Content = append(node.Content, utils.CloneYAMLNode(key), cloned)
			if properties != nil {
				properties[key.Value] = cloned
			}
		}
	}
	return nil
}

func mergeSequenceNode(node *yaml.Node, merge *yaml.Node) {
	for _, child := range merge.Content {
		node.Content = append(node.Content, utils.CloneYAMLNode(child))
	}
}
