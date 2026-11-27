// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"context"
	"iter"
	"regexp"
	"strings"
	"sync"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
)

// Each document comparison shares one lazy usage map per side across its workers.
// Standalone comparisons have no document context.
type documentComparison struct {
	left, right func() schemaUsage
}

func newDocumentComparison(left, right *index.SpecIndex) documentComparison {
	return documentComparison{
		left:  sync.OnceValue(func() schemaUsage { return buildSchemaUsage(left) }),
		right: sync.OnceValue(func() schemaUsage { return buildSchemaUsage(right) }),
	}
}

func (d documentComparison) compareSchemas(l, r *base.SchemaProxy) *SchemaChanges {
	c := schemaComparison{documentComparison: d}
	return c.compareSchemas(l, r)
}

type schemaDirection uint8

const (
	requestSchema schemaDirection = 1 << iota
	responseSchema
	sharedSchema = requestSchema | responseSchema
)

type schemaUsage map[*yaml.Node]schemaDirection

func (d documentComparison) exclusively(l, r *base.SchemaProxy, direction schemaDirection) bool {
	return d.left != nil && l != nil && r != nil &&
		d.left().direction(l) == direction && d.right().direction(r) == direction
}

// requiredAdditionRule checks a closed response's declared names once, even when many become required.
func (d documentComparison) requiredAdditionRule(l, r *base.SchemaProxy, old, next *base.Schema) func(string) bool {
	breaking := BreakingAdded(CompSchema, PropRequired)
	if !breaking || !d.exclusively(l, r, responseSchema) {
		return func(string) bool { return breaking }
	}
	if !restrictsProperties(old.AdditionalProperties.Value) && !restrictsProperties(old.UnevaluatedProperties.Value) {
		return func(string) bool { return false }
	}
	// Existing properties were already accepted by the old object. For dictionary keys,
	// compare the rest of the contract once: patterns or applicators can admit new fields.
	declared := make(map[string]bool)
	for key := range old.Properties.Value.KeysFromOldest() {
		declared[key.Value] = true
	}
	otherChanges := sync.OnceValue(func() bool {
		view := copySchemaPublicFields(old)
		view.Required = next.Required
		return view.Hash() != next.Hash()
	})
	return func(name string) bool { return !declared[name] && otherChanges() }
}

func restrictsProperties(rule *base.SchemaDynamicValue[*base.SchemaProxy, bool]) bool {
	return rule != nil && (rule.IsA() || !rule.B)
}

// A ref use can be narrower than the target it describes. Require proof for both.
func (usage schemaUsage) direction(proxy *base.SchemaProxy) schemaDirection {
	d := usage[utils.NodeAlias(proxy.GetValueNode())]
	if d != 0 && proxy.IsReference() {
		resolved, _, err, _ := low.LocateRefEnd(proxy.GetContext(), proxy.GetReferenceNode(), proxy.GetIndex(), 0)
		target := usage[utils.NodeAlias(resolved)]
		if err != nil || target == 0 {
			return 0
		}
		return d | target
	}
	return d
}

type usageObject uint8

const (
	pathItemObject usageObject = iota
	operationObject
	callbackObject
	payloadObject // parameter, header, request body, response, media type or encoding
	schemaObject
)

type usageWalker struct {
	index   *index.SpecIndex
	visited [schemaObject + 1]schemaUsage
	targets map[string]*yaml.Node
	unknown bool
	// schemas are the component schemas, which a discriminator can select by name.
	schemas *yaml.Node
	// refSiblings reads the keywords next to $ref, as the schema model does with TransformSiblingRefs. Without it, the
	// model reads a schema with a $ref as the referenced schema alone.
	refSiblings bool
	// children lists, for each schema, the component schemas that build on it with allOf, directly or through other
	// schemas. It is built for the first discriminator that needs it.
	children map[*yaml.Node][]*yaml.Node
}

func buildSchemaUsage(idx *index.SpecIndex) schemaUsage {
	if idx == nil || idx.GetRootNode() == nil || len(idx.GetAllSchemaIds()) != 0 {
		return nil
	}
	root := idx.GetRootNode()
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	w := usageWalker{
		index:       idx,
		targets:     make(map[string]*yaml.Node),
		schemas:     usageField(usageField(root, v3.ComponentsLabel), v3.SchemasLabel),
		refSiblings: idx.GetConfig() != nil && idx.GetConfig().TransformSiblingRefs,
	}
	for i := range w.visited {
		w.visited[i] = make(schemaUsage)
	}
	for key, value := range usageFields(root) {
		switch key {
		case v3.JSONSchemaDialectLabel:
			if !knownUsageDialect(value) {
				return nil
			}
		case v3.PathsLabel:
			w.openAPIEntries(value, pathItemObject, 0)
		case v3.WebhooksLabel:
			w.each(value, pathItemObject, sharedSchema)
		}
	}
	if w.unknown {
		return nil
	}
	return w.visited[schemaObject]
}

