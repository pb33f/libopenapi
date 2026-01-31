// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io
// SPDX-License-Identifier: MIT

package bundler

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"go.yaml.in/yaml/v4"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
)

// ErrInvalidModel is returned when the model is not usable.
var ErrInvalidModel = errors.New("invalid model")

// buildV3ModelFromBytes is a helper that parses bytes and builds a v3 model.
// Returns the model and any build errors. The model may be non-nil even when err is non-nil
// (e.g., circular reference warnings), allowing bundling to proceed with warnings.
func buildV3ModelFromBytes(bytes []byte, configuration *datamodel.DocumentConfiguration) (*v3.Document, error) {
	doc, err := libopenapi.NewDocumentWithConfiguration(bytes, configuration)
	if err != nil {
		return nil, err
	}

	v3Doc, buildErr := doc.BuildV3Model()
	if v3Doc == nil {
		return nil, errors.Join(ErrInvalidModel, buildErr)
	}
	// Return both model and error - caller decides how to handle warnings/errors
	return &v3Doc.Model, buildErr
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
	model, err := buildV3ModelFromBytes(bytes, configuration)
	if model == nil {
		return nil, err
	}

	bundledBytes, e := bundleWithConfig(model, nil, configuration)
	return bundledBytes, errors.Join(err, e)
}

// BundleBytesComposed will take a byte slice of an OpenAPI specification and return a composed bundled version of it.
// this is the same as BundleBytes, but it will compose the bundling instead of inline it.
//
// Composed means that every external file will have references lifted out and added to the `components` section of the document.
// Names will be preserved where possible, conflicts will dealt with by using a delimiter and appending a number.
func BundleBytesComposed(bytes []byte, configuration *datamodel.DocumentConfiguration, compositionConfig *BundleCompositionConfig) ([]byte, error) {
	doc, err := libopenapi.NewDocumentWithConfiguration(bytes, configuration)
	if err != nil {
		return nil, err
	}

	v3Doc, err := doc.BuildV3Model()
	if v3Doc == nil || err != nil {
		return nil, errors.Join(ErrInvalidModel, err)
	}

	bundledBytes, e := compose(&v3Doc.Model, compositionConfig)
	return bundledBytes, errors.Join(err, e)
}

// BundleBytesComposedWithOrigins returns a bundled spec with origin tracking for navigation.
// This enables consumers to map bundled components back to their original file locations.
func BundleBytesComposedWithOrigins(bytes []byte, configuration *datamodel.DocumentConfiguration, compositionConfig *BundleCompositionConfig) (*BundleResult, error) {
	doc, err := libopenapi.NewDocumentWithConfiguration(bytes, configuration)
	if err != nil {
		return nil, err
	}

	v3Doc, err := doc.BuildV3Model()
	if v3Doc == nil || err != nil {
		return nil, errors.Join(ErrInvalidModel, err)
	}

	result, e := composeWithOrigins(&v3Doc.Model, compositionConfig)
	return result, errors.Join(err, e)
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
	return bundleWithConfig(model, nil, nil)
}

// BundleBytesWithConfig will take a byte slice of an OpenAPI specification and return a bundled version of it,
// with additional configuration options for inline bundling behavior.
//
// Use the BundleInlineConfig to enable features like ResolveDiscriminatorExternalRefs which copies external
// schemas referenced by discriminator mappings to the root document's components section.
func BundleBytesWithConfig(bytes []byte, configuration *datamodel.DocumentConfiguration, bundleConfig *BundleInlineConfig) ([]byte, error) {
	model, err := buildV3ModelFromBytes(bytes, configuration)
	if model == nil {
		return nil, err
	}

	bundledBytes, e := bundleWithConfig(model, bundleConfig, configuration)
	return bundledBytes, errors.Join(err, e)
}

// BundleDocumentWithConfig will take a v3.Document and return a bundled version of it,
// with additional configuration options for inline bundling behavior.
//
// Use the BundleInlineConfig to enable features like ResolveDiscriminatorExternalRefs which copies external
// schemas referenced by discriminator mappings to the root document's components section.
func BundleDocumentWithConfig(model *v3.Document, bundleConfig *BundleInlineConfig) ([]byte, error) {
	return bundleWithConfig(model, bundleConfig, nil)
}

