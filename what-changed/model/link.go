// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
)

type LinkChanges struct {
	PropertyChanges
	ExtensionChanges *ExtensionChanges
	ServerChanges    *ServerChanges
}

func (l *LinkChanges) TotalChanges() int {
	c := l.PropertyChanges.TotalChanges()
	if l.ExtensionChanges != nil {
		c += l.ExtensionChanges.TotalChanges()
	}
	if l.ServerChanges != nil {
		c += l.ServerChanges.TotalChanges()
	}
	return c
}

func (l *LinkChanges) TotalBreakingChanges() int {
	c := l.PropertyChanges.TotalBreakingChanges()
	if l.ServerChanges != nil {
		c += l.ServerChanges.TotalBreakingChanges()
	}
	return c
}

func CompareLinks(l, r *v3.Link) *LinkChanges {
	if low.AreEqual(l, r) {
		return nil
	}

	var props []*PropertyCheck
	var changes []*Change

	// operation ref
	props = append(props, &PropertyCheck{
		LeftNode:  l.OperationRef.ValueNode,
		RightNode: r.OperationRef.ValueNode,
		Label:     v3.OperationRefLabel,
		Changes:   &changes,
		Breaking:  true,
		Original:  l,
		New:       r,
	})

	// operation id
	props = append(props, &PropertyCheck{
		LeftNode:  l.OperationId.ValueNode,
		RightNode: r.OperationId.ValueNode,
		Label:     v3.OperationIdLabel,
		Changes:   &changes,
		Breaking:  true,
		Original:  l,
		New:       r,
	})

	// request body
	props = append(props, &PropertyCheck{
		LeftNode:  l.RequestBody.ValueNode,
		RightNode: r.RequestBody.ValueNode,
		Label:     v3.RequestBodyLabel,
		Changes:   &changes,
		Breaking:  true,
		Original:  l,
		New:       r,
	})

	// description
	props = append(props, &PropertyCheck{
		LeftNode:  l.Description.ValueNode,
		RightNode: r.Description.ValueNode,
		Label:     v3.DescriptionLabel,
		Changes:   &changes,
		Breaking:  false,
		Original:  l,
		New:       r,
	})

	CheckProperties(props)
	lc := new(LinkChanges)
	lc.ExtensionChanges = CompareExtensions(l.Extensions, r.Extensions)

	// server
	if !l.Server.IsEmpty() && !r.Server.IsEmpty() {
		if !low.AreEqual(l.Server.Value, r.Server.Value) {
			lc.ServerChanges = CompareServers(l.Server.Value, r.Server.Value)
		}
	}
	if !l.Server.IsEmpty() && r.Server.IsEmpty() {
		CreateChange(&changes, PropertyRemoved, v3.ServerLabel,
			l.Server.ValueNode, nil, true,
			l.Server.Value, nil)
	}
	if l.Server.IsEmpty() && !r.Server.IsEmpty() {
		CreateChange(&changes, PropertyAdded, v3.ServerLabel,
			nil, r.Server.ValueNode, true,
			nil, r.Server.Value)
	}

	// parameters
	lValues := make(map[string]low.ValueReference[string])
	rValues := make(map[string]low.ValueReference[string])
	for i := range l.Parameters.Value {
		lValues[i.Value] = l.Parameters.Value[i]
	}
	for i := range r.Parameters.Value {
		rValues[i.Value] = r.Parameters.Value[i]
	}
	for k := range lValues {
		if _, ok := rValues[k]; !ok {
			CreateChange(&changes, ObjectRemoved, v3.ParametersLabel,
				lValues[k].ValueNode, nil, true,
				k, nil)
			continue
		}
		if lValues[k].Value != rValues[k].Value {
			CreateChange(&changes, Modified, v3.ParametersLabel,
				lValues[k].ValueNode, rValues[k].ValueNode, true,
				k, k)
		}

	}
	for k := range rValues {
		if _, ok := lValues[k]; !ok {
			CreateChange(&changes, ObjectAdded, v3.ParametersLabel,
				nil, rValues[k].ValueNode, true,
				nil, k)
		}
	}

	lc.Changes = changes
	return lc
}
