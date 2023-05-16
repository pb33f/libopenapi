// Copyright 2022-2033 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

// Package index contains an OpenAPI indexer that will very quickly scan through an OpenAPI specification (all versions)
// and extract references to all the important nodes you might want to look up, as well as counts on total objects.
//
// When extracting references, the index can determine if the reference is local to the file (recommended) or the
// reference is located in another local file, or a remote file. The index will then attempt to load in those remote
// files and look up the references there, or continue following the chain.
//
// When the index loads in a local or remote file, it will also index that remote spec as well. This means everything
// is indexed and stored as a tree, depending on how deep the remote references go.
package index

import (
	"fmt"
	"github.com/pb33f/libopenapi/utils"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"golang.org/x/sync/syncmap"
	"gopkg.in/yaml.v3"
	"strings"
	"sync"
)

// NewSpecIndexWithConfig will create a new index of an OpenAPI or Swagger spec. It uses the same logic as NewSpecIndex
// except it sets a base URL for resolving relative references, except it also allows for granular control over
// how the index is set up.
func NewSpecIndexWithConfig(rootNode *yaml.Node, config *SpecIndexConfig) *SpecIndex {
	index := new(SpecIndex)
	if config != nil && config.seenRemoteSources == nil {
		config.seenRemoteSources = &syncmap.Map{}
	}
	config.remoteLock = &sync.Mutex{}
	index.config = config
	index.parentIndex = config.ParentIndex
	index.uri = config.uri
	if rootNode == nil || len(rootNode.Content) <= 0 {
		return index
	}
	boostrapIndexCollections(rootNode, index)
	return createNewIndex(rootNode, index)
}

// NewSpecIndex will create a new index of an OpenAPI or Swagger spec. It's not resolved or converted into anything
// other than a raw index of every node for every content type in the specification. This process runs as fast as
// possible so dependencies looking through the tree, don't need to walk the entire thing over, and over.
//
// Deprecated: Use NewSpecIndexWithConfig instead, this function will be removed in the future because it
// defaults to allowing remote references and file references. This is a potential security risk and should be controlled by
// providing a SpecIndexConfig that explicitly sets the AllowRemoteLookup and AllowFileLookup to true.
// This function also does not support specifications with relative references that may not exist locally.
//   - https://github.com/pb33f/libopenapi/issues/73
func NewSpecIndex(rootNode *yaml.Node) *SpecIndex {
	index := new(SpecIndex)
	index.config = CreateOpenAPIIndexConfig()
	boostrapIndexCollections(rootNode, index)
	return createNewIndex(rootNode, index)
}

func createNewIndex(rootNode *yaml.Node, index *SpecIndex) *SpecIndex {
	// there is no node! return an empty index.
	if rootNode == nil {
		return index
	}

	// boot index.
	results := index.ExtractRefs(index.root.Content[0], index.root, []string{}, 0, false, "")

	// map poly refs
	poly := make([]*Reference, len(index.polymorphicRefs))
	z := 0
	for i := range index.polymorphicRefs {
		poly[z] = index.polymorphicRefs[i]
		z++
	}

	// pull out references
	index.ExtractComponentsFromRefs(results)
	index.ExtractComponentsFromRefs(poly)

	index.ExtractExternalDocuments(index.root)
	index.GetPathCount()

	countFuncs := []func() int{
		index.GetOperationCount,
		index.GetComponentSchemaCount,
		index.GetGlobalTagsCount,
		index.GetComponentParameterCount,
		index.GetOperationsParameterCount,
	}

	var wg sync.WaitGroup
	wg.Add(len(countFuncs))
	runIndexFunction(countFuncs, &wg) // run as fast as we can.
	wg.Wait()

	// these functions are aggregate and can only run once the rest of the datamodel is ready
	countFuncs = []func() int{
		index.GetInlineUniqueParamCount,
		index.GetOperationTagsCount,
		index.GetGlobalLinksCount,
		index.GetGlobalCallbacksCount,
	}

	wg.Add(len(countFuncs))
	runIndexFunction(countFuncs, &wg) // run as fast as we can.
	wg.Wait()

	// these have final calculation dependencies
	index.GetInlineDuplicateParamCount()
	index.GetAllDescriptionsCount()
	index.GetTotalTagsCount()

	// do a copy!
	index.config.seenRemoteSources.Range(func(k, v any) bool {
		index.seenRemoteSources[k.(string)] = v.(*yaml.Node)
		return true
	})
	return index
}

// GetRootNode returns document root node.
func (index *SpecIndex) GetRootNode() *yaml.Node {
	return index.root
}

// GetGlobalTagsNode returns document root tags node.
func (index *SpecIndex) GetGlobalTagsNode() *yaml.Node {
	return index.tagsNode
}

