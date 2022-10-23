// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/v2"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/what-changed/core"
	"gopkg.in/yaml.v3"
)

type SecurityRequirementChanges struct {
	core.PropertyChanges
}

func (s *SecurityRequirementChanges) TotalChanges() int {
	return s.PropertyChanges.TotalChanges()
}

func (s *SecurityRequirementChanges) TotalBreakingChanges() int {
	return s.PropertyChanges.TotalBreakingChanges()
}

func CompareSecurityRequirement(l, r *v2.SecurityRequirement) *SecurityRequirementChanges {
	if low.AreEqual(l, r) {
		return nil
	}
	var changes []*core.Change
	lKeys := make([]string, len(l.Values.Value))
	rKeys := make([]string, len(r.Values.Value))
	lValues := make(map[string]low.ValueReference[[]low.ValueReference[string]])
	rValues := make(map[string]low.ValueReference[[]low.ValueReference[string]])
	var n, z int
	for i := range l.Values.Value {
		lKeys[n] = i.Value
		lValues[i.Value] = l.Values.Value[i]
		n++
	}
	for i := range r.Values.Value {
		rKeys[z] = i.Value
		rValues[i.Value] = r.Values.Value[i]
		z++
	}
	removed := func(z int, vn *yaml.Node, name string) {
		core.CreateChange(&changes, core.ObjectRemoved, v3.SecurityDefinitionLabel,
			vn, nil, true, name, nil)
	}

	added := func(z int, vn *yaml.Node, name string) {
		core.CreateChange(&changes, core.ObjectAdded, v3.SecurityDefinitionLabel,
			nil, vn, false, nil, name)
	}

	for z = range lKeys {
		if z < len(rKeys) {
			if _, ok := rValues[lKeys[z]]; !ok {
				removed(z, lValues[lKeys[z]].ValueNode, lKeys[z])
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
						removed(t, lRoleValues[lRoleKeys[t]].ValueNode, lRoleKeys[t])
						continue
					}
				}
				if t >= len(rRoleKeys) {
					if _, ok := rRoleValues[lRoleKeys[t]]; !ok {
						removed(t, lRoleValues[lRoleKeys[t]].ValueNode, lRoleKeys[t])
					}
				}
			}
			for t = range rRoleKeys {
				if t < len(lRoleKeys) {
					if _, ok := lRoleValues[rRoleKeys[t]]; !ok {
						added(t, rRoleValues[rRoleKeys[t]].ValueNode, rRoleKeys[t])
						continue
					}
				}
				if t >= len(lRoleKeys) {
					added(t, rRoleValues[rRoleKeys[t]].ValueNode, rRoleKeys[t])
				}
			}

		}
		if z >= len(rKeys) {
			if _, ok := rValues[lKeys[z]]; !ok {
				removed(z, lValues[lKeys[z]].ValueNode, lKeys[z])
			}
		}
	}
	for z = range rKeys {
		if z < len(lKeys) {
			if _, ok := lValues[rKeys[z]]; !ok {
				added(z, rValues[rKeys[z]].ValueNode, rKeys[z])
				continue
			}
		}
		if z >= len(lKeys) {
			if _, ok := lValues[rKeys[z]]; !ok {
				added(z, rValues[rKeys[z]].ValueNode, rKeys[z])
			}
		}
	}

	sc := new(SecurityRequirementChanges)
	sc.Changes = changes
	if sc.TotalChanges() <= 0 {
		return nil
	}
	return sc
}
