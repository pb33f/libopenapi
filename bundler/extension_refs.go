// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io
// SPDX-License-Identifier: MIT

package bundler

import (
	"context"

	"go.yaml.in/yaml/v4"

	"github.com/pb33f/libopenapi/index"
)

// resolveExtensionRefs resolves $ref pointers within extension fields (x-*).
// Extensions are stored as raw *yaml.Node and don't go through the MarshalYAMLInline()
// chain during rendering, so refs inside extensions need to be resolved separately.
//
// NOTE: This mutates the model's yaml.Node objects in-place.
// This follows the pattern used in compose() for composed bundling.
func resolveExtensionRefs(rolodex *index.Rolodex) {
	if rolodex == nil {
		return
	}

	// Process root index
	resolveExtensionRefsFromIndex(rolodex.GetRootIndex(), rolodex)

	// Process all external indexes
	for _, idx := range rolodex.GetIndexes() {
		resolveExtensionRefsFromIndex(idx, rolodex)
	}
}

func resolveExtensionRefsFromIndex(idx *index.SpecIndex, rolodex *index.Rolodex) {
	if idx == nil {
		return
	}

	extensionRefs := idx.GetExtensionRefsSequenced()
	ctx := context.Background()

	for _, ref := range extensionRefs {
		// Skip invalid refs and circular refs (already detected by indexer)
		if ref.Node == nil || ref.FullDefinition == "" || ref.Circular {
			continue
		}

		// Resolve the reference
		resolvedContent := resolveExtensionRefContent(ctx, ref, rolodex)
		if resolvedContent != nil {
			replaceRefNodeWithContent(ref.Node, resolvedContent)
		}
	}
}

func resolveExtensionRefContent(ctx context.Context, ref *index.Reference, rolodex *index.Rolodex) *yaml.Node {
	// Try FindComponent first (handles #/components/... refs)
	if ref.Index != nil {
		foundRef := ref.Index.FindComponent(ctx, ref.FullDefinition)
		if foundRef != nil && foundRef.Node != nil {
			// Deep copy to avoid mutating original component
			return deepCopyNode(foundRef.Node)
		}
	}

	// Try loading file directly via rolodex
	rFile, err := rolodex.Open(ref.FullDefinition)
	if err != nil || rFile == nil {
		return nil
	}

	// Try as YAML
	node, err := rFile.GetContentAsYAMLNode()
	if err == nil && node != nil {
		return deepCopyNode(node)
	}

	// Fallback: treat as raw text (e.g., shell scripts, markdown)
	content := rFile.GetContent()
	if content != "" {
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: content,
			Style: yaml.LiteralStyle,
		}
	}

	return nil
}

// deepCopyNode creates a deep copy of a yaml.Node tree.
func deepCopyNode(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}

	// unwrap document nodes
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}

	// create copy
	nodeCopy := &yaml.Node{
		Kind:        node.Kind,
		Style:       node.Style,
		Tag:         node.Tag,
		Value:       node.Value,
		Anchor:      node.Anchor,
		Alias:       node.Alias,
		HeadComment: node.HeadComment,
		LineComment: node.LineComment,
		FootComment: node.FootComment,
		Line:        node.Line,
		Column:      node.Column,
	}

	// deep copy children
	if len(node.Content) > 0 {
		nodeCopy.Content = make([]*yaml.Node, len(node.Content))
		for i, child := range node.Content {
			nodeCopy.Content[i] = deepCopyNode(child)
		}
	}

	return nodeCopy
}

func replaceRefNodeWithContent(refNode, content *yaml.Node) {
	if refNode == nil || content == nil {
		return
	}

	// replace refNode in-place with resolved content
	refNode.Kind = content.Kind
	refNode.Style = content.Style
	refNode.Tag = content.Tag
	refNode.Value = content.Value
	refNode.Anchor = content.Anchor
	refNode.Alias = content.Alias
	refNode.Content = content.Content
	refNode.HeadComment = content.HeadComment
	refNode.LineComment = content.LineComment
	refNode.FootComment = content.FootComment
}