// BundleCompositionConfig is used to configure the composition of OpenAPI documents when using BundleDocumentComposed.
type BundleCompositionConfig struct {
	Delimiter           string // Delimiter is used to separate clashing names. Defaults to `__`.
	StrictValidation    bool   // StrictValidation will cause bundling to fail on invalid OpenAPI specs (e.g. $ref with siblings)
}

// BundleInlineConfig provides configuration options for inline bundling.
//
// Example usage:
//   // Inline everything including local refs
//   inlineTrue := true
//   config := &BundleInlineConfig{
//       InlineLocalRefs: &inlineTrue,
//   }
//   bundled, err := BundleBytesWithConfig(specBytes, docConfig, config)
type BundleInlineConfig struct {
	// ResolveDiscriminatorExternalRefs when true, copies external schemas referenced
	// by discriminator mappings to the root document's components section.
	// This ensures the bundled output is valid and self-contained when discriminators
	// in external files reference other schemas in those external files.
	// Default: false (preserves existing behavior of keeping external refs as-is)
	ResolveDiscriminatorExternalRefs bool

	// InlineLocalRefs controls whether local component references are inlined during bundling.
	// When nil, falls back to DocumentConfiguration.BundleInlineRefs.
	// - false: preserve local refs like #/components/schemas/Pet (discriminator-safe, default behavior)
	// - true: inline all refs including local component refs
	// Default: nil (uses DocumentConfiguration.BundleInlineRefs)
	InlineLocalRefs *bool
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

// composeWithOrigins performs composed bundling and returns origin tracking information
func composeWithOrigins(model *v3.Document, compositionConfig *BundleCompositionConfig) (*BundleResult, error) {
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

	discriminatorMappings := collectDiscriminatorMappingNodes(rolodex)

	cf := &handleIndexConfig{
		idx:                   rolodex.GetRootIndex(),
		model:                 model,
		indexes:               indexes,
		seen:                  sync.Map{},
		refMap:                orderedmap.New[string, *processRef](),
		compositionConfig:     compositionConfig,
		discriminatorMappings: discriminatorMappings,
		origins:               make(ComponentOriginMap),
	}
	if err := handleIndex(cf); err != nil {
		return nil, err
	}

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

	rootIndex := rolodex.GetRootIndex()
	remapIndex(rootIndex, processedNodes)

	for _, idx := range indexes {
		remapIndex(idx, processedNodes)
	}

	updateDiscriminatorMappingsComposed(discriminatorMappings, processedNodes, rolodex)

	// anything that could not be recomposed and needs inlining
	inlinedPaths := make(map[string]*yaml.Node)
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
					// Track this inlined content for reuse
					if pr.ref != nil {
						inlinedPaths[pr.ref.FullDefinition] = pointerRef.Node
					}
					continue
				}
			}
		}
		pr.seqRef.Node.Content = pr.ref.Node.Content
		// Track this inlined content for reuse
		if pr.ref != nil {
			inlinedPaths[pr.ref.FullDefinition] = pr.ref.Node
		}
	}

	// Fix any remaining absolute path references that match inlined content
	// Also check the root index
	allIndexes := append(indexes, rolodex.GetRootIndex())
	for _, idx := range allIndexes {
		for _, seqRef := range idx.GetRawReferencesSequenced() {
			if isRef, _, refVal := utils.IsNodeRefValue(seqRef.Node); isRef {
				// Check if this is an absolute path that should have been inlined
				if filepath.IsAbs(refVal) {
					// Try to find matching inlined content
					for inlinedPath, inlinedNode := range inlinedPaths {
						// Match if paths are the same or if they refer to the same file
						if refVal == inlinedPath {
							seqRef.Node.Content = inlinedNode.Content
							break
						}
					}
				}
			}
		}
	}

	b, err := model.Render()
	errs = append(errs, err)

	result := &BundleResult{
		Bytes:   b,
		Origins: cf.origins,
	}

	return result, errors.Join(errs...)
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

	discriminatorMappings := collectDiscriminatorMappingNodes(rolodex)

	cf := &handleIndexConfig{
		idx:                   rolodex.GetRootIndex(),
		model:                 model,
		indexes:               indexes,
		seen:                  sync.Map{},
		refMap:                orderedmap.New[string, *processRef](),
		compositionConfig:     compositionConfig,
		discriminatorMappings: discriminatorMappings,
		origins:               make(ComponentOriginMap),
	}
	if err := handleIndex(cf); err != nil {
		return nil, err
	}

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

	rootIndex := rolodex.GetRootIndex()
	remapIndex(rootIndex, processedNodes)

	for _, idx := range indexes {
		remapIndex(idx, processedNodes)
	}

	updateDiscriminatorMappingsComposed(discriminatorMappings, processedNodes, rolodex)

	// anything that could not be recomposed and needs inlining
	inlinedPaths := make(map[string]*yaml.Node)
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
					// Track this inlined content for reuse
					if pr.ref != nil {
						inlinedPaths[pr.ref.FullDefinition] = pointerRef.Node
					}
					continue
				}
			}
		}
		pr.seqRef.Node.Content = pr.ref.Node.Content
		// Track this inlined content for reuse
		if pr.ref != nil {
			inlinedPaths[pr.ref.FullDefinition] = pr.ref.Node
		}
	}

	// Fix any remaining absolute path references that match inlined content
	// Also check the root index
	allIndexes := append(indexes, rolodex.GetRootIndex())
	for _, idx := range allIndexes {
		for _, seqRef := range idx.GetRawReferencesSequenced() {
			if isRef, _, refVal := utils.IsNodeRefValue(seqRef.Node); isRef {
				// Check if this is an absolute path that should have been inlined
				if filepath.IsAbs(refVal) {
					// Try to find matching inlined content
					for inlinedPath, inlinedNode := range inlinedPaths {
						// Match if paths are the same or if they refer to the same file
						if refVal == inlinedPath {
							seqRef.Node.Content = inlinedNode.Content
							break
						}
					}
				}
			}
		}
	}

	b, err := model.Render()
	errs = append(errs, err)

	return b, errors.Join(errs...)
}

