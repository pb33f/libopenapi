// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/orderedmap"
)

// EncodingChanges represent all the changes made to an Encoding object
type EncodingChanges struct {
	*PropertyChanges
	HeaderChanges         map[string]*HeaderChanges   `json:"headers,omitempty" yaml:"headers,omitempty"`
	EncodingChanges       map[string]*EncodingChanges `json:"encoding,omitempty" yaml:"encoding,omitempty"`
	PrefixEncodingChanges []*EncodingChanges          `json:"prefixEncoding,omitempty" yaml:"prefixEncoding,omitempty"`
	ItemEncodingChanges   *EncodingChanges            `json:"itemEncoding,omitempty" yaml:"itemEncoding,omitempty"`
}

// GetAllChanges returns a slice of all changes made between Encoding objects
func (e *EncodingChanges) GetAllChanges() []*Change {
	if e == nil {
		return nil
	}
	var changes []*Change
	changes = append(changes, e.Changes...)
	for k := range e.HeaderChanges {
		changes = append(changes, e.HeaderChanges[k].GetAllChanges()...)
	}
	for k := range e.EncodingChanges {
		changes = append(changes, e.EncodingChanges[k].GetAllChanges()...)
	}
	for k := range e.PrefixEncodingChanges {
		changes = append(changes, e.PrefixEncodingChanges[k].GetAllChanges()...)
	}
	changes = append(changes, e.ItemEncodingChanges.GetAllChanges()...)
	return changes
}

// TotalChanges returns the total number of changes made between two Encoding objects
func (e *EncodingChanges) TotalChanges() int {
	if e == nil {
		return 0
	}
	c := e.PropertyChanges.TotalChanges()
	if e.HeaderChanges != nil {
		for i := range e.HeaderChanges {
			c += e.HeaderChanges[i].TotalChanges()
		}
	}
	for k := range e.EncodingChanges {
		c += e.EncodingChanges[k].TotalChanges()
	}
	for k := range e.PrefixEncodingChanges {
		c += e.PrefixEncodingChanges[k].TotalChanges()
	}
	c += e.ItemEncodingChanges.TotalChanges()
	return c
}

// TotalBreakingChanges returns the number of changes made between two Encoding objects that were breaking.
func (e *EncodingChanges) TotalBreakingChanges() int {
	if e == nil {
		return 0
	}
	c := e.PropertyChanges.TotalBreakingChanges()
	if e.HeaderChanges != nil {
		for i := range e.HeaderChanges {
			c += e.HeaderChanges[i].TotalBreakingChanges()
		}
	}
	for k := range e.EncodingChanges {
		c += e.EncodingChanges[k].TotalBreakingChanges()
	}
	for k := range e.PrefixEncodingChanges {
		c += e.PrefixEncodingChanges[k].TotalBreakingChanges()
	}
	c += e.ItemEncodingChanges.TotalBreakingChanges()
	return c
}

// CompareEncoding returns a pointer to *EncodingChanges that contain all changes made between a left and right
// set of Encoding objects.
func CompareEncoding(l, r *v3.Encoding) *EncodingChanges {
	var changes []*Change
	props := make([]*PropertyCheck, 0, 4)

	props = append(props,
		NewPropertyCheck(CompEncoding, PropContentType,
			l.ContentType.ValueNode, r.ContentType.ValueNode,
			v3.ContentTypeLabel, &changes, l, r),
		NewPropertyCheck(CompEncoding, PropStyle,
			l.Style.ValueNode, r.Style.ValueNode,
			v3.StyleLabel, &changes, l, r),
		NewPropertyCheck(CompEncoding, PropExplode,
			l.Explode.ValueNode, r.Explode.ValueNode,
			v3.ExplodeLabel, &changes, l, r),
		NewPropertyCheck(CompEncoding, PropAllowReserved,
			l.AllowReserved.ValueNode, r.AllowReserved.ValueNode,
			v3.AllowReservedLabel, &changes, l, r),
	)

	// check everything.
	CheckProperties(props)
	ec := new(EncodingChanges)

	// headers
	ec.HeaderChanges = CheckMapForChanges(l.Headers.Value, r.Headers.Value, &changes, v3.HeadersLabel, CompareHeadersV3)

	// nested encoding, prefixEncoding, itemEncoding (OpenAPI 3.2+)
	ec.EncodingChanges, ec.PrefixEncodingChanges, ec.ItemEncodingChanges = compareEncodingFields(
		CompEncoding, l.Encoding.Value, r.Encoding.Value, l.PrefixEncoding, r.PrefixEncoding,
		l.ItemEncoding, r.ItemEncoding, &changes, true)
	ec.PropertyChanges = NewPropertyChanges(changes)
	if ec.TotalChanges() <= 0 {
		return nil
	}
	return ec
}

// compareEncodingFields compares the encoding, prefixEncoding and itemEncoding fields shared by the Media Type
// Object and the Encoding Object. prefixEncoding entries are compared by position, because each one applies to the
// part at its position. The encoding map uses the component's breaking rules only when withEncodingRule is set.
// The Media Type Object's encoding map predates the rules system and keeps its hardcoded behavior.
func compareEncodingFields(component string,
	lEnc, rEnc *orderedmap.Map[low.KeyReference[string], low.ValueReference[*v3.Encoding]],
	lPrefix, rPrefix low.NodeReference[[]low.ValueReference[*v3.Encoding]],
	lItem, rItem low.NodeReference[*v3.Encoding],
	changes *[]*Change, withEncodingRule bool,
) (map[string]*EncodingChanges, []*EncodingChanges, *EncodingChanges) {
	var encChanges map[string]*EncodingChanges
	if withEncodingRule {
		encChanges = CheckMapForChangesWithRules(lEnc, rEnc, changes, v3.EncodingLabel, CompareEncoding,
			component, PropEncoding)
	} else {
		encChanges = CheckMapForChanges(lEnc, rEnc, changes, v3.EncodingLabel, CompareEncoding)
	}

	var prefixChanges []*EncodingChanges
	lp, rp := lPrefix.Value, rPrefix.Value
	for i := 0; i < len(lp) || i < len(rp); i++ {
		switch {
		case i >= len(rp):
			CreateChange(changes, ObjectRemoved, v3.PrefixEncodingLabel, lp[i].ValueNode, nil,
				BreakingRemoved(component, PropPrefixEncoding), lp[i].Value, nil)
		case i >= len(lp):
			CreateChange(changes, ObjectAdded, v3.PrefixEncodingLabel, nil, rp[i].ValueNode,
				BreakingAdded(component, PropPrefixEncoding), nil, rp[i].Value)
		default:
			if ch := CompareEncoding(lp[i].Value, rp[i].Value); ch != nil {
				prefixChanges = append(prefixChanges, ch)
			}
		}
	}

	var itemChanges *EncodingChanges
	switch {
	case !lItem.IsEmpty() && !rItem.IsEmpty():
		itemChanges = CompareEncoding(lItem.Value, rItem.Value)
	case !lItem.IsEmpty():
		CreateChange(changes, ObjectRemoved, v3.ItemEncodingLabel, lItem.ValueNode, nil,
			BreakingRemoved(component, PropItemEncoding), lItem.Value, nil)
	case !rItem.IsEmpty():
		CreateChange(changes, ObjectAdded, v3.ItemEncodingLabel, nil, rItem.ValueNode,
			BreakingAdded(component, PropItemEncoding), nil, rItem.Value)
	}
	return encChanges, prefixChanges, itemChanges
}
