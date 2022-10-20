// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

import (
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	v2 "github.com/pb33f/libopenapi/datamodel/low/v2"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"gopkg.in/yaml.v3"
	"reflect"
	"sort"
	"strings"
)

type ParameterChanges struct {
	PropertyChanges
	SchemaChanges    *SchemaChanges
	ExtensionChanges *ExtensionChanges

	// V2 change types
	ItemsChanges *ItemsChanges

	// v3 change types
	ExampleChanges map[string]*ExampleChanges
}

// TotalChanges returns a count of everything that changed
func (p *ParameterChanges) TotalChanges() int {
	c := p.PropertyChanges.TotalChanges()
	if p.SchemaChanges != nil {
		c += p.SchemaChanges.TotalChanges()
	}
	if len(p.ExampleChanges) > 0 {
		for i := range p.ExampleChanges {
			c += p.ExampleChanges[i].TotalChanges()
		}
	}
	if p.ExtensionChanges != nil {
		c += p.ExtensionChanges.TotalChanges()
	}

	return c
}

// TotalBreakingChanges always returns 0 for ExternalDoc objects, they are non-binding.
func (p *ParameterChanges) TotalBreakingChanges() int {
	c := p.PropertyChanges.TotalBreakingChanges()
	if p.SchemaChanges != nil {
		c += p.SchemaChanges.TotalBreakingChanges()
	}
	return c
}

func addPropertyCheck(props *[]*PropertyCheck,
	lvn, rvn *yaml.Node, lv, rv any, changes *[]*Change, label string, breaking bool) {
	*props = append(*props, &PropertyCheck{
		LeftNode:  lvn,
		RightNode: rvn,
		Label:     label,
		Changes:   changes,
		Breaking:  breaking,
		Original:  lv,
		New:       rv,
	})
}

func addOpenAPIParameterProperties(left, right low.IsParameter, changes *[]*Change) []*PropertyCheck {
	var props []*PropertyCheck

	// style
	addPropertyCheck(&props, left.GetStyle().ValueNode, right.GetStyle().ValueNode,
		left.GetStyle(), right.GetStyle(), changes, v3.StyleLabel, false)

	// allow reserved
	addPropertyCheck(&props, left.GetAllowReserved().ValueNode, right.GetAllowReserved().ValueNode,
		left.GetAllowReserved(), right.GetAllowReserved(), changes, v3.AllowReservedLabel, true)

	// explode
	addPropertyCheck(&props, left.GetExplode().ValueNode, right.GetExplode().ValueNode,
		left.GetExplode(), right.GetExplode(), changes, v3.ExplodeLabel, false)

	// deprecated
	addPropertyCheck(&props, left.GetDeprecated().ValueNode, right.GetDeprecated().ValueNode,
		left.GetDeprecated(), right.GetDeprecated(), changes, v3.DeprecatedLabel, false)

	// example
	addPropertyCheck(&props, left.GetExample().ValueNode, right.GetExample().ValueNode,
		left.GetExample(), right.GetExample(), changes, v3.ExampleLabel, false)

	return props
}

