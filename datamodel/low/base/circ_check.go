// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package base

import (
	"github.com/pb33f/libopenapi/index"
)

func CheckSchemaProxyForCircularRefs(s *SchemaProxy) bool {
	allCircs := s.GetIndex().GetRolodex().GetRootIndex().GetCircularReferences()
	safeCircularRefs := s.GetIndex().GetRolodex().GetSafeCircularReferences()
	ignoredCircularRefs := s.GetIndex().GetRolodex().GetIgnoredCircularReferences()
	combinedCircularRefs := append(safeCircularRefs, ignoredCircularRefs...)
	combinedCircularRefs = append(combinedCircularRefs, allCircs...)
	for _, ref := range combinedCircularRefs {
		// hash the root node of the schema reference
		rh := index.HashNode(s.GetValueNode())
		lph := index.HashNode(ref.LoopPoint.Node)
		if rh == lph {
			return true // nope
		}
	}
	return false
}