// SetCircularReferences is a convenience method for the resolver to pass in circular references
// if the resolver is used.
func (index *SpecIndex) SetCircularReferences(refs []*CircularReferenceResult) {
	index.circularReferences = refs
}

// GetCircularReferences will return any circular reference results that were found by the resolver.
func (index *SpecIndex) GetCircularReferences() []*CircularReferenceResult {
	return index.circularReferences
}

// GetPathsNode returns document root node.
func (index *SpecIndex) GetPathsNode() *yaml.Node {
	return index.pathsNode
}

// GetDiscoveredReferences will return all unique references found in the spec
func (index *SpecIndex) GetDiscoveredReferences() map[string]*Reference {
	return index.allRefs
}

// GetPolyReferences will return every polymorphic reference in the doc
func (index *SpecIndex) GetPolyReferences() map[string]*Reference {
	return index.polymorphicRefs
}

// GetPolyAllOfReferences will return every 'allOf' polymorphic reference in the doc
func (index *SpecIndex) GetPolyAllOfReferences() []*Reference {
	return index.polymorphicAllOfRefs
}

// GetPolyAnyOfReferences will return every 'anyOf' polymorphic reference in the doc
func (index *SpecIndex) GetPolyAnyOfReferences() []*Reference {
	return index.polymorphicAnyOfRefs
}

// GetPolyOneOfReferences will return every 'allOf' polymorphic reference in the doc
func (index *SpecIndex) GetPolyOneOfReferences() []*Reference {
	return index.polymorphicOneOfRefs
}

// GetAllCombinedReferences will return the number of unique and polymorphic references discovered.
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

// GetRefsByLine will return all references and the lines at which they were found.
func (index *SpecIndex) GetRefsByLine() map[string]map[int]bool {
	return index.refsByLine
}

// GetLinesWithReferences will return a map of lines that have a $ref
func (index *SpecIndex) GetLinesWithReferences() map[int]bool {
	return index.linesWithRefs
}

// GetMappedReferences will return all references that were mapped successfully to actual property nodes.
// this collection is completely unsorted, traversing it may produce random results when resolving it and
// encountering circular references can change results depending on where in the collection the resolver started
// its journey through the index.
func (index *SpecIndex) GetMappedReferences() map[string]*Reference {
	return index.allMappedRefs
}

// GetMappedReferencesSequenced will return all references that were mapped successfully to nodes, performed in sequence
// as they were read in from the document.
func (index *SpecIndex) GetMappedReferencesSequenced() []*ReferenceMapped {
	return index.allMappedRefsSequenced
}

// GetOperationParameterReferences will return all references to operation parameters
func (index *SpecIndex) GetOperationParameterReferences() map[string]map[string]map[string][]*Reference {
	return index.paramOpRefs
}

// GetAllSchemas will return references to all schemas found in the document both inline and those under components
// The first elements of at the top of the slice, are all the inline references (using GetAllInlineSchemas),
// and then following on are all the references extracted from the components section (using GetAllComponentSchemas).
func (index *SpecIndex) GetAllSchemas() []*Reference {

	componentSchemas := index.GetAllComponentSchemas()
	inlineSchemas := index.GetAllInlineSchemas()

	combined := make([]*Reference, len(inlineSchemas)+len(componentSchemas))
	i := 0
	for x := range inlineSchemas {
		combined[i] = inlineSchemas[x]
		i++
	}
	for x := range componentSchemas {
		combined[i] = componentSchemas[x]
		i++
	}
	return combined
}

// GetAllInlineSchemaObjects will return all schemas that are inline (not inside components) and that are also typed
// as 'object' or 'array' (not primitives).
func (index *SpecIndex) GetAllInlineSchemaObjects() []*Reference {
	return index.allInlineSchemaObjectDefinitions
}

// GetAllInlineSchemas will return all schemas defined in the components section of the document.
func (index *SpecIndex) GetAllInlineSchemas() []*Reference {
	return index.allInlineSchemaDefinitions
}

// GetAllComponentSchemas will return all schemas defined in the components section of the document.
func (index *SpecIndex) GetAllComponentSchemas() map[string]*Reference {
	return index.allComponentSchemaDefinitions
}

// GetAllSecuritySchemes will return all security schemes / definitions found in the document.
func (index *SpecIndex) GetAllSecuritySchemes() map[string]*Reference {
	return index.allSecuritySchemes
}

// GetAllHeaders will return all headers found in the document (under components)
func (index *SpecIndex) GetAllHeaders() map[string]*Reference {
	return index.allHeaders
}

// GetAllExternalDocuments will return all external documents found
func (index *SpecIndex) GetAllExternalDocuments() map[string]*Reference {
	return index.allExternalDocuments
}

