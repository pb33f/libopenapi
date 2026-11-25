// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package jsonschema

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"slices"
	"strconv"
	"strings"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/utils"
	wc "github.com/pb33f/libopenapi/what-changed/model"
)

// Compare uses the schema comparator on authored constraints. Each referenced
// resource is compared once, independent of the number of references to it.
func Compare(left, right *Document, rules *wc.BreakingRulesConfig) (*wc.SchemaChanges, error) {
	if left == nil || right == nil {
		return nil, fmt.Errorf("both JSON Schema inputs are required")
	}
	if (left.Root == nil) != (right.Root == nil) {
		return nil, fmt.Errorf("a schema collection requires a collection on both sides")
	}
	c := comparison{left: left, right: right, rules: rules}
	if err := c.resources(); err != nil {
		return nil, err
	}
	if len(c.changes) == 0 {
		return nil, nil
	}
	slices.SortStableFunc(c.changes, func(a, b *wc.Change) int {
		return cmp.Or(strings.Compare(a.SchemaResource, b.SchemaResource), strings.Compare(a.Path, b.Path), strings.Compare(a.Property, b.Property), cmp.Compare(a.ChangeType, b.ChangeType), strings.Compare(a.Original, b.Original), strings.Compare(a.New, b.New))
	})
	return &wc.SchemaChanges{PropertyChanges: wc.NewPropertyChanges(c.changes)}, nil
}

type comparison struct {
	annotationOnly               bool
	originalUsage, modifiedUsage map[*yaml.Node]evaluationContext
	appliedAdditions             map[*Resource]bool
	inline                       map[*Resource]bool
	pairs                        map[*Resource]*Resource
	matched                      map[*Resource]bool
	left, right                  *Document
	rules                        *wc.BreakingRulesConfig
	changes                      []*wc.Change
}

// MatchResources pairs the original and modified resources by resolved ID, then
// by source path and embedded pointer. A comparison of two single schemas always
// pairs their roots. Unmatched resources have no entry in the returned map.
func MatchResources(left, right *Document) map[*Resource]*Resource {
	pairs, _ := matchResources(left, right)
	return pairs
}

func matchResources(left, right *Document) (map[*Resource]*Resource, map[*Resource]bool) {
	if left == nil || right == nil {
		return nil, nil
	}
	byID := make(map[string]*Resource, len(right.Resources))
	type origin struct{ file, pointer string }
	byOrigin := make(map[origin]*Resource, len(right.Resources))
	used := make(map[*Resource]bool)
	pairs := make(map[*Resource]*Resource)
	for _, r := range right.Resources {
		byID[r.ID] = r
		byOrigin[origin{r.Source.relative, r.Pointer}] = r
	}
	if left.Root != nil && right.Root != nil {
		pairs[left.Root] = right.Root
		used[right.Root] = true
	}
	for _, l := range left.Resources {
		if pairs[l] == nil {
			if r := byID[l.ID]; r != nil && !used[r] {
				pairs[l] = r
				used[r] = true
			}
		}
	}
	for _, l := range left.Resources {
		r := pairs[l]
		if r == nil {
			file := l.Source.relative
			if left.Root != nil && right.Root != nil && l.Source == left.Root.Source {
				file = right.Root.Source.relative
			}
			r = byOrigin[origin{file, l.Pointer}]
			if used[r] {
				r = nil
			}
			if r != nil {
				used[r] = true
				pairs[l] = r
			}
		}
	}
	return pairs, used
}

