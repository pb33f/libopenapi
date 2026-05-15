// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io
// SPDX-License-Identifier: MIT

package bundler

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/pb33f/libopenapi/utils"
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

func resolveExtensionRefContent(ctx context.Context, ref *index.Reference, _ *index.Rolodex) *yaml.Node {
	// Use FindComponent which handles all reference types including:
	// - #/components/... refs (local component lookups)
	// - File refs (via lookupRolodex internally)
	// - Both YAML and raw text files
	if ref.Index != nil {
		foundRef := ref.Index.FindComponent(ctx, ref.FullDefinition)
		if foundRef != nil && foundRef.Node != nil {
			// Deep copy to avoid mutating original component
			return deepCopyNode(foundRef.Node)
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

// rewriteExtensionRefsForComposedBundle rebases $ref values found under x-* extension
// keys from their original source file location to the bundled root document.
func rewriteExtensionRefsForComposedBundle(rolodex *index.Rolodex) {
	if rolodex == nil {
		return
	}
	rootIdx := rolodex.GetRootIndex()
	if rootIdx == nil {
		return
	}
	rewriteExtensionRefsForComposedIndex(rootIdx, rootIdx)
	for _, idx := range rolodex.GetIndexes() {
		rewriteExtensionRefsForComposedIndex(idx, rootIdx)
	}
}

func rewriteExtensionRefsForComposedIndex(sourceIdx, rootIdx *index.SpecIndex) {
	if sourceIdx == nil || rootIdx == nil {
		return
	}
	if sourceIdx.GetSpecAbsolutePath() == rootIdx.GetSpecAbsolutePath() {
		return
	}
	walkAndRewriteComposedExtensionRefs(sourceIdx.GetRootNode(), sourceIdx, rootIdx, false)
}

func walkAndRewriteComposedExtensionRefs(node *yaml.Node, sourceIdx, rootIdx *index.SpecIndex, inExtension bool) {
	if node == nil {
		return
	}
	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			walkAndRewriteComposedExtensionRefs(child, sourceIdx, rootIdx, inExtension)
		}
	case yaml.SequenceNode:
		for _, child := range node.Content {
			walkAndRewriteComposedExtensionRefs(child, sourceIdx, rootIdx, inExtension)
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]
			if keyNode == nil {
				continue
			}
			childInExtension := inExtension || strings.HasPrefix(keyNode.Value, "x-")
			if childInExtension && keyNode.Value == "$ref" && valueNode != nil && valueNode.Kind == yaml.ScalarNode {
				valueNode.Value = rebaseExtensionRefForComposed(valueNode.Value, sourceIdx, rootIdx)
				continue
			}
			walkAndRewriteComposedExtensionRefs(valueNode, sourceIdx, rootIdx, childInExtension)
		}
	}
}

func rebaseExtensionRefForComposed(refValue string, sourceIdx, rootIdx *index.SpecIndex) string {
	pathPart, fragment := splitRefPathAndFragment(refValue)
	if pathPart == "" {
		if fragment == "" || sourceIdx == nil || sourceIdx.GetSpecAbsolutePath() == "" || isExternalRefURI(sourceIdx.GetSpecAbsolutePath()) {
			return refValue
		}
		pathPart = sourceIdx.GetSpecAbsolutePath()
	}
	if isExternalRefURI(pathPart) {
		return refValue
	}
	sourceDir := specDir(sourceIdx)
	rootDir := specDir(rootIdx)
	if sourceDir == "" || rootDir == "" {
		return refValue
	}
	targetPath := pathPart
	if !filepath.IsAbs(targetPath) {
		targetPath = utils.CheckPathOverlap(sourceDir, targetPath, string(os.PathSeparator))
	}
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return refValue
	}
	relTarget, err := filepath.Rel(rootDir, absTarget)
	if err != nil {
		return filepath.ToSlash(absTarget) + fragment
	}
	return filepath.ToSlash(relTarget) + fragment
}

func splitRefPathAndFragment(refValue string) (string, string) {
	pathPart, fragment, found := strings.Cut(refValue, "#")
	if found {
		return pathPart, "#" + fragment
	}
	return refValue, ""
}

func isExternalRefURI(refPath string) bool {
	return strings.HasPrefix(refPath, "http://") ||
		strings.HasPrefix(refPath, "https://") ||
		strings.HasPrefix(refPath, "urn:")
}

func specDir(idx *index.SpecIndex) string {
	if idx == nil {
		return ""
	}
	specPath := idx.GetSpecAbsolutePath()
	if specPath == "" || isExternalRefURI(specPath) {
		return ""
	}
	if filepath.Ext(specPath) == "" {
		return specPath
	}
	return filepath.Dir(specPath)
}