// GetAllExamples will return all examples found in the document (under components)
func (index *SpecIndex) GetAllExamples() map[string]*Reference {
	return index.allExamples
}

// GetAllDescriptions will return all descriptions found in the document
func (index *SpecIndex) GetAllDescriptions() []*DescriptionReference {
	return index.allDescriptions
}

// GetAllEnums will return all enums found in the document
func (index *SpecIndex) GetAllEnums() []*EnumReference {
	return index.allEnums
}

// GetAllObjectsWithProperties will return all objects with properties found in the document
func (index *SpecIndex) GetAllObjectsWithProperties() []*ObjectReference {
	return index.allObjectsWithProperties
}

// GetAllSummaries will return all summaries found in the document
func (index *SpecIndex) GetAllSummaries() []*DescriptionReference {
	return index.allSummaries
}

// GetAllRequestBodies will return all requestBodies found in the document (under components)
func (index *SpecIndex) GetAllRequestBodies() map[string]*Reference {
	return index.allRequestBodies
}

// GetAllLinks will return all links found in the document (under components)
func (index *SpecIndex) GetAllLinks() map[string]*Reference {
	return index.allLinks
}

// GetAllParameters will return all parameters found in the document (under components)
func (index *SpecIndex) GetAllParameters() map[string]*Reference {
	return index.allParameters
}

// GetAllResponses will return all responses found in the document (under components)
func (index *SpecIndex) GetAllResponses() map[string]*Reference {
	return index.allResponses
}

// GetAllCallbacks will return all links found in the document (under components)
func (index *SpecIndex) GetAllCallbacks() map[string]*Reference {
	return index.allCallbacks
}

// GetInlineOperationDuplicateParameters will return a map of duplicates located in operation parameters.
func (index *SpecIndex) GetInlineOperationDuplicateParameters() map[string][]*Reference {
	return index.paramInlineDuplicateNames
}

// GetReferencesWithSiblings will return a map of all the references with sibling nodes (illegal)
func (index *SpecIndex) GetReferencesWithSiblings() map[string]Reference {
	return index.refsWithSiblings
}

// GetAllReferences will return every reference found in the spec, after being de-duplicated.
func (index *SpecIndex) GetAllReferences() map[string]*Reference {
	return index.allRefs
}

// GetAllSequencedReferences will return every reference (in sequence) that was found (non-polymorphic)
func (index *SpecIndex) GetAllSequencedReferences() []*Reference {
	return index.rawSequencedRefs
}

// GetSchemasNode will return the schema's node found in the spec
func (index *SpecIndex) GetSchemasNode() *yaml.Node {
	return index.schemasNode
}

// GetParametersNode will return the schema's node found in the spec
func (index *SpecIndex) GetParametersNode() *yaml.Node {
	return index.parametersNode
}

// GetReferenceIndexErrors will return any errors that occurred when indexing references
func (index *SpecIndex) GetReferenceIndexErrors() []error {
	return index.refErrors
}

// GetOperationParametersIndexErrors any errors that occurred when indexing operation parameters
func (index *SpecIndex) GetOperationParametersIndexErrors() []error {
	return index.operationParamErrors
}

// GetAllPaths will return all paths indexed in the document
func (index *SpecIndex) GetAllPaths() map[string]map[string]*Reference {
	return index.pathRefs
}

// GetOperationTags will return all references to all tags found in operations.
func (index *SpecIndex) GetOperationTags() map[string]map[string][]*Reference {
	return index.operationTagsRefs
}

// GetAllParametersFromOperations will return all paths indexed in the document
func (index *SpecIndex) GetAllParametersFromOperations() map[string]map[string]map[string][]*Reference {
	return index.paramOpRefs
}

// GetRootSecurityReferences will return all root security settings
func (index *SpecIndex) GetRootSecurityReferences() []*Reference {
	return index.rootSecurity
}

// GetSecurityRequirementReferences will return all security requirement definitions found in the document
func (index *SpecIndex) GetSecurityRequirementReferences() map[string]map[string][]*Reference {
	return index.securityRequirementRefs
}

// GetRootSecurityNode will return the root security node
func (index *SpecIndex) GetRootSecurityNode() *yaml.Node {
	return index.rootSecurityNode
}

// GetRootServersNode will return the root servers node
func (index *SpecIndex) GetRootServersNode() *yaml.Node {
	return index.rootServersNode
}

// GetAllRootServers will return all root servers defined
func (index *SpecIndex) GetAllRootServers() []*Reference {
	return index.serversRefs
}

// GetAllOperationsServers will return all operation overrides for servers.
func (index *SpecIndex) GetAllOperationsServers() map[string]map[string][]*Reference {
	return index.opServersRefs
}

