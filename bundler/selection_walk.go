// Copyright 2023-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package bundler

import (
	"strings"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/internal/specpath"
)

// refSelections maps the documents that are not OpenAPI descriptions to the nodes references select
// in them, each with the path it takes effect at. Paths in such a document do not tell whether a
// mapping is a schema, so the context of what a reference selects decides, as in the index, see
// specpath.Selections. Collect them before composing moves or rewrites any node.
type refSelections map[string][]specpath.Selection

func collectRefSelections(rolodex *index.Rolodex) refSelections {
	rootIdx := rolodex.GetRootIndex()
	indexes := rolodex.GetIndexes()
	if rootIdx != nil {
		indexes = append([]*index.SpecIndex{rootIdx}, indexes...)
	}
	var refs []specpath.Reference
	roots := make(map[string]*yaml.Node)
	relative := make(map[string]bool)
	for _, idx := range indexes {
		from := idx.GetSpecAbsolutePath()
		if idx != rootIdx && !specpath.IsOpenAPIDocument(idx.GetRootNode()) {
			relative[from] = true
			roots[from] = idx.GetRootNode()
		}
		mapped := idx.GetMappedReferences()
		for _, ref := range idx.GetRawReferencesSequenced() {
			located := mapped[ref.FullDefinition]
			if located == nil || located.Node == nil {
				continue
			}
			target, _, _ := strings.Cut(ref.FullDefinition, "#")
			refs = append(refs, specpath.Reference{
				From:       from,
				Holder:     ref.Node,
				SourcePath: ref.SourcePath,
				Target:     target,
				Selected:   located.Node,
			})
		}
	}
	selections := make(refSelections)
	for doc, docSelections := range specpath.Selections(refs, roots, relative) {
		if relative[doc] {
			selections[doc] = docSelections
		}
	}
	return selections
}

// selectionWalk is what the walks of one document share. The nodes references select are walked
// first, each under the source path of its reference, then the document from its root. A value one
// selection reads as literal data is walked when another reads it as structure, as in the index,
// and the walk from the root leaves alone what any selection decided.
type selectionWalk struct {
	visited  map[*yaml.Node]struct{}
	literal  map[*yaml.Node]struct{}
	selected bool
}

func newSelectionWalk() *selectionWalk {
	return &selectionWalk{
		visited: make(map[*yaml.Node]struct{}),
		literal: make(map[*yaml.Node]struct{}),
	}
}

// walkSelected calls walk for each selection in the document at specPath, then for its root.
func (w *selectionWalk) walkSelected(selections refSelections, specPath string, root *yaml.Node, walk func(node *yaml.Node, path []string)) {
	w.selected = true
	for _, selection := range selections[specPath] {
		walk(selection.Node, selection.Path)
	}
	w.selected = false
	walk(root, nil)
}

// enter reports whether node is still to be walked, and marks it walked.
func (w *selectionWalk) enter(node *yaml.Node) bool {
	if _, ok := w.visited[node]; ok {
		return false
	}
	if _, ok := w.literal[node]; ok && !w.selected {
		return false
	}
	w.visited[node] = struct{}{}
	return true
}

// isLiteral reports whether value, at path within holder, is literal data, and records what a
// selection decides.
func (w *selectionWalk) isLiteral(path []string, value, holder *yaml.Node) bool {
	if !specpath.IsLiteralValue(path, value, holder) {
		return false
	}
	if w.selected {
		w.literal[value] = struct{}{}
	}
	return true
}
