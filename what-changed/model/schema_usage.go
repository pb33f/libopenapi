// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
)

// schemaDirection records which side of an exchange can carry a schema.
// A request schema describes data that a client sends. A response schema
// describes data that a server returns.
type schemaDirection uint8

const (
	directionRequest schemaDirection = 1 << iota
	directionResponse
	directionBoth = directionRequest | directionResponse
)

// schemaUsage maps the nodes of one OpenAPI 3 document to the directions that
// can reach them from its operations and webhooks.
type schemaUsage struct {
	nodes map[*yaml.Node]schemaDirection
}

// schemaUsageByIndex holds the usage of each document while CompareDocuments
// runs. A schema finds the usage of its document through its index. A schema
// whose index is not here has no known direction. Comparisons that run at the
// same time can share a document, so each record counts its comparisons.
var (
	schemaUsageMu      sync.RWMutex
	schemaUsageByIndex = make(map[*index.SpecIndex]*trackedSchemaUsage)
)

type trackedSchemaUsage struct {
	usage *schemaUsage
	refs  int
}

// trackSchemaUsage records the usage of each document index for one comparison
// and returns a function that releases it again.
func trackSchemaUsage(indexes ...*index.SpecIndex) func() {
	schemaUsageMu.Lock()
	defer schemaUsageMu.Unlock()
	var tracked []*index.SpecIndex
	for _, idx := range indexes {
		if idx == nil {
			continue
		}
		entry, ok := schemaUsageByIndex[idx]
		if !ok {
			usage := buildSchemaUsage(idx.GetRootNode())
			if usage == nil {
				continue
			}
			entry = &trackedSchemaUsage{usage: usage}
			schemaUsageByIndex[idx] = entry
		}
		entry.refs++
		tracked = append(tracked, idx)
	}
	return func() {
		schemaUsageMu.Lock()
		defer schemaUsageMu.Unlock()
		for _, idx := range tracked {
			if entry := schemaUsageByIndex[idx]; entry.refs > 1 {
				entry.refs--
			} else {
				delete(schemaUsageByIndex, idx)
			}
		}
	}
}

// schemaPairDirection returns the union of the directions that reach the left
// and right schemas. known is false when either side has no usage record, so
// callers keep their direction-free result.
func schemaPairDirection(l, r *base.SchemaProxy) (direction schemaDirection, known bool) {
	if l == nil && r == nil {
		return 0, false
	}
	for _, proxy := range []*base.SchemaProxy{l, r} {
		if proxy == nil {
			continue
		}
		d, ok := proxyDirection(proxy)
		if !ok {
			return 0, false
		}
		direction |= d
	}
	return direction, true
}

func proxyDirection(proxy *base.SchemaProxy) (schemaDirection, bool) {
	node := proxy.GetValueNode()
	if node == nil {
		return 0, false
	}
	schemaUsageMu.RLock()
	entry, ok := schemaUsageByIndex[proxy.GetIndex()]
	schemaUsageMu.RUnlock()
	if !ok {
		return 0, false
	}
	return entry.usage.nodes[node], true
}

// buildSchemaUsage walks every operation and webhook of an OpenAPI 3 document.
// Request bodies and parameters mark their subtrees as request data. Responses
// mark their subtrees as response data. Callbacks, webhooks, and any other
// operation field mark both, because their direction is not the direction of
// the API's own operations. Each local $ref passes the direction to its target.
//
// The walk gives up, and returns nil, when it cannot see every place that a
// schema is used: an external or unresolvable $ref, a $dynamicRef, or a
// discriminator mapping that it cannot resolve.
func buildSchemaUsage(root *yaml.Node) *schemaUsage {
	if root != nil && root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	if root == nil || root.Kind != yaml.MappingNode {
		return nil
	}
	w := &usageWalker{
		root:    root,
		nodes:   make(map[*yaml.Node]schemaDirection),
		visited: make(map[*yaml.Node]struct{}),
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		switch root.Content[i].Value {
		case "paths":
			w.eachValue(root.Content[i+1], w.pathItem)
		case "webhooks":
			w.eachValue(root.Content[i+1], func(n *yaml.Node) { w.walk(n, directionBoth) })
		}
	}
	if w.unsure {
		return nil
	}
	return &schemaUsage{nodes: w.nodes}
}

type usageWalker struct {
	root  *yaml.Node
	nodes map[*yaml.Node]schemaDirection
	// visited holds the path items and operations already read, so that
	// $ref cycles between them end.
	visited map[*yaml.Node]struct{}
	unsure  bool
}

// first reports whether node is a path item or operation not read before.
func (w *usageWalker) first(node *yaml.Node) bool {
	if _, done := w.visited[node]; done {
		return false
	}
	w.visited[node] = struct{}{}
	return true
}