func addSwaggerParameterProperties(left, right low.IsParameter, changes *[]*Change) []*PropertyCheck {
	var props []*PropertyCheck

	// type
	addPropertyCheck(&props, left.GetType().ValueNode, right.GetType().ValueNode,
		left.GetType(), right.GetType(), changes, v3.TypeLabel, true)

	// format
	addPropertyCheck(&props, left.GetFormat().ValueNode, right.GetFormat().ValueNode,
		left.GetFormat(), right.GetFormat(), changes, v3.FormatLabel, true)

	// collection format
	addPropertyCheck(&props, left.GetCollectionFormat().ValueNode, right.GetCollectionFormat().ValueNode,
		left.GetCollectionFormat(), right.GetCollectionFormat(), changes, v3.CollectionFormatLabel, true)

	// maximum
	addPropertyCheck(&props, left.GetMaximum().ValueNode, right.GetMaximum().ValueNode,
		left.GetMaximum(), right.GetMaximum(), changes, v3.MaximumLabel, true)

	// minimum
	addPropertyCheck(&props, left.GetMinimum().ValueNode, right.GetMinimum().ValueNode,
		left.GetMinimum(), right.GetMinimum(), changes, v3.MinimumLabel, true)

	// exclusive maximum
	addPropertyCheck(&props, left.GetExclusiveMaximum().ValueNode, right.GetExclusiveMaximum().ValueNode,
		left.GetExclusiveMaximum(), right.GetExclusiveMaximum(), changes, v3.ExclusiveMaximumLabel, true)

	// exclusive minimum
	addPropertyCheck(&props, left.GetExclusiveMinimum().ValueNode, right.GetExclusiveMinimum().ValueNode,
		left.GetExclusiveMinimum(), right.GetExclusiveMinimum(), changes, v3.ExclusiveMinimumLabel, true)

	// max length
	addPropertyCheck(&props, left.GetMaxLength().ValueNode, right.GetMaxLength().ValueNode,
		left.GetMaxLength(), right.GetMaxLength(), changes, v3.MaxLengthLabel, true)

	// min length
	addPropertyCheck(&props, left.GetMinLength().ValueNode, right.GetMinLength().ValueNode,
		left.GetMinLength(), right.GetMinLength(), changes, v3.MinLengthLabel, true)

	// pattern
	addPropertyCheck(&props, left.GetPattern().ValueNode, right.GetPattern().ValueNode,
		left.GetPattern(), right.GetPattern(), changes, v3.PatternLabel, true)

	// max items
	addPropertyCheck(&props, left.GetMaxItems().ValueNode, right.GetMaxItems().ValueNode,
		left.GetMaxItems(), right.GetMaxItems(), changes, v3.MaxItemsLabel, true)

	// min items
	addPropertyCheck(&props, left.GetMinItems().ValueNode, right.GetMinItems().ValueNode,
		left.GetMinItems(), right.GetMinItems(), changes, v3.MinItemsLabel, true)

	// unique items
	addPropertyCheck(&props, left.GetUniqueItems().ValueNode, right.GetUniqueItems().ValueNode,
		left.GetUniqueItems(), right.GetUniqueItems(), changes, v3.UniqueItemsLabel, true)

	// multiple of
	addPropertyCheck(&props, left.GetMultipleOf().ValueNode, right.GetMultipleOf().ValueNode,
		left.GetMultipleOf(), right.GetMultipleOf(), changes, v3.MultipleOfLabel, true)

	return props
}

func addCommonParameterProperties(left, right low.IsParameter, changes *[]*Change) []*PropertyCheck {
	var props []*PropertyCheck

	addPropertyCheck(&props, left.GetName().ValueNode, right.GetName().ValueNode,
		left.GetName(), right.GetName(), changes, v3.NameLabel, true)

	// in
	addPropertyCheck(&props, left.GetIn().ValueNode, right.GetIn().ValueNode,
		left.GetIn(), right.GetIn(), changes, v3.InLabel, true)

	// description
	addPropertyCheck(&props, left.GetDescription().ValueNode, right.GetDescription().ValueNode,
		left.GetDescription(), right.GetDescription(), changes, v3.DescriptionLabel, false)

	// required
	addPropertyCheck(&props, left.GetRequired().ValueNode, right.GetRequired().ValueNode,
		left.GetRequired(), right.GetRequired(), changes, v3.RequiredLabel, true)

	// allow empty value
	addPropertyCheck(&props, left.GetAllowEmptyValue().ValueNode, right.GetAllowEmptyValue().ValueNode,
		left.GetAllowEmptyValue(), right.GetAllowEmptyValue(), changes, v3.AllowEmptyValueLabel, true)

	return props
}

