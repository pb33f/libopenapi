// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import "gopkg.in/yaml.v3"

// SearchIndexForReference searches the index for a reference, first looking through the mapped references
// and then externalSpecIndex for a match. If no match is found, it will recursively search the child indexes
// extracted when parsing the OpenAPI Spec.
func (index *SpecIndex) SearchIndexForReference(ref string) []*Reference {
    if r, ok := index.allMappedRefs[ref]; ok {
        if r.Node.Kind == yaml.DocumentNode {
            // the reference is an entire document, so we need to dig down a level and rewire the reference.
            r.Node = r.Node.Content[0]
        }
        return []*Reference{r}
    }
    if r, ok := index.externalSpecIndex[ref]; ok {
        return []*Reference{
            {
                Node:       r.root.Content[0],
                Name:       ref,
                Definition: ref,
            },
        }
    }
    for c := range index.children {
        found := goFindMeSomething(index.children[c], ref)
        if found != nil {
            return found
        }
    }
    return nil
}

func goFindMeSomething(i *SpecIndex, ref string) []*Reference {
    return i.SearchIndexForReference(ref)
}
