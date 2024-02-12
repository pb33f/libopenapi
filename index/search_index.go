// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

type ContextKey string

const (
	CurrentPathKey ContextKey = "currentPath"
	FoundIndexKey  ContextKey = "foundIndex"
)

func (index *SpecIndex) SearchIndexForReferenceByReference(fullRef *Reference) (*Reference, *SpecIndex) {
	r, idx, _ := index.SearchIndexForReferenceByReferenceWithContext(context.Background(), fullRef)
	return r, idx
}

// SearchIndexForReference searches the index for a reference, first looking through the mapped references
// and then externalSpecIndex for a match. If no match is found, it will recursively search the child indexes
// extracted when parsing the OpenAPI Spec.
func (index *SpecIndex) SearchIndexForReference(ref string) (*Reference, *SpecIndex) {
	return index.SearchIndexForReferenceByReference(&Reference{FullDefinition: ref})
}

func (index *SpecIndex) SearchIndexForReferenceWithContext(ctx context.Context, ref string) (*Reference, *SpecIndex, context.Context) {
	return index.SearchIndexForReferenceByReferenceWithContext(ctx, &Reference{FullDefinition: ref})
}

func (index *SpecIndex) SearchIndexForReferenceByReferenceWithContext(ctx context.Context, searchRef *Reference) (*Reference, *SpecIndex, context.Context) {
	if index.cache != nil {
		if v, ok := index.cache.Load(searchRef.FullDefinition); ok {
			return v.(*Reference), v.(*Reference).Index, context.WithValue(ctx, CurrentPathKey, v.(*Reference).RemoteLocation)
		}
	}

	ref := searchRef.FullDefinition
	refAlt := ref
	absPath := index.specAbsolutePath
	if searchRef.RemoteLocation != "" {
		absPath = searchRef.RemoteLocation
	}
	if absPath == "" {
		absPath = index.config.BasePath
	}
	var roloLookup string
	uri := strings.Split(ref, "#/")
	if len(uri) == 2 {
		if uri[0] != "" {
			if strings.HasPrefix(uri[0], "http") {
				roloLookup = searchRef.FullDefinition
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
		} else {

			if filepath.Ext(uri[1]) != "" {
				roloLookup = absPath
			} else {
				roloLookup = ""
			}

			ref = fmt.Sprintf("#/%s", uri[1])
			refAlt = fmt.Sprintf("%s#/%s", absPath, uri[1])

		}
	} else {
		if filepath.IsAbs(uri[0]) {
			roloLookup = uri[0]
		} else {
			if strings.HasPrefix(uri[0], "http") {
				roloLookup = ref
			} else {
				if filepath.Ext(absPath) != "" {
					absPath = filepath.Dir(absPath)
				}
				roloLookup, _ = filepath.Abs(filepath.Join(absPath, uri[0]))
			}
		}
		ref = uri[0]
	}
	if strings.Contains(ref, "%") {
		// decode the url.
		ref, _ = url.QueryUnescape(ref)
		refAlt, _ = url.QueryUnescape(refAlt)
	}

	if r, ok := index.allMappedRefs[ref]; ok {
		index.cache.Store(ref, r)
		return r, r.Index, context.WithValue(ctx, CurrentPathKey, r.RemoteLocation)
	}

	if r, ok := index.allMappedRefs[refAlt]; ok {
		index.cache.Store(refAlt, r)
		return r, r.Index, context.WithValue(ctx, CurrentPathKey, r.RemoteLocation)
	}

	if r, ok := index.allComponentSchemaDefinitions.Load(refAlt); ok {
		ref := r.(*Reference)

		index.cache.Store(refAlt, r)
		return ref, ref.Index, context.WithValue(ctx, CurrentPathKey, ref.RemoteLocation)
	}

	// check the rolodex for the reference.
	if roloLookup != "" {

		if strings.Contains(roloLookup, "#") {
			roloLookup = strings.Split(roloLookup, "#")[0]
		}
		if filepath.Base(roloLookup) == "root.yaml" {
			return nil, index, ctx
		}
		rFile, err := index.rolodex.Open(roloLookup)
		if err != nil {
			return nil, index, ctx
		}

		// extract the index from the rolodex file.
		if rFile != nil {

			n := rFile.GetFullPath()
			refParsed := ref

			// do we have a relative reference and an exact match on the suffix?
			if strings.HasPrefix(ref, "./") {
				refParsed = strings.ReplaceAll(ref, "./", "")
			}

			if strings.HasSuffix(n, refParsed) {
				node, _ := rFile.GetContentAsYAMLNode()
				if node != nil {
					return &Reference{
						FullDefinition: n,
						Definition:     n,
						IsRemote:       true,
						RemoteLocation: n,
						Index:          rFile.GetIndex(),
						Node:           node.Content[0],
						ParentNode:     node,
					}, rFile.GetIndex(), ctx
				} else {
					return nil, index, ctx
				}
			}

			idx := rFile.GetIndex()
			if index.resolver != nil {
				index.resolver.indexesVisited++
			}
			if idx != nil {

				// check mapped refs.
				if r, ok := idx.allMappedRefs[ref]; ok {
					index.cache.Store(ref, r)
					idx.cache.Store(ref, r)
					return r, r.Index, context.WithValue(ctx, CurrentPathKey, r.RemoteLocation)
				}

				// build a collection of all the inline schemas and search them
				// for the reference.
				var d []*Reference
				d = append(d, idx.allInlineSchemaDefinitions...)
				d = append(d, idx.allRefSchemaDefinitions...)
				d = append(d, idx.allInlineSchemaObjectDefinitions...)
				for _, s := range d {
					if s.FullDefinition == ref {
						idx.cache.Store(ref, s)
						index.cache.Store(ref, s)
						return s, s.Index, context.WithValue(ctx, CurrentPathKey, s.RemoteLocation)
					}
				}

				// does component exist in the root?
				node, _ := rFile.GetContentAsYAMLNode()
				if node != nil {
					var found *Reference
					exp := strings.Split(ref, "#/")
					compId := ref

					if len(exp) == 2 {
						compId = fmt.Sprintf("#/%s", exp[1])
						found = FindComponent(node, compId, exp[0], idx)
					}
					if found == nil {
						found = idx.FindComponent(ref)
					}

					if found != nil {
						idx.cache.Store(ref, found)
						index.cache.Store(ref, found)
						return found, found.Index, context.WithValue(ctx, CurrentPathKey, found.RemoteLocation)
					}
				}
			}
		}
	}

	if index.logger != nil {
		index.logger.Error("unable to locate reference anywhere in the rolodex", "reference", ref)
	}
	return nil, index, ctx
}
