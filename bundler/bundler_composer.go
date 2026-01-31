// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package bundler

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"

	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	v3low "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

type processRef struct {
	idx          *index.SpecIndex
	ref          *index.Reference
	seqRef       *index.Reference
	refPointer   string
	name         string
	location     []string
	wasRenamed   bool   // true when component was renamed due to collision
	originalName string // original name before collision renaming
}

type handleIndexConfig struct {
	idx                   *index.SpecIndex
	model                 *v3.Document
	indexes               []*index.SpecIndex
	refMap                *orderedmap.Map[string, *processRef]
	seen                  sync.Map
	inlineRequired        []*processRef
	compositionConfig     *BundleCompositionConfig
	discriminatorMappings []*yaml.Node
	origins               ComponentOriginMap // component origins for navigation
}

// handleIndex will recursively explore the indexes and their references, building a map of references
// to be processed later. It will also check for circular references and avoid infinite loops.
// everything is stored in the handleIndexConfig, which is passed around to avoid passing too many parameters.
func handleIndex(c *handleIndexConfig) error {
	mappedReferences := c.idx.GetMappedReferences()
	sequencedReferences := c.idx.GetRawReferencesSequenced()
	var indexesToExplore []*index.SpecIndex

	for _, sequenced := range sequencedReferences {
		mappedReference := mappedReferences[sequenced.FullDefinition]

		// Check for invalid sibling properties if strict validation is enabled
		if c.compositionConfig.StrictValidation &&
				c.idx.GetConfig().SpecInfo.VersionNumeric == 3.0 &&
				sequenced.HasSiblingProperties {
			siblingKeys := make([]string, 0, len(sequenced.SiblingProperties))
			for key := range sequenced.SiblingProperties {
				siblingKeys = append(siblingKeys, key)
			}
			return fmt.Errorf("invalid OpenAPI 3.0 specification: $ref cannot have sibling properties. Found $ref '%s' with siblings %v at line %d, column %d",
				sequenced.FullDefinition, siblingKeys, sequenced.Node.Line, sequenced.Node.Column)
		}

		// if we're in the root document, don't bundle anything.
		refExp := strings.Split(sequenced.FullDefinition, "#/")
		var foundIndex *index.SpecIndex

		// make sure to use the correct index.
		// https://github.com/pb33f/libopenapi/issues/397
		for _, i := range c.indexes {
			if i.GetSpecAbsolutePath() == refExp[0] {
				foundIndex = i
				if mappedReference != nil && !mappedReference.Circular {

					lookup := sequenced.FullDefinition
					mr := i.FindComponent(context.Background(), lookup)
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
		if err := handleIndex(c); err != nil {
			return err
		}
	}
	return nil
}

// openAPIRootKeys contains known OpenAPI root-level keys that should NOT be
// recomposed as components. OpenAPI root keys are always lowercase per spec.
// Package-level to avoid allocation on each call.
var openAPIRootKeys = map[string]bool{
	"openapi":          true,
	"info":             true,
	"jsonSchemaDialect": true,
	"servers":          true,
	"paths":            true,
	"webhooks":         true,
	"components":       true,
	"security":         true,
	"tags":             true,
	"externalDocs":     true,
}

// isOpenAPIRootKey returns true if the key is a known OpenAPI root-level key
// that should NOT be recomposed as a component. The check is case-sensitive
// because OpenAPI root keys are always lowercase, allowing component names
// like "Paths" or "INFO" to be recomposed normally.
func isOpenAPIRootKey(key string) bool {
	return openAPIRootKeys[key]
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

		// extract fragment from the full definition.
		segs := strings.Split(pr.ref.FullDefinition, "#/")
		location = strings.Split(segs[1], "/")
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

	unknown := func(procRef *processRef, config *handleIndexConfig) {
		if l := config.idx.GetLogger(); l != nil {
			l.Warn("[bundler] unable to compose reference, not sure where it goes.", "$ref", procRef.ref.FullDefinition)
		}
		// no idea what do with this, so we will inline it.
		config.inlineRequired = append(cf.inlineRequired, procRef)
	}

	if len(location) > 0 {
		pr.location = location
		if location[0] == v3low.ComponentsLabel {
			if len(location) > 1 {
				switch location[1] {
				case v3low.SchemasLabel:
					if len(location) > 2 && components.Schemas != nil {
						return checkReferenceAndCapture(location[2], cf.compositionConfig.Delimiter,
							v3low.SchemasLabel, pr, idx, components.Schemas, buildSchema, cf.origins)
					}

				case v3low.ResponsesLabel:
					if len(location) > 2 && components.Responses != nil {
						return checkReferenceAndCapture(location[2], cf.compositionConfig.Delimiter,
							v3low.ResponsesLabel, pr, idx, components.Responses, buildResponse, cf.origins)
					}

				case v3low.ParametersLabel:
					if len(location) > 2 && components.Parameters != nil {
						return checkReferenceAndCapture(location[2], cf.compositionConfig.Delimiter,
							v3low.ParametersLabel, pr, idx, components.Parameters, buildParameter, cf.origins)
					}

				case v3low.HeadersLabel:
					if len(location) > 2 && components.Headers != nil {
						return checkReferenceAndCapture(location[2], cf.compositionConfig.Delimiter,
							v3low.HeadersLabel, pr, idx, components.Headers, buildHeader, cf.origins)
					}

				case v3low.RequestBodiesLabel:
					if len(location) > 2 && components.RequestBodies != nil {
						return checkReferenceAndCapture(location[2], cf.compositionConfig.Delimiter,
							v3low.RequestBodiesLabel, pr, idx, components.RequestBodies, buildRequestBody, cf.origins)
					}

				case v3low.ExamplesLabel:
					if len(location) > 2 && components.Examples != nil {
						return checkReferenceAndCapture(location[2], cf.compositionConfig.Delimiter,
							v3low.ExamplesLabel, pr, idx, components.Examples, buildExample, cf.origins)
					}

				case v3low.LinksLabel:
					if len(location) > 2 && components.Links != nil {
						return checkReferenceAndCapture(location[2], cf.compositionConfig.Delimiter,
							v3low.LinksLabel, pr, idx, components.Links, buildLink, cf.origins)
					}

				case v3low.CallbacksLabel:
					if len(location) > 2 && components.Callbacks != nil {
						return checkReferenceAndCapture(location[2], cf.compositionConfig.Delimiter,
							v3low.CallbacksLabel, pr, idx, components.Callbacks, buildCallback, cf.origins)
					}

				case v3low.PathItemsLabel:
					if len(location) > 2 && components.PathItems != nil {
						return checkReferenceAndCapture(location[2], cf.compositionConfig.Delimiter,
							v3low.PathItemsLabel, pr, idx, components.PathItems, buildPathItem, cf.origins)
					}
				}
			}
		} else {
			// handle single-segment JSON pointers (e.g., #/NonRequired)
			if len(location) == 1 && location[0] != "" {
				componentName := location[0]

				// decode URL-encoded characters (e.g., "My%20Schema" -> "My Schema")
				if decoded, err := url.PathUnescape(componentName); err == nil {
					componentName = decoded
				}
				// process JSON Pointer escapes per RFC 6901 (~1 before ~0 to avoid mangling "~0")
				if strings.Contains(componentName, "~") {
					componentName = strings.ReplaceAll(componentName, "~1", "/")
					componentName = strings.ReplaceAll(componentName, "~0", "~")
				}

				// skip known OpenAPI root-level keys that are not reusable components
				if isOpenAPIRootKey(componentName) {
					unknown(pr, cf)
					return nil
				}

				// preserve original name before collision handling
				pr.originalName = componentName

				if importType, ok := DetectOpenAPIComponentType(pr.ref.Node); ok {
					switch importType {
					case v3low.SchemasLabel:
						if components.Schemas != nil {
							pr.name = checkForCollision(componentName, delim, pr, components.Schemas)
							pr.location = []string{v3low.ComponentsLabel, v3low.SchemasLabel, pr.name}
							return checkReferenceAndCapture(pr.name, delim, v3low.SchemasLabel, pr, idx, components.Schemas, buildSchema, cf.origins)
						}
					case v3low.ResponsesLabel:
						if components.Responses != nil {
							pr.name = checkForCollision(componentName, delim, pr, components.Responses)
							pr.location = []string{v3low.ComponentsLabel, v3low.ResponsesLabel, pr.name}
							return checkReferenceAndCapture(pr.name, delim, v3low.ResponsesLabel, pr, idx, components.Responses, buildResponse, cf.origins)
						}
					case v3low.ParametersLabel:
						if components.Parameters != nil {
							pr.name = checkForCollision(componentName, delim, pr, components.Parameters)
							pr.location = []string{v3low.ComponentsLabel, v3low.ParametersLabel, pr.name}
							return checkReferenceAndCapture(pr.name, delim, v3low.ParametersLabel, pr, idx, components.Parameters, buildParameter, cf.origins)
						}
					case v3low.HeadersLabel:
						if components.Headers != nil {
							pr.name = checkForCollision(componentName, delim, pr, components.Headers)
							pr.location = []string{v3low.ComponentsLabel, v3low.HeadersLabel, pr.name}
							return checkReferenceAndCapture(pr.name, delim, v3low.HeadersLabel, pr, idx, components.Headers, buildHeader, cf.origins)
						}
					case v3low.RequestBodiesLabel:
						if components.RequestBodies != nil {
							pr.name = checkForCollision(componentName, delim, pr, components.RequestBodies)
							pr.location = []string{v3low.ComponentsLabel, v3low.RequestBodiesLabel, pr.name}
							return checkReferenceAndCapture(pr.name, delim, v3low.RequestBodiesLabel, pr, idx, components.RequestBodies, buildRequestBody, cf.origins)
						}
					case v3low.ExamplesLabel:
						if components.Examples != nil {
							pr.name = checkForCollision(componentName, delim, pr, components.Examples)
							pr.location = []string{v3low.ComponentsLabel, v3low.ExamplesLabel, pr.name}
							return checkReferenceAndCapture(pr.name, delim, v3low.ExamplesLabel, pr, idx, components.Examples, buildExample, cf.origins)
						}
					case v3low.LinksLabel:
						if components.Links != nil {
							pr.name = checkForCollision(componentName, delim, pr, components.Links)
							pr.location = []string{v3low.ComponentsLabel, v3low.LinksLabel, pr.name}
							return checkReferenceAndCapture(pr.name, delim, v3low.LinksLabel, pr, idx, components.Links, buildLink, cf.origins)
						}
					case v3low.CallbacksLabel:
						if components.Callbacks != nil {
							pr.name = checkForCollision(componentName, delim, pr, components.Callbacks)
							pr.location = []string{v3low.ComponentsLabel, v3low.CallbacksLabel, pr.name}
							return checkReferenceAndCapture(pr.name, delim, v3low.CallbacksLabel, pr, idx, components.Callbacks, buildCallback, cf.origins)
						}
					case v3low.PathItemsLabel:
						if components.PathItems != nil {
							pr.name = checkForCollision(componentName, delim, pr, components.PathItems)
							pr.location = []string{v3low.ComponentsLabel, v3low.PathItemsLabel, pr.name}
							return checkReferenceAndCapture(pr.name, delim, v3low.PathItemsLabel, pr, idx, components.PathItems, buildPathItem, cf.origins)
						}
					}
				}
			}
			// type detection failed or multi-segment non-component path - inline instead
			unknown(pr, cf)
		}
	} else {
		unknown(pr, cf)
	}
	return nil
}