func (c *comparison) resources() error {
	pairs, used := matchResources(c.left, c.right)
	c.pairs, c.matched = pairs, used
	c.compareReferences()
	for _, l := range c.left.Resources {
		if c.inline[l] {
			continue
		}
		if err := c.compareResource(l, pairs[l]); err != nil {
			return err
		}
	}
	for _, r := range c.right.Resources {
		if !used[r] {
			if err := c.compareResource(nil, r); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *comparison) compareResource(l, r *Resource) error {
	if l == nil || r == nil {
		c.membership(l, r, "$", "schema", nil, nil, l != nil || c.appliedAdditions[r])
		return nil
	}
	return c.schema(l, r, l.Node, r.Node, "$", "type", 1)
}

func (c *comparison) schema(l, r *Resource, left, right *yaml.Node, path, keyword string, polarity int) error {
	if sameNode(left, right) {
		return nil
	}
	usage := c.originalUsage[left] | c.modifiedUsage[right]
	if keyword == "contentSchema" || c.annotationOnly {
		previous := c.annotationOnly
		c.annotationOnly = usage == 0
		switch usage {
		case positiveContext:
			polarity = 1
		case negativeContext:
			polarity = -1
		case mixedContext:
			polarity = 0
		}
		defer func() { c.annotationOnly = previous }()
	}
	polarity = combinedPolarity(polarity, usage)
	if left.Kind == yaml.ScalarNode || right.Kind == yaml.ScalarNode {
		if left.Kind == right.Kind && strings.EqualFold(left.Value, right.Value) {
			return nil
		}
		a, b := left, right
		if polarity < 0 {
			a, b = right, left
		}
		breaking := polarity == 0 || !strings.EqualFold(a.Value, "false") && !strings.EqualFold(b.Value, "true")
		start := len(c.changes)
		wc.CreateChangeWithEncoding(&c.changes, wc.Modified, "schema", left, right, breaking, nil, nil)
		c.finish(c.changes[start], l, r, path, keyword, 1, breaking)
		return nil
	}
	// The existing low model owns keyword semantics. Child schemas and references
	// carry separate authored locations and must not expand inside this comparison.
	lp, err := constraintProxy(left, l)
	if err != nil {
		return err
	}
	rp, err := constraintProxy(right, r)
	if err != nil {
		return err
	}
	if diff := wc.CompareSchemas(lp, rp); diff != nil {
		dependencies := make(map[*wc.Change]bool, len(diff.DependentRequiredChanges))
		for _, change := range diff.DependentRequiredChanges {
			dependencies[change] = true
		}
		for _, ch := range diff.GetAllChanges() {
			property := ch.Property
			if property == "schemaDialect" {
				property = "$schema"
				ch.Property = property
			}
			editPath := PropertyPath(path, property)
			a, b := member(left, property), member(right, property)
			if dependencies[ch] {
				property = "dependentRequired"
				editPath = PropertyPath(PropertyPath(path, property), ch.Property)
				a, b = member(member(left, property), ch.Property), member(member(right, property), ch.Property)
				ch.OriginalEncoded, ch.NewEncoded = encodedLiteral(a), encodedLiteral(b)
				if a != nil && b != nil && equivalentConstraint(property, a, b) {
					continue
				}
			}
			breaking := compatibility(ch, property)
			direction := polarity
			if impact, ok := validationChange(property, a, b, polarity); ok {
				direction, breaking = 1, impact
			}
			c.finish(ch, l, r, editPath, property, direction, breaking)
			c.changes = append(c.changes, ch)
		}
	}
	for _, keyword := range []string{"$anchor", "$ref", "minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum", "multipleOf", "minContains", "maxContains"} {
		a, b := member(left, keyword), member(right, keyword)
		if a == nil && b == nil || a != nil && b != nil && a.Value == b.Value {
			continue
		}
		kind := wc.Modified
		if a == nil {
			kind = wc.PropertyAdded
		} else if b == nil {
			kind = wc.PropertyRemoved
		}
		start := len(c.changes)
		breaking := keyword == "$ref" || a != nil
		if exactNumber(keyword) && a != nil && b != nil {
			x, ok := new(big.Rat).SetString(a.Value)
			y, ok2 := new(big.Rat).SetString(b.Value)
			if ok && ok2 && x.Cmp(y) == 0 {
				continue
			}
		}
		wc.CreateChange(&c.changes, kind, keyword, a, b, breaking, nil, nil)
		if exactNumber(keyword) {
			breaking = compatibility(c.changes[start], keyword)
		}
		direction := polarity
		if impact, ok := validationChange(keyword, a, b, polarity); ok {
			direction, breaking = 1, impact
		}
		c.finish(c.changes[start], l, r, PropertyPath(path, keyword), keyword, direction, breaking)
	}
	for _, keyword := range literalKeywords {
		a, b := member(left, keyword), member(right, keyword)
		if a == b {
			continue
		}
		if keyword == "enum" {
			c.compareEnum(l, r, path, a, b, polarity)
			continue
		}
		if sameLiteral(a, b) {
			continue
		}
		if keyword == "type" && a != nil && b != nil && equivalentConstraint(keyword, a, b) {
			continue
		}
		breaking, _ := validationChange(keyword, a, b, polarity)
		if keyword == "$vocabulary" {
			breaking = true
		}
		c.recordValue(l, r, path, keyword, a, b, breaking)
	}
	c.compareAnnotations(l, r, left, right, path)
	return c.compareChildren(l, r, left, right, path, polarity)
}

// Keywords outside the supported vocabularies are annotations. Keep them out of
// the API schema model, which also recognizes legacy OpenAPI-only keywords.
func nativeConstraint(keyword string) bool {
	switch keyword {
	case "$schema", "$id", "$comment", "$dynamicRef", "$dynamicAnchor", "$vocabulary",
		"type", "required", "enum", "const", "maxLength", "minLength", "pattern", "format",
		"minItems", "maxItems", "uniqueItems", "minProperties", "maxProperties", "dependentRequired",
		"title", "description", "default", "examples", "deprecated", "readOnly", "writeOnly", "contentMediaType", "contentEncoding":
		return true
	}
	return false
}

func annotation(keyword string) bool {
	return childKind(keyword) == 0 && !nativeConstraint(keyword) && !exactNumber(keyword) && keyword != "$ref" && keyword != "$anchor"
}

func (c *comparison) compareAnnotations(l, r *Resource, left, right *yaml.Node, path string) {
	record := func(key string, a, b *yaml.Node) {
		if a != nil && b != nil && low.HashYAMLNodeSlice([]*yaml.Node{a}) == low.HashYAMLNodeSlice([]*yaml.Node{b}) {
			return
		}
		kind := wc.Modified
		if b == nil {
			kind = wc.PropertyRemoved
		} else if a == nil {
			kind = wc.PropertyAdded
		}
		wc.CreateChangeWithEncoding(&c.changes, kind, key, a, b, false, nil, nil)
		c.finish(c.changes[len(c.changes)-1], l, r, PropertyPath(path, key), key, 1, false)
	}
	for i := 0; i < len(left.Content); i += 2 {
		key := left.Content[i].Value
		if annotation(key) {
			record(key, left.Content[i+1], member(right, key))
		}
	}
	for i := 0; i < len(right.Content); i += 2 {
		key := right.Content[i].Value
		if annotation(key) && member(left, key) == nil {
			record(key, nil, right.Content[i+1])
		}
	}
}

func constraintProxy(n *yaml.Node, r *Resource) (*base.SchemaProxy, error) {
	view := *n
	view.Content = make([]*yaml.Node, 0, len(n.Content))
	for i := 0; i < len(n.Content); i += 2 {
		k := n.Content[i].Value
		if nativeConstraint(k) && !slices.Contains(literalKeywords[:], k) {
			view.Content = append(view.Content, n.Content[i:i+2]...)
		}
	}
	var proxy base.SchemaProxy
	if err := proxy.Build(context.Background(), nil, &view, r.Source.Index); err != nil {
		return nil, err
	}
	if proxy.Schema() == nil {
		return nil, fmt.Errorf("%s:%d: %w", r.Source.Name, n.Line, proxy.GetBuildError())
	}
	return &proxy, nil
}

type childKey struct{ keyword, key string }

func childNodes(n *yaml.Node) map[childKey]*yaml.Node {
	var nodes map[childKey]*yaml.Node
	_ = children(n, func(kw, k string, child *yaml.Node) error {
		if nodes == nil {
			nodes = make(map[childKey]*yaml.Node)
		}
		nodes[childKey{kw, k}] = child
		return nil
	})
	return nodes
}

func sameNode(a, b *yaml.Node) bool {
	if a == b {
		return true
	}
	if a.Kind != b.Kind || a.Tag != b.Tag || a.Value != b.Value || len(a.Content) != len(b.Content) {
		return false
	}
	for i := range a.Content {
		if !sameNode(a.Content[i], b.Content[i]) {
			return false
		}
	}
	return true
}

var literalKeywords = [...]string{"type", "enum", "const", "examples", "default", "$vocabulary"}

// encodedLiteral retains typed values in report records and their hashes.
func encodedLiteral(n *yaml.Node) string {
	if n == nil {
		return ""
	}
	switch n.Kind {
	case yaml.MappingNode:
		fields := make([]string, 0, len(n.Content)/2)
		for i := 0; i < len(n.Content); i += 2 {
			key, _ := json.Marshal(n.Content[i].Value)
			fields = append(fields, string(key)+":"+encodedLiteral(n.Content[i+1]))
		}
		slices.Sort(fields)
		return "{" + strings.Join(fields, ",") + "}"
	case yaml.SequenceNode:
		values := make([]string, 0, len(n.Content))
		for _, child := range n.Content {
			values = append(values, encodedLiteral(child))
		}
		return "[" + strings.Join(values, ",") + "]"
	}
	switch n.ShortTag() {
	case "!!int", "!!float":
		return n.Value
	case "!!null":
		return "null"
	case "!!bool":
		return strings.ToLower(n.Value)
	}
	quoted, _ := json.Marshal(n.Value)
	return string(quoted)
}

func (c *comparison) recordValue(l, r *Resource, path, keyword string, a, b *yaml.Node, breaking bool) {
	kind := wc.Modified
	if a == nil {
		kind = wc.PropertyAdded
	} else if b == nil {
		kind = wc.PropertyRemoved
	}
	wc.CreateChange(&c.changes, kind, keyword, a, b, breaking, nil, nil)
	ch := c.changes[len(c.changes)-1]
	ch.OriginalEncoded, ch.NewEncoded = encodedLiteral(a), encodedLiteral(b)
	c.finish(ch, l, r, PropertyPath(path, keyword), keyword, 1, breaking)
}

func (c *comparison) compareEnum(l, r *Resource, path string, a, b *yaml.Node, polarity int) {
	var values []*yaml.Node
	if a != nil {
		values = append(values, a.Content...)
	}
	boundary := len(values)
	if b != nil {
		values = append(values, b.Content...)
	}
	if len(values) == 0 {
		if (a == nil) != (b == nil) {
			c.recordValue(l, r, path, "enum", a, b, polarity == 0 || (polarity > 0) == (a == nil))
		}
		return
	}
	ids := literalIDs(values)
	old, current := make(map[int]bool, boundary), make(map[int]bool, len(values)-boundary)
	for i, id := range ids {
		if i < boundary {
			old[id] = true
		} else {
			current[id] = true
		}
	}
	for i, id := range ids {
		if i < boundary && !current[id] {
			// Removing an entire enum relaxes its constraint; removing one choice narrows it.
			c.recordValue(l, r, path, "enum", values[i], nil, polarity == 0 || (polarity > 0) == (b != nil))
			current[id] = true
		}
		if i >= boundary && !old[id] {
			c.recordValue(l, r, path, "enum", nil, values[i], polarity == 0 || (polarity > 0) == (a == nil))
			old[id] = true
		}
	}
}

func sameLiteral(a, b *yaml.Node) bool {
	if a == nil || b == nil {
		return a == b
	}
	ids := literalIDs([]*yaml.Node{a, b})
	return ids[0] == ids[1]
}

func literalIDs(nodes []*yaml.Node) []int {
	ids := utils.JSONValueIDs(nodes)
	var fallback map[string]int
	for i, id := range ids {
		if id != 0 {
			continue
		}
		// The shared interner bounds arbitrary-precision arithmetic. Keep distinct
		// out-of-range literals distinguishable without expanding their exponents.
		if fallback == nil {
			fallback = make(map[string]int)
		}
		wire := encodedLiteral(nodes[i])
		if fallback[wire] == 0 {
			fallback[wire] = -len(fallback) - 1
		}
		ids[i] = fallback[wire]
	}
	return ids
}

func typeAllows(n *yaml.Node, typ string) bool {
	if n == nil {
		return true
	}
	if n.Kind == yaml.SequenceNode {
		return slices.ContainsFunc(n.Content, func(item *yaml.Node) bool { return typeAllows(item, typ) })
	}
	return n.Value == typ || n.Value == "number" && typ == "integer"
}

func equivalentConstraint(keyword string, a, b *yaml.Node) bool {
	forward, _ := validationChange(keyword, a, b, 1)
	reverse, _ := validationChange(keyword, b, a, 1)
	return !forward && !reverse
}

// validationChange checks whether b rejects values accepted by a. Reversing the
// operands gives the dual check inside not. Other keywords use their native rule.
func validationChange(keyword string, a, b *yaml.Node, polarity int) (breaking, handled bool) {
	if polarity < 0 {
		a, b = b, a
	}
	switch keyword {
	case "type":
		for _, typ := range []string{"integer", "number", "object", "array", "string", "null", "boolean"} {
			if typeAllows(a, typ) && !typeAllows(b, typ) {
				return true, true
			}
		}
	case "const", "pattern":
		breaking = b != nil
	case "uniqueItems":
		unique := func(n *yaml.Node) bool { return n != nil && strings.EqualFold(n.Value, "true") }
		breaking = unique(b) && !unique(a)
	case "dependentRequired":
		if b != nil {
			for _, req := range b.Content {
				if a == nil || !slices.ContainsFunc(a.Content, func(item *yaml.Node) bool { return item.Value == req.Value }) {
					return true, true
				}
			}
		}
	case "multipleOf":
		if a == nil || b == nil {
			return polarity == 0 || b != nil, true
		}
		x, ok1 := new(big.Rat).SetString(a.Value)
		y, ok2 := new(big.Rat).SetString(b.Value)
		breaking = !ok1 || !ok2 || y.Sign() == 0 || !x.Quo(x, y).IsInt()
	default:
		return false, false
	}
	return polarity == 0 || breaking, true
}

func childMembership(keyword string, whole, added bool, polarity int) bool {
	if keyword == "contentSchema" {
		return false
	}
	if keyword == "$defs" {
		return !added
	}
	if polarity == 0 {
		return true
	}
	switch keyword {
	case "anyOf":
		// Adding the first disjunction restricts the schema. Adding an
		// alternative to an existing disjunction widens it.
		return (whole == added) == (polarity > 0)
	case "allOf", "dependentSchemas":
		return added == (polarity > 0)
	}
	return true
}

func exactNumber(k string) bool {
	switch k {
	case "minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum", "multipleOf", "minContains", "maxContains":
		return true
	}
	return false
}

func (c *comparison) compareChildren(l, r *Resource, left, right *yaml.Node, path string, polarity int) error {
	old, newNodes := childNodes(left), childNodes(right)
	// allOf, anyOf, and oneOf have unordered operands. Match identical branches
	// before comparing the remaining positions. prefixItems is an ordered tuple.
	for _, kw := range []string{"allOf", "anyOf", "oneOf"} {
		if member(left, kw) == nil || member(right, kw) == nil {
			continue
		}
		byHash := make(map[string][]childKey)
		for i := range member(right, kw).Content {
			k := childKey{kw, strconv.Itoa(i)}
			if n := newNodes[k]; n != nil {
				hash := low.HashYAMLNodeSlice([]*yaml.Node{n})
				byHash[hash] = append(byHash[hash], k)
			}
		}
		for i := range member(left, kw).Content {
			k := childKey{kw, strconv.Itoa(i)}
			if n := old[k]; n != nil {
				hash := low.HashYAMLNodeSlice([]*yaml.Node{n})
				if matches := byHash[hash]; len(matches) > 0 {
					delete(old, k)
					delete(newNodes, matches[0])
					byHash[hash] = matches[1:]
				}
			}
		}
		var remainingLeft, remainingRight []childKey
		for i := range member(left, kw).Content {
			k := childKey{kw, strconv.Itoa(i)}
			if old[k] != nil {
				remainingLeft = append(remainingLeft, k)
			}
		}
		for i := range member(right, kw).Content {
			k := childKey{kw, strconv.Itoa(i)}
			if newNodes[k] != nil {
				remainingRight = append(remainingRight, k)
			}
		}
		for i := 0; i < min(len(remainingLeft), len(remainingRight)); i++ {
			lk, rk := remainingLeft[i], remainingRight[i]
			a, b := old[lk], newNodes[rk]
			delete(old, lk)
			delete(newNodes, rk)
			if err := c.compareChild(l, r, a, b, childPath(path, rk), kw, polarity); err != nil {
				return err
			}
		}
	}
	for k, a := range old {
		b := newNodes[k]
		delete(newNodes, k)
		p := childPath(path, k)
		if b == nil {
			if owner := c.left.schemas[a]; owner != l && c.pairs[owner] == nil {
				continue
			}
			breaking := childMembership(k.keyword, member(right, k.keyword) == nil, false, polarity)
			c.membership(l, r, p, k.keyword, a, nil, breaking)
			continue
		}
		direction := polarity
		if k.keyword == "contains" && (member(left, "maxContains") != nil || member(right, "maxContains") != nil) {
			direction = 0
		}
		if err := c.compareChild(l, r, a, b, p, k.keyword, direction); err != nil {
			return err
		}
	}
	for k, b := range newNodes {
		breaking := childMembership(k.keyword, member(left, k.keyword) == nil, true, polarity)
		if owner := c.right.schemas[b]; owner != r && !c.matched[owner] {
			if breaking && !c.annotationOnly {
				if c.appliedAdditions == nil {
					c.appliedAdditions = make(map[*Resource]bool)
				}
				c.appliedAdditions[owner] = true
			}
			continue
		}
		c.membership(l, r, childPath(path, k), k.keyword, nil, b, breaking)
	}
	return nil
}

func (c *comparison) compareChild(l, r *Resource, a, b *yaml.Node, path, keyword string, polarity int) error {
	switch keyword {
	case "not":
		polarity = -polarity
	case "oneOf", "if":
		polarity = 0
	case "$defs":
		polarity = 1
	}
	lo, ro := c.left.schemas[a], c.right.schemas[b]
	switch {
	case lo == l && ro == r:
		return c.schema(l, r, a, b, path, keyword, polarity)
	case lo != l && ro != r:
		if c.pairs[lo] != ro && (c.pairs[lo] != nil || c.matched[ro]) {
			wc.CreateChange(&c.changes, wc.Modified, keyword, member(a, "$id"), member(b, "$id"), true, nil, nil)
			c.finish(c.changes[len(c.changes)-1], l, r, path, keyword, 1, true)
		}
	case lo == l && !c.matched[ro]:
		// An inline schema acquired its first $id. Its constraints now belong to the new resource.
		c.matched[ro] = true
		return c.schema(l, ro, a, b, "$", keyword, polarity)
	case ro == r && c.pairs[lo] == nil:
		if c.inline == nil {
			c.inline = make(map[*Resource]bool)
		}
		c.inline[lo] = true
		return c.schema(lo, r, a, b, path, keyword, polarity)
	default:
		// A matched resource moved to or from an existing inline use.
		wc.CreateChangeWithEncoding(&c.changes, wc.Modified, keyword, a, b, true, nil, nil)
		c.finish(c.changes[len(c.changes)-1], l, r, path, keyword, 1, true)
	}
	return nil
}

// PropertyPath appends an object key to a JSONPath without changing its identity.
func PropertyPath(path, key string) string {
	valid := key != ""
	for i, c := range key {
		if !(c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || i > 0 && c >= '0' && c <= '9') {
			valid = false
			break
		}
	}
	if valid {
		return path + "." + key
	}
	return path + "[" + strconv.Quote(key) + "]"
}

func childPath(path string, k childKey) string {
	p := PropertyPath(path, k.keyword)
	switch childKind(k.keyword) {
	case schemaMap:
		return PropertyPath(p, k.key)
	case schemaList:
		return p + "[" + k.key + "]"
	}
	return p
}

func (c *comparison) membership(l, r *Resource, path, kw string, a, b *yaml.Node, breaking bool) {
	if l == nil && b == nil {
		b = r.Node
	}
	if r == nil && a == nil {
		a = l.Node
	}
	kind := wc.ObjectAdded
	if b == nil {
		kind = wc.ObjectRemoved
	}
	start := len(c.changes)
	wc.CreateChangeWithEncoding(&c.changes, kind, kw, a, b, breaking, nil, nil)
	c.finish(c.changes[start], l, r, path, kw, 1, breaking)
}

func compatibility(ch *wc.Change, property string) bool {
	switch property {
	case "required":
		return ch.ChangeType != wc.PropertyRemoved
	case "$id", "$dynamicAnchor":
		return ch.ChangeType != wc.PropertyAdded
	case "minimum", "exclusiveMinimum", "minLength", "minItems", "minProperties", "minContains", "maximum", "exclusiveMaximum", "maxLength", "maxItems", "maxProperties", "maxContains":
		if ch.ChangeType == wc.PropertyRemoved {
			return false
		}
		if ch.ChangeType == wc.PropertyAdded {
			return true
		}
		a, ok := new(big.Rat).SetString(ch.Original)
		b, ok2 := new(big.Rat).SetString(ch.New)
		if ok && ok2 {
			sign := b.Cmp(a)
			if strings.HasPrefix(property, "max") || property == "exclusiveMaximum" {
				return sign < 0
			}
			return sign > 0
		}
	}
	if rule := standaloneDefaults.GetRule(wc.CompSchema, property); rule != nil {
		if value := operationRule(rule, ch.ChangeType); value != nil {
			return *value
		}
	}
	return true
}

var standaloneDefaults = wc.GenerateDefaultBreakingRules()

func operationRule(rule *wc.BreakingChangeRule, kind int) *bool {
	switch kind {
	case wc.PropertyAdded, wc.ObjectAdded:
		return rule.Added
	case wc.PropertyRemoved, wc.ObjectRemoved:
		return rule.Removed
	default:
		return rule.Modified
	}
}

func (c *comparison) finish(ch *wc.Change, l, r *Resource, path, property string, polarity int, breaking bool) {
	switch property {
	case "description", "title", "$comment", "default", "examples", "readOnly", "writeOnly", "deprecated", "format", "contentMediaType", "contentEncoding":
		breaking = false
	case "$id", "$anchor", "$dynamicAnchor":
	default:
		if polarity < 0 {
			switch property {
			case "type", "required", "minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum", "minLength", "maxLength", "minContains", "maxContains", "minItems", "maxItems", "minProperties", "maxProperties":
				if property == "type" && ch.Property != "schema" {
					breaking = true
				} else {
					breaking = !breaking
				}
			default:
				breaking = true
			}
		} else if polarity == 0 {
			breaking = true
		}
	}
	if c.annotationOnly || property == "contentSchema" {
		breaking = false
	}
	if c.rules != nil {
		ruleKey := property
		if property == "$schema" {
			ruleKey = wc.PropSchemaDialect
		}
		if rule := c.rules.GetRule(wc.CompSchema, ruleKey); rule != nil {
			if override := operationRule(rule, ch.ChangeType); override != nil {
				breaking = *override
			}
		}
	}
	ch.Breaking, ch.Path, ch.Type = breaking, path, "schema"
	if l != nil {
		ch.SchemaResource = l.ID
		if ch.Context.OriginalLine != nil {
			ch.Context.OriginalDocument = l.Source.Name
			ch.Context.DocumentLocation = l.Source.Name
		}
	}
	if r != nil {
		ch.SchemaResource = r.ID
		if ch.Context.NewLine != nil {
			ch.Context.ModifiedDocument = r.Source.Name
			ch.Context.DocumentLocation = r.Source.Name
		}
	}
}
