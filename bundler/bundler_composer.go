// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package bundler

import (
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	v3low "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"strings"
	"sync"
)

type processRef struct {
	idx    *index.SpecIndex
	ref    *index.Reference
	seqRef *index.Reference
	name   string
}

type handleIndexConfig struct {
	idx     *index.SpecIndex
	model   *v3.Document
	indexes []*index.SpecIndex
	refMap  *orderedmap.Map[string, *processRef]
	seen    sync.Map
	depth   int
}

func handleIndex(c *handleIndexConfig) {
	c.depth++
	if c.depth > 1000 {
		c.idx.GetLogger().Error("[bundler] too deep, we're over 1000 levels deep")
		return
	}

	mappedReferences := c.idx.GetMappedReferences()
	sequencedReferences := c.idx.GetRawReferencesSequenced()
	var indexesToExplore []*index.SpecIndex

	for _, sequenced := range sequencedReferences {
		mappedReference := mappedReferences[sequenced.FullDefinition]

		// if we're in the root document, don't bundle anything.
		refExp := strings.Split(sequenced.FullDefinition, "#/")
		var foundIndex *index.SpecIndex
		if len(refExp) == 2 {

			// make sure to use the correct index.
			// https://github.com/pb33f/libopenapi/issues/397
			for _, i := range c.indexes {
				if i.GetSpecAbsolutePath() == refExp[0] {
					foundIndex = i
					if mappedReference != nil && !mappedReference.Circular {
						mr := i.FindComponent(sequenced.Definition)
						if mr != nil {
							// found the component; this is the one we want to use.
							mappedReference = mr
							break
						}
					}
				}
			}
			// check if we have seen this index before, if so - skip it, otherwise we will be going around forever.
			if _, ok := c.seen.Load(sequenced.FullDefinition); ok {
				continue
			}
			if foundIndex != nil && mappedReference != nil {
				// store the reference to be composed in the root.
				if kk := c.refMap.GetOrZero(mappedReference.FullDefinition); kk == nil {
					c.refMap.Set(mappedReference.FullDefinition, &processRef{
						idx:    foundIndex,
						ref:    mappedReference,
						seqRef: sequenced,
						name:   mappedReference.Name,
					})
				}
				c.seen.Store(mappedReference.FullDefinition, mappedReference) // TODO: replace with map.
				indexesToExplore = append(indexesToExplore, foundIndex)
			}
		}
	}

	for _, idx := range indexesToExplore {
		c.idx = idx
		handleIndex(c)
	}
}

func processReference(model *v3.Document, pr *processRef) {

	idx := pr.idx
	location := strings.Split(pr.ref.Definition, "/")[1:]

	var err error
	if len(location) > 0 {
		if location[0] == v3low.ComponentsLabel {
			var components *v3.Components
			if model.Components != nil {
				components = model.Components
			} else {
				components, err = buildComponents(idx)
				if err != nil {
					idx.GetLogger().Error("[bundler] unable to build components", "error", err)
					return
				}
				model.Components = components
			}

			if len(location) > 1 {
				switch location[1] {
				case v3low.SchemasLabel:
					if len(location) > 2 {
						schemaName := location[2]
						if components.Schemas != nil {
							err = checkReferenceAndBubbleUp(schemaName, pr, idx, components.Schemas, buildSchema)
							if err != nil {
								return
							}
						}
					}

				case v3low.ResponsesLabel:
					if len(location) > 2 {
						responseCode := location[2]
						if components.Responses != nil {
							err = checkReferenceAndBubbleUp(responseCode, pr, idx, components.Responses, buildResponse)
							if err != nil {
								return
							}
						}
					}

				case v3low.ParametersLabel:
					if len(location) > 2 {
						paramName := location[2]
						if components.Parameters != nil {
							err = checkReferenceAndBubbleUp(paramName, pr, idx, components.Parameters, buildParameter)
							if err != nil {
								return
							}
						}
					}

				case v3low.HeadersLabel:
					if len(location) > 2 {
						headerName := location[2]
						if components.Headers != nil {
							err = checkReferenceAndBubbleUp(headerName, pr, idx, components.Headers, buildHeader)
							if err != nil {
								return
							}
						}
					}

				case v3low.RequestBodiesLabel:
					if len(location) > 2 {
						requestBodyName := location[2]
						if components.RequestBodies != nil {
							err = checkReferenceAndBubbleUp(requestBodyName, pr, idx, components.RequestBodies, buildRequestBody)
							if err != nil {
								return
							}
						}
					}
				case v3low.ExamplesLabel:
					if len(location) > 2 {
						exampleName := location[2]
						if components.Examples != nil {
							err = checkReferenceAndBubbleUp(exampleName, pr, idx, components.Examples, buildExample)
							if err != nil {
								return
							}
						}
					}

				case v3low.LinksLabel:
					if len(location) > 2 {
						linksName := location[2]
						if components.Links != nil {
							err = checkReferenceAndBubbleUp(linksName, pr, idx, components.Links, buildLink)
							if err != nil {
								return
							}
						}
					}

				case v3low.CallbacksLabel:
					if len(location) > 2 {
						callbacks := location[2]
						if components.Callbacks != nil {
							err = checkReferenceAndBubbleUp(callbacks, pr, idx, components.Callbacks, buildCallback)
							if err != nil {
								return
							}
						}
					}

				case v3low.PathItemsLabel:
					if len(location) > 2 {
						pathItem := location[2]
						if components.Callbacks != nil {
							err = checkReferenceAndBubbleUp(pathItem, pr, idx, components.PathItems, buildPathItem)
							if err != nil {
								return
							}
						}
					}
				}
			}
		}
	}
}