// resolveBundleInlineConfig resolves the inlineLocalRefs setting from the fallback chain:
// 1. BundleInlineConfig.InlineLocalRefs (explicit per-call)
// 2. DocumentConfiguration.BundleInlineRefs (document-wide default)
// 3. false (system default - preserve local refs)
func resolveBundleInlineConfig(bundleConfig *BundleInlineConfig, docConfig *datamodel.DocumentConfiguration) bool {
	if bundleConfig != nil && bundleConfig.InlineLocalRefs != nil {
		return *bundleConfig.InlineLocalRefs
	}
	if docConfig != nil {
		return docConfig.BundleInlineRefs
	}
	return false // system default
}

func bundleWithConfig(model *v3.Document, config *BundleInlineConfig, docConfig *datamodel.DocumentConfiguration) ([]byte, error) {
	if model == nil {
		return nil, errors.New("model cannot be nil")
	}

	inlineLocalRefs := resolveBundleInlineConfig(config, docConfig)

	// enable bundling mode to preserve local component refs during marshalling
	// when inlineLocalRefs is true, skip bundling mode to inline everything
	if !inlineLocalRefs {
		highbase.SetBundlingMode(true)
		defer highbase.SetBundlingMode(false)
	}

	if model.Rolodex != nil {
		// copy external schemas referenced by discriminator mappings to root components
		// ensures bundled output is valid and self-contained
		if config != nil && config.ResolveDiscriminatorExternalRefs {
			resolveDiscriminatorExternalRefs(model)
		}

		// resolve extension refs before rendering (mutates model's extension nodes in-place)
		// extensions are raw yaml nodes that bypass MarshalYAMLInline()
		resolveExtensionRefs(model.Rolodex)
	}

	// render inline - discriminator mappings and circular refs are preserved via SchemaProxy.MarshalYAMLInline()
	return model.RenderInline()
}