// GetAllExternalIndexes will return all indexes for external documents
func (index *SpecIndex) GetAllExternalIndexes() map[string]*SpecIndex {
	return index.externalSpecIndex
}

// SetAllowCircularReferenceResolving will flip a bit that can be used by any consumers to determine if they want
// to allow or disallow circular references to be resolved or visited
func (index *SpecIndex) SetAllowCircularReferenceResolving(allow bool) {
	index.allowCircularReferences = allow
}

// AllowCircularReferenceResolving will return a bit that allows developers to determine what to do with circular refs.
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

// GetPathCount will return the number of paths found in the spec
func (index *SpecIndex) GetPathCount() int {
	if index.root == nil {
		return -1
	}

	if index.pathCount > 0 {
		return index.pathCount
	}
	pc := 0
	for i, n := range index.root.Content[0].Content {
		if i%2 == 0 {
			if n.Value == "paths" {
				pn := index.root.Content[0].Content[i+1].Content
				index.pathsNode = index.root.Content[0].Content[i+1]
				pc = len(pn) / 2
			}
		}
	}
	index.pathCount = pc
	return pc
}

// ExtractExternalDocuments will extract the number of externalDocs nodes found in the document.
func (index *SpecIndex) ExtractExternalDocuments(node *yaml.Node) []*Reference {
	if node == nil {
		return nil
	}
	var found []*Reference
	if len(node.Content) > 0 {
		for i, n := range node.Content {
			if utils.IsNodeMap(n) || utils.IsNodeArray(n) {
				found = append(found, index.ExtractExternalDocuments(n)...)
			}

			if i%2 == 0 && n.Value == "externalDocs" {
				docNode := node.Content[i+1]
				_, urlNode := utils.FindKeyNode("url", docNode.Content)
				if urlNode != nil {
					ref := &Reference{
						Definition: urlNode.Value,
						Name:       urlNode.Value,
						Node:       docNode,
					}
					index.externalDocumentsRef = append(index.externalDocumentsRef, ref)
				}
			}
		}
	}
	index.externalDocumentsCount = len(index.externalDocumentsRef)
	return found
}

// GetGlobalTagsCount will return the number of tags found in the top level 'tags' node of the document.
func (index *SpecIndex) GetGlobalTagsCount() int {
	if index.root == nil {
		return -1
	}

	if index.globalTagsCount > 0 {
		return index.globalTagsCount
	}

	for i, n := range index.root.Content[0].Content {
		if i%2 == 0 {
			if n.Value == "tags" {
				tagsNode := index.root.Content[0].Content[i+1]
				if tagsNode != nil {
					index.tagsNode = tagsNode
					index.globalTagsCount = len(tagsNode.Content) // tags is an array, don't divide by 2.
					for x, tagNode := range index.tagsNode.Content {

						_, name := utils.FindKeyNode("name", tagNode.Content)
						_, description := utils.FindKeyNode("description", tagNode.Content)

						var desc string
						if description == nil {
							desc = ""
						}
						if name != nil {
							ref := &Reference{
								Definition: desc,
								Name:       name.Value,
								Node:       tagNode,
								Path:       fmt.Sprintf("$.tags[%d]", x),
							}
							index.globalTagRefs[name.Value] = ref
						}
					}
				}
			}
		}
	}
	return index.globalTagsCount
}

// GetOperationTagsCount will return the number of operation tags found (tags referenced in operations)
func (index *SpecIndex) GetOperationTagsCount() int {
	if index.root == nil {
		return -1
	}

	if index.operationTagsCount > 0 {
		return index.operationTagsCount
	}

	// this is an aggregate count function that can only be run after operations
	// have been calculated.
	seen := make(map[string]bool)
	count := 0
	for _, path := range index.operationTagsRefs {
		for _, method := range path {
			for _, tag := range method {
				if !seen[tag.Name] {
					seen[tag.Name] = true
					count++
				}
			}
		}
	}
	index.operationTagsCount = count
	return index.operationTagsCount
}

// GetTotalTagsCount will return the number of global and operation tags found that are unique.
func (index *SpecIndex) GetTotalTagsCount() int {
	if index.root == nil {
		return -1
	}
	if index.totalTagsCount > 0 {
		return index.totalTagsCount
	}

	seen := make(map[string]bool)
	count := 0

	for _, gt := range index.globalTagRefs {
		// TODO: do we still need this?
		if !seen[gt.Name] {
			seen[gt.Name] = true
			count++
		}
	}
	for _, ot := range index.operationTagsRefs {
		for _, m := range ot {
			for _, t := range m {
				if !seen[t.Name] {
					seen[t.Name] = true
					count++
				}
			}
		}
	}
	index.totalTagsCount = count
	return index.totalTagsCount
}

