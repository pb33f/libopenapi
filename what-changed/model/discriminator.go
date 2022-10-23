// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low/base"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/what-changed/core"
)

// DiscriminatorChanges represents changes made to a Discriminator OpenAPI object
type DiscriminatorChanges struct {
	core.PropertyChanges
	MappingChanges []*core.Change
}

// TotalChanges returns a count of everything changed within the Discriminator object
func (d *DiscriminatorChanges) TotalChanges() int {
	l := 0
	if k := d.PropertyChanges.TotalChanges(); k > 0 {
		l += k
	}
	if k := len(d.MappingChanges); k > 0 {
		l += k
	}
	return l
}

// TotalBreakingChanges returns the number of breaking changes made by the Discriminator
func (d *DiscriminatorChanges) TotalBreakingChanges() int {
	return d.PropertyChanges.TotalBreakingChanges() + core.CountBreakingChanges(d.MappingChanges)
}

// CompareDiscriminator will check a left (original) and right (new) Discriminator object for changes
// and will return a pointer to DiscriminatorChanges
func CompareDiscriminator(l, r *base.Discriminator) *DiscriminatorChanges {
	dc := new(DiscriminatorChanges)
	var changes []*core.Change
	var props []*core.PropertyCheck
	var mappingChanges []*core.Change

	// Name (breaking change)
	props = append(props, &core.PropertyCheck{
		LeftNode:  l.PropertyName.ValueNode,
		RightNode: r.PropertyName.ValueNode,
		Label:     v3.PropertyNameLabel,
		Changes:   &changes,
		Breaking:  true,
		Original:  l,
		New:       r,
	})

	// check properties
	core.CheckProperties(props)

	// flatten maps
	lMap := core.FlattenLowLevelMap[string](l.Mapping)
	rMap := core.FlattenLowLevelMap[string](r.Mapping)

	// check for removals, modifications and moves
	for i := range lMap {
		core.CheckForObjectAdditionOrRemoval[string](lMap, rMap, i, &mappingChanges, false, true)
		// if the existing tag exists, let's check it.
		if rMap[i] != nil {
			if lMap[i].Value != rMap[i].Value {
				core.CreateChange(&mappingChanges, core.Modified, i, lMap[i].GetValueNode(),
					rMap[i].GetValueNode(), true, lMap[i].GetValue(), rMap[i].GetValue())
			}
		}
	}

	for i := range rMap {
		if lMap[i] == nil {
			core.CreateChange(&mappingChanges, core.ObjectAdded, i, nil,
				rMap[i].GetValueNode(), false, nil, rMap[i].GetValue())
		}
	}

	dc.Changes = changes
	dc.MappingChanges = mappingChanges
	if dc.TotalChanges() <= 0 {
		return nil
	}
	return dc

}