func usageFields(node *yaml.Node) iter.Seq2[string, *yaml.Node] {
	return func(yield func(string, *yaml.Node) bool) {
		node = utils.ResolveMergeNodes(node)
		if node == nil || node.Kind != yaml.MappingNode {
			return
		}
		for i := 0; i+1 < len(node.Content); i += 2 {
			if !yield(utils.NodeAlias(node.Content[i]).Value, node.Content[i+1]) {
				return
			}
		}
	}
}

// Paths and Responses allow extension fields alongside their entries.
func (w *usageWalker) openAPIEntries(node *yaml.Node, object usageObject, dir schemaDirection) {
	for key, value := range usageFields(node) {
		if !strings.HasPrefix(key, "x-") {
			w.walk(value, object, dir)
		}
	}
}

// each visits the members of an OpenAPI map or array without interpreting their names as keywords.
func (w *usageWalker) each(node *yaml.Node, object usageObject, dir schemaDirection) {
	node = utils.NodeAlias(node)
	if node.Kind == yaml.SequenceNode {
		for _, item := range node.Content {
			w.walk(item, object, dir)
		}
		return
	}
	for _, value := range usageFields(node) {
		w.walk(value, object, dir)
	}
}

func (w *usageWalker) walk(node *yaml.Node, object usageObject, dir schemaDirection) {
	node = utils.NodeAlias(node)
	if w.unknown || node == nil || node.Kind != yaml.MappingNode {
		return
	}
	visited := w.visited[object]
	if prior, seen := visited[node]; seen && prior&dir == dir {
		return
	}
	visited[node] |= dir
	sameInstance := dir
	if object == schemaObject {
		if utils.ResolveMergeNodes(node) != node {
			// The index may return a distinct expanded mapping for each reference to this schema.
			w.unknown = true
			return
		}
		// A field declared by a subschema can bypass its parent's unevaluatedProperties rule.
		for key, value := range usageFields(node) {
			if key == v3.UnevaluatedPropertiesLabel && !(utils.IsNodeBoolValue(value) && utils.NodeAlias(value).Value == "true") {
				sameInstance = sharedSchema
				break
			}
		}
	}
	for key, value := range usageFields(node) {
		if key == v3.RefLabel {
			ref := utils.NodeAlias(value).Value
			if target := w.localTarget(ref); target != nil {
				w.walk(target, object, sameInstance)
				continue
			}
			w.unknown = true
			return
		}
		switch object {
		case pathItemObject, operationObject:
			w.operationField(key, value, dir)
		case callbackObject:
			if !strings.HasPrefix(key, "x-") {
				w.walk(value, pathItemObject, sharedSchema)
			}
		case payloadObject:
			switch key {
			case v3.SchemaLabel, v3.ItemSchemaLabel:
				w.walk(value, schemaObject, dir)
			case v3.ContentLabel, v3.HeadersLabel, v3.EncodingLabel, v3.PrefixEncodingLabel:
				w.each(value, payloadObject, dir)
			case v3.ItemEncodingLabel:
				w.walk(value, payloadObject, dir)
			}
		case schemaObject:
			w.schemaField(node, key, value, dir, sameInstance)
		}
	}
}

func (w *usageWalker) operationField(key string, value *yaml.Node, inherited schemaDirection) {
	switch {
	case utils.IsHttpVerb(key) || key == v3.QueryLabel:
		w.walk(value, operationObject, inherited)
	case key == v3.AdditionalOperationsLabel:
		w.each(value, operationObject, inherited)
	case key == v3.ParametersLabel:
		w.each(value, payloadObject, inherited|requestSchema)
	case key == v3.RequestBodyLabel:
		w.walk(value, payloadObject, inherited|requestSchema)
	case key == v3.ResponsesLabel:
		w.openAPIEntries(value, payloadObject, inherited|responseSchema)
	case key == v3.CallbacksLabel:
		w.each(value, callbackObject, sharedSchema)
	}
}

func (w *usageWalker) schemaField(node *yaml.Node, key string, value *yaml.Node, dir, sameInstance schemaDirection) {
	switch key {
	case v3.AllOfLabel, v3.AnyOfLabel, v3.DependentSchemasLabel:
		w.each(value, schemaObject, sameInstance)
	case v3.ThenLabel, v3.ElseLabel:
		w.walk(value, schemaObject, sameInstance)
	case v3.PropertiesLabel, v3.PatternPropertiesLabel, v3.PrefixItemsLabel:
		w.each(value, schemaObject, dir)
	case v3.ItemsLabel, v3.AdditionalPropertiesLabel, v3.UnevaluatedPropertiesLabel,
		v3.UnevaluatedItemsLabel, v3.PropertyNamesLabel, base.ContentSchemaLabel:
		w.walk(value, schemaObject, dir)
	// Narrowing a predicate can widen the enclosing schema (or change which branch matches).
	case v3.NotLabel, v3.IfLabel, v3.ContainsLabel:
		w.walk(value, schemaObject, sharedSchema)
	case v3.OneOfLabel:
		w.each(value, schemaObject, sharedSchema)
	case v3.DiscriminatorLabel:
		w.discriminator(node, value, sameInstance)
	case v3.SchemaDialectLabel:
		if !knownUsageDialect(value) {
			w.unknown = true
		}
	case v3.DynamicRefLabel, "$recursiveRef", "$id":
		// Resource scopes need more than local pointer reachability.
		w.unknown = true
	}
}

