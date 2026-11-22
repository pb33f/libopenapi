// Copyright 2023-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"context"
	"sort"
	"strings"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/internal/specpath"
)

// guessedValue is the value of enum, const, default or a schema examples array in a document that
// is not an OpenAPI description, where no path rule places the mapping holding the keyword at a
// Schema Object. Whether the value is instance data then depends on how other documents reference
// that mapping: a file holding only const is a schema when a schema references it, and a catalog
// with an entry named properties is not one when a reference selects another entry. While the
// rolodex indexes its documents, the walk records such values instead of deciding them, and
// settleGuessedValues extracts the references in those that turn out not to be instance data.
type guessedValue struct {
	path     []string   // mapping keys from the document root to the keyword
	node     *yaml.Node // the keyword value
	holder   *yaml.Node // the mapping holding the keyword, as written
	view     *yaml.Node // holder with merge keys expanded
	literal  bool       // the guess from the keywords of holder
	ctx      context.Context
	level    int
	poly     bool
	polyName string
	settled  bool
}

func (index *SpecIndex) deferGuessedValue(
	node, view *yaml.Node,
	state *extractRefsState,
	literal bool,
	level int,
	poly bool,
	polyName string,
) {
	index.guessedValues = append(index.guessedValues, &guessedValue{
		path:     append([]string(nil), state.seenPath...),
		node:     node,
		holder:   state.source,
		view:     view,
		literal:  literal,
		ctx:      state.ctx,
		level:    level,
		poly:     poly,
		polyName: polyName,
	})
}

// valueContext locates the nodes of one document and the selections references make in it, so
// each guessed value is decided by walking up from its holder instead of searching the document.
type valueContext struct {
	tree      *specpath.Tree
	selected  map[*yaml.Node][]specpath.Selection
	enclosing map[*yaml.Node]struct{} // selected nodes and every node above them
}

func newValueContext(tree *specpath.Tree, selections []specpath.Selection) *valueContext {
	c := &valueContext{
		tree:      tree,
		selected:  make(map[*yaml.Node][]specpath.Selection),
		enclosing: make(map[*yaml.Node]struct{}),
	}
	for _, selection := range selections {
		c.selected[selection.Node] = append(c.selected[selection.Node], selection)
		if specpath.HoldsExamplePayload(selection.Path) {
			// Example payload says nothing of the structure around the node it selects.
			continue
		}
		for node := selection.Node; node != nil; node, _, _ = c.tree.Parent(node) {
			if _, seen := c.enclosing[node]; seen {
				break
			}
			c.enclosing[node] = struct{}{}
		}
	}
	return c
}

// literal reports whether the value is instance data. Each selection that holds the value, or sits
// inside it, decides by the path of its reference; the value is instance data only when all of
// them agree. With no such selection, or no context at all, the guess stands.
func (c *valueContext) literal(v *guessedValue) bool {
	if c == nil {
		return v.literal
	}
	if _, inside := c.enclosing[v.node]; inside {
		// The keyword lies outside a selected node, so it says nothing about it.
		return false
	}
	var above []string // keys from the holder up, nearest first
	covered := false
	for node := v.holder; node != nil; {
		for _, selection := range c.selected[node] {
			path := make([]string, 0, len(selection.Path)+len(above)+1)
			path = append(path, selection.Path...)
			for i := len(above) - 1; i >= 0; i-- {
				path = append(path, above[i])
			}
			path = append(path, v.path[len(v.path)-1])
			if !specpath.IsLiteralValue(path, v.node, v.view) {
				return false
			}
			covered = true
		}
		parent, key, keyed := c.tree.Parent(node)
		if keyed {
			above = append(above, key)
		}
		node = parent
	}
	if !covered {
		return v.literal
	}
	return true
}

// extractGuessedValue extracts and looks up the references in a value that is not instance data.
func (index *SpecIndex) extractGuessedValue(v *guessedValue) {
	polyBefore := make(map[string]struct{}, len(index.polymorphicRefs))
	for key := range index.polymorphicRefs {
		polyBefore[key] = struct{}{}
	}
	found := index.ExtractRefs(v.ctx, v.node, v.view, append([]string(nil), v.path...), v.level, v.poly, v.polyName)

	seen := make(map[string]struct{}, len(found))
	refs := make([]*Reference, 0, len(found))
	for _, ref := range found {
		if _, ok := seen[ref.FullDefinition]; !ok {
			seen[ref.FullDefinition] = struct{}{}
			refs = append(refs, ref)
		}
	}
	var polyKeys []string
	for key := range index.polymorphicRefs {
		if _, ok := polyBefore[key]; !ok {
			polyKeys = append(polyKeys, key)
		}
	}
	sort.Strings(polyKeys)
	poly := make([]*Reference, len(polyKeys))
	for i, key := range polyKeys {
		poly[i] = index.polymorphicRefs[key]
	}
	index.ExtractComponentsFromRefs(v.ctx, refs)
	index.ExtractComponentsFromRefs(v.ctx, poly)
}

