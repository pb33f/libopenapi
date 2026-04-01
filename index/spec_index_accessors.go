// Copyright 2022-2033 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package index

import (
	"sort"

	"go.yaml.in/yaml/v4"
)

func (index *SpecIndex) SetCircularReferences(refs []*CircularReferenceResult) {
	index.circularReferences = refs
}

func (index *SpecIndex) GetCircularReferences() []*CircularReferenceResult {
	return index.circularReferences
}

func (index *SpecIndex) GetTagCircularReferences() []*CircularReferenceResult {
	return index.tagCircularReferences
}

func (index *SpecIndex) SetIgnoredPolymorphicCircularReferences(refs []*CircularReferenceResult) {
	index.polyCircularReferences = refs
}

func (index *SpecIndex) SetIgnoredArrayCircularReferences(refs []*CircularReferenceResult) {
	index.arrayCircularReferences = refs
}

func (index *SpecIndex) GetIgnoredPolymorphicCircularReferences() []*CircularReferenceResult {
	return index.polyCircularReferences
}

func (index *SpecIndex) GetIgnoredArrayCircularReferences() []*CircularReferenceResult {
	return index.arrayCircularReferences
}

func (index *SpecIndex) GetPathsNode() *yaml.Node {
	return index.pathsNode
}

func (index *SpecIndex) GetDiscoveredReferences() map[string]*Reference {
	return index.allRefs
}

func (index *SpecIndex) GetPolyReferences() map[string]*Reference {
	return index.polymorphicRefs
}

func (index *SpecIndex) GetPolyAllOfReferences() []*Reference {
	return index.polymorphicAllOfRefs
}

func (index *SpecIndex) GetPolyAnyOfReferences() []*Reference {
	return index.polymorphicAnyOfRefs
}

func (index *SpecIndex) GetPolyOneOfReferences() []*Reference {
	return index.polymorphicOneOfRefs
}

func (index *SpecIndex) GetAllCombinedReferences() map[string]*Reference {
	combined := make(map[string]*Reference)
	for k, ref := range index.allRefs {
		combined[k] = ref
	}
	for k, ref := range index.polymorphicRefs {
		combined[k] = ref
	}
	return combined
}

func (index *SpecIndex) GetRefsByLine() map[string]map[int]bool {
	return index.refsByLine
}

func (index *SpecIndex) GetLinesWithReferences() map[int]bool {
	return index.linesWithRefs
}

func (index *SpecIndex) GetMappedReferences() map[string]*Reference {
	return index.allMappedRefs
}

func (index *SpecIndex) SetMappedReferences(mappedRefs map[string]*Reference) {
	index.allMappedRefs = mappedRefs
}

func (index *SpecIndex) GetRawReferencesSequenced() []*Reference {
	return index.rawSequencedRefs
}

func (index *SpecIndex) GetExtensionRefsSequenced() []*Reference {
	var extensionRefs []*Reference
	for _, ref := range index.rawSequencedRefs {
		if ref.IsExtensionRef {
			extensionRefs = append(extensionRefs, ref)
		}
	}
	return extensionRefs
}

func (index *SpecIndex) GetMappedReferencesSequenced() []*ReferenceMapped {
	return index.allMappedRefsSequenced
}

func (index *SpecIndex) GetOperationParameterReferences() map[string]map[string]map[string][]*Reference {
	return index.paramOpRefs
}

func (index *SpecIndex) GetAllSchemas() []*Reference {
	componentSchemas := index.GetAllComponentSchemas()
	inlineSchemas := index.GetAllInlineSchemas()
	refSchemas := index.GetAllReferenceSchemas()
	combined := make([]*Reference, len(inlineSchemas)+len(componentSchemas)+len(refSchemas))
	i := 0
	for x := range inlineSchemas {
		combined[i] = inlineSchemas[x]
		i++
	}
	for x := range componentSchemas {
		combined[i] = componentSchemas[x]
		i++
	}
	for x := range refSchemas {
		combined[i] = refSchemas[x]
		i++
	}
	sort.Slice(combined, func(i, j int) bool {
		return combined[i].Node.Line < combined[j].Node.Line
	})
	return combined
}

func (index *SpecIndex) GetAllInlineSchemaObjects() []*Reference {
	return index.allInlineSchemaObjectDefinitions
}

func (index *SpecIndex) GetAllInlineSchemas() []*Reference {
	return index.allInlineSchemaDefinitions
}

func (index *SpecIndex) GetAllReferenceSchemas() []*Reference {
	return index.allRefSchemaDefinitions
}

func (index *SpecIndex) GetAllComponentSchemas() map[string]*Reference {
	if index == nil {
		return nil
	}
	index.allComponentSchemasLock.RLock()
	if index.allComponentSchemas != nil {
		defer index.allComponentSchemasLock.RUnlock()
		return index.allComponentSchemas
	}
	index.allComponentSchemasLock.RUnlock()

	index.allComponentSchemasLock.Lock()
	defer index.allComponentSchemasLock.Unlock()
	if index.allComponentSchemas == nil {
		index.allComponentSchemas = syncMapToMap[string, *Reference](index.allComponentSchemaDefinitions)
	}
	return index.allComponentSchemas
}

