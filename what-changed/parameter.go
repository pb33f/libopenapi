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
)

type ParameterChanges struct {
	PropertyChanges
	SchemaChanges    *SchemaChanges
	ExtensionChanges *ExtensionChanges

	// V2 change types
	// ItemsChanges

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

		// todo: items
		// todo: default
		// todo: enums

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
		if lParam.Example.Value != nil && rParam.Example.Value != nil {
			if low.GenerateHashString(lParam.Example.Value) != low.GenerateHashString(rParam.Example.Value) {
				CreateChange(&changes, Modified, v3.ExampleLabel,
					lParam.Example.GetValueNode(), rParam.Example.GetValueNode(), false,
					lParam.Example.GetValue(), rParam.Example.GetValue())
			}
		}
		if lParam.Example.Value == nil && rParam.Example.Value != nil {
			CreateChange(&changes, PropertyAdded, v3.ExampleLabel,
				nil, rParam.Example.GetValueNode(), false,
				nil, rParam.Example.GetValue())

		}
		if lParam.Example.Value != nil && rParam.Example.Value == nil {
			CreateChange(&changes, PropertyRemoved, v3.ExampleLabel,
				lParam.Example.GetValueNode(), nil, false,
				lParam.Example.GetValue(), nil)

		}

		// examples
		checkParameterExamples(lParam, rParam, changes, pc)

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

func checkParameterExamples(lParam *v3.Parameter, rParam *v3.Parameter, changes []*Change, pc *ParameterChanges) {
	lExpHashes := make(map[string]string)
	rExpHashes := make(map[string]string)
	lExpValues := make(map[string]low.ValueReference[*base.Example])
	rExpValues := make(map[string]low.ValueReference[*base.Example])
	if lParam != nil && lParam.Examples.Value != nil {
		for k := range lParam.Examples.Value {
			lExpHashes[k.Value] = fmt.Sprintf("%x", lParam.Examples.Value[k].Value.Hash())
			lExpValues[k.Value] = lParam.Examples.Value[k]
		}
	}
	if rParam != nil && rParam.Examples.Value != nil {
		for k := range rParam.Examples.Value {
			rExpHashes[k.Value] = fmt.Sprintf("%x", rParam.Examples.Value[k].Value.Hash())
			rExpValues[k.Value] = rParam.Examples.Value[k]
		}
	}
	expChanges := make(map[string]*ExampleChanges)

	// check left example hashes
	for k := range lExpHashes {
		rhash := rExpHashes[k]
		if rhash == "" {
			CreateChange(&changes, ObjectRemoved, v3.ExamplesLabel,
				lExpValues[k].GetValueNode(), nil, false,
				lExpValues[k].GetValue(), nil)
			continue
		}
		if lExpHashes[k] == rExpHashes[k] {
			continue
		}
		expChanges[k] = CompareExamples(lExpValues[k].Value, rExpValues[k].Value)
	}

	//check right example hashes
	for k := range rExpHashes {
		lhash := lExpHashes[k]
		if lhash == "" {
			CreateChange(&changes, ObjectAdded, v3.ExamplesLabel,
				nil, lExpValues[k].GetValueNode(), false,
				nil, lExpValues[k].GetValue())
			continue
		}
	}

	if len(expChanges) > 0 {
		pc.ExampleChanges = expChanges
	}
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
