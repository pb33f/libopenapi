// Copyright 2023-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"context"
	"strings"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/internal/specpath"
	"github.com/pb33f/libopenapi/utils"
)

type extractRefsState struct {
	mergeStart    int
	source        *yaml.Node // the mapping or sequence walked, before merge keys are expanded
	ctx           context.Context
	scope         *SchemaIdScope
	parentBaseURI string
	seenPath      []string
	lastAppended  bool
	level         int
	poly          bool
	polyName      string
	prev          string
}

func (index *SpecIndex) initializeExtractRefsState(
	ctx context.Context,
	nodeID string,
	seenPath []string,
	level int,
	poly bool,
	pName string,
) extractRefsState {
	scope := GetSchemaIdScope(ctx)
	if scope == nil {
		scope = NewSchemaIdScope(index.specAbsolutePath)
		ctx = WithSchemaIdScope(ctx, scope)
	}

	parentBaseURI := scope.BaseUri
	if nodeID != "" && !specpath.UnderOpenAPIExamplePath(seenPath) {
		resolvedNodeID, _ := ResolveSchemaId(nodeID, parentBaseURI)
		if resolvedNodeID == "" {
			resolvedNodeID = nodeID
		}
		scope = scope.Copy()
		scope.PushId(resolvedNodeID)
		ctx = WithSchemaIdScope(ctx, scope)
	}

	return extractRefsState{
		ctx:           ctx,
		scope:         scope,
		parentBaseURI: parentBaseURI,
		seenPath:      seenPath,
		level:         level,
		poly:          poly,
		polyName:      pName,
	}
}

func (index *SpecIndex) walkExtractRefs(node, parent *yaml.Node, state *extractRefsState, walk *extractRefsTraversal) []*Reference {
	if node == nil || len(node.Content) == 0 {
		return nil
	}

	var found []*Reference
	for i, n := range node.Content {
		// In YAML mapping nodes, Content alternates key-value: even indices (0, 2, 4...)
		// are keys, odd indices (1, 3, 5...) are values.
		if i%2 == 0 {
			if stop := index.handleExtractRefsKey(node, parent, state, i, &found); stop {
				continue
			}
		}

		if utils.IsNodeMap(n) || utils.IsNodeArray(n) {
			inherited := i >= state.mergeStart
			if inherited {
				walk.aliasDepth++
			}
			found = append(found, index.walkChildExtractRefs(n, node, state, walk)...)
			if inherited {
				walk.aliasDepth--
			}
			if walk.stopped {
				break
			}
		}

		index.unwindExtractRefsPath(node, state, i)
	}

	return found
}

func (index *SpecIndex) walkChildExtractRefs(node, parent *yaml.Node, state *extractRefsState, walk *extractRefsTraversal) []*Reference {
	literal, guessed := specpath.ClassifyValue(state.seenPath, node, parent)
	if guessed && index.deferGuessedValues {
		level, poly, polyName := index.childExtractRefsState(state)
		index.deferGuessedValue(node, parent, state, literal, level, poly, polyName)
		if !literal {
			// Leave the state for the siblings as walking the value would.
			state.level, state.poly, state.polyName = level, poly, polyName
		}
		return nil
	}
	if literal {
		return nil
	}
	state.level, state.poly, state.polyName = index.childExtractRefsState(state)
	return index.extractRefs(state.ctx, node, parent, state.seenPath, state.level, state.poly, state.polyName, walk)
}

func (index *SpecIndex) childExtractRefsState(state *extractRefsState) (int, bool, string) {
	level, poly, polyName := state.level+1, state.poly, state.polyName
	if isPoly, _ := index.checkPolymorphicNode(state.prev); isPoly {
		poly = true
		if state.prev != "" {
			polyName = state.prev
		}
	}
	return level, poly, polyName
}

func (index *SpecIndex) handleExtractRefsKey(
	node, parent *yaml.Node,
	state *extractRefsState,
	keyIndex int,
	found *[]*Reference,
) bool {
	keyNode := node.Content[keyIndex]
	state.lastAppended = false
	if keyNode == nil {
		return false
	}

	if isSchemaContainingNode(keyNode.Value) && !utils.IsNodeArray(node) && keyIndex+1 < len(node.Content) {
		index.collectInlineSchemaDefinition(parent, node, state.seenPath, keyIndex)
	}

	if isMapOfSchemaContainingNode(keyNode.Value) && !utils.IsNodeArray(node) && keyIndex+1 < len(node.Content) {
		if shouldSkipMapSchemaCollection(state.seenPath) {
			// Keep the segment: the keys below it are names, not keywords.
			state.seenPath = append(state.seenPath, keyNode.Value)
			state.lastAppended = true
			state.prev = keyNode.Value
			return true
		}
		index.collectMapSchemaDefinitions(parent, node, state.seenPath, keyIndex)
	}

	if isArrayOfSchemaContainingNode(keyNode.Value) && !utils.IsNodeArray(node) && keyIndex+1 < len(node.Content) {
		index.collectArraySchemaDefinitions(parent, node, state.seenPath, keyIndex)
	}

	if keyNode.Value == "$ref" {
		if ref := index.extractReferenceAt(node, parent, keyIndex, state.seenPath, state.scope, state.poly, state.polyName); ref != nil {
			*found = append(*found, ref)
		}
	}

	if keyNode.Value == "$id" {
		index.registerSchemaIDAt(node, keyIndex, state.seenPath, state.parentBaseURI)
	}

	if keyNode.Value != "$ref" && keyNode.Value != "$id" && keyNode.Value != "" {
		action := index.extractNodeMetadata(node, parent, state.seenPath, keyIndex)
		state.lastAppended = action.appendSegment
		if action.appendSegment {
			state.seenPath = append(state.seenPath, strings.ReplaceAll(keyNode.Value, "/", "~1"))
			state.prev = keyNode.Value
		}
		return action.stop
	}

	return false
}

// shouldSkipMapSchemaCollection reports whether seenPath is inside an extension or an example,
// where maps of schemas are not collected. A property named x-... is a schema, not an extension.
func shouldSkipMapSchemaCollection(seenPath []string) bool {
	for i := range seenPath {
		if specpath.IsOpenAPIExtensionSegment(seenPath, i) {
			return true
		}
	}
	return specpath.UnderOpenAPIExamplePath(seenPath)
}

func (index *SpecIndex) unwindExtractRefsPath(node *yaml.Node, state *extractRefsState, currentIndex int) {
	if currentIndex >= len(node.Content)-1 {
		return
	}
	next := node.Content[currentIndex+1]
	if currentIndex%2 != 0 && state.lastAppended &&
		next != nil && !utils.IsNodeArray(next) && !utils.IsNodeMap(next) && len(state.seenPath) > 0 {
		state.seenPath = state.seenPath[:len(state.seenPath)-1]
		state.lastAppended = false
	}
}