// discriminator follows the schemas its mapping and default mapping name. Next to a oneOf or an anyOf it can only
// select one of the listed schemas, which the walk follows already. Otherwise it is on a parent schema, and can
// select any component schema that builds on the parent with allOf by name.
func (w *usageWalker) discriminator(holder, node *yaml.Node, dir schemaDirection) {
	for key, value := range usageFields(node) {
		switch key {
		case base.MappingLabel:
			for _, target := range usageFields(value) {
				w.discriminatorTarget(target, dir)
			}
		case base.DefaultMappingLabel:
			w.discriminatorTarget(value, dir)
		}
	}
	if w.unknown {
		return
	}
	for key := range usageFields(holder) {
		if key == v3.OneOfLabel || key == v3.AnyOfLabel {
			return
		}
	}
	for _, child := range w.allOfChildren(holder) {
		w.walk(child, schemaObject, dir)
	}
}

// componentName is the pattern of a component name, so a mapping value that does not match it is a relative URI.
var componentName = regexp.MustCompile(`^[a-zA-Z0-9.\-_]+$`)

// discriminatorTarget follows one mapping value, which is a reference or a schema name. A value that is not a schema
// of this document, such as a file or a URL, cannot be followed.
func (w *usageWalker) discriminatorTarget(value *yaml.Node, dir schemaDirection) {
	// The usage is dropped once it is unknown, so the other targets do not need to be resolved.
	if w.unknown {
		return
	}
	ref := utils.NodeAlias(value).Value
	if !strings.HasPrefix(ref, "#/") {
		if !componentName.MatchString(ref) {
			w.unknown = true
			return
		}
		ref = "#/components/schemas/" + ref
	}
	schema := w.localTarget(ref)
	// A pointer can lead to a value that cannot be a schema, such as a string. A boolean schema has nothing to walk.
	if schema == nil || (utils.NodeAlias(schema).Kind != yaml.MappingNode && !utils.IsNodeBoolValue(schema)) {
		w.unknown = true
		return
	}
	w.walk(schema, schemaObject, dir)
}

// allOfChildren returns the component schemas that build on parent with allOf, directly or through other schemas.
func (w *usageWalker) allOfChildren(parent *yaml.Node) []*yaml.Node {
	if w.children == nil {
		w.children = make(map[*yaml.Node][]*yaml.Node)
		for _, schema := range usageFields(w.schemas) {
			schema = utils.NodeAlias(schema)
			ancestors := make(map[*yaml.Node]bool)
			w.ancestors(schema, ancestors)
			for ancestor := range ancestors {
				w.children[ancestor] = append(w.children[ancestor], schema)
			}
		}
	}
	return w.children[parent]
}

// ancestors collects node and the schemas it builds on with allOf. A schema with a $ref builds on the schema it refers
// to as well, so the references are followed too, and an allOf next to a $ref only counts when the model reads it. A
// reference that cannot be followed could lead to a parent, so it makes the usage unknown.
func (w *usageWalker) ancestors(node *yaml.Node, seen map[*yaml.Node]bool) {
	node = utils.NodeAlias(node)
	if w.unknown || node == nil || seen[node] {
		return
	}
	seen[node] = true
	for key, value := range usageFields(node) {
		switch {
		case key == v3.RefLabel:
			target := w.localTarget(utils.NodeAlias(value).Value)
			if target == nil {
				w.unknown = true
				return
			}
			w.ancestors(target, seen)
		case key == v3.AllOfLabel && (w.refSiblings || usageField(node, v3.RefLabel) == nil):
			for _, item := range utils.NodeAlias(value).Content {
				w.ancestors(item, seen)
			}
		}
	}
}

// usageField returns the value of one field of a mapping, or nil when the mapping does not have it.
func usageField(node *yaml.Node, name string) *yaml.Node {
	for key, value := range usageFields(node) {
		if key == name {
			return value
		}
	}
	return nil
}

// Other dialects may introduce applicators that the OpenAPI schema walk cannot follow.
func knownUsageDialect(value *yaml.Node) bool {
	switch strings.TrimSuffix(utils.NodeAlias(value).Value, "#") {
	case utils.SchemaSource, "https://spec.openapis.org/oas/3.1/dialect/base", "https://spec.openapis.org/oas/3.2/dialect/base":
		return true
	}
	return false
}

func (w *usageWalker) localTarget(ref string) *yaml.Node {
	if node := w.targets[ref]; node != nil {
		return node
	}
	if strings.HasPrefix(ref, "#/") {
		if target := w.index.FindComponentInRoot(context.Background(), ref); target != nil {
			w.targets[ref] = target.Node
			return target.Node
		}
	}
	return nil
}
