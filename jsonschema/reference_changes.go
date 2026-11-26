// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package jsonschema

import (
	"net/url"
	"strings"

	"github.com/pb33f/go-yaml"
	wc "github.com/pb33f/libopenapi/what-changed/model"
)

// A target can be evaluated both normally and under negation. Keep both
// possibilities when independent entry points or a reference cycle reach it.
type evaluationContext uint8

const (
	positiveContext evaluationContext = 1 << iota
	negativeContext
	mixedContext = positiveContext | negativeContext
)

func (ctx evaluationContext) child(parent *yaml.Node, keyword string) evaluationContext {
	if ctx == 0 || keyword == "contentSchema" {
		return 0
	}
	switch keyword {
	case "$defs":
		return positiveContext
	case "not":
		return (ctx&positiveContext)<<1 | (ctx&negativeContext)>>1
	case "oneOf", "if":
		return mixedContext
	case "contains":
		if member(parent, "maxContains") != nil {
			return mixedContext
		}
	}
	return ctx
}

func combinedPolarity(polarity int, usage evaluationContext) int {
	if polarity > 0 && usage&negativeContext != 0 || polarity < 0 && usage&positiveContext != 0 {
		return 0
	}
	return polarity
}

// Trace evaluation contexts, never schema comparisons, through reference
// edges. Each schema enters the work list at most once for each polarity.
func (d *Document) referenceUsage() map[*yaml.Node]evaluationContext {
	if !d.hasAdverseContext || len(d.References) == 0 {
		return nil
	}
	uses := make(map[*yaml.Node]evaluationContext)
	outgoing := make(map[*yaml.Node][]*yaml.Node, len(d.References))
	type work struct {
		node    *yaml.Node
		context evaluationContext
	}
	var pending []work
	visit := func(node *yaml.Node, ctx evaluationContext) {
		delta := ctx &^ uses[node]
		if delta != 0 {
			uses[node] |= delta
			pending = append(pending, work{node, delta})
		}
	}
	for _, ref := range d.References {
		outgoing[ref.From] = append(outgoing[ref.From], ref.To)
		visit(ref.To, ref.context)
	}
	for len(pending) > 0 {
		step := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		for _, target := range outgoing[step.node] {
			visit(target, step.context)
		}
		_ = children(step.node, func(keyword, _ string, child *yaml.Node) error {
			// Declarations and content annotations do not evaluate their children here.
			if keyword != "$defs" && keyword != "contentSchema" {
				visit(child, step.context.child(step.node, keyword))
			}
			return nil
		})
	}
	return uses
}

type schemaOccurrence struct {
	owner   *Resource
	pointer string
}

// Use the comparator's own operand pairing for every reference below a
// reordered allOf, anyOf, or oneOf branch. An empty destination marks a removal.
func (c *comparison) recordOperandMove(a, b *yaml.Node) {
	if len(c.left.References) == 0 || len(c.right.References) == 0 {
		return
	}
	old := c.left.schemaPointers[a]
	if old == "" {
		return // A named resource already has its own stable namespace.
	}
	owner := c.left.schemas[a]
	newPath := ""
	if b != nil {
		if c.pairs[owner] != c.right.schemas[b] {
			return
		}
		newPath = c.right.schemaPointers[b]
		if old == newPath {
			return
		}
	}
	if c.operandMoves == nil {
		c.operandMoves = make(map[schemaOccurrence]string)
	}
	c.operandMoves[schemaOccurrence{owner, old}] = newPath
}

func (c *comparison) matchedPointer(owner *Resource, pointer string) (string, bool) {
	if c.operandMoves != nil {
		for prefix := pointer; prefix != ""; {
			if to, ok := c.operandMoves[schemaOccurrence{owner, prefix}]; ok {
				if to == "" {
					return "", false
				}
				return to + pointer[len(prefix):], true
			}
			prefix = prefix[:strings.LastIndexByte(prefix, '/')]
		}
	}
	return pointer, true
}