// GetGlobalCallbacksCount for each response of each operation method, multiple callbacks can be defined
func (index *SpecIndex) GetGlobalCallbacksCount() int {
	if index.root == nil {
		return -1
	}

	if index.globalCallbacksCount > 0 {
		return index.globalCallbacksCount
	}

	// index.pathRefsLock.Lock()
	for path, p := range index.pathRefs {
		for _, m := range p {

			// look through method for callbacks
			callbacks, _ := yamlpath.NewPath("$..callbacks")
			res, _ := callbacks.Find(m.Node)

			if len(res) > 0 {
				for _, callback := range res[0].Content {
					if utils.IsNodeMap(callback) {

						ref := &Reference{
							Definition: m.Name,
							Name:       m.Name,
							Node:       callback,
						}

						if index.callbacksRefs[path] == nil {
							index.callbacksRefs[path] = make(map[string][]*Reference)
						}
						if len(index.callbacksRefs[path][m.Name]) > 0 {
							index.callbacksRefs[path][m.Name] = append(index.callbacksRefs[path][m.Name], ref)
						} else {
							index.callbacksRefs[path][m.Name] = []*Reference{ref}
						}
						index.globalCallbacksCount++
					}
				}
			}
		}
	}
	// index.pathRefsLock.Unlock()
	return index.globalCallbacksCount
}

// GetGlobalLinksCount for each response of each operation method, multiple callbacks can be defined
func (index *SpecIndex) GetGlobalLinksCount() int {
	if index.root == nil {
		return -1
	}

	if index.globalLinksCount > 0 {
		return index.globalLinksCount
	}

	// index.pathRefsLock.Lock()
	for path, p := range index.pathRefs {
		for _, m := range p {

			// look through method for links
			links, _ := yamlpath.NewPath("$..links")
			res, _ := links.Find(m.Node)

			if len(res) > 0 {
				for _, link := range res[0].Content {
					if utils.IsNodeMap(link) {

						ref := &Reference{
							Definition: m.Name,
							Name:       m.Name,
							Node:       link,
						}
						if index.linksRefs[path] == nil {
							index.linksRefs[path] = make(map[string][]*Reference)
						}
						if len(index.linksRefs[path][m.Name]) > 0 {
							index.linksRefs[path][m.Name] = append(index.linksRefs[path][m.Name], ref)
						}
						index.linksRefs[path][m.Name] = []*Reference{ref}
						index.globalLinksCount++
					}
				}
			}
		}
	}
	// index.pathRefsLock.Unlock()
	return index.globalLinksCount
}

// GetRawReferenceCount will return the number of raw references located in the document.
func (index *SpecIndex) GetRawReferenceCount() int {
	return len(index.rawSequencedRefs)
}