func CompareParameters(l, r any) *ParameterChanges {

	var changes []*Change
	var props []*PropertyCheck

	pc := new(ParameterChanges)
	var lSchema *base.SchemaProxy
	var rSchema *base.SchemaProxy
	var lext, rext map[low.KeyReference[string]]low.ValueReference[any]

	if reflect.TypeOf(&v2.Parameter{}) == reflect.TypeOf(l) && reflect.TypeOf(&v2.Parameter{}) == reflect.TypeOf(r) {
		lParam := l.(*v2.Parameter)
		rParam := r.(*v2.Parameter)

		// perform hash check to avoid further processing
		if low.AreEqual(lParam, rParam) {
			return nil
		}

		props = append(props, addSwaggerParameterProperties(lParam, rParam, &changes)...)
		props = append(props, addCommonParameterProperties(lParam, rParam, &changes)...)

		// extract schema
		if lParam != nil {
			lSchema = lParam.Schema.Value
			lext = lParam.Extensions
		}
		if rParam != nil {
			rext = rParam.Extensions
			rSchema = rParam.Schema.Value
		}

		// items
		if !lParam.Items.IsEmpty() && !rParam.Items.IsEmpty() {
			if lParam.Items.Value.Hash() != rParam.Items.Value.Hash() {
				pc.ItemsChanges = CompareItems(lParam.Items.Value, rParam.Items.Value)
			}
		}
		if lParam.Items.IsEmpty() && !rParam.Items.IsEmpty() {
			CreateChange(&changes, ObjectAdded, v3.ItemsLabel,
				nil, rParam.Items.ValueNode, true, nil,
				rParam.Items.Value)
		}
		if !lParam.Items.IsEmpty() && rParam.Items.IsEmpty() {
			CreateChange(&changes, ObjectRemoved, v3.ItemsLabel,
				lParam.Items.ValueNode, nil, true, lParam.Items.Value,
				nil)
		}

		// default
		if !lParam.Default.IsEmpty() && !rParam.Default.IsEmpty() {
			if low.GenerateHashString(lParam.Default.Value) != low.GenerateHashString(lParam.Default.Value) {
				CreateChange(&changes, Modified, v3.DefaultLabel,
					lParam.Items.ValueNode, rParam.Items.ValueNode, true, lParam.Items.Value,
					rParam.Items.ValueNode)
			}
		}
		if lParam.Default.IsEmpty() && !rParam.Default.IsEmpty() {
			CreateChange(&changes, ObjectAdded, v3.DefaultLabel,
				nil, rParam.Default.ValueNode, true, nil,
				rParam.Default.Value)
		}
		if !lParam.Default.IsEmpty() && rParam.Items.IsEmpty() {
			CreateChange(&changes, ObjectRemoved, v3.ItemsLabel,
				lParam.Items.ValueNode, nil, true, lParam.Items.Value,
				nil)
		}

		// enum
		if len(lParam.Enum.Value) > 0 || len(rParam.Enum.Value) > 0 {
			ExtractStringValueSliceChanges(lParam.Enum.Value, rParam.Enum.Value, &changes, v3.EnumLabel)
		}
	}

	// OpenAPI
	if reflect.TypeOf(&v3.Parameter{}) == reflect.TypeOf(l) && reflect.TypeOf(&v3.Parameter{}) == reflect.TypeOf(r) {

		lParam := l.(*v3.Parameter)
		rParam := r.(*v3.Parameter)

		// perform hash check to avoid further processing
		if low.AreEqual(lParam, rParam) {
			return nil
		}

		props = append(props, addOpenAPIParameterProperties(lParam, rParam, &changes)...)
		props = append(props, addCommonParameterProperties(lParam, rParam, &changes)...)
		if lParam != nil {
			lext = lParam.Extensions
			lSchema = lParam.Schema.Value
		}
		if rParam != nil {
			rext = rParam.Extensions
			rSchema = rParam.Schema.Value
		}

		// example
		checkParameterExample(lParam.Example, rParam.Example, changes)

		// examples
		CheckMapForChanges(lParam.Examples.Value, rParam.Examples.Value, &changes, v3.ExamplesLabel, CompareExamples)

		// todo: content

	}
	CheckProperties(props)

	if lSchema != nil && rSchema != nil {
		pc.SchemaChanges = CompareSchemas(lSchema, rSchema)
	}
	if lSchema != nil && rSchema == nil {
		CreateChange(&changes, ObjectRemoved, v3.SchemaLabel,
			lSchema.GetValueNode(), nil, true, lSchema,
			nil)
	}

	if lSchema == nil && rSchema != nil {
		CreateChange(&changes, ObjectAdded, v3.SchemaLabel,
			nil, rSchema.GetValueNode(), true, nil,
			rSchema)
	}

	pc.Changes = changes
	pc.ExtensionChanges = CompareExtensions(lext, rext)

	if pc.TotalChanges() > 0 {
		return pc
	}
	return nil
}

func ExtractStringValueSliceChanges(lParam, rParam []low.ValueReference[string], changes *[]*Change, label string) {
	lKeys := make([]string, len(lParam))
	rKeys := make([]string, len(rParam))
	lValues := make(map[string]low.ValueReference[string])
	rValues := make(map[string]low.ValueReference[string])
	for i := range lParam {
		lKeys[i] = strings.ToLower(lParam[i].Value)
		lValues[lKeys[i]] = lParam[i]
	}
	for i := range rParam {
		rKeys[i] = strings.ToLower(rParam[i].Value)
		rValues[lKeys[i]] = rParam[i]
	}
	sort.Strings(lKeys)
	sort.Strings(rKeys)

	for i := range lKeys {
		if i < len(rKeys) {
			if lKeys[i] != rKeys[i] {
				CreateChange(changes, Modified, label,
					lValues[lKeys[i]].ValueNode,
					rValues[rKeys[i]].ValueNode,
					true,
					lValues[lKeys[i]].Value,
					rValues[rKeys[i]].ValueNode)
			}
			continue
		}
		if i >= len(rKeys) {
			CreateChange(changes, PropertyRemoved, label,
				lValues[lKeys[i]].ValueNode,
				nil,
				true,
				lValues[lKeys[i]].Value,
				nil)
		}
	}
	for i := range rKeys {
		if i >= len(lKeys) {
			CreateChange(changes, PropertyAdded, label,
				nil,
				rValues[rKeys[i]].ValueNode,
				false,
				nil,
				rValues[rKeys[i]].ValueNode)
		}
	}
}

