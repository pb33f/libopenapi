// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/v2"
	"github.com/pb33f/libopenapi/datamodel/low/v3"
	"gopkg.in/yaml.v3"
	"reflect"
)

type SecurityRequirementChanges struct {
	PropertyChanges
}

func (s *SecurityRequirementChanges) TotalChanges() int {
	return s.PropertyChanges.TotalChanges()
}

func (s *SecurityRequirementChanges) TotalBreakingChanges() int {
	return s.PropertyChanges.TotalBreakingChanges()
}

func removedSecurityRequirement(vn *yaml.Node, name string, changes *[]*Change) {
	CreateChange(changes, ObjectRemoved, v3.SecurityLabel,
		vn, nil, true, name, nil)
}

func addedSecurityRequirement(vn *yaml.Node, name string, changes *[]*Change) {
	CreateChange(changes, ObjectAdded, v3.SecurityLabel,
		nil, vn, false, nil, name)
}

func CompareSecurityRequirement(l, r any) *SecurityRequirementChanges {

	var changes []*Change
	sc := new(SecurityRequirementChanges)

	if reflect.TypeOf(&v2.SecurityRequirement{}) == reflect.TypeOf(l) &&
		reflect.TypeOf(&v2.SecurityRequirement{}) == reflect.TypeOf(r) {

		lSec := l.(*v2.SecurityRequirement)
		rSec := r.(*v2.SecurityRequirement)

		if low.AreEqual(lSec, rSec) {
			return nil
		}
		checkSecurityRequirement(lSec.Values.Value, rSec.Values.Value, &changes)

	}

	if reflect.TypeOf(&v3.SecurityRequirement{}) == reflect.TypeOf(l) &&
		reflect.TypeOf(&v3.SecurityRequirement{}) == reflect.TypeOf(r) {

		lSec := l.(*v3.SecurityRequirement)
		rSec := r.(*v3.SecurityRequirement)

		if low.AreEqual(lSec, rSec) {
			return nil
		}

		// can we find anyone to dance with?
		findPartner := func(key string,
			search map[low.KeyReference[string]]map[low.KeyReference[string]]low.ValueReference[[]low.ValueReference[string]]) map[low.KeyReference[string]]low.ValueReference[[]low.ValueReference[string]] {
			for k := range search {
				if k.Value == key {
					return search[k]
				}
			}
			return nil
		}

		// Yes, this exists.
		lValues := make(map[low.KeyReference[string]]map[low.KeyReference[string]]low.ValueReference[[]low.ValueReference[string]])
		rValues := make(map[low.KeyReference[string]]map[low.KeyReference[string]]low.ValueReference[[]low.ValueReference[string]])
		for i := range lSec.ValueRequirements {
			for k := range lSec.ValueRequirements[i].Value {
				lValues[k] = lSec.ValueRequirements[i].Value
			}
		}
		for i := range rSec.ValueRequirements {
			for k := range rSec.ValueRequirements[i].Value {
				rValues[k] = rSec.ValueRequirements[i].Value
			}
		}

		// look through left and right slices to see if we recognize anything.
		for k := range lValues {
			if p := findPartner(k.Value, rValues); p != nil {
				checkSecurityRequirement(lValues[k], p, &changes)
				continue
			}
			CreateChange(&changes, ObjectRemoved, v3.SecurityLabel,
				k.KeyNode, nil, true, lValues[k], nil)
		}
		for k := range rValues {
			if ok := findPartner(k.Value, lValues); ok == nil {
				CreateChange(&changes, ObjectAdded, v3.SecurityLabel,
					nil, k.KeyNode, false, nil, rValues[k])
			}
		}
	}

	sc.Changes = changes
	return sc
}

func checkSecurityRequirement(lSec, rSec map[low.KeyReference[string]]low.ValueReference[[]low.ValueReference[string]],
	changes *[]*Change) {

	lKeys := make([]string, len(lSec))
	rKeys := make([]string, len(rSec))
	lValues := make(map[string]low.ValueReference[[]low.ValueReference[string]])
	rValues := make(map[string]low.ValueReference[[]low.ValueReference[string]])
	var n, z int
	for i := range lSec {
		lKeys[n] = i.Value
		lValues[i.Value] = lSec[i]
		n++
	}
	for i := range rSec {
		rKeys[z] = i.Value
		rValues[i.Value] = rSec[i]
		z++
	}

	for z = range lKeys {
		if z < len(rKeys) {
			if _, ok := rValues[lKeys[z]]; !ok {
				removedSecurityRequirement(lValues[lKeys[z]].ValueNode, lKeys[z], changes)
				continue
			}

			lValue := lValues[lKeys[z]].Value
			rValue := rValues[lKeys[z]].Value

			// check if actual values match up
			lRoleKeys := make([]string, len(lValue))
			rRoleKeys := make([]string, len(rValue))
			lRoleValues := make(map[string]low.ValueReference[string])
			rRoleValues := make(map[string]low.ValueReference[string])
			var t, k int
			for i := range lValue {
				lRoleKeys[t] = lValue[i].Value
				lRoleValues[lValue[i].Value] = lValue[i]
				t++
			}
			for i := range rValue {
				rRoleKeys[k] = rValue[i].Value
				rRoleValues[rValue[i].Value] = rValue[i]
				k++
			}

			for t = range lRoleKeys {
				if t < len(rRoleKeys) {
					if _, ok := rRoleValues[lRoleKeys[t]]; !ok {
						removedSecurityRequirement(lRoleValues[lRoleKeys[t]].ValueNode, lRoleKeys[t], changes)
						continue
					}
				}
				if t >= len(rRoleKeys) {
					if _, ok := rRoleValues[lRoleKeys[t]]; !ok {
						removedSecurityRequirement(lRoleValues[lRoleKeys[t]].ValueNode, lRoleKeys[t], changes)
					}
				}
			}
			for t = range rRoleKeys {
				if t < len(lRoleKeys) {
					if _, ok := lRoleValues[rRoleKeys[t]]; !ok {
						addedSecurityRequirement(rRoleValues[rRoleKeys[t]].ValueNode, rRoleKeys[t], changes)
						continue
					}
				}
				if t >= len(lRoleKeys) {
					addedSecurityRequirement(rRoleValues[rRoleKeys[t]].ValueNode, rRoleKeys[t], changes)
				}
			}

		}
		if z >= len(rKeys) {
			if _, ok := rValues[lKeys[z]]; !ok {
				removedSecurityRequirement(lValues[lKeys[z]].ValueNode, lKeys[z], changes)
			}
		}
	}
	for z = range rKeys {
		if z < len(lKeys) {
			if _, ok := lValues[rKeys[z]]; !ok {
				addedSecurityRequirement(rValues[rKeys[z]].ValueNode, rKeys[z], changes)
				continue
			}
		}
		if z >= len(lKeys) {
			if _, ok := lValues[rKeys[z]]; !ok {
				addedSecurityRequirement(rValues[rKeys[z]].ValueNode, rKeys[z], changes)
			}
		}
	}
}

func CompareSecurityRequirementV3(l, r *v3.SecurityRequirement) *SecurityRequirementChanges {
	return CompareSecurityRequirement(l, r)
}

func CompareSecurityRequirementV2(l, r *v2.SecurityRequirement) *SecurityRequirementChanges {
	return CompareSecurityRequirement(l, r)
}