// settleGuessedValues decides the values the indexes recorded while the rolodex indexed them, once
// every document is indexed and the references into each are known. Extracting references can load
// further documents and select further nodes, so this repeats until no value changes.
func (r *Rolodex) settleGuessedValues() {
	defer func() {
		r.recordSettledValues()
		r.settlingValues.Store(false)
	}()
	found := newFoundSelections()
	for {
		pending := make(map[string]*SpecIndex)
		for _, idx := range r.GetIndexes() {
			for _, v := range idx.guessedValues {
				if !v.settled {
					pending[idx.specAbsolutePath] = idx
					break
				}
			}
		}
		if len(pending) == 0 {
			return
		}
		found.read(r)
		selections := found.selector.Selections()
		paths := make([]string, 0, len(pending))
		for path := range pending {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		extracted := false
		for _, path := range paths {
			idx := pending[path]
			var values *valueContext
			if docSelections := selections[path]; len(docSelections) > 0 {
				values = newValueContext(found.selector.Tree(path), docSelections)
			}
			var idsBefore map[string]*SchemaIdEntry
			extractedHere := false
			for _, v := range idx.guessedValues {
				if v.settled || values.literal(v) {
					continue
				}
				if !extractedHere {
					idsBefore = idx.GetAllSchemaIds()
					extractedHere = true
				}
				v.settled = true
				idx.extractGuessedValue(v)
			}
			if extractedHere {
				r.registerNewSchemaIds(idx, idsBefore)
				extracted = true
			}
		}
		if !extracted {
			return
		}
	}
}

// recordSettledValues keeps the decision on every guessed value, so a resolver walking into one
// treats it as the index did, whatever path it walked in from. A node a merge key reaches from
// several mappings is data only when it is data under each of them, as the index otherwise holds
// the references in it.
func (r *Rolodex) recordSettledValues() {
	for _, idx := range r.GetIndexes() {
		for _, v := range idx.guessedValues {
			if r.settledValues == nil {
				r.settledValues = make(map[*yaml.Node]bool)
			}
			literal, recorded := r.settledValues[v.node]
			r.settledValues[v.node] = !v.settled && (literal || !recorded)
		}
	}
}

// settledValue reports whether node is a guessed value the rolodex settled, and if so whether it
// turned out to be instance data.
func (r *Rolodex) settledValue(node *yaml.Node) (literal, settled bool) {
	if r == nil || len(r.settledValues) == 0 {
		return false, false
	}
	literal, settled = r.settledValues[node]
	return literal, settled
}

// registerNewSchemaIds adds to the global registry the $id registrations idx gained since before.
// The rolodex aggregated those of idx when it was added, before its guessed values were decided.
func (r *Rolodex) registerNewSchemaIds(idx *SpecIndex, before map[string]*SchemaIdEntry) {
	for key, entry := range idx.GetAllSchemaIds() {
		if _, ok := before[key]; !ok {
			_ = r.RegisterGlobalSchemaId(entry)
		}
	}
}

// foundSelections gives a specpath.Selector the documents of the rolodex and the references in
// them as the settle finds them, so a pass reads only what the passes before it did not, and the
// selections already made are not made again.
type foundSelections struct {
	selector *specpath.Selector
	indexes  map[*SpecIndex]int // how many raw references of each index were read
}

func newFoundSelections() *foundSelections {
	return &foundSelections{
		selector: specpath.NewSelector(),
		indexes:  make(map[*SpecIndex]int),
	}
}

// read adds the documents and references the rolodex gained since the last read. An index looks
// up the references it extracts as it extracts them, so one without a mapping by now failed its
// lookup and selects nothing.
func (f *foundSelections) read(r *Rolodex) {
	indexes := r.GetIndexes()
	if r.rootIndex != nil {
		indexes = append([]*SpecIndex{r.rootIndex}, indexes...)
	}
	for _, idx := range indexes {
		if _, ok := f.indexes[idx]; !ok {
			f.indexes[idx] = 0
			if idx != r.rootIndex && !specpath.IsOpenAPIDocument(idx.root) {
				f.selector.AddDocument(idx.specAbsolutePath, idx.root)
			}
		}
	}
	for _, idx := range indexes {
		raw := idx.GetRawReferencesSequenced()
		mapped := idx.GetMappedReferences()
		for _, ref := range raw[f.indexes[idx]:] {
			located := mapped[ref.FullDefinition]
			if located == nil || located.Node == nil {
				continue
			}
			target, _, _ := strings.Cut(ref.FullDefinition, "#")
			f.selector.AddReference(specpath.Reference{
				From:       idx.specAbsolutePath,
				Holder:     ref.Node,
				SourcePath: ref.SourcePath,
				Target:     target,
				Selected:   located.Node,
			})
		}
		f.indexes[idx] = len(raw)
	}
}