// GetComponentSchemaCount will return the number of schemas located in the 'components' or 'definitions' node.
func (index *SpecIndex) GetComponentSchemaCount() int {
	if index.root == nil {
		return -1
	}

	if index.schemaCount > 0 {
		return index.schemaCount
	}

	for i, n := range index.root.Content[0].Content {
		if i%2 == 0 {

			// servers
			if n.Value == "servers" {
				index.rootServersNode = index.root.Content[0].Content[i+1]
				if i+1 < len(index.root.Content[0].Content) {
					serverDefinitions := index.root.Content[0].Content[i+1]
					for x, def := range serverDefinitions.Content {
						ref := &Reference{
							Definition: "servers",
							Name:       "server",
							Node:       def,
							Path:       fmt.Sprintf("$.servers[%d]", x),
							ParentNode: index.rootServersNode,
						}
						index.serversRefs = append(index.serversRefs, ref)
					}
				}
			}

			// root security definitions
			if n.Value == "security" {
				index.rootSecurityNode = index.root.Content[0].Content[i+1]
				if i+1 < len(index.root.Content[0].Content) {
					securityDefinitions := index.root.Content[0].Content[i+1]
					for x, def := range securityDefinitions.Content {
						if len(def.Content) > 0 {
							name := def.Content[0]
							ref := &Reference{
								Definition: name.Value,
								Name:       name.Value,
								Node:       def,
								Path:       fmt.Sprintf("$.security[%d]", x),
							}
							index.rootSecurity = append(index.rootSecurity, ref)
						}
					}
				}
			}

			if n.Value == "components" {
				_, schemasNode := utils.FindKeyNode("schemas", index.root.Content[0].Content[i+1].Content)

				// while we are here, go ahead and extract everything in components.
				_, parametersNode := utils.FindKeyNode("parameters", index.root.Content[0].Content[i+1].Content)
				_, requestBodiesNode := utils.FindKeyNode("requestBodies", index.root.Content[0].Content[i+1].Content)
				_, responsesNode := utils.FindKeyNode("responses", index.root.Content[0].Content[i+1].Content)
				_, securitySchemesNode := utils.FindKeyNode("securitySchemes", index.root.Content[0].Content[i+1].Content)
				_, headersNode := utils.FindKeyNode("headers", index.root.Content[0].Content[i+1].Content)
				_, examplesNode := utils.FindKeyNode("examples", index.root.Content[0].Content[i+1].Content)
				_, linksNode := utils.FindKeyNode("links", index.root.Content[0].Content[i+1].Content)
				_, callbacksNode := utils.FindKeyNode("callbacks", index.root.Content[0].Content[i+1].Content)

				// extract schemas
				if schemasNode != nil {
					index.extractDefinitionsAndSchemas(schemasNode, "#/components/schemas/")
					index.schemasNode = schemasNode
					index.schemaCount = len(schemasNode.Content) / 2
				}

				// extract parameters
				if parametersNode != nil {
					index.extractComponentParameters(parametersNode, "#/components/parameters/")
					index.componentLock.Lock()
					index.parametersNode = parametersNode
					index.componentLock.Unlock()
				}

				// extract requestBodies
				if requestBodiesNode != nil {
					index.extractComponentRequestBodies(requestBodiesNode, "#/components/requestBodies/")
					index.requestBodiesNode = requestBodiesNode
				}

				// extract responses
				if responsesNode != nil {
					index.extractComponentResponses(responsesNode, "#/components/responses/")
					index.responsesNode = responsesNode
				}

				// extract security schemes
				if securitySchemesNode != nil {
					index.extractComponentSecuritySchemes(securitySchemesNode, "#/components/securitySchemes/")
					index.securitySchemesNode = securitySchemesNode
				}

				// extract headers
				if headersNode != nil {
					index.extractComponentHeaders(headersNode, "#/components/headers/")
					index.headersNode = headersNode
				}

				// extract examples
				if examplesNode != nil {
					index.extractComponentExamples(examplesNode, "#/components/examples/")
					index.examplesNode = examplesNode
				}

				// extract links
				if linksNode != nil {
					index.extractComponentLinks(linksNode, "#/components/links/")
					index.linksNode = linksNode
				}

				// extract callbacks
				if callbacksNode != nil {
					index.extractComponentCallbacks(callbacksNode, "#/components/callbacks/")
					index.callbacksNode = callbacksNode
				}

			}

			// swagger
			if n.Value == "definitions" {
				schemasNode := index.root.Content[0].Content[i+1]
				if schemasNode != nil {

					// extract schemas
					index.extractDefinitionsAndSchemas(schemasNode, "#/definitions/")
					index.schemasNode = schemasNode
					index.schemaCount = len(schemasNode.Content) / 2
				}
			}

			// swagger
			if n.Value == "parameters" {
				parametersNode := index.root.Content[0].Content[i+1]
				if parametersNode != nil {
					// extract params
					index.extractComponentParameters(parametersNode, "#/parameters/")
					index.componentLock.Lock()
					index.parametersNode = parametersNode
					index.componentLock.Unlock()
				}
			}

			if n.Value == "responses" {
				responsesNode := index.root.Content[0].Content[i+1]
				if responsesNode != nil {

					// extract responses
					index.extractComponentResponses(responsesNode, "#/responses/")
					index.responsesNode = responsesNode
				}
			}

			if n.Value == "securityDefinitions" {
				securityDefinitionsNode := index.root.Content[0].Content[i+1]
				if securityDefinitionsNode != nil {

					// extract security definitions.
					index.extractComponentSecuritySchemes(securityDefinitionsNode, "#/securityDefinitions/")
					index.securitySchemesNode = securityDefinitionsNode
				}
			}

		}
	}
	return index.schemaCount
}

// GetComponentParameterCount returns the number of parameter components defined
func (index *SpecIndex) GetComponentParameterCount() int {
	if index.root == nil {
		return -1
	}

	if index.componentParamCount > 0 {
		return index.componentParamCount
	}

	for i, n := range index.root.Content[0].Content {
		if i%2 == 0 {
			// openapi 3
			if n.Value == "components" {
				_, parametersNode := utils.FindKeyNode("parameters", index.root.Content[0].Content[i+1].Content)
				if parametersNode != nil {
					index.componentLock.Lock()
					index.parametersNode = parametersNode
					index.componentParamCount = len(parametersNode.Content) / 2
					index.componentLock.Unlock()
				}
			}
			// openapi 2
			if n.Value == "parameters" {
				parametersNode := index.root.Content[0].Content[i+1]
				if parametersNode != nil {
					index.componentLock.Lock()
					index.parametersNode = parametersNode
					index.componentParamCount = len(parametersNode.Content) / 2
					index.componentLock.Unlock()
				}
			}
		}
	}
	return index.componentParamCount
}

