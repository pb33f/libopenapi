// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	v2 "github.com/pb33f/libopenapi/datamodel/low/v2"
)

// ExamplesChanges represents changes made between Swagger Examples objects (Not OpenAPI 3).
type ExamplesChanges struct {
	*PropertyChanges
}

// GetAllChanges returns a slice of all changes made between Examples objects
func (a *ExamplesChanges) GetAllChanges() []*Change {
	return a.Changes
}

// TotalChanges represents the total number of changes made between Example instances.
func (a *ExamplesChanges) TotalChanges() int {
	return a.PropertyChanges.TotalChanges()
}

// TotalBreakingChanges will always return 0. Examples cannot break a contract.
func (a *ExamplesChanges) TotalBreakingChanges() int {
	return 0 // not supported.
}

// CompareExamplesV2 compares two Swagger Examples objects, returning a pointer to
// ExamplesChanges if anything was found.
func CompareExamplesV2(l, r *v2.Examples) *ExamplesChanges {

	lHashes := make(map[string]string)
	rHashes := make(map[string]string)
	lValues := make(map[string]low.ValueReference[any])
	rValues := make(map[string]low.ValueReference[any])

	for k := range l.Values {
		lHashes[k.Value] = low.GenerateHashString(l.Values[k].Value)
		lValues[k.Value] = l.Values[k]
	}

	for k := range r.Values {
		rHashes[k.Value] = low.GenerateHashString(r.Values[k].Value)
		rValues[k.Value] = r.Values[k]
	}
	var changes []*Change

	// check left example hashes
	for k := range lHashes {
		rhash := rHashes[k]
		if rhash == "" {
			CreateChange(&changes, ObjectRemoved, k,
				lValues[k].GetValueNode(), nil, false,
				lValues[k].GetValue(), nil)
			continue
		}
		if lHashes[k] == rHashes[k] {
			continue
		}
		CreateChange(&changes, Modified, k,
			lValues[k].GetValueNode(), rValues[k].GetValueNode(), false,
			lValues[k].GetValue(), lValues[k].GetValue())

	}

	//check right example hashes
	for k := range rHashes {
		lhash := lHashes[k]
		if lhash == "" {
			CreateChange(&changes, ObjectAdded, k,
				nil, lValues[k].GetValueNode(), false,
				nil, lValues[k].GetValue())
			continue
		}
	}

	ex := new(ExamplesChanges)
	ex.PropertyChanges = NewPropertyChanges(changes)
	if ex.TotalChanges() <= 0 {
		return nil
	}
	return ex
}
