// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

func (index *SpecIndex) SearchIndexForReference(ref string) []*Reference {

    if r, ok := index.allMappedRefs[ref]; ok {
        return []*Reference{r}
    }
    for c := range index.children {
        found := index.children[c].SearchIndexForReference(ref)
        if found != nil {
            return found
        }
    }
    return nil
}
