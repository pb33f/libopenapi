// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/datamodel/low/v3"
)

// preservedObjectBranch returns a comparison view for an object moved into a
// composition. Uncertain wrappers retain the normal structural comparison.
func preservedObjectBranch(l, r *base.SchemaProxy, old, wrapper *base.Schema) *base.Schema {
	if l.IsReference() || r.IsReference() || old == nil || wrapper == nil ||
		old.Properties.Value == nil || old.Properties.Value.Len() == 0 || hasObjectComposition(old) {
		return nil
	}
	oldNull, ok := objectOrNullType(old)
	if !ok {
		return nil
	}
	branches := wrapper.AnyOf.Value
	oneOf := schemaNodeHasSingleKey(r.GetValueNode(), v3.OneOfLabel)
	if oneOf {
		branches = wrapper.OneOf.Value
		if len(branches) != 2 {
			return nil
		}
	} else if !schemaNodeHasSingleKey(r.GetValueNode(), v3.AnyOfLabel) {
		return nil
	}

	var selected *base.Schema
	hasNull := false
	for _, entry := range branches {
		proxy := entry.Value
		if proxy == nil || base.CheckSchemaProxyForCircularRefs(proxy) {
			return nil
		}
		branch := proxy.Schema()
		if branch == nil {
			return nil
		}
		// A pure null alternative proves null admission without evaluating a schema.
		if !proxy.IsReference() && schemaNodeHasSingleKey(proxy.GetValueNode(), v3.TypeLabel) &&
			branch.Type.Value.A == "null" {
			hasNull = true
			continue
		}
		branchNull, object := objectOrNullType(branch)
		if object && !branchNull && !hasObjectComposition(branch) {
			if selected != nil {
				return nil
			}
			selected = branch
		} else if oneOf {
			// Only object versus null is proven disjoint here. Other oneOf
			// alternatives can reject values by matching more than once.
			return nil
		}
	}
	if (oldNull || oneOf) && !hasNull {
		return nil
	}
	return selected
}

// objectOrNullType accepts only explicit object types, optionally with null.
// Other unions and OAS 3.0 nullable need a wider proof than this shortcut makes.
func objectOrNullType(schema *base.Schema) (nullable, ok bool) {
	if schema.Nullable.Value || schema.Type.IsEmpty() {
		return false, false
	}
	if schema.Type.Value.IsA() {
		return false, schema.Type.Value.A == "object"
	}
	object := false
	for _, value := range schema.Type.Value.B {
		switch value.Value {
		case "object":
			object = true
		case "null":
			nullable = true
		default:
			return false, false
		}
	}
	return nullable, object
}

func hasObjectComposition(schema *base.Schema) bool {
	return len(schema.OneOf.Value) > 0 || len(schema.AnyOf.Value) > 0 ||
		len(schema.AllOf.Value) > 0 || len(schema.PrefixItems.Value) > 0 || schema.Not.Value != nil
}

// Keep genuine widening visible while suppressing the moved object's removal.
func checkObjectCompositionAlternatives(old, wrapper, selected *base.Schema, changes *[]*Change) {
	branches, label := wrapper.AnyOf.Value, v3.AnyOfLabel
	if len(wrapper.OneOf.Value) > 0 {
		branches, label = wrapper.OneOf.Value, v3.OneOfLabel
	}
	oldNull, _ := objectOrNullType(old)
	oldNull = oldNull && objectConstraintsAdmitNull(old)
	for _, entry := range branches {
		proxy := entry.Value
		branch := proxy.Schema()
		if branch == selected || (oldNull && !proxy.IsReference() &&
			schemaNodeHasSingleKey(proxy.GetValueNode(), v3.TypeLabel) && branch.Type.Value.A == "null") {
			continue
		}
		CreateChange(changes, ObjectAdded, label, nil, proxy.GetValueNode(),
			schemaCompositionChangeBreaking(label, ObjectAdded), nil, proxy)
	}
}

// Object assertions do not restrict null. Other assertions need an explicit
// proof before an added null alternative can be treated as already present.
var nullableObjectKeys = map[string]struct{}{
	v3.TypeLabel: {}, v3.PropertiesLabel: {}, v3.RequiredLabel: {},
	v3.AdditionalPropertiesLabel: {}, v3.PatternPropertiesLabel: {},
	v3.PropertyNamesLabel: {}, v3.MinPropertiesLabel: {}, v3.MaxPropertiesLabel: {},
	v3.DependentSchemasLabel: {}, base.DependentRequiredLabel: {}, v3.UnevaluatedPropertiesLabel: {},
	v3.TitleLabel: {}, v3.DescriptionLabel: {}, v3.DefaultLabel: {},
	v3.ExampleLabel: {}, v3.ExamplesLabel: {}, v3.DeprecatedLabel: {},
	v3.ReadOnlyLabel: {}, v3.WriteOnlyLabel: {}, v3.XMLLabel: {}, v3.ExternalDocsLabel: {},
	v3.EnumLabel: {}, v3.ConstLabel: {},
}

func objectConstraintsAdmitNull(schema *base.Schema) bool {
	if !schemaNodeHasOnlyAllowedKeys(schema.RootNode, nullableObjectKeys) {
		return false
	}
	if schema.Const.Value != nil && schema.Const.Value.Tag != "!!null" {
		return false
	}
	if !schema.Enum.IsEmpty() {
		for _, value := range schema.Enum.Value {
			if value.Value.Tag == "!!null" {
				return true
			}
		}
		return false
	}
	return true
}
