// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package base

func CheckSchemaProxyForCircularRefs(s *SchemaProxy) bool {
	rolo := s.GetIndex().GetRolodex()
	if rolo == nil {
		return false // no rolodex, so no circular references
	}
	allCircs := rolo.GetRootIndex().GetCircularReferences()
	safeCircularRefs := rolo.GetSafeCircularReferences()
	ignoredCircularRefs := rolo.GetIgnoredCircularReferences()
	combinedCircularRefs := append(safeCircularRefs, ignoredCircularRefs...)
	combinedCircularRefs = append(combinedCircularRefs, allCircs...)
	for _, ref := range combinedCircularRefs {
		// hash the root node of the schema reference
		if ref.LoopPoint.FullDefinition == s.GetReference() || ref.LoopPoint.Definition == s.GetReference() {
			return true
		}
	}
	return false
}
