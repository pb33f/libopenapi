// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

import v3 "github.com/pb33f/libopenapi/datamodel/low/v3"

type EncodingChanges struct {
	ParameterChanges
	HeaderChanges map[string]*HeaderChanges
}

func (e *EncodingChanges) TotalChanges() int {
	c := e.PropertyChanges.TotalChanges()
	if e.HeaderChanges != nil {
		for i := range e.HeaderChanges {
			c += e.HeaderChanges[i].TotalChanges()
		}
	}
	return c
}

func (e *EncodingChanges) TotalBreakingChanges() int {
	c := e.PropertyChanges.TotalBreakingChanges()
	if e.HeaderChanges != nil {
		for i := range e.HeaderChanges {
			c += e.HeaderChanges[i].TotalBreakingChanges()
		}
	}
	return c
}

func CompareEncoding(l, r *v3.Encoding) *EncodingChanges {

	var changes []*Change
	var props []*PropertyCheck

	// ContentType
	props = append(props, &PropertyCheck{
		LeftNode:  l.ContentType.ValueNode,
		RightNode: r.ContentType.ValueNode,
		Label:     v3.ContentTypeLabel,
		Changes:   &changes,
		Breaking:  true,
		Original:  l,
		New:       r,
	})

	// Explode
	props = append(props, &PropertyCheck{
		LeftNode:  l.Explode.ValueNode,
		RightNode: r.Explode.ValueNode,
		Label:     v3.ExplodeLabel,
		Changes:   &changes,
		Breaking:  true,
		Original:  l,
		New:       r,
	})

	// AllowReserved
	props = append(props, &PropertyCheck{
		LeftNode:  l.AllowReserved.ValueNode,
		RightNode: r.AllowReserved.ValueNode,
		Label:     v3.AllowReservedLabel,
		Changes:   &changes,
		Breaking:  false,
		Original:  l,
		New:       r,
	})

	// check everything.
	CheckProperties(props)
	ec := new(EncodingChanges)

	// headers
	ec.HeaderChanges = CheckMapForChanges(l.Headers.Value, r.Headers.Value, &changes, v3.HeadersLabel, CompareHeadersV3)
	ec.Changes = changes
	if ec.TotalChanges() <= 0 {
		return nil
	}
	return ec
}
