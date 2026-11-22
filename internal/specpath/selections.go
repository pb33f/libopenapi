// Copyright 2023-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package specpath

import (
	"slices"
	"strings"

	"github.com/pb33f/go-yaml"
)

// IsOpenAPIDocument reports whether root, a document or its top mapping, is an OpenAPI or Swagger
// description. Paths in such a document place each mapping on their own; in any other document a
// path is only relative to whatever references the document.
func IsOpenAPIDocument(root *yaml.Node) bool {
	if root != nil && root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	if root == nil || root.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		if key := root.Content[i]; key != nil {
			switch key.Value {
			case "openapi", "swagger":
				return true
			}
		}
	}
	return false
}

// Tree records the parent of every node of a document, so the path between two of its nodes is
// found by walking up from the lower one instead of searching down from the upper one.
type Tree struct {
	parents map[*yaml.Node]treeParent
}

// treeParent is the node holding a node, and the mapping key it sits under. A sequence element,
// the content of a document and an alias target sit under no key, as the index walk records them.
type treeParent struct {
	node  *yaml.Node
	key   string
	keyed bool
}

// NewTree records the parents of root and everything below it. A node reachable more than once
// through aliases keeps the first parent found in document order.
func NewTree(root *yaml.Node) *Tree {
	t := &Tree{parents: make(map[*yaml.Node]treeParent)}
	t.link(nil, root, treeParent{})
	return t
}

func (t *Tree) link(parent, node *yaml.Node, at treeParent) {
	if node == nil {
		return
	}
	if _, seen := t.parents[node]; seen {
		return
	}
	at.node = parent
	t.parents[node] = at
	switch node.Kind {
	case yaml.DocumentNode, yaml.SequenceNode:
		for _, child := range node.Content {
			t.link(node, child, treeParent{})
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			if node.Content[i] != nil {
				t.link(node, node.Content[i+1], treeParent{key: node.Content[i].Value, keyed: true})
			}
		}
	case yaml.AliasNode:
		t.link(node, node.Alias, treeParent{})
	}
}

// Parent returns the node holding node, and the mapping key node sits under when it has one.
// parent is nil for the root and for a node outside the tree.
func (t *Tree) Parent(node *yaml.Node) (parent *yaml.Node, key string, keyed bool) {
	at := t.parents[node]
	return at.node, at.key, at.keyed
}

// Path returns the mapping keys from ancestor down to node, and whether ancestor holds node or is
// node itself.
func (t *Tree) Path(ancestor, node *yaml.Node) ([]string, bool) {
	var above []string // keys from node up, nearest first
	for current := node; current != nil; {
		if current == ancestor {
			slices.Reverse(above)
			return above, true
		}
		parent, key, keyed := t.Parent(current)
		if keyed {
			above = append(above, key)
		}
		current = parent
	}
	return nil, false
}

// Reference is a $ref as Selections reads it.
type Reference struct {
	From       string     // the document holding the $ref
	Holder     *yaml.Node // the mapping holding the $ref
	SourcePath []string   // the mapping keys from the root of From down to Holder
	Target     string     // the document the reference selects a node in
	Selected   *yaml.Node // the node it selects
}

// Selection is a node a reference selects, and the path at which its content takes effect.
type Selection struct {
	Node *yaml.Node
	Path []string
}

// Selections returns, for each document, the nodes references select in it, each with the path it
// takes effect at. The source path of a reference held in an OpenAPI description is that path. In
// any other document, marked relative, a source path only starts at the root of that document, so
// a reference held inside a node another reference selects takes effect below the path of that
// selection: a file holding only $ref passes on the context it is referenced from. A reference
// that nothing selected reaches keeps its own source path. roots holds the root node of each
// relative document.
func Selections(refs []Reference, roots map[string]*yaml.Node, relative map[string]bool) map[string][]Selection {
	s := NewSelector()
	for doc := range relative {
		s.AddDocument(doc, roots[doc])
	}
	for _, ref := range refs {
		s.AddReference(ref)
	}
	return s.Selections()
}

// Selector computes Selections as documents and references are found, so the selections made so
// far are not made again when more turn up.
type Selector struct {
	refs     []Reference
	roots    map[string]*yaml.Node // the root of each relative document
	held     map[string][]int      // the references each relative document holds
	trees    map[string]*Tree
	targeted map[string]bool // documents a reference selects a node in
	// What selections from OpenAPI descriptions reach. They only grow as references are added.
	rooted  map[string][]chained
	seen    map[*yaml.Node]map[string]struct{}
	covered []bool
}

// chained is a selection and the references it was reached through, most recent first.
type chained struct {
	selection Selection
	chain     *chain
}

// chain is a list of references. A chain that takes a reference twice has gone round a cycle,
// which only lengthens the path of a selection already made, so it stops there.
type chain struct {
	ref  int
	prev *chain
}

func (c *chain) taken(ref int) bool {
	for ; c != nil; c = c.prev {
		if c.ref == ref {
			return true
		}
	}
	return false
}

// NewSelector returns a Selector with no documents or references.
func NewSelector() *Selector {
	return &Selector{
		roots:    make(map[string]*yaml.Node),
		held:     make(map[string][]int),
		trees:    make(map[string]*Tree),
		targeted: make(map[string]bool),
		rooted:   make(map[string][]chained),
		seen:     make(map[*yaml.Node]map[string]struct{}),
	}
}

