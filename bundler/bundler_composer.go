// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package bundler

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"sync"

	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	v3low "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"go.yaml.in/yaml/v4"
)

type processRef struct {
	idx               *index.SpecIndex
	ref               *index.Reference
	seqRef            *index.Reference
	refPointer        string
	name              string
	location          []string
	wasRenamed        bool   // true when component was renamed due to collision
	originalName      string // original name before collision renaming
	fromDiscriminator bool   // true if created from discriminator mapping (skip inline handling)
}

// discriminatorMappingWithContext stores a mapping node with its source index
// and the pre-computed canonical key for processedNodes lookup
type discriminatorMappingWithContext struct {
	node         *yaml.Node       // The YAML node containing the mapping value
	sourceIdx    *index.SpecIndex // The index where the mapping was found
	canonicalKey string           // Pre-computed key: ref.FullDefinition at enqueue time (before any mutation)
	targetIdx    *index.SpecIndex // The index where the resolved ref actually lives (may differ from sourceIdx)
}

type handleIndexConfig struct {
	idx                   *index.SpecIndex
	rootIdx               *index.SpecIndex
	model                 *v3.Document
	indexes               []*index.SpecIndex
	refMap                *orderedmap.Map[string, *processRef]
	seen                  sync.Map
	inlineRequired        []*processRef
	compositionConfig     *BundleCompositionConfig
	discriminatorMappings []*discriminatorMappingWithContext // mapping nodes with source context
	origins               ComponentOriginMap                 // component origins for navigation
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
			// Avoid recomposing components that resolve back to the root document.
			if c.rootIdx != nil && foundIndex.GetSpecAbsolutePath() == c.rootIdx.GetSpecAbsolutePath() {
				continue
			}
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
	"openapi":           true,
	"info":              true,
	"jsonSchemaDialect": true,
	"servers":           true,
	"paths":             true,
	"webhooks":          true,
	"components":        true,
	"security":          true,
	"tags":              true,
	"externalDocs":      true,
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

// enqueueDiscriminatorMappingTargets ensures mapping targets are composed into components.
// This handles cases where a schema is ONLY referenced via discriminator mapping.
func enqueueDiscriminatorMappingTargets(
	mappings []*discriminatorMappingWithContext,
	cf *handleIndexConfig,
	rootIdx *index.SpecIndex,
) {
	for _, mapping := range mappings {
		refValue := mapping.node.Value

		// Skip empty values
		if refValue == "" {
			continue
		}

		// Skip external URLs and URNs - they're not local refs to compose
		if strings.HasPrefix(refValue, "http://") ||
			strings.HasPrefix(refValue, "https://") ||
			strings.HasPrefix(refValue, "urn:") {
			continue
		}

		// Only skip #/ refs if we're in the ROOT index.
		// In external files, #/components/... refers to THAT file's components,
		// which must still be composed into the root document.
		if strings.HasPrefix(refValue, "#/") && mapping.sourceIdx == rootIdx {
			continue
		}

		// Resolve using source index context
		ref, foundIdx := mapping.sourceIdx.SearchIndexForReference(refValue)
		if ref == nil {
			ref, foundIdx = resolveDiscriminatorMappingTarget(mapping.sourceIdx, refValue)
		}
		if ref == nil {
			continue // Can't resolve - will be caught later
		}

		// Cache the canonical key and target index NOW, before any mutation can occur.
		// This key will be used later in updateDiscriminatorMappingsComposed
		// to correctly look up the processedNodes entry.
		// The targetIdx may differ from sourceIdx if the ref resolves to a different file.
		mapping.canonicalKey = ref.FullDefinition
		mapping.targetIdx = foundIdx

		// Skip if already in refMap (avoid duplicates)
		if cf.refMap.GetOrZero(ref.FullDefinition) != nil {
			continue
		}

		// Derive name from reference - use ref.Name if set, otherwise extract from FullDefinition
		name := ref.Name
		if name == "" {
			name = deriveNameFromFullDefinition(ref.FullDefinition)
		}

		pr := &processRef{
			ref:               ref,
			seqRef:            ref,
			idx:               foundIdx,
			name:              name,
			fromDiscriminator: true, // NEW FLAG: prevents inline handling
		}
		cf.refMap.Set(ref.FullDefinition, pr)
	}
}

// resolveDiscriminatorMappingTarget attempts to resolve a mapping value as a whole-file reference.
// This is a fallback for cases where SearchIndexForReference returns nil for bare file refs.
func resolveDiscriminatorMappingTarget(
	sourceIdx *index.SpecIndex,
	refValue string,
) (*index.Reference, *index.SpecIndex) {
	if sourceIdx == nil {
		return nil, nil
	}

	rolodex := sourceIdx.GetRolodex()
	if rolodex == nil {
		return nil, nil
	}

	absPath := refValue
	if !filepath.IsAbs(absPath) && !strings.HasPrefix(absPath, "http") {
		base := sourceIdx.GetSpecAbsolutePath()
		if base == "" {
			if cfg := sourceIdx.GetConfig(); cfg != nil {
				base = cfg.BasePath
			}
		}
		if base != "" && filepath.Ext(base) != "" {
			base = filepath.Dir(base)
		}
		if base != "" {
			if p, err := filepath.Abs(utils.CheckPathOverlap(base, refValue, string(filepath.Separator))); err == nil {
				absPath = p
			}
		}
	}

	rFile, err := rolodex.OpenWithContext(context.Background(), absPath)
	if err != nil || rFile == nil {
		return nil, nil
	}

	if rFile.GetIndex() == nil {
		if cfg := sourceIdx.GetConfig(); cfg != nil {
			if idxFile, ok := rFile.(index.CanBeIndexed); ok {
				_, _ = idxFile.Index(cfg)
			}
		}
	}

	idx := rFile.GetIndex()
	node, _ := rFile.GetContentAsYAMLNode()
	if node != nil && node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}

	ref := &index.Reference{
		FullDefinition: absPath,
		Definition:     absPath,
		Name:           filepath.Base(absPath),
		Index:          idx,
		Node:           node,
		IsRemote:       true,
		RemoteLocation: absPath,
	}

	return ref, idx
}

// handleDiscriminatorMappingIndexes ensures indexes discovered only via discriminator mappings
// are explored so their internal refs are composed.
func handleDiscriminatorMappingIndexes(
	cf *handleIndexConfig,
	rootIdx *index.SpecIndex,
	rolodex *index.Rolodex,
) error {
	for _, mapping := range cf.discriminatorMappings {
		if mapping.targetIdx == nil {
			continue
		}
		if mapping.targetIdx == rootIdx {
			continue
		}
		if _, ok := cf.seen.Load(mapping.targetIdx.GetSpecAbsolutePath()); ok {
			continue
		}

		// Refresh indexes in case new ones were loaded during mapping resolution.
		if rolodex != nil {
			cf.indexes = rolodex.GetIndexes()
		}

		cf.idx = mapping.targetIdx
		if err := handleIndex(cf); err != nil {
			return err
		}
	}
	cf.idx = rootIdx
	return nil
}

func deriveNameFromFullDefinition(fullDef string) string {
	if idx := strings.Index(fullDef, "#"); idx != -1 {
		fullDef = fullDef[:idx]
	}
	baseName := filepath.Base(fullDef)
	return strings.TrimSuffix(baseName, filepath.Ext(baseName))
}
