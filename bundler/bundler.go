// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io
// SPDX-License-Identifier: MIT

package bundler

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"gopkg.in/yaml.v3"
)

// discriminatorMapping tracks a discriminator mapping that references a schema
type discriminatorMapping struct {
	mappingNode *yaml.Node
	originalRef string
}

// ErrInvalidModel is returned when the model is not usable.
var ErrInvalidModel = errors.New("invalid model")

// discoverDiscriminatorMappings finds all discriminator mappings in schemas that reference other schemas
func discoverDiscriminatorMappings(idx *index.SpecIndex) map[string][]*discriminatorMapping {
	discriminatorMappings := make(map[string][]*discriminatorMapping)

	for _, schema := range idx.GetAllSchemas() {
		if schema.Node != nil {
			findDiscriminatorMappingsInNode(schema.Node, discriminatorMappings)
		}
	}

	return discriminatorMappings
}

// findDiscriminatorMappingsInNode recursively searches for discriminator mappings in a YAML node
func findDiscriminatorMappingsInNode(node *yaml.Node, mappings map[string][]*discriminatorMapping) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.MappingNode:
		processMappingNode(node, mappings)
	case yaml.SequenceNode:
		for _, child := range node.Content {
			findDiscriminatorMappingsInNode(child, mappings)
		}
	}
}

// processMappingNode handles discriminator discovery in mapping nodes
func processMappingNode(node *yaml.Node, mappings map[string][]*discriminatorMapping) {
	for i := 0; i < len(node.Content); i += 2 {
		if i+1 >= len(node.Content) {
			continue
		}

		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		if keyNode.Value == "discriminator" && valueNode.Kind == yaml.MappingNode {
			processDiscriminatorNode(valueNode, mappings)
		}

		// Recursively search in child nodes
		findDiscriminatorMappingsInNode(valueNode, mappings)
	}
}

// processDiscriminatorNode processes a discriminator node to extract mappings
func processDiscriminatorNode(discriminatorNode *yaml.Node, mappings map[string][]*discriminatorMapping) {
	for i := 0; i < len(discriminatorNode.Content); i += 2 {
		if i+1 >= len(discriminatorNode.Content) {
			continue
		}

		keyNode := discriminatorNode.Content[i]
		valueNode := discriminatorNode.Content[i+1]

		if keyNode.Value == "mapping" && valueNode.Kind == yaml.MappingNode {
			extractMappingReferences(valueNode, mappings)
		}
	}
}

// extractMappingReferences extracts all mapping references from a mapping node
func extractMappingReferences(mappingNode *yaml.Node, mappings map[string][]*discriminatorMapping) {
	for i := 0; i < len(mappingNode.Content); i += 2 {
		if i+1 >= len(mappingNode.Content) {
			continue
		}

		mappingValueNode := mappingNode.Content[i+1]
		if mappingValueNode.Value != "" {
			mapping := &discriminatorMapping{
				mappingNode: mappingValueNode,
				originalRef: mappingValueNode.Value,
			}
			mappings[mappingValueNode.Value] = append(mappings[mappingValueNode.Value], mapping)
		}
	}
}

// updateDiscriminatorMappings updates discriminator mappings when their referenced schemas are moved/inlined
func updateDiscriminatorMappings(mappings []*discriminatorMapping, newRef string) {
	for _, mapping := range mappings {
		if mapping.mappingNode != nil {
			mapping.mappingNode.Value = newRef
		}
	}
}

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