// AddDocument marks doc, with its root node, as relative. A document is added before the
// references it holds.
func (s *Selector) AddDocument(doc string, root *yaml.Node) {
	s.roots[doc] = root
}

// AddReference adds a reference and what it selects through the documents marked relative.
func (s *Selector) AddReference(ref Reference) {
	i := len(s.refs)
	s.refs = append(s.refs, ref)
	s.covered = append(s.covered, false)
	s.targeted[ref.Target] = true
	if _, relative := s.roots[ref.From]; !relative {
		s.propagate(s.rootedState(), ref.Target, Selection{Node: ref.Selected, Path: slices.Clip(ref.SourcePath)}, nil)
		return
	}
	s.held[ref.From] = append(s.held[ref.From], i)
	for _, reached := range s.rooted[ref.From] {
		if rel, inside := s.Tree(ref.From).Path(reached.selection.Node, ref.Holder); inside {
			s.covered[i] = true
			if !reached.chain.taken(i) {
				s.propagate(s.rootedState(), ref.Target, s.below(reached.selection, rel, i), &chain{ref: i, prev: reached.chain})
			}
		}
	}
}

// Selections returns the selections in each document, see the function Selections.
func (s *Selector) Selections() map[string][]Selection {
	selections := make(map[string][]Selection, len(s.rooted))
	for doc, reached := range s.rooted {
		for _, r := range reached {
			selections[doc] = append(selections[doc], r.selection)
		}
	}
	// The rest are held in documents, or parts of them, that no selection reaches, and take effect
	// at their own source path as if their document were a root. They leave alone the documents
	// that selections from OpenAPI descriptions reach. Documents nothing references go first, and
	// each fallback passes its path on before the next is taken, so a reference that a fallback
	// reaches takes effect at the path of that fallback instead of its own.
	fallen := &selectionState{
		selections: make(map[string][]chained),
		seen:       make(map[*yaml.Node]map[string]struct{}),
		covered:    slices.Clone(s.covered),
		skip:       s.rooted,
	}
	fallback := func(i int) {
		fallen.covered[i] = true
		ref := s.refs[i]
		s.propagate(fallen, ref.Target, Selection{Node: ref.Selected, Path: slices.Clip(ref.SourcePath)}, &chain{ref: i})
	}
	for _, unrooted := range []bool{true, false} {
		for i, ref := range s.refs {
			if _, relative := s.roots[ref.From]; relative && !fallen.covered[i] && (!unrooted || !s.targeted[ref.From]) {
				fallback(i)
			}
		}
	}
	for doc, reached := range fallen.selections {
		for _, r := range reached {
			selections[doc] = append(selections[doc], r.selection)
		}
	}
	return selections
}

// selectionState is where propagate records selections: the rooted state of the Selector, or a
// fallback state that leaves alone the documents in skip.
type selectionState struct {
	selections map[string][]chained
	seen       map[*yaml.Node]map[string]struct{}
	covered    []bool
	skip       map[string][]chained
}

func (s *Selector) rootedState() *selectionState {
	return &selectionState{selections: s.rooted, seen: s.seen, covered: s.covered}
}

// propagate records a selection in doc and every selection the references held inside it lead to.
func (s *Selector) propagate(state *selectionState, doc string, selection Selection, c *chain) {
	queue := []pending{{doc: doc, reached: chained{selection: selection, chain: c}}}
	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		if !state.add(next.doc, next.reached) {
			continue
		}
		for _, i := range s.held[next.doc] {
			rel, inside := s.Tree(next.doc).Path(next.reached.selection.Node, s.refs[i].Holder)
			if !inside {
				continue
			}
			state.covered[i] = true
			if next.reached.chain.taken(i) {
				continue
			}
			queue = append(queue, pending{
				doc:     s.refs[i].Target,
				reached: chained{selection: s.below(next.reached.selection, rel, i), chain: &chain{ref: i, prev: next.reached.chain}},
			})
		}
	}
}

type pending struct {
	doc     string
	reached chained
}

// below returns the selection reference i makes, held at rel below selection.
func (s *Selector) below(selection Selection, rel []string, i int) Selection {
	path := make([]string, 0, len(selection.Path)+len(rel))
	path = append(path, selection.Path...)
	path = append(path, rel...)
	return Selection{Node: s.refs[i].Selected, Path: path}
}

// add records reached in doc, unless the document is skipped or the selection is already made.
func (state *selectionState) add(doc string, reached chained) bool {
	if _, skip := state.skip[doc]; skip {
		// A fallback guesses at a context the document already has.
		return false
	}
	key := strings.Join(reached.selection.Path, "\x00")
	paths := state.seen[reached.selection.Node]
	if paths == nil {
		paths = make(map[string]struct{})
		state.seen[reached.selection.Node] = paths
	}
	if _, ok := paths[key]; ok {
		return false
	}
	paths[key] = struct{}{}
	state.selections[doc] = append(state.selections[doc], reached)
	return true
}

// Tree returns the parents of the nodes in doc, a relative document.
func (s *Selector) Tree(doc string) *Tree {
	tree := s.trees[doc]
	if tree == nil {
		tree = NewTree(s.roots[doc])
		s.trees[doc] = tree
	}
	return tree
}