// GetOperationCount returns the number of operations (for all paths) located in the document
func (index *SpecIndex) GetOperationCount() int {
	if index.root == nil {
		return -1
	}

	if index.pathsNode == nil {
		return -1
	}

	if index.operationCount > 0 {
		return index.operationCount
	}

	opCount := 0

	for x, p := range index.pathsNode.Content {
		if x%2 == 0 {

			var method *yaml.Node
			if utils.IsNodeArray(index.pathsNode) {
				method = index.pathsNode.Content[x]
			} else {
				method = index.pathsNode.Content[x+1]
			}

			// extract methods for later use.
			for y, m := range method.Content {
				if y%2 == 0 {

					// check node is a valid method
					valid := false
					for _, methodType := range methodTypes {
						if m.Value == methodType {
							valid = true
						}
					}
					if valid {
						ref := &Reference{
							Definition: m.Value,
							Name:       m.Value,
							Node:       method.Content[y+1],
						}
						index.pathRefsLock.Lock()
						if index.pathRefs[p.Value] == nil {
							index.pathRefs[p.Value] = make(map[string]*Reference)
						}
						index.pathRefs[p.Value][ref.Name] = ref
						index.pathRefsLock.Unlock()
						// update
						opCount++
					}
				}
			}
		}
	}

	index.operationCount = opCount
	return opCount
}

