// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"fmt"
	"path/filepath"
	"strings"
)

func (index *SpecIndex) SearchIndexForReferenceByReference(fullRef *Reference) *Reference {

	ref := fullRef.FullDefinition

	absPath := index.specAbsolutePath
	if absPath == "" {
		absPath = index.config.BasePath
	}
	var roloLookup string
	uri := strings.Split(ref, "#/")
	if len(uri) == 2 {
		if uri[0] != "" {
			if strings.HasPrefix(uri[0], "http") {
				roloLookup = fullRef.FullDefinition
			} else {
				if filepath.IsAbs(uri[0]) {
					roloLookup = uri[0]
				} else {
					if filepath.Ext(absPath) != "" {
						absPath = filepath.Dir(absPath)
					}
					roloLookup, _ = filepath.Abs(filepath.Join(absPath, uri[0]))
				}
			}
		}
		ref = fmt.Sprintf("#/%s", uri[1])
	} else {
		if filepath.IsAbs(uri[0]) {
			roloLookup = uri[0]
		} else {
			if filepath.Ext(absPath) != "" {
				absPath = filepath.Dir(absPath)
			}
			roloLookup, _ = filepath.Abs(filepath.Join(absPath, uri[0]))
		}

		ref = uri[0]
	}

	if r, ok := index.allMappedRefs[ref]; ok {
		return r
	}

	// check the rolodex for the reference.
	if roloLookup != "" {
		rFile, err := index.rolodex.Open(roloLookup)
		if err != nil {
			return nil
		}

		// extract the index from the rolodex file.
		idx := rFile.GetIndex()
		index.resolver.indexesVisited++
		if idx != nil {

			// check mapped refs.
			if r, ok := idx.allMappedRefs[ref]; ok {
				return r
			}

			// build a collection of all the inline schemas and search them
			// for the reference.
			var d []*Reference
			d = append(d, idx.allInlineSchemaDefinitions...)
			d = append(d, idx.allRefSchemaDefinitions...)
			d = append(d, idx.allInlineSchemaObjectDefinitions...)
			for _, s := range d {
				if s.Definition == ref {
					return s
				}
			}

			// does component exist in the root?
			node, _ := rFile.GetContentAsYAMLNode()
			if node != nil {
				found := idx.FindComponent(ref, node)
				if found != nil {
					return found
				}
			}
		}
	}

	fmt.Printf("unable to locate reference: %s, within index: %s\n", ref, index.specAbsolutePath)
	return nil
}

// SearchIndexForReference searches the index for a reference, first looking through the mapped references
// and then externalSpecIndex for a match. If no match is found, it will recursively search the child indexes
// extracted when parsing the OpenAPI Spec.
func (index *SpecIndex) SearchIndexForReference(ref string) *Reference {
	return index.SearchIndexForReferenceByReference(&Reference{FullDefinition: ref})
}
