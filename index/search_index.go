// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

// SearchIndexForReference searches the index for a reference, first looking through the mapped references
// and then externalSpecIndex for a match. If no match is found, it will recursively search the child indexes
// extracted when parsing the OpenAPI Spec.
func (index *SpecIndex) SearchIndexForReference(ref string) []*Reference {
	if r, ok := index.allMappedRefs[ref]; ok {
		return []*Reference{r}
	}
	for c := range index.children {
		found := goFindMeSomething(index.children[c], ref)
		if found != nil {
			return found
		}
	}
	return nil
}

func (index *SpecIndex) SearchAncestryForSeenURI(uri string) *SpecIndex {
	if index.parentIndex == nil {
		return nil
	}
	if index.uri[0] != uri {
		return index.parentIndex.SearchAncestryForSeenURI(uri)
	}
	return index
}

func goFindMeSomething(i *SpecIndex, ref string) []*Reference {
	return i.SearchIndexForReference(ref)
}