// GetOperationsParameterCount returns the number of parameters defined in paths and operations.
// this method looks in top level (path level) and inside each operation (get, post etc.). Parameters can
// be hiding within multiple places.
func (index *SpecIndex) GetOperationsParameterCount() int {
	if index.root == nil {
		return -1
	}

	if index.pathsNode == nil {
		return -1
	}

	if index.operationParamCount > 0 {
		return index.operationParamCount
	}

	// parameters are sneaky, they can be in paths, in path operations or in components.
	// sometimes they are refs, sometimes they are inline definitions, just for fun.
	// some authors just LOVE to mix and match them all up.
	// check paths first
	for x, pathItemNode := range index.pathsNode.Content {
		if x%2 == 0 {

			var pathPropertyNode *yaml.Node
			if utils.IsNodeArray(index.pathsNode) {
				pathPropertyNode = index.pathsNode.Content[x]
			} else {
				pathPropertyNode = index.pathsNode.Content[x+1]
			}

			// extract methods for later use.
			for y, prop := range pathPropertyNode.Content {
				if y%2 == 0 {

					// while we're here, lets extract any top level servers
					if prop.Value == "servers" {
						serversNode := pathPropertyNode.Content[y+1]
						if index.opServersRefs[pathItemNode.Value] == nil {
							index.opServersRefs[pathItemNode.Value] = make(map[string][]*Reference)
						}
						var serverRefs []*Reference
						for i, serverRef := range serversNode.Content {
							ref := &Reference{
								Definition: serverRef.Value,
								Name:       serverRef.Value,
								Node:       serverRef,
								ParentNode: prop,
								Path:       fmt.Sprintf("$.paths.%s.servers[%d]", pathItemNode.Value, i),
							}
							serverRefs = append(serverRefs, ref)
						}
						index.opServersRefs[pathItemNode.Value]["top"] = serverRefs
					}

					// top level params
					if prop.Value == "parameters" {

						// let's look at params, check if they are refs or inline.
						params := pathPropertyNode.Content[y+1].Content
						index.scanOperationParams(params, pathItemNode, "top")
					}

					// method level params.
					if isHttpMethod(prop.Value) {
						for z, httpMethodProp := range pathPropertyNode.Content[y+1].Content {
							if z%2 == 0 {
								if httpMethodProp.Value == "parameters" {
									params := pathPropertyNode.Content[y+1].Content[z+1].Content
									index.scanOperationParams(params, pathItemNode, prop.Value)
								}

								// extract operation tags if set.
								if httpMethodProp.Value == "tags" {
									tags := pathPropertyNode.Content[y+1].Content[z+1]

									if index.operationTagsRefs[pathItemNode.Value] == nil {
										index.operationTagsRefs[pathItemNode.Value] = make(map[string][]*Reference)
									}

									var tagRefs []*Reference
									for _, tagRef := range tags.Content {
										ref := &Reference{
											Definition: tagRef.Value,
											Name:       tagRef.Value,
											Node:       tagRef,
										}
										tagRefs = append(tagRefs, ref)
									}
									index.operationTagsRefs[pathItemNode.Value][prop.Value] = tagRefs
								}

								// extract description and summaries
								if httpMethodProp.Value == "description" {
									desc := pathPropertyNode.Content[y+1].Content[z+1].Value
									ref := &Reference{
										Definition: desc,
										Name:       "description",
										Node:       pathPropertyNode.Content[y+1].Content[z+1],
									}
									if index.operationDescriptionRefs[pathItemNode.Value] == nil {
										index.operationDescriptionRefs[pathItemNode.Value] = make(map[string]*Reference)
									}

									index.operationDescriptionRefs[pathItemNode.Value][prop.Value] = ref
								}
								if httpMethodProp.Value == "summary" {
									summary := pathPropertyNode.Content[y+1].Content[z+1].Value
									ref := &Reference{
										Definition: summary,
										Name:       "summary",
										Node:       pathPropertyNode.Content[y+1].Content[z+1],
									}

									if index.operationSummaryRefs[pathItemNode.Value] == nil {
										index.operationSummaryRefs[pathItemNode.Value] = make(map[string]*Reference)
									}

									index.operationSummaryRefs[pathItemNode.Value][prop.Value] = ref
								}

								// extract servers from method operation.
								if httpMethodProp.Value == "servers" {
									serversNode := pathPropertyNode.Content[y+1].Content[z+1]

									var serverRefs []*Reference
									for i, serverRef := range serversNode.Content {
										ref := &Reference{
											Definition: "servers",
											Name:       "servers",
											Node:       serverRef,
											ParentNode: httpMethodProp,
											Path:       fmt.Sprintf("$.paths.%s.%s.servers[%d]", pathItemNode.Value, prop.Value, i),
										}
										serverRefs = append(serverRefs, ref)
									}

									if index.opServersRefs[pathItemNode.Value] == nil {
										index.opServersRefs[pathItemNode.Value] = make(map[string][]*Reference)
									}

									index.opServersRefs[pathItemNode.Value][prop.Value] = serverRefs
								}

							}
						}
					}
				}
			}
		}
	}

	// Now that all the paths and operations are processed, lets pick out everything from our pre
	// mapped refs and populate our ready to roll index of component params.
	for key, component := range index.allMappedRefs {
		if strings.Contains(key, "/parameters/") {
			index.paramCompRefs[key] = component
			index.paramAllRefs[key] = component
		}
	}

	// now build main index of all params by combining comp refs with inline params from operations.
	// use the namespace path:::param for inline params to identify them as inline.
	for path, params := range index.paramOpRefs {
		for mName, mValue := range params {
			for pName, pValue := range mValue {
				if !strings.HasPrefix(pName, "#") {
					index.paramInlineDuplicateNames[pName] = append(index.paramInlineDuplicateNames[pName], pValue...)
					for i := range pValue {
						if pValue[i] != nil {
							_, in := utils.FindKeyNodeTop("in", pValue[i].Node.Content)
							if in != nil {
								index.paramAllRefs[fmt.Sprintf("%s:::%s:::%s", path, mName, in.Value)] = pValue[i]
							} else {
								index.paramAllRefs[fmt.Sprintf("%s:::%s", path, mName)] = pValue[i]
							}
						}
					}
				}
			}
		}
	}

	index.operationParamCount = len(index.paramCompRefs) + len(index.paramInlineDuplicateNames)
	return index.operationParamCount
}

// GetInlineDuplicateParamCount returns the number of inline duplicate parameters (operation params)
func (index *SpecIndex) GetInlineDuplicateParamCount() int {
	if index.componentsInlineParamDuplicateCount > 0 {
		return index.componentsInlineParamDuplicateCount
	}
	dCount := len(index.paramInlineDuplicateNames) - index.countUniqueInlineDuplicates()
	index.componentsInlineParamDuplicateCount = dCount
	return dCount
}

// GetInlineUniqueParamCount returns the number of unique inline parameters (operation params)
func (index *SpecIndex) GetInlineUniqueParamCount() int {
	return index.countUniqueInlineDuplicates()
}

// GetAllDescriptionsCount will collect together every single description found in the document
func (index *SpecIndex) GetAllDescriptionsCount() int {
	return len(index.allDescriptions)
}

// GetAllSummariesCount will collect together every single summary found in the document
func (index *SpecIndex) GetAllSummariesCount() int {
	return len(index.allSummaries)
}

// CheckForSeenRemoteSource will check to see if we have already seen this remote source and return it,
// to avoid making duplicate remote calls for document data.
func (index *SpecIndex) CheckForSeenRemoteSource(url string) (bool, *yaml.Node) {
	if index.config == nil || index.config.seenRemoteSources == nil {
		return false, nil
	}
	j, _ := index.config.seenRemoteSources.Load(url)
	if j != nil {
		return true, j.(*yaml.Node)
	}
	return false, nil
}