// BundleBytesComposed will take a byte slice of an OpenAPI specification and return a composed bundled version of it.
// this is the same as BundleBytes, but it will compose the bundling instead of inline it.
func BundleBytesComposed(bytes []byte, configuration *datamodel.DocumentConfiguration, compositionConfig *BundleCompositionConfig) ([]byte, error) {
	doc, err := libopenapi.NewDocumentWithConfiguration(bytes, configuration)
	if err != nil {
		return nil, err
	}

	v3Doc, errs := doc.BuildV3Model()
	err = errors.Join(errs...)
	if v3Doc == nil || len(errs) > 0 {
		return nil, errors.Join(ErrInvalidModel, err)
	}

	bundledBytes, e := compose(&v3Doc.Model, compositionConfig)
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

// BundleCompositionConfig is used to configure the composition of OpenAPI documents when using BundleDocumentComposed.
type BundleCompositionConfig struct {
	Delimiter string // Delimiter is used to separate clashing names. Defaults to `__`.
}

// BundleDocumentComposed will take a v3.Document and return a composed bundled version of it. Composed means
// that every external file will have references lifted out and added to the `components` section of the document.
// Names will be preserved where possible, conflicts will be appended with a number. If the type of the reference cannot
// be determined, it will be added to the `components` section as a `Schema` type, a warning will be logged.
// The document model will be mutated permanently.
//
// Circular references will not be resolved and will be skipped.
func BundleDocumentComposed(model *v3.Document, compositionConfig *BundleCompositionConfig) ([]byte, error) {
	return compose(model, compositionConfig)
}

func compose(model *v3.Document, compositionConfig *BundleCompositionConfig) ([]byte, error) {
	if compositionConfig == nil {
		compositionConfig = &BundleCompositionConfig{
			Delimiter: "__",
		}
	} else {
		if compositionConfig.Delimiter == "" {
			compositionConfig.Delimiter = "__"
		}
		if strings.Contains(compositionConfig.Delimiter, "#") ||
			strings.Contains(compositionConfig.Delimiter, "/") {
			return nil, errors.New("composition delimiter cannot contain '#' or '/' characters")
		}
		if strings.Contains(compositionConfig.Delimiter, " ") {
			return nil, errors.New("composition delimiter cannot contain spaces")
		}
	}

	if model == nil || model.Rolodex == nil {
		return nil, errors.New("model or rolodex is nil")
	}

	rolodex := model.Rolodex
	indexes := rolodex.GetIndexes()
	rootIndex := rolodex.GetRootIndex()

	// discover discriminator mappings across all indexes
	allDiscriminatorMappings := make(map[string][]*discriminatorMapping)

	// First check the root index
	rootMappings := discoverDiscriminatorMappings(rootIndex)
	for ref, mappingList := range rootMappings {
		allDiscriminatorMappings[ref] = append(allDiscriminatorMappings[ref], mappingList...)
	}

	// Then check all other indexes
	for _, idx := range indexes {
		mappings := discoverDiscriminatorMappings(idx)
		for ref, mappingList := range mappings {
			allDiscriminatorMappings[ref] = append(allDiscriminatorMappings[ref], mappingList...)
		}
	}

	cf := &handleIndexConfig{
		idx:                   rootIndex,
		model:                 model,
		indexes:               indexes,
		seen:                  sync.Map{},
		refMap:                orderedmap.New[string, *processRef](),
		compositionConfig:     compositionConfig,
		discriminatorMappings: allDiscriminatorMappings,
	}
	// recursive function to handle the indexes, we need a different approach to composition vs. inlining.
	handleIndex(cf)

	processedNodes := orderedmap.New[string, *processRef]()
	var errs []error
	for _, ref := range cf.refMap.FromOldest() {
		err := processReference(model, ref, cf)
		errs = append(errs, err)
		processedNodes.Set(ref.ref.FullDefinition, ref)
	}

	slices.SortFunc(indexes, func(i, j *index.SpecIndex) int {
		if i.GetSpecAbsolutePath() < j.GetSpecAbsolutePath() {
			return 1
		}
		return 0
	})

	remapIndex(rootIndex, processedNodes)

	for _, idx := range indexes {
		remapIndex(idx, processedNodes)
	}

	// update discriminator mappings after all references have been processed
	for originalRef, mappings := range allDiscriminatorMappings {
		if processedRef := processedNodes.GetOrZero(originalRef); processedRef != nil {
			var newRef string
			if len(processedRef.location) > 0 {
				newRef = "#/" + strings.Join(processedRef.location, "/")
			} else {
				newRef = "#/components/schemas/" + processedRef.name
			}
			updateDiscriminatorMappings(mappings, newRef)
		} else {
			// Check for external file references that may match processed nodes
			var bestMatch *processRef
			for _, processedRef := range processedNodes.FromOldest() {
				refExp := strings.Split(processedRef.ref.FullDefinition, "#/")
				if len(refExp) == 2 {
					externalFile := refExp[0]
					if externalFile != "" {
						matched := false

						if originalRef == externalFile || strings.HasPrefix(originalRef, externalFile+"#/") {
							matched = true
						}

						if !matched && strings.HasPrefix(originalRef, "./") {
							rootIndexPath := rolodex.GetRootIndex().GetSpecAbsolutePath()
							baseDir := filepath.Dir(rootIndexPath)

							mappingRefParts := strings.Split(originalRef, "#/")
							if len(mappingRefParts) == 2 {
								relativeFile := mappingRefParts[0]
								absFile := filepath.Join(baseDir, relativeFile)

								if absFile == externalFile || strings.HasPrefix(processedRef.ref.FullDefinition, absFile+"#/") {
									matched = true
								}
							}
						}

						if matched {
							if bestMatch == nil ||
								(len(processedRef.location) > len(bestMatch.location)) ||
								(len(processedRef.location) == len(bestMatch.location) &&
									len(processedRef.name) > len(bestMatch.name)) {
								bestMatch = processedRef
							}
						}
					}
				}
			}

			if bestMatch != nil {
				var newRef string
				if len(bestMatch.location) > 0 && len(bestMatch.location) >= 3 &&
					bestMatch.location[0] == "components" && bestMatch.location[1] == "schemas" {
					newRef = "#/" + strings.Join(bestMatch.location, "/")
				} else {
					componentsSchemaName := ""
					if model.Components != nil && model.Components.Schemas != nil {
						for schemaName := range model.Components.Schemas.FromOldest() {
							if strings.HasSuffix(schemaName, bestMatch.name) ||
								strings.Contains(schemaName, bestMatch.name) {
								componentsSchemaName = schemaName
								break
							}
						}
					}

					if componentsSchemaName != "" {
						newRef = "#/components/schemas/" + componentsSchemaName
					} else {
						newRef = "#/components/schemas/" + bestMatch.name
					}
				}
				updateDiscriminatorMappings(mappings, newRef)
			}
		}
	}

	// anything that could not be recomposed and needs inlining
	for _, pr := range cf.inlineRequired {
		if pr.refPointer != "" {

			// if the ref is a pointer to an external pointer, then we need to stitch it.
			uri := strings.Split(pr.refPointer, "#/")
			if len(uri) == 2 {
				if uri[0] != "" {
					if !filepath.IsAbs(uri[0]) && !strings.HasPrefix(uri[0], "http") {
						// if the uri is not absolute, then we need to make it absolute.
						uri[0] = filepath.Join(filepath.Dir(pr.idx.GetSpecAbsolutePath()), uri[0])
					}
					pointerRef := pr.idx.FindComponent(context.Background(), strings.Join(uri, "#/"))
					pr.seqRef.Node.Content = pointerRef.Node.Content
					continue
				}
			}
		}
		pr.seqRef.Node.Content = pr.ref.Node.Content
	}

	b, err := model.Render()
	errs = append(errs, err)

	return b, errors.Join(errs...)
}

func bundle(model *v3.Document) ([]byte, error) {
	rolodex := model.Rolodex
	indexes := rolodex.GetIndexes()
	rootIndex := rolodex.GetRootIndex()

	// discover discriminator mappings across all indexes
	allDiscriminatorMappings := make(map[string][]*discriminatorMapping)

	// First check the root index
	rootMappings := discoverDiscriminatorMappings(rootIndex)
	for ref, mappingList := range rootMappings {
		allDiscriminatorMappings[ref] = append(allDiscriminatorMappings[ref], mappingList...)
	}

	// Then check all other indexes
	for _, idx := range indexes {
		mappings := discoverDiscriminatorMappings(idx)
		for ref, mappingList := range mappings {
			allDiscriminatorMappings[ref] = append(allDiscriminatorMappings[ref], mappingList...)
		}
	}

	// compact function.
	compact := func(idx *index.SpecIndex, root bool) {
		mappedReferences := idx.GetMappedReferences()
		sequencedReferences := idx.GetRawReferencesSequenced()
		for _, sequenced := range sequencedReferences {
			mappedReference := mappedReferences[sequenced.FullDefinition]

			refExp := strings.Split(sequenced.FullDefinition, "#/")
			if len(refExp) == 2 {
				if root {
					for _, i := range indexes {
						if i.GetSpecAbsolutePath() == refExp[0] {
							if mappedReference != nil && !mappedReference.Circular {
								mr := i.FindComponent(context.Background(), sequenced.Definition)
								if mr != nil {
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
				hasDiscriminatorMapping := false

				if _, exists := allDiscriminatorMappings[sequenced.FullDefinition]; exists {
					hasDiscriminatorMapping = true
				}

				if !hasDiscriminatorMapping {
					refExp := strings.Split(sequenced.FullDefinition, "#/")
					if len(refExp) == 2 {
						externalFile := refExp[0]
						if externalFile != "" {
							for mappingRef, _ := range allDiscriminatorMappings {
								matched := false

								if mappingRef == externalFile || strings.HasPrefix(mappingRef, externalFile+"#/") {
									matched = true
								}

								if !matched && strings.HasPrefix(mappingRef, "./") {
									rootIndexPath := rolodex.GetRootIndex().GetSpecAbsolutePath()
									baseDir := filepath.Dir(rootIndexPath)

									mappingRefParts := strings.Split(mappingRef, "#/")
									if len(mappingRefParts) == 2 {
										relativeFile := mappingRefParts[0]
										absFile := filepath.Join(baseDir, relativeFile)

										if absFile == externalFile || strings.HasPrefix(sequenced.FullDefinition, absFile+"#/") {
											matched = true
										}
									}
								}

								if matched {
									hasDiscriminatorMapping = true
									break
								}
							}
						}
					}
				}

				if hasDiscriminatorMapping {
					if idx.GetLogger() != nil {
						idx.GetLogger().Debug("[bundler] preserving schema referenced by discriminator mapping",
							"ref", sequenced.Definition)
					}
					continue
				}

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

	// Update discriminator mappings to point to components after bundling
	for originalRef, mappings := range allDiscriminatorMappings {
		if strings.HasPrefix(originalRef, "./") || strings.Contains(originalRef, "/") {
			mappingRefParts := strings.Split(originalRef, "#/")
			if len(mappingRefParts) == 2 {
				fragmentName := mappingRefParts[1]

				if model.Components != nil && model.Components.Schemas != nil {
					for schemaName := range model.Components.Schemas.FromOldest() {
						if strings.Contains(schemaName, fragmentName) ||
							strings.HasSuffix(schemaName, fragmentName) {
							newRef := "#/components/schemas/" + schemaName
							updateDiscriminatorMappings(mappings, newRef)
							break
						}
					}
				}
			}
		}
	}

	return model.Render()
}
