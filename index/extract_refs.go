// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"context"
	"fmt"
	"strings"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/go-yaml/plugin/loaderlimits"
	"github.com/pb33f/libopenapi/utils"
)

func isSchemaContainingNode(v string) bool {
	switch v {
	case "schema", "items", "additionalProperties", "contains", "not",
		"unevaluatedItems", "unevaluatedProperties":
		return true
	}
	return false
}

func isMapOfSchemaContainingNode(v string) bool {
	switch v {
	case "properties", "patternProperties", "$defs":
		return true
	}
	return false
}

func isArrayOfSchemaContainingNode(v string) bool {
	switch v {
	case "allOf", "anyOf", "oneOf", "prefixItems":
		return true
	}
	return false
}

// Limit plugins are immutable after construction and shared by extraction walks.
var defaultExtractRefsLimits loaderlimits.Plugin
var unlimitedExtractRefsLimits = loaderlimits.New(loaderlimits.AliasNone())

// ExtractRefs will return a deduplicated slice of references for every unique ref found in the document.
// The total number of refs, will generally be much higher, you can extract those from GetRawReferenceCount()
// Excessive YAML alias expansion stops extraction and is reported by GetReferenceIndexErrors.
func (index *SpecIndex) ExtractRefs(ctx context.Context, node, parent *yaml.Node, seenPath []string, level int, poly bool, pName string) []*Reference {
	walk := &extractRefsTraversal{}
	if index.config != nil && index.config.AllowExcessiveAliases {
		walk.limits = unlimitedExtractRefsLimits
	}
	return index.extractRefs(ctx, node, parent, seenPath, level, poly, pName, walk)
}

type extractRefsTraversal struct {
	limits     *loaderlimits.Plugin
	ancestry   map[*yaml.Node]struct{}
	nodes      int
	aliasNodes int
	aliasDepth int
	stopped    bool
}

func (walk *extractRefsTraversal) visit() error {
	walk.nodes++
	if walk.aliasDepth == 0 {
		return nil
	}
	walk.aliasNodes++
	limits := walk.limits
	if limits == nil {
		limits = &defaultExtractRefsLimits
	}
	return limits.CheckAlias(walk.aliasNodes, walk.nodes)
}

// Ancestry tracks canonical source nodes, not the shallow views created during
// merge expansion. A shared anchor is visited again at each distinct path.
func (index *SpecIndex) extractRefs(ctx context.Context, node, parent *yaml.Node, seenPath []string, level int, poly bool, pName string, walk *extractRefsTraversal) []*Reference {
	if node == nil || walk.stopped {
		return nil
	}
	err := walk.visit()
	aliased := node.Kind == yaml.AliasNode && err == nil
	if aliased {
		walk.aliasDepth++
		node = utils.NodeAlias(node)
		err = walk.visit()
	}
	if err != nil {
		walk.stopped = true
		// A stopped traversal cannot provide a complete model.
		index.refErrors = append(index.refErrors, &ResolvingError{
			ErrorRef: fmt.Errorf("%w; set AllowExcessiveAliases for trusted documents", err),
			Node:     node, Path: "$." + strings.Join(seenPath, "."),
		})
		if aliased {
			walk.aliasDepth--
		}
		return nil
	}
	if node == nil {
		if aliased {
			walk.aliasDepth--
		}
		return nil
	}
	view, nodeID := resolveExtractRefsNode(node)
	tracked := aliased || view != node
	if tracked {
		if _, active := walk.ancestry[node]; active {
			if aliased {
				walk.aliasDepth--
			}
			return nil
		}
		if walk.ancestry == nil {
			walk.ancestry = make(map[*yaml.Node]struct{})
		}
		walk.ancestry[node] = struct{}{}
	}
	state := index.initializeExtractRefsState(ctx, nodeID, seenPath, level, poly, pName)
	state.mergeStart = len(view.Content)
	if view != node {
		// Merge views retain explicit pairs first, then append inherited pairs.
		// Only inherited descendants consume the alias expansion budget.
		state.mergeStart = 0
		for i := 0; i < len(node.Content); i += 2 {
			if key := node.Content[i]; key != nil && key.Tag != "!!merge" {
				state.mergeStart += 2
			}
		}
	}
	found := index.walkExtractRefs(view, parent, &state, walk)
	if tracked {
		delete(walk.ancestry, node)
	}
	if aliased {
		walk.aliasDepth--
	}
	index.refCount = len(index.allRefs)
	return found
}

// resolveExtractRefsNode finds merge keys and the schema ID in one scan. Ordinary
// mappings retain their identity; merged mappings use the existing read-only view.
func resolveExtractRefsNode(node *yaml.Node) (*yaml.Node, string) {
	if node.Kind != yaml.MappingNode && node.Tag != "!!map" {
		return node, ""
	}
	var idNode *yaml.Node
	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i]
		if key == nil {
			continue
		}
		if key.Tag == "!!merge" {
			view := utils.ResolveMergeNodes(node)
			return view, FindSchemaIdInNode(view)
		}
		if idNode == nil && node.Kind == yaml.MappingNode && key.Value == "$id" && i+1 < len(node.Content) && utils.IsNodeStringValue(node.Content[i+1]) {
			idNode = node.Content[i+1]
		}
	}
	if idNode != nil {
		return node, idNode.Value
	}
	return node, ""
}