var operationKeys = map[string]bool{
	"get": true, "put": true, "post": true, "delete": true, "options": true,
	"head": true, "patch": true, "trace": true, "query": true,
}

func (w *usageWalker) eachValue(node *yaml.Node, visit func(*yaml.Node)) {
	if node == nil || node.Kind != yaml.MappingNode {
		return
	}
	for i := 1; i < len(node.Content); i += 2 {
		visit(node.Content[i])
	}
}

func (w *usageWalker) pathItem(node *yaml.Node) {
	if node.Kind != yaml.MappingNode || !w.first(node) {
		return
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key, value := node.Content[i].Value, node.Content[i+1]
		switch {
		case key == "$ref":
			if target := w.resolveOrGiveUp(value.Value); target != nil {
				w.pathItem(target)
			}
		case key == "parameters":
			w.walk(value, directionRequest)
		case operationKeys[key]:
			w.operation(value)
		case key == "additionalOperations":
			w.eachValue(value, w.operation)
		default:
			w.walk(value, directionBoth)
		}
	}
}

func (w *usageWalker) operation(node *yaml.Node) {
	if node.Kind != yaml.MappingNode || !w.first(node) {
		return
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		value := node.Content[i+1]
		switch node.Content[i].Value {
		case "$ref":
			// libopenapi also reads an operation from a $ref.
			if target := w.resolveOrGiveUp(value.Value); target != nil {
				w.operation(target)
			}
		case "parameters", "requestBody":
			w.walk(value, directionRequest)
		case "responses":
			w.walk(value, directionResponse)
		default:
			w.walk(value, directionBoth)
		}
	}
}

func (w *usageWalker) walk(node *yaml.Node, direction schemaDirection) {
	if node == nil || node.Kind == yaml.ScalarNode || w.unsure {
		return
	}
	// a node already reached in this direction needs no second visit.
	if w.nodes[node]&direction == direction {
		return
	}
	w.nodes[node] |= direction
	switch node.Kind {
	case yaml.AliasNode:
		w.walk(node.Alias, direction)
	case yaml.SequenceNode:
		for _, item := range node.Content {
			w.walk(item, direction)
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			key, value := node.Content[i].Value, node.Content[i+1]
			switch key {
			case "$ref":
				w.walk(w.resolveOrGiveUp(value.Value), direction)
			case "$dynamicRef":
				w.unsure = true
				return
			case "discriminator":
				w.discriminatorMapping(value, direction)
			}
			w.walk(value, direction)
		}
	}
}

func (w *usageWalker) discriminatorMapping(node *yaml.Node, direction schemaDirection) {
	if node.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value != "mapping" {
			continue
		}
		w.eachValue(node.Content[i+1], func(target *yaml.Node) {
			ref := target.Value
			if !strings.HasPrefix(ref, "#") {
				ref = "#/components/schemas/" + ref
			}
			w.walk(w.resolveOrGiveUp(ref), direction)
		})
	}
}

func (w *usageWalker) resolveOrGiveUp(ref string) *yaml.Node {
	target := w.resolve(ref)
	if target == nil {
		w.unsure = true
	}
	return target
}

// resolve follows a local JSON pointer such as #/components/schemas/Widget.
// It returns nil for any other reference.
func (w *usageWalker) resolve(ref string) *yaml.Node {
	if !strings.HasPrefix(ref, "#/") {
		return nil
	}
	node := w.root
	for _, token := range strings.Split(ref[2:], "/") {
		token, err := url.PathUnescape(token)
		if err != nil {
			return nil
		}
		token = strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
		node = pointerChild(node, token)
		if node == nil {
			return nil
		}
	}
	return node
}

func pointerChild(node *yaml.Node, token string) *yaml.Node {
	for node.Kind == yaml.AliasNode {
		node = node.Alias
	}
	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			if node.Content[i].Value == token {
				return node.Content[i+1]
			}
		}
	case yaml.SequenceNode:
		if i, err := strconv.Atoi(token); err == nil && i >= 0 && i < len(node.Content) {
			return node.Content[i]
		}
	}
	return nil
}

// requiredChangeBreaking returns the breaking classification of an added and a
// removed required entry. A new required entry only narrows what a server may
// return, so it does not break a schema that only responses use. A removed
// required entry only relaxes what a client must send, so it does not break a
// schema that only requests use. Every other case keeps the configured rule.
func requiredChangeBreaking(l, r *base.SchemaProxy) (added, removed bool) {
	added = BreakingAdded(CompSchema, PropRequired)
	removed = BreakingRemoved(CompSchema, PropRequired)
	if !added && !removed {
		return added, removed
	}
	direction, known := schemaPairDirection(l, r)
	if !known {
		return added, removed
	}
	if direction == directionResponse {
		added = false
	}
	if direction == directionRequest {
		removed = false
	}
	return added, removed
}