// externalSchemaRef represents an external schema that needs to be copied to the root document's components.
type externalSchemaRef struct {
	idx         *index.SpecIndex  // Source index where the schema is defined
	ref         *index.Reference  // The reference object
	schemaName  string            // The target name in components
	fullDef     string            // The full definition path (e.g., "/path/to/file.yaml#/components/schemas/Cat")
	originalRef string            // The original reference string (e.g., "#/components/schemas/Cat")
}

// resolveDiscriminatorExternalRefs handles copying external schemas referenced by discriminators
// to the root document's components section and rewrites the references.
func resolveDiscriminatorExternalRefs(model *v3.Document) {
	if model == nil || model.Rolodex == nil {
		return
	}

	rolodex := model.Rolodex
	rootIdx := rolodex.GetRootIndex()

	// Collect all external schemas referenced by discriminators
	externalSchemas := collectExternalDiscriminatorSchemas(rolodex, rootIdx)
	if len(externalSchemas) == 0 {
		return
	}

	// Ensure model has Components (buildComponents always succeeds with valid rootIdx,
	// and rootIdx must be valid since collectExternalDiscriminatorSchemas would panic otherwise)
	if model.Components == nil {
		model.Components, _ = buildComponents(rootIdx)
	}

	// Build existing names map from current components for collision detection
	existingNames := make(map[string]bool)
	for pair := model.Components.Schemas.First(); pair != nil; pair = pair.Next() {
		existingNames[pair.Key()] = true
	}

	// Copy schemas to components and build ref mapping
	// We need to map both local refs (like #/components/schemas/Cat) and
	// external refs (like ./external.yaml#/components/schemas/Cat) to the new location
	refMapping := make(map[string]string)
	for _, extSchema := range externalSchemas {
		// externalSchemas has unique fullDef values (from map iteration in collectExternalDiscriminatorSchemas)
		newRef := copySchemaToComponents(model, extSchema, existingNames)

		// Map the local ref format (used in external files)
		refMapping[extSchema.originalRef] = newRef

		// Also map external ref formats that might be used in the root document
		// e.g., "./vehicles/car.yaml#/components/schemas/Car"
		// The external ref format is: relative path from root + JSON pointer
		if extSchema.idx != nil {
			rootPath := rootIdx.GetSpecAbsolutePath()
			extPath := extSchema.idx.GetSpecAbsolutePath()
			if rootPath != "" && extPath != "" {
				// Calculate relative path from root to external file
				relPath, err := filepath.Rel(filepath.Dir(rootPath), extPath)
				if err == nil {
					// Normalize path separators to forward slashes for cross-platform compatibility
					// OpenAPI refs always use forward slashes regardless of OS
					relPath = filepath.ToSlash(relPath)

					// Build external ref format: ./relpath#/components/schemas/Name
					externalRefFormat := relPath + extSchema.originalRef
					refMapping[externalRefFormat] = newRef
					// Also try with "./" prefix
					if !strings.HasPrefix(relPath, ".") && !strings.HasPrefix(relPath, "/") {
						refMapping["./"+externalRefFormat] = newRef
					}
				}
			}
		}
	}

	// Rewrite discriminator mapping refs and oneOf/anyOf refs
	rewriteInlineDiscriminatorRefs(rolodex, refMapping)
}

