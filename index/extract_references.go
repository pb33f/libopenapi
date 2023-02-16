// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
    "fmt"
    "github.com/pb33f/libopenapi/utils"
    "strings"
)

// ExtractComponentsFromRefs returns located components from references. The returned nodes from here
// can be used for resolving as they contain the actual object properties.
func (index *SpecIndex) ExtractComponentsFromRefs(refs []*Reference) []*Reference {
    var found []*Reference

    //run this async because when things get recursive, it can take a while
    c := make(chan bool)

    locate := func(ref *Reference, refIndex int, sequence []*ReferenceMapped) {
        located := index.FindComponent(ref.Definition, ref.Node)
        if located != nil {
            index.refLock.Lock()
            if index.allMappedRefs[ref.Definition] == nil {
                found = append(found, located)
                index.allMappedRefs[ref.Definition] = located
                sequence[refIndex] = &ReferenceMapped{
                    Reference:  located,
                    Definition: ref.Definition,
                }
            }
            index.refLock.Unlock()
        } else {

            _, path := utils.ConvertComponentIdIntoFriendlyPathSearch(ref.Definition)
            indexError := &IndexingError{
                Err:  fmt.Errorf("component '%s' does not exist in the specification", ref.Definition),
                Node: ref.Node,
                Path: path,
            }
            index.refErrors = append(index.refErrors, indexError)
        }
        c <- true
    }

    var refsToCheck []*Reference
    for _, ref := range refs {

        // check reference for backslashes (hah yeah seen this too!)
        if strings.Contains(ref.Definition, "\\") { // this was from blazemeter.com haha!
            _, path := utils.ConvertComponentIdIntoFriendlyPathSearch(ref.Definition)
            indexError := &IndexingError{
                Err:  fmt.Errorf("component '%s' contains a backslash '\\'. It's not valid", ref.Definition),
                Node: ref.Node,
                Path: path,
            }
            index.refErrors = append(index.refErrors, indexError)
            continue

        }
        refsToCheck = append(refsToCheck, ref)
    }
    mappedRefsInSequence := make([]*ReferenceMapped, len(refsToCheck))
    for r := range refsToCheck {
        // expand our index of all mapped refs
        go locate(refsToCheck[r], r, mappedRefsInSequence)
    }

    completedRefs := 0
    for completedRefs < len(refsToCheck) {
        select {
        case <-c:
            completedRefs++
        }
    }
    for m := range mappedRefsInSequence {
        if mappedRefsInSequence[m] != nil {
            index.allMappedRefsSequenced = append(index.allMappedRefsSequenced, mappedRefsInSequence[m])
        }
    }
    return found
}