func checkParameterExample(expLeft, expRight low.NodeReference[any], changes []*Change) {
	if !expLeft.IsEmpty() && !expRight.IsEmpty() {
		if low.GenerateHashString(expLeft.GetValue()) != low.GenerateHashString(expRight.GetValue()) {
			CreateChange(&changes, Modified, v3.ExampleLabel,
				expLeft.GetValueNode(), expRight.GetValueNode(), false,
				expLeft.GetValue(), expRight.GetValue())
		}
	}
	if expLeft.Value == nil && expRight.Value != nil {
		CreateChange(&changes, PropertyAdded, v3.ExampleLabel,
			nil, expRight.GetValueNode(), false,
			nil, expRight.GetValue())

	}
	if expLeft.Value != nil && expRight.Value == nil {
		CreateChange(&changes, PropertyRemoved, v3.ExampleLabel,
			expLeft.GetValueNode(), nil, false,
			expLeft.GetValue(), nil)

	}
}

func CheckMapForChanges[T any, R any](expLeft, expRight map[low.KeyReference[string]]low.ValueReference[T],
	changes *[]*Change, label string, compareFunc func(l, r T) R) map[string]R {

	lHashes := make(map[string]string)
	rHashes := make(map[string]string)
	lValues := make(map[string]low.ValueReference[T])
	rValues := make(map[string]low.ValueReference[T])

	for k := range expLeft {
		lHashes[k.Value] = fmt.Sprintf("%x", low.GenerateHashString(expLeft[k].Value))
		lValues[k.Value] = expLeft[k]
	}

	for k := range expRight {
		rHashes[k.Value] = fmt.Sprintf("%x", low.GenerateHashString(expRight[k].Value))
		rValues[k.Value] = expRight[k]
	}

	expChanges := make(map[string]R)

	// check left example hashes
	for k := range lHashes {
		rhash := rHashes[k]
		if rhash == "" {
			CreateChange(changes, ObjectRemoved, label,
				lValues[k].GetValueNode(), nil, false,
				lValues[k].GetValue(), nil)
			continue
		}
		if lHashes[k] == rHashes[k] {
			continue
		}
		// run comparison.
		expChanges[k] = compareFunc(lValues[k].Value, rValues[k].Value)
	}

	//check right example hashes
	for k := range rHashes {
		lhash := lHashes[k]
		if lhash == "" {
			CreateChange(changes, ObjectAdded, v3.ExamplesLabel,
				nil, lValues[k].GetValueNode(), false,
				nil, lValues[k].GetValue())
			continue
		}
	}

	return expChanges
}

func checkParameterContent(lParam *v3.Parameter, rParam *v3.Parameter, changes []*Change, pc *ParameterChanges) {
	lConHashes := make(map[string]string)
	rConHashes := make(map[string]string)
	lConValues := make(map[string]low.ValueReference[*v3.MediaType])
	rConValues := make(map[string]low.ValueReference[*v3.MediaType])
	if lParam != nil && lParam.Content.Value != nil {
		for k := range lParam.Content.Value {
			lConHashes[k.Value] = fmt.Sprintf("%x", lParam.Content.Value[k].Value.Hash())
			lConValues[k.Value] = lParam.Content.Value[k]
		}
	}
	if rParam != nil && rParam.Content.Value != nil {
		for k := range rParam.Content.Value {
			rConHashes[k.Value] = fmt.Sprintf("%x", rParam.Content.Value[k].Value.Hash())
			rConValues[k.Value] = rParam.Content.Value[k]
		}
	}
	expChanges := make(map[string]*ExampleChanges)

	// check left example hashes
	for k := range lConHashes {
		rhash := rConHashes[k]
		if rhash == "" {
			CreateChange(&changes, ObjectRemoved, v3.ExamplesLabel,
				lConValues[k].GetValueNode(), nil, false,
				lConValues[k].GetValue(), nil)
			continue
		}
		if lConHashes[k] == rConHashes[k] {
			continue
		}

		// Compare media types.
		//expChanges[k] = CompareM(lConValues[k].Value, rConValues[k].Value)
		// todo: start here <--------

	}

	//check right example hashes
	for k := range rConHashes {
		lhash := lConHashes[k]
		if lhash == "" {
			CreateChange(&changes, ObjectAdded, v3.ExamplesLabel,
				nil, lConValues[k].GetValueNode(), false,
				nil, lConValues[k].GetValue())
			continue
		}
	}

	if len(expChanges) > 0 {
		pc.ExampleChanges = expChanges
	}
}