// collectExternalDiscriminatorSchemas identifies external schemas referenced by discriminators
// that need to be copied to the root document's components section.
func collectExternalDiscriminatorSchemas(rolodex *index.Rolodex, rootIdx *index.SpecIndex) []*externalSchemaRef {
	var result []*externalSchemaRef

	// Use existing infrastructure to collect pinned refs
	pinned := make(map[string]struct{})

	// Collect from all indexes (root and external)
	collectDiscriminatorMappingValues(rootIdx, rootIdx.GetRootNode(), pinned)
	for _, idx := range rolodex.GetIndexes() {
		collectDiscriminatorMappingValues(idx, idx.GetRootNode(), pinned)
	}

	// Pre-build index lookup map for O(1) lookups instead of O(N) per ref
	indexByPath := make(map[string]*index.SpecIndex)
	for _, idx := range rolodex.GetIndexes() {
		indexByPath[idx.GetSpecAbsolutePath()] = idx
	}

	rootPath := rootIdx.GetSpecAbsolutePath()

	// Convert pinned refs to externalSchemaRef structs
	for fullDef := range pinned {
		// Parse the full definition to get the original ref
		// Format: "/absolute/path/to/file.yaml#/components/schemas/SchemaName"
		parts := strings.Split(fullDef, "#")
		filePath := parts[0]
		jsonPointer := "#" + parts[1]

		// Skip if this is from the root document (not external)
		if filePath == rootPath {
			continue
		}

		// find the index for this file using pre-built map (O(1) lookup)
		sourceIdx, ok := indexByPath[filePath]
		if !ok {
			// defensive: skip if index not found (shouldn't happen with valid specs)
			continue
		}

		// find the actual reference - this was already found when pinning
		ref, _ := sourceIdx.SearchIndexForReference(jsonPointer)

		// Extract schema name from the JSON pointer
		// e.g., "#/components/schemas/Cat" -> "Cat"
		pointerParts := strings.Split(strings.TrimPrefix(parts[1], "/"), "/")
		schemaName := pointerParts[len(pointerParts)-1]

		result = append(result, &externalSchemaRef{
			idx:         sourceIdx,
			ref:         ref,
			schemaName:  schemaName,
			fullDef:     fullDef,
			originalRef: jsonPointer,
		})
	}

	return result
}

// copySchemaToComponents copies an external schema to the root document's components section.
// Returns the new reference string (e.g., "#/components/schemas/Cat").
// existingNames is updated with the new name to track collisions across multiple calls.
func copySchemaToComponents(model *v3.Document, extSchema *externalSchemaRef, existingNames map[string]bool) string {
	// Build the schema from the YAML node
	// extSchema.ref.Node is always valid (validated when collecting external schemas)
	schema, _ := buildSchema(extSchema.ref.Node, extSchema.idx)

	// Check for naming collisions and get unique name
	finalName := extSchema.schemaName
	if existingNames[finalName] {
		finalName = calculateCollisionNameInline(finalName, extSchema.fullDef, "__", existingNames)
	}

	// Track this name to prevent future collisions
	existingNames[finalName] = true

	// Add to components
	model.Components.Schemas.Set(finalName, schema)

	return fmt.Sprintf("#/components/schemas/%s", finalName)
}

// calculateCollisionNameInline generates a unique name for a schema to avoid collisions.
// It first tries appending the source filename, then falls back to numeric suffixes.
func calculateCollisionNameInline(name, fullDef, delimiter string, existingNames map[string]bool) string {
	// Extract filename from the full definition path
	parts := strings.Split(fullDef, "#")
	filePath := parts[0]
	baseName := filepath.Base(filePath)
	// Remove extension
	baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))

	// Try filename-based name first
	candidate := fmt.Sprintf("%s%s%s", name, delimiter, baseName)
	if !existingNames[candidate] {
		return candidate
	}

	// If filename-based collision exists, try numeric suffixes
	for i := 1; ; i++ {
		candidate = fmt.Sprintf("%s%s%s%s%d", name, delimiter, baseName, delimiter, i)
		if !existingNames[candidate] {
			return candidate
		}
	}
}

// rewriteInlineDiscriminatorRefs updates discriminator mapping refs and oneOf/anyOf refs
// to point to the newly copied component locations.
func rewriteInlineDiscriminatorRefs(rolodex *index.Rolodex, refMapping map[string]string) {
	if len(refMapping) == 0 {
		return
	}

	// Collect all discriminator mapping nodes
	mappingNodes := collectDiscriminatorMappingNodes(rolodex)

	// Update discriminator mapping values
	for _, mappingNode := range mappingNodes {
		originalValue := mappingNode.Value
		if newRef, ok := refMapping[originalValue]; ok {
			mappingNode.Value = newRef
		}
	}

	// Also update oneOf/anyOf $ref values in all indexes
	allIndexes := append(rolodex.GetIndexes(), rolodex.GetRootIndex())
	for _, idx := range allIndexes {
		updateOneOfAnyOfRefs(idx.GetRootNode(), refMapping)
	}
}

