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
	idx      *index.SpecIndex
	ref      *index.Reference
	seqRef   *index.Reference
	name     string
	location []string
}

type handleIndexConfig struct {
	idx               *index.SpecIndex
	model             *v3.Document
	indexes           []*index.SpecIndex
	refMap            *orderedmap.Map[string, *processRef]
	seen              sync.Map
	inlineRequired    []*processRef
	compositionConfig *BundleCompositionConfig
}

// handleIndex will recursively explore the indexes and their references, building a map of references
// to be processed later. It will also check for circular references and avoid infinite loops.
// everything is stored in the handleIndexConfig, which is passed around to avoid passing too many parameters.
func handleIndex(c *handleIndexConfig) {
	mappedReferences := c.idx.GetMappedReferences()
	sequencedReferences := c.idx.GetRawReferencesSequenced()
	var indexesToExplore []*index.SpecIndex

	for _, sequenced := range sequencedReferences {
		mappedReference := mappedReferences[sequenced.FullDefinition]

		// if we're in the root document, don't bundle anything.
		refExp := strings.Split(sequenced.FullDefinition, "#/")
		var foundIndex *index.SpecIndex

		// make sure to use the correct index.
		// https://github.com/pb33f/libopenapi/issues/397
		for _, i := range c.indexes {
			if i.GetSpecAbsolutePath() == refExp[0] {
				foundIndex = i
				if mappedReference != nil && !mappedReference.Circular {

					lookup := sequenced.Definition
					if !strings.Contains(lookup, "#/") {
						lookup = sequenced.FullDefinition
					}
					mr := i.FindComponent(lookup)
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
			if _, ok := c.seen.Load(foundIndex.GetSpecAbsolutePath()); !ok {
				c.seen.Store(foundIndex.GetSpecAbsolutePath(), mappedReference) // TODO: replace with map.
				indexesToExplore = append(indexesToExplore, foundIndex)
			}
		}
	}

	for _, idx := range indexesToExplore {
		c.idx = idx
		handleIndex(c)
	}
}

// processReference will extract a reference from the current index, and transform it into a first class
// top-level component in the root OpenAPI document.
func processReference(model *v3.Document, pr *processRef, cf *handleIndexConfig) error {

	idx := pr.idx
	var components *v3.Components
	var err error

	delim := cf.compositionConfig.Delimiter

	if model.Components != nil {
		components = model.Components
	} else {
		components, err = buildComponents(idx)
		if err != nil {
			return err
		}
		model.Components = components
	}

	var location []string
	if strings.Contains(pr.ref.FullDefinition, "#/") {
		location = strings.Split(pr.ref.Definition, "/")[1:]
	} else {
		// make sure the sequence ref and pr ref have the same full definition.
		pr.ref.FullDefinition = pr.seqRef.FullDefinition
		// this is a root document reference, there is no way to get the location from the fragment.
		// first, lets try to determine the type of the import, if we can.
		if importType, ok := DetectOpenAPIComponentType(pr.ref.Node); ok {

			// cool, using the filename as the reference name, check if we have any collisions.
			switch importType {
			case v3low.SchemasLabel:
				location = handleFileImport(pr, v3low.SchemasLabel, delim, components.Schemas)
			case v3low.ResponsesLabel:
				location = handleFileImport(pr, v3low.ResponsesLabel, delim, components.Responses)
			case v3low.ParametersLabel:
				location = handleFileImport(pr, v3low.ParametersLabel, delim, components.Parameters)
			case v3low.HeadersLabel:
				location = handleFileImport(pr, v3low.HeadersLabel, delim, components.Headers)
			case v3low.RequestBodiesLabel:
				location = handleFileImport(pr, v3low.RequestBodiesLabel, delim, components.RequestBodies)
			case v3low.ExamplesLabel:
				location = handleFileImport(pr, v3low.ExamplesLabel, delim, components.Examples)
			case v3low.LinksLabel:
				location = handleFileImport(pr, v3low.LinksLabel, delim, components.Links)
			case v3low.CallbacksLabel:
				location = handleFileImport(pr, v3low.CallbacksLabel, delim, components.Callbacks)
			case v3low.PathItemsLabel:
				location = handleFileImport(pr, v3low.PathItemsLabel, delim, components.PathItems)
			}
		} else {
			// the only choice we can make here to be accurate is to inline instead of recompose.
			cf.inlineRequired = append(cf.inlineRequired, pr)
		}
	}

	if len(location) > 0 {
		pr.location = location
		if location[0] == v3low.ComponentsLabel {

			if len(location) > 1 {
				switch location[1] {
				case v3low.SchemasLabel:
					if len(location) > 2 {
						schemaName := location[2]
						if components.Schemas != nil {
							return checkReferenceAndBubbleUp(schemaName, cf.compositionConfig.Delimiter,
								pr, idx, components.Schemas, buildSchema)
						}
					}

				case v3low.ResponsesLabel:
					if len(location) > 2 {
						responseCode := location[2]
						if components.Responses != nil {
							return checkReferenceAndBubbleUp(responseCode, cf.compositionConfig.Delimiter,
								pr, idx, components.Responses, buildResponse)
						}
					}

				case v3low.ParametersLabel:
					if len(location) > 2 {
						paramName := location[2]
						if components.Parameters != nil {
							return checkReferenceAndBubbleUp(paramName, cf.compositionConfig.Delimiter,
								pr, idx, components.Parameters, buildParameter)
						}
					}

				case v3low.HeadersLabel:
					if len(location) > 2 {
						headerName := location[2]
						if components.Headers != nil {
							return checkReferenceAndBubbleUp(headerName, cf.compositionConfig.Delimiter,
								pr, idx, components.Headers, buildHeader)
						}
					}

				case v3low.RequestBodiesLabel:
					if len(location) > 2 {
						requestBodyName := location[2]
						if components.RequestBodies != nil {
							return checkReferenceAndBubbleUp(requestBodyName, cf.compositionConfig.Delimiter,
								pr, idx, components.RequestBodies, buildRequestBody)
						}
					}
				case v3low.ExamplesLabel:
					if len(location) > 2 {
						exampleName := location[2]
						if components.Examples != nil {
							return checkReferenceAndBubbleUp(exampleName, cf.compositionConfig.Delimiter,
								pr, idx, components.Examples, buildExample)
						}
					}

				case v3low.LinksLabel:
					if len(location) > 2 {
						linksName := location[2]
						if components.Links != nil {
							return checkReferenceAndBubbleUp(linksName, cf.compositionConfig.Delimiter,
								pr, idx, components.Links, buildLink)
						}
					}

				case v3low.CallbacksLabel:
					if len(location) > 2 {
						callbacks := location[2]
						if components.Callbacks != nil {
							return checkReferenceAndBubbleUp(callbacks, cf.compositionConfig.Delimiter,
								pr, idx, components.Callbacks, buildCallback)
						}
					}

				case v3low.PathItemsLabel:
					if len(location) > 2 {
						pathItem := location[2]
						if components.Callbacks != nil {
							return checkReferenceAndBubbleUp(pathItem, cf.compositionConfig.Delimiter,
								pr, idx, components.PathItems, buildPathItem)
						}
					}
				}
			}
		}
	}
	return nil
}
