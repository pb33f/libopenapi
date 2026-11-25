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
	wc "github.com/pb33f/libopenapi/what-changed/model"
)

// Compare uses the schema comparator on authored constraints. Each referenced
// resource is compared once, independent of the number of references to it.
func Compare(left, right *Document, rules *wc.BreakingRulesConfig) (*wc.SchemaChanges, error) {
	if left == nil || right == nil {
		return nil, fmt.Errorf("both JSON Schema inputs are required")
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
	pairs       map[*Resource]*Resource
	matched     map[*Resource]bool
	left, right *Document
	rules       *wc.BreakingRulesConfig
	changes     []*wc.Change
}

func (c *comparison) resources() error {
	byID := make(map[string]*Resource, len(c.right.Resources))
	type origin struct{ file, pointer string }
	byOrigin := make(map[origin]*Resource, len(c.right.Resources))
	used := make(map[*Resource]bool)
	pairs := make(map[*Resource]*Resource)
	c.pairs, c.matched = pairs, used
	for _, r := range c.right.Resources {
		byID[r.ID] = r
		byOrigin[origin{r.Source.relative, r.Pointer}] = r
	}
	if c.left.Root != nil && c.right.Root != nil {
		pairs[c.left.Root] = c.right.Root
		used[c.right.Root] = true
	}
	for _, l := range c.left.Resources {
		if pairs[l] == nil {
			if r := byID[l.ID]; r != nil && !used[r] {
				pairs[l] = r
				used[r] = true
			}
		}
	}
	for _, l := range c.left.Resources {
		r := pairs[l]
		if r == nil {
			file := l.Source.relative
			if c.left.Root != nil && c.right.Root != nil && l.Source == c.left.Root.Source {
				file = c.right.Root.Source.relative
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
	for _, l := range c.left.Resources {
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
		c.membership(l, r, "$", "schema", nil, nil, true)
		return nil
	}
	return c.schema(l, r, l.Node, r.Node, "$", 1)
}

func (c *comparison) schema(l, r *Resource, left, right *yaml.Node, path string, polarity int) error {
	if sameNode(left, right) {
		return nil
	}
	if left.Kind == yaml.ScalarNode || right.Kind == yaml.ScalarNode {
		if left.Kind == right.Kind && strings.EqualFold(left.Value, right.Value) {
			return nil
		}
		breaking := !strings.EqualFold(left.Value, "false") && !strings.EqualFold(right.Value, "true")
		start := len(c.changes)
		wc.CreateChangeWithEncoding(&c.changes, wc.Modified, "schema", left, right, breaking, nil, nil)
		c.finish(c.changes[start], l, r, path, "type", polarity, breaking)
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
		for _, ch := range diff.GetAllChanges() {
			property := ch.Property
			if property == "schemaDialect" {
				property = "$schema"
			}
			editPath := PropertyPath(path, property)
			for _, dep := range diff.DependentRequiredChanges {
				if ch == dep {
					editPath = PropertyPath(PropertyPath(path, "dependentRequired"), property)
					if a, ok := ch.OriginalObject.([]string); ok {
						value, _ := json.Marshal(a)
						ch.OriginalEncoded = string(value)
					}
					if b, ok := ch.NewObject.([]string); ok {
						value, _ := json.Marshal(b)
						ch.NewEncoded = string(value)
					}
					property = "dependentRequired"
					break
				}
			}
			c.finish(ch, l, r, editPath, property, polarity, compatibility(ch, property))
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
		c.finish(c.changes[start], l, r, PropertyPath(path, keyword), keyword, polarity, breaking)
	}
	return c.compareChildren(l, r, left, right, path, polarity)
}

func constraintProxy(n *yaml.Node, r *Resource) (*base.SchemaProxy, error) {
	view := *n
	view.Content = make([]*yaml.Node, 0, len(n.Content))
	for i := 0; i < len(n.Content); i += 2 {
		k := n.Content[i].Value
		if childKind(k) == 0 && k != "$ref" && k != "$anchor" && !exactNumber(k) {
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
			direction := polarity
			if kw == "oneOf" {
				direction = 0
			}
			if c.left.schemas[a] == l && c.right.schemas[b] == r {
				if err := c.schema(l, r, a, b, childPath(path, rk), direction); err != nil {
					return err
				}
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
			c.membership(l, r, p, k.keyword, a, nil, true)
			continue
		}
		direction := polarity
		switch k.keyword {
		case "not":
			direction = -direction
		case "oneOf", "if":
			direction = 0
		case "$defs":
			direction = 1
		}
		leftOwner, rightOwner := c.left.schemas[a], c.right.schemas[b]
		if leftOwner != l && rightOwner != r {
			if c.pairs[leftOwner] != rightOwner && (c.pairs[leftOwner] != nil || c.matched[rightOwner]) {
				wc.CreateChange(&c.changes, wc.Modified, k.keyword, member(a, "$id"), member(b, "$id"), true, nil, nil)
				c.finish(c.changes[len(c.changes)-1], l, r, p, k.keyword, 1, true)
			}
			continue
		}
		if err := c.schema(l, r, a, b, p, direction); err != nil {
			return err
		}
	}
	for k, b := range newNodes {
		if owner := c.right.schemas[b]; owner != r && !c.matched[owner] {
			continue
		}
		breaking := k.keyword != "$defs" && k.keyword != "anyOf"
		c.membership(l, r, childPath(path, k), k.keyword, nil, b, breaking)
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
		breaking = false
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
	case "description", "title", "$comment", "default", "examples", "readOnly", "writeOnly", "deprecated":
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
	if c.rules != nil {
		if rule := c.rules.GetRule(wc.CompSchema, property); rule != nil {
			override := rule.Modified
			switch ch.ChangeType {
			case wc.PropertyAdded, wc.ObjectAdded:
				override = rule.Added
			case wc.PropertyRemoved, wc.ObjectRemoved:
				override = rule.Removed
			}
			if override != nil {
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