// updateOneOfAnyOfRefs recursively walks a YAML node tree to update oneOf/anyOf $ref values.
func updateOneOfAnyOfRefs(n *yaml.Node, refMapping map[string]string) {
	if n == nil {
		return
	}

	if n.Kind == yaml.DocumentNode && len(n.Content) > 0 {
		n = n.Content[0]
	}

	switch n.Kind {
	case yaml.SequenceNode:
		for _, c := range n.Content {
			updateOneOfAnyOfRefs(c, refMapping)
		}
		return
	case yaml.MappingNode:
	default:
		return
	}

	var hasDiscriminator bool
	var oneOfNode, anyOfNode *yaml.Node

	// First pass: check for discriminator and find oneOf/anyOf
	for i := 0; i < len(n.Content); i += 2 {
		k, v := n.Content[i], n.Content[i+1]
		switch k.Value {
		case "discriminator":
			hasDiscriminator = true
		case "oneOf":
			oneOfNode = v
		case "anyOf":
			anyOfNode = v
		}
	}

	// Update refs in oneOf/anyOf if this schema has a discriminator
	if hasDiscriminator {
		updateUnionRefs(oneOfNode, refMapping)
		updateUnionRefs(anyOfNode, refMapping)
	}

	// Recursively process all children
	for i := 0; i < len(n.Content); i += 2 {
		updateOneOfAnyOfRefs(n.Content[i+1], refMapping)
	}
}

// updateUnionRefs updates $ref values in a oneOf or anyOf sequence.
func updateUnionRefs(seq *yaml.Node, refMapping map[string]string) {
	if seq == nil || seq.Kind != yaml.SequenceNode {
		return
	}
	for _, item := range seq.Content {
		if item.Kind != yaml.MappingNode {
			continue
		}
		for i := 0; i < len(item.Content); i += 2 {
			k, v := item.Content[i], item.Content[i+1]
			if k.Value == "$ref" && v.Kind == yaml.ScalarNode {
				if newRef, ok := refMapping[v.Value]; ok {
					v.Value = newRef
				}
			}
		}
	}
}

func collectDiscriminatorMappingValues(idx *index.SpecIndex, n *yaml.Node, pinned map[string]struct{}) {
	if n.Kind == yaml.DocumentNode && len(n.Content) > 0 {
		n = n.Content[0]
	}

	switch n.Kind {
	case yaml.SequenceNode:
		for _, c := range n.Content {
			collectDiscriminatorMappingValues(idx, c, pinned)
		}
		return
	case yaml.MappingNode:
	default:
		return
	}

	var discriminator, oneOf, anyOf *yaml.Node

	for i := 0; i < len(n.Content); i += 2 {
		k, v := n.Content[i], n.Content[i+1]
		switch k.Value {
		case "discriminator":
			discriminator = v
		case "oneOf":
			oneOf = v
		case "anyOf":
			anyOf = v
		}
		collectDiscriminatorMappingValues(idx, v, pinned)
	}

	if discriminator != nil {
		walkDiscriminatorMapping(idx, discriminator, pinned)
		walkUnionRefs(idx, oneOf, pinned)
		walkUnionRefs(idx, anyOf, pinned)
	}
}

func walkDiscriminatorMapping(idx *index.SpecIndex, discriminatorNode *yaml.Node, pinned map[string]struct{}) {
	if discriminatorNode.Kind != yaml.MappingNode {
		return
	}

	for i := 0; i < len(discriminatorNode.Content); i += 2 {
		if discriminatorNode.Content[i].Value == "mapping" {
			mappingNode := discriminatorNode.Content[i+1]

			for j := 0; j < len(mappingNode.Content); j += 2 {
				refValue := mappingNode.Content[j+1].Value

				if ref, refIdx := idx.SearchIndexForReference(refValue); ref != nil {
					fullDef := fmt.Sprintf("%s%s", refIdx.GetSpecAbsolutePath(), ref.Definition)
					pinned[fullDef] = struct{}{}
				}
			}
		}
	}
}