func (index *SpecIndex) GetAllSecuritySchemes() map[string]*Reference {
	return syncMapToMap[string, *Reference](index.allSecuritySchemes)
}

func (index *SpecIndex) GetAllHeaders() map[string]*Reference {
	return index.allHeaders
}

func (index *SpecIndex) GetAllExternalDocuments() map[string]*Reference {
	return index.allExternalDocuments
}

func (index *SpecIndex) GetAllExamples() map[string]*Reference {
	return index.allExamples
}

func (index *SpecIndex) GetAllDescriptions() []*DescriptionReference {
	return index.allDescriptions
}

func (index *SpecIndex) GetAllEnums() []*EnumReference {
	return index.allEnums
}

func (index *SpecIndex) GetAllObjectsWithProperties() []*ObjectReference {
	return index.allObjectsWithProperties
}

func (index *SpecIndex) GetAllSummaries() []*DescriptionReference {
	return index.allSummaries
}

func (index *SpecIndex) GetAllRequestBodies() map[string]*Reference {
	return index.allRequestBodies
}

func (index *SpecIndex) GetAllLinks() map[string]*Reference {
	return index.allLinks
}

func (index *SpecIndex) GetAllParameters() map[string]*Reference {
	return index.allParameters
}

func (index *SpecIndex) GetAllResponses() map[string]*Reference {
	return index.allResponses
}

func (index *SpecIndex) GetAllCallbacks() map[string]*Reference {
	return index.allCallbacks
}

func (index *SpecIndex) GetAllComponentPathItems() map[string]*Reference {
	return index.allComponentPathItems
}

func (index *SpecIndex) GetInlineOperationDuplicateParameters() map[string][]*Reference {
	return index.paramInlineDuplicateNames
}

func (index *SpecIndex) GetReferencesWithSiblings() map[string]Reference {
	return index.refsWithSiblings
}

func (index *SpecIndex) GetAllReferences() map[string]*Reference {
	return index.allRefs
}

func (index *SpecIndex) GetAllSequencedReferences() []*Reference {
	return index.rawSequencedRefs
}

func (index *SpecIndex) GetSchemasNode() *yaml.Node {
	return index.schemasNode
}

func (index *SpecIndex) GetParametersNode() *yaml.Node {
	return index.parametersNode
}

func (index *SpecIndex) GetReferenceIndexErrors() []error {
	return index.refErrors
}

func (index *SpecIndex) GetOperationParametersIndexErrors() []error {
	return index.operationParamErrors
}

func (index *SpecIndex) GetAllPaths() map[string]map[string]*Reference {
	return index.pathRefs
}

func (index *SpecIndex) GetOperationTags() map[string]map[string][]*Reference {
	return index.operationTagsRefs
}

func (index *SpecIndex) GetAllParametersFromOperations() map[string]map[string]map[string][]*Reference {
	return index.paramOpRefs
}

func (index *SpecIndex) GetRootSecurityReferences() []*Reference {
	return index.rootSecurity
}

func (index *SpecIndex) GetSecurityRequirementReferences() map[string]map[string][]*Reference {
	return index.securityRequirementRefs
}

func (index *SpecIndex) GetRootSecurityNode() *yaml.Node {
	return index.rootSecurityNode
}

func (index *SpecIndex) GetRootServersNode() *yaml.Node {
	return index.rootServersNode
}

func (index *SpecIndex) GetAllRootServers() []*Reference {
	return index.serversRefs
}

func (index *SpecIndex) GetAllOperationsServers() map[string]map[string][]*Reference {
	return index.opServersRefs
}

func (index *SpecIndex) SetAllowCircularReferenceResolving(allow bool) {
	index.allowCircularReferences = allow
}

func (index *SpecIndex) AllowCircularReferenceResolving() bool {
	return index.allowCircularReferences
}

func (index *SpecIndex) checkPolymorphicNode(name string) (bool, string) {
	switch name {
	case "anyOf":
		return true, "anyOf"
	case "allOf":
		return true, "allOf"
	case "oneOf":
		return true, "oneOf"
	}
	return false, ""
}

func (index *SpecIndex) RegisterSchemaId(entry *SchemaIdEntry) error {
	index.schemaIdRegistryLock.Lock()
	defer index.schemaIdRegistryLock.Unlock()
	if index.schemaIdRegistry == nil {
		index.schemaIdRegistry = make(map[string]*SchemaIdEntry)
	}
	_, err := registerSchemaIdToRegistry(index.schemaIdRegistry, entry, index.logger, "local index")
	return err
}

func (index *SpecIndex) GetSchemaById(uri string) *SchemaIdEntry {
	index.schemaIdRegistryLock.RLock()
	defer index.schemaIdRegistryLock.RUnlock()
	if index.schemaIdRegistry == nil {
		return nil
	}
	return index.schemaIdRegistry[uri]
}

func (index *SpecIndex) GetAllSchemaIds() map[string]*SchemaIdEntry {
	index.schemaIdRegistryLock.RLock()
	defer index.schemaIdRegistryLock.RUnlock()
	return copySchemaIdRegistry(index.schemaIdRegistry)
}
