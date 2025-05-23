// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io
// SPDX-License-Identifier: MIT

package bundler

import (
	"errors"
	"fmt"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	v3low "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
)

// ErrInvalidModel is returned when the model is not usable.
var ErrInvalidModel = errors.New("invalid model")

// BundleBytes will take a byte slice of an OpenAPI specification and return a bundled version of it.
// This is useful for when you want to take a specification with external references, and you want to bundle it
// into a single document.
//
// This function will 'resolve' all references in the specification and return a single document. The resulting
// document will be a valid OpenAPI specification, containing no references.
//
// Circular references will not be resolved and will be skipped.
func BundleBytes(bytes []byte, configuration *datamodel.DocumentConfiguration) ([]byte, error) {
	doc, err := libopenapi.NewDocumentWithConfiguration(bytes, configuration)
	if err != nil {
		return nil, err
	}

	v3Doc, errs := doc.BuildV3Model()
	err = errors.Join(errs...)
	if v3Doc == nil {
		return nil, errors.Join(ErrInvalidModel, err)
	}

	bundledBytes, e := bundle(&v3Doc.Model)
	return bundledBytes, errors.Join(err, e)
}

// BundleDocument will take a v3.Document and return a bundled version of it.
// This is useful for when you want to take a document that has been built
// from a specification with external references, and you want to bundle it
// into a single document.
//
// This function will 'resolve' all references in the specification and return a single document. The resulting
// document will be a valid OpenAPI specification, containing no references.
//
// Circular references will not be resolved and will be skipped.
func BundleDocument(model *v3.Document) ([]byte, error) {
	return bundle(model)
}

// BundleDocumentComposed will take a v3.Document and return a composed bundled version of it. Composed means
// that every external file will have references lifted out and added to the `components` section of the document.
// Names will be preserved where possible, conflicts will be appended with a number. If the type of the reference cannot
// be determined, it will be added to the `components` section as a `Schema` type, a warning will be logged.
// The document model will be mutated permanently.
//
// Circular references will not be resolved and will be skipped.
func BundleDocumentComposed(model *v3.Document) ([]byte, error) {
	return compose(model)
}

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

func calculateCollisionName(name, pointer string) string {
	jsonPointer := strings.Split(pointer, "#/")
	if len(jsonPointer) == 2 {

		// TODO: make delimiter configurable.
		// count the number of collisions by splitting the name by the __ delimiter.
		nameSegments := strings.Split(name, "__")
		if len(nameSegments) > 1 {

			if len(nameSegments) == 2 {
				return fmt.Sprintf("%s__%s", name, "1")
			}
			if len(nameSegments) == 3 {
				count, err := strconv.Atoi(nameSegments[2])
				if err != nil {
					return fmt.Sprintf("%s__%s", name, "X")
				}
				count++
				nameSegments[2] = strconv.Itoa(count)
				return strings.Join(nameSegments, "__")
			}

		} else {

			// the first collision attempt will be to use the last segment of the location as a postfix.
			// this will be the last segment of the path.

			uri := jsonPointer[0]
			b := filepath.Base(uri)
			fileName := fmt.Sprintf("%s__%s", name, strings.Replace(b, filepath.Ext(b), "", 1))
			return fileName

		}

	}

	// TODO: handle full file imports.
	return name
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

func compose(model *v3.Document) ([]byte, error) {

	rolodex := model.Rolodex
	indexes := rolodex.GetIndexes()

	cf := &handleIndexConfig{
		idx:     rolodex.GetRootIndex(),
		model:   model,
		indexes: indexes,
		seen:    sync.Map{},
		depth:   0,
		refMap:  orderedmap.New[string, *processRef](),
	}
	// recursive function to handle the indexes, we need a very different approach to composition vs. inlining.
	handleIndex(cf)

	processedNodes := orderedmap.New[string, *processRef]()

	for _, ref := range cf.refMap.FromOldest() {
		processReference(model, ref)
		processedNodes.Set(ref.ref.FullDefinition, ref)
	}

	rootIndex := rolodex.GetRootIndex()
	remapIndex(rootIndex, processedNodes)

	slices.SortFunc(indexes, func(i, j *index.SpecIndex) int {
		if i.GetSpecAbsolutePath() < j.GetSpecAbsolutePath() {
			return 1
		}
		return 0
	})

	for _, idx := range indexes {
		remapIndex(idx, processedNodes)
	}

	return model.Render()
}

func bundle(model *v3.Document) ([]byte, error) {
	rolodex := model.Rolodex
	indexes := rolodex.GetIndexes()
	//indexMap := make(map[string]*index.SpecIndex)
	// compact function.
	compact := func(idx *index.SpecIndex, root bool) {
		mappedReferences := idx.GetMappedReferences()
		sequencedReferences := idx.GetRawReferencesSequenced()
		for _, sequenced := range sequencedReferences {
			mappedReference := mappedReferences[sequenced.FullDefinition]

			// if we're in the root document, don't bundle anything.
			refExp := strings.Split(sequenced.FullDefinition, "#/")
			if len(refExp) == 2 {

				// make sure to use the correct index.
				// https://github.com/pb33f/libopenapi/issues/397
				if root {

					for _, i := range indexes {
						if i.GetSpecAbsolutePath() == refExp[0] {
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
				}

				if refExp[0] == sequenced.Index.GetSpecAbsolutePath() || refExp[0] == "" {
					if root {
						idx.GetLogger().Debug("[bundler] skipping local root reference",
							"ref", sequenced.Definition)
						continue
					}
				}
			}

			if mappedReference != nil && !mappedReference.Circular {
				sequenced.Node.Content = mappedReference.Node.Content
				continue
			}

			if mappedReference != nil && mappedReference.Circular {
				if idx.GetLogger() != nil {
					idx.GetLogger().Warn("[bundler] skipping circular reference",
						"ref", sequenced.FullDefinition)
				}
			}
		}
	}

	for _, idx := range indexes {
		compact(idx, false)
	}
	compact(rolodex.GetRootIndex(), true)
	return model.Render()
}