func walkUnionRefs(idx *index.SpecIndex, seq *yaml.Node, pinned map[string]struct{}) {
	if seq == nil || seq.Kind != yaml.SequenceNode {
		return
	}
	for _, item := range seq.Content {
		if item.Kind != yaml.MappingNode {
			continue
		}
		for i := 0; i < len(item.Content); i += 2 {
			k, v := item.Content[i], item.Content[i+1]
			if k.Value != "$ref" || v.Kind != yaml.ScalarNode {
				continue
			}
			if ref, refIdx := idx.SearchIndexForReference(v.Value); ref != nil {
				full := fmt.Sprintf("%s%s", refIdx.GetSpecAbsolutePath(), ref.Definition)
				pinned[full] = struct{}{}
			}
		}
	}
}

// collectDiscriminatorMappingNodes gathers all discriminator mapping value nodes from the document tree.
func collectDiscriminatorMappingNodes(rolodex *index.Rolodex) []*yaml.Node {
	var mappingNodes []*yaml.Node

	collectDiscriminatorMappingNodesFromIndex(rolodex.GetRootIndex(), rolodex.GetRootIndex().GetRootNode(), &mappingNodes)
	for _, idx := range rolodex.GetIndexes() {
		collectDiscriminatorMappingNodesFromIndex(idx, idx.GetRootNode(), &mappingNodes)
	}

	return mappingNodes
}

// collectDiscriminatorMappingNodesFromIndex recursively walks a YAML node tree to find discriminator mapping nodes.
func collectDiscriminatorMappingNodesFromIndex(idx *index.SpecIndex, n *yaml.Node, mappingNodes *[]*yaml.Node) {
	if n.Kind == yaml.DocumentNode && len(n.Content) > 0 {
		n = n.Content[0]
	}

	switch n.Kind {
	case yaml.SequenceNode:
		for _, c := range n.Content {
			collectDiscriminatorMappingNodesFromIndex(idx, c, mappingNodes)
		}
		return
	case yaml.MappingNode:
	default:
		return
	}

	var discriminator *yaml.Node

	for i := 0; i < len(n.Content); i += 2 {
		k, v := n.Content[i], n.Content[i+1]
		switch k.Value {
		case "discriminator":
			discriminator = v
		}
		collectDiscriminatorMappingNodesFromIndex(idx, v, mappingNodes)
	}

	if discriminator != nil && discriminator.Kind == yaml.MappingNode {
		for i := 0; i < len(discriminator.Content); i += 2 {
			if discriminator.Content[i].Value == "mapping" {
				mappingNode := discriminator.Content[i+1]
				for j := 0; j < len(mappingNode.Content); j += 2 {
					*mappingNodes = append(*mappingNodes, mappingNode.Content[j+1])
				}
			}
		}
	}
}

// updateDiscriminatorMappingsComposed updates discriminator mapping references to point to composed component locations.
func updateDiscriminatorMappingsComposed(mappingNodes []*yaml.Node, processedNodes *orderedmap.Map[string, *processRef], rolodex *index.Rolodex) {
	for _, mappingNode := range mappingNodes {
		originalValue := mappingNode.Value

		if !strings.Contains(originalValue, "#/") {
			continue
		}

		var matchingIdx *index.SpecIndex

		// Search root index first
		if ref, refIdx := rolodex.GetRootIndex().SearchIndexForReference(originalValue); ref != nil {
			matchingIdx = refIdx
		} else {
			// Search all other indexes
			for _, idx := range rolodex.GetIndexes() {
				if ref, refIdx := idx.SearchIndexForReference(originalValue); ref != nil {
					matchingIdx = refIdx
					break
				}
			}
		}

		if matchingIdx != nil {
			newRef := renameRef(matchingIdx, originalValue, processedNodes)
			if newRef != originalValue {
				mappingNode.Value = newRef
			}
		}
	}
}