func (c *comparison) compareReferences() {
	if len(c.left.References) == 0 || len(c.right.References) == 0 {
		return
	}
	type site struct {
		owner            *Resource
		pointer, keyword string
	}
	current := make(map[site]*Reference, len(c.right.References))
	for i := range c.right.References {
		ref := &c.right.References[i]
		current[site{c.right.schemas[ref.From], c.right.schemaPointers[ref.From], ref.Keyword}] = ref
	}
	var byFile map[site]*Reference
	for _, before := range c.left.References {
		owner := c.left.schemas[before.From]
		pointer, paired := c.matchedPointer(owner, c.left.schemaPointers[before.From])
		if !paired {
			continue
		}
		key := site{c.pairs[owner], pointer, before.Keyword}
		after := current[key]
		if after == nil && (len(c.pairs) != len(c.left.Resources) || len(c.pairs) != len(c.right.Resources)) {
			// Adding or dropping an embedded $id can change the base at an
			// otherwise unchanged reference site.
			if byFile == nil {
				byFile = make(map[site]*Reference, len(c.right.References))
				for i := range c.right.References {
					ref := &c.right.References[i]
					r := c.right.schemas[ref.From]
					byFile[site{c.right.schemas[r.Source.Node], r.Pointer + c.right.schemaPointers[ref.From], ref.Keyword}] = ref
				}
			}
			file := c.pairs[c.left.schemas[owner.Source.Node]]
			candidate := byFile[site{file, owner.Pointer + key.pointer, key.keyword}]
			if candidate != nil && c.sameTarget(before.From, candidate.From) {
				after = candidate
				key.owner, key.pointer = c.right.schemas[after.From], c.right.schemaPointers[after.From]
			}
		}
		// The regular keyword comparison reports edits to the authored text.
		if after == nil || before.Value != after.Value || c.sameTarget(before.To, after.To) {
			continue
		}
		a, b := member(before.From, before.Keyword), member(after.From, after.Keyword)
		wc.CreateChange(&c.changes, wc.Modified, before.Keyword, a, b, true, nil, nil)
		change := c.changes[len(c.changes)-1]
		change.Reference = before.Value
		change.Original = c.left.targetAddress(before.To)
		change.New = c.right.targetAddress(after.To)
		evaluating := before.context | after.context | c.originalUsage[before.From] | c.modifiedUsage[after.From]
		c.finish(change, owner, key.owner, PropertyPath(schemaPointerPath(key.pointer), key.keyword), key.keyword, 1, evaluating != 0)
	}
}

func (c *comparison) sameTarget(a, b *yaml.Node) bool {
	old, current := c.left.schemas[a], c.right.schemas[b]
	oldPointer, newPointer := c.left.schemaPointers[a], c.right.schemaPointers[b]
	if c.pairs[old] == current {
		return oldPointer == newPointer
	}
	// An inline target can acquire or lose its own $id at the same location.
	// A matched resource that moved to this slot still has a different identity.
	if c.pairs[old] != nil && c.matched[current] {
		return false
	}
	return c.pairs[c.left.schemas[old.Source.Node]] == c.right.schemas[current.Source.Node] &&
		old.Pointer+oldPointer == current.Pointer+newPointer
}

func (d *Document) targetAddress(node *yaml.Node) string {
	id := d.schemas[node].ID
	if pointer := d.schemaPointers[node]; pointer != "" {
		return id + "#" + (&url.URL{Fragment: pointer}).EscapedFragment()
	}
	return id
}

// Convert a schema-bearing pointer to a JSONPath. A property named "0" is
// an object key, while a prefixItems slot is an array index.
func schemaPointerPath(pointer string) string {
	path := "$"
	if pointer == "" {
		return path
	}
	var container int
	for part := range strings.SplitSeq(strings.TrimPrefix(pointer, "/"), "/") {
		key := strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
		switch container {
		case schemaList:
			path += "[" + key + "]"
			container = 0
		case schemaMap:
			path = PropertyPath(path, key)
			container = 0
		default:
			path = PropertyPath(path, key)
			container = childKind(key)
		}
	}
	return path
}
