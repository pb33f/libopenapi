// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestSpecIndex_ExtractRefsStripe(t *testing.T) {
	stripe, _ := ioutil.ReadFile("../test_specs/stripe.yaml")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(stripe, &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())

	assert.Len(t, index.allRefs, 385)
	assert.Equal(t, 537, len(index.allMappedRefs))
	combined := index.GetAllCombinedReferences()
	assert.Equal(t, 537, len(combined))

	assert.Len(t, index.rawSequencedRefs, 1972)
	assert.Equal(t, 246, index.pathCount)
	assert.Equal(t, 402, index.operationCount)
	assert.Equal(t, 537, index.schemaCount)
	assert.Equal(t, 0, index.globalTagsCount)
	assert.Equal(t, 0, index.globalLinksCount)
	assert.Equal(t, 0, index.componentParamCount)
	assert.Equal(t, 143, index.operationParamCount)
	assert.Equal(t, 88, index.componentsInlineParamDuplicateCount)
	assert.Equal(t, 55, index.componentsInlineParamUniqueCount)
	assert.Equal(t, 1516, index.enumCount)
	assert.Len(t, index.GetAllEnums(), 1516)
	assert.Len(t, index.GetPolyAllOfReferences(), 0)
	assert.Len(t, index.GetPolyOneOfReferences(), 275)
	assert.Len(t, index.GetPolyAnyOfReferences(), 553)
	assert.Len(t, index.GetAllReferenceSchemas(), 1972)
	assert.NotNil(t, index.GetRootServersNode())
	assert.Len(t, index.GetAllRootServers(), 1)

	// not required, but flip the circular result switch on and off.
	assert.False(t, index.AllowCircularReferenceResolving())
	index.SetAllowCircularReferenceResolving(true)
	assert.True(t, index.AllowCircularReferenceResolving())

	// simulate setting of circular references, also pointless but needed for coverage.
	assert.Nil(t, index.GetCircularReferences())
	index.SetCircularReferences([]*CircularReferenceResult{new(CircularReferenceResult)})
	assert.Len(t, index.GetCircularReferences(), 1)

	assert.Len(t, index.GetRefsByLine(), 537)
	assert.Len(t, index.GetLinesWithReferences(), 1972)
	assert.Len(t, index.GetAllExternalDocuments(), 0)
	assert.Len(t, index.GetAllExternalIndexes(), 0)
}

func TestSpecIndex_Asana(t *testing.T) {
	asana, _ := ioutil.ReadFile("../test_specs/asana.yaml")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(asana, &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())

	assert.Len(t, index.allRefs, 152)
	assert.Len(t, index.allMappedRefs, 171)
	combined := index.GetAllCombinedReferences()
	assert.Equal(t, 171, len(combined))
	assert.Equal(t, 118, index.pathCount)
	assert.Equal(t, 152, index.operationCount)
	assert.Equal(t, 135, index.schemaCount)
	assert.Equal(t, 26, index.globalTagsCount)
	assert.Equal(t, 0, index.globalLinksCount)
	assert.Equal(t, 30, index.componentParamCount)
	assert.Equal(t, 107, index.operationParamCount)
	assert.Equal(t, 8, index.componentsInlineParamDuplicateCount)
	assert.Equal(t, 69, index.componentsInlineParamUniqueCount)
}

func TestSpecIndex_DigitalOcean(t *testing.T) {
	do, _ := os.ReadFile("../test_specs/digitalocean.yaml")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(do, &rootNode)

	baseURL, _ := url.Parse("https://raw.githubusercontent.com/digitalocean/openapi/main/specification")
	index := NewSpecIndexWithConfig(&rootNode, &SpecIndexConfig{
		BaseURL:           baseURL,
		AllowRemoteLookup: true,
		AllowFileLookup:   true,
	})

	assert.Len(t, index.GetAllExternalIndexes(), 291)
	assert.NotNil(t, index)
}

func TestSpecIndex_DigitalOcean_FullCheckoutLocalResolve(t *testing.T) {
	// this is a full checkout of the digitalocean API repo.
	tmp, _ := os.MkdirTemp("", "openapi")
	cmd := exec.Command("git", "clone", "https://github.com/digitalocean/openapi", tmp)
	defer os.RemoveAll(filepath.Join(tmp, "openapi"))
	err := cmd.Run()
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}
	spec, _ := filepath.Abs(filepath.Join(tmp, "specification", "DigitalOcean-public.v2.yaml"))
	doLocal, _ := ioutil.ReadFile(spec)
	var rootNode yaml.Node
	_ = yaml.Unmarshal(doLocal, &rootNode)

	config := CreateOpenAPIIndexConfig()
	config.BasePath = filepath.Join(tmp, "specification")

	index := NewSpecIndexWithConfig(&rootNode, config)

	assert.NotNil(t, index)
	assert.Len(t, index.GetAllExternalIndexes(), 296)

	ref := index.SearchIndexForReference("resources/apps/apps_list_instanceSizes.yml")
	assert.NotNil(t, ref)
	assert.Equal(t, "operationId", ref[0].Node.Content[0].Value)

	ref = index.SearchIndexForReference("examples/ruby/domains_create.yml")
	assert.NotNil(t, ref)
	assert.Equal(t, "lang", ref[0].Node.Content[0].Value)

	ref = index.SearchIndexForReference("../../shared/responses/server_error.yml")
	assert.NotNil(t, ref)
	assert.Equal(t, "description", ref[0].Node.Content[0].Value)

	ref = index.SearchIndexForReference("../models/options.yml")
	assert.NotNil(t, ref)
}

func TestSpecIndex_DigitalOcean_LookupsNotAllowed(t *testing.T) {
	asana, _ := ioutil.ReadFile("../test_specs/digitalocean.yaml")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(asana, &rootNode)

	baseURL, _ := url.Parse("https://raw.githubusercontent.com/digitalocean/openapi/main/specification")
	index := NewSpecIndexWithConfig(&rootNode, &SpecIndexConfig{
		BaseURL: baseURL,
	})

	// no lookups allowed, bits have not been set, so there should just be a bunch of errors.
	assert.Len(t, index.GetAllExternalIndexes(), 0)
	assert.True(t, len(index.GetReferenceIndexErrors()) > 0)
}

func TestSpecIndex_BaseURLError(t *testing.T) {
	asana, _ := ioutil.ReadFile("../test_specs/digitalocean.yaml")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(asana, &rootNode)

	// this should fail because the base url is not a valid url and digital ocean won't be able to resolve
	// anything.
	baseURL, _ := url.Parse("https://githerbs.com/fresh/herbs/for/you")
	index := NewSpecIndexWithConfig(&rootNode, &SpecIndexConfig{
		BaseURL:           baseURL,
		AllowRemoteLookup: true,
		AllowFileLookup:   true,
	})

	assert.Len(t, index.GetAllExternalIndexes(), 0)
}

func TestSpecIndex_k8s(t *testing.T) {
	asana, _ := ioutil.ReadFile("../test_specs/k8s.json")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(asana, &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())

	assert.Len(t, index.allRefs, 558)
	assert.Equal(t, 563, len(index.allMappedRefs))
	combined := index.GetAllCombinedReferences()
	assert.Equal(t, 563, len(combined))
	assert.Equal(t, 436, index.pathCount)
	assert.Equal(t, 853, index.operationCount)
	assert.Equal(t, 563, index.schemaCount)
	assert.Equal(t, 0, index.globalTagsCount)
	assert.Equal(t, 58, index.operationTagsCount)
	assert.Equal(t, 0, index.globalLinksCount)
	assert.Equal(t, 0, index.componentParamCount)
	assert.Equal(t, 36, index.operationParamCount)
	assert.Equal(t, 26, index.componentsInlineParamDuplicateCount)
	assert.Equal(t, 10, index.componentsInlineParamUniqueCount)
	assert.Equal(t, 58, index.GetTotalTagsCount())
	assert.Equal(t, 2524, index.GetRawReferenceCount())
}

func TestSpecIndex_PetstoreV2(t *testing.T) {
	asana, _ := ioutil.ReadFile("../test_specs/petstorev2.json")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(asana, &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())

	assert.Len(t, index.allRefs, 6)
	assert.Len(t, index.allMappedRefs, 6)
	assert.Equal(t, 14, index.pathCount)
	assert.Equal(t, 20, index.operationCount)
	assert.Equal(t, 6, index.schemaCount)
	assert.Equal(t, 3, index.globalTagsCount)
	assert.Equal(t, 3, index.operationTagsCount)
	assert.Equal(t, 0, index.globalLinksCount)
	assert.Equal(t, 1, index.componentParamCount)
	assert.Equal(t, 1, index.GetComponentParameterCount())
	assert.Equal(t, 11, index.operationParamCount)
	assert.Equal(t, 5, index.componentsInlineParamDuplicateCount)
	assert.Equal(t, 6, index.componentsInlineParamUniqueCount)
	assert.Equal(t, 3, index.GetTotalTagsCount())
	assert.Equal(t, 2, len(index.GetSecurityRequirementReferences()))
}

func TestSpecIndex_XSOAR(t *testing.T) {
	xsoar, _ := ioutil.ReadFile("../test_specs/xsoar.json")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(xsoar, &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())
	assert.Len(t, index.allRefs, 209)
	assert.Equal(t, 85, index.pathCount)
	assert.Equal(t, 88, index.operationCount)
	assert.Equal(t, 245, index.schemaCount)
	assert.Equal(t, 207, len(index.allMappedRefs))
	assert.Equal(t, 0, index.globalTagsCount)
	assert.Equal(t, 0, index.operationTagsCount)
	assert.Equal(t, 0, index.globalLinksCount)
	assert.Len(t, index.GetRootSecurityReferences(), 1)
	assert.NotNil(t, index.GetRootSecurityNode())
}

func TestSpecIndex_PetstoreV3(t *testing.T) {
	petstore, _ := ioutil.ReadFile("../test_specs/petstorev3.json")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(petstore, &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())

	assert.Len(t, index.allRefs, 7)
	assert.Len(t, index.allMappedRefs, 7)
	assert.Equal(t, 13, index.pathCount)
	assert.Equal(t, 19, index.operationCount)
	assert.Equal(t, 8, index.schemaCount)
	assert.Equal(t, 3, index.globalTagsCount)
	assert.Equal(t, 3, index.operationTagsCount)
	assert.Equal(t, 0, index.globalLinksCount)
	assert.Equal(t, 0, index.componentParamCount)
	assert.Equal(t, 9, index.operationParamCount)
	assert.Equal(t, 4, index.componentsInlineParamDuplicateCount)
	assert.Equal(t, 5, index.componentsInlineParamUniqueCount)
	assert.Equal(t, 3, index.GetTotalTagsCount())
	assert.Equal(t, 90, index.GetAllDescriptionsCount())
	assert.Equal(t, 19, index.GetAllSummariesCount())
	assert.Len(t, index.GetAllDescriptions(), 90)
	assert.Len(t, index.GetAllSummaries(), 19)
}

var mappedRefs = 15

func TestSpecIndex_BurgerShop(t *testing.T) {
	burgershop, _ := ioutil.ReadFile("../test_specs/burgershop.openapi.yaml")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(burgershop, &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())

	assert.Len(t, index.allRefs, mappedRefs)
	assert.Len(t, index.allMappedRefs, mappedRefs)
	assert.Equal(t, mappedRefs, len(index.GetMappedReferences()))
	assert.Equal(t, mappedRefs, len(index.GetMappedReferencesSequenced()))

	assert.Equal(t, 6, index.pathCount)
	assert.Equal(t, 6, index.GetPathCount())

	assert.Equal(t, 6, len(index.GetAllComponentSchemas()))
	assert.Equal(t, 56, len(index.GetAllSchemas()))

	assert.Equal(t, 34, len(index.GetAllSequencedReferences()))
	assert.NotNil(t, index.GetSchemasNode())
	assert.NotNil(t, index.GetParametersNode())

	assert.Equal(t, 5, index.operationCount)
	assert.Equal(t, 5, index.GetOperationCount())

	assert.Equal(t, 6, index.schemaCount)
	assert.Equal(t, 6, index.GetComponentSchemaCount())

	assert.Equal(t, 2, index.globalTagsCount)
	assert.Equal(t, 2, index.GetGlobalTagsCount())
	assert.Equal(t, 2, index.GetTotalTagsCount())

	assert.Equal(t, 2, index.operationTagsCount)
	assert.Equal(t, 2, index.GetOperationTagsCount())

	assert.Equal(t, 3, index.globalLinksCount)
	assert.Equal(t, 3, index.GetGlobalLinksCount())

	assert.Equal(t, 1, index.globalCallbacksCount)
	assert.Equal(t, 1, index.GetGlobalCallbacksCount())

	assert.Equal(t, 2, index.componentParamCount)
	assert.Equal(t, 2, index.GetComponentParameterCount())

	assert.Equal(t, 4, index.operationParamCount)
	assert.Equal(t, 4, index.GetOperationsParameterCount())

	assert.Equal(t, 0, index.componentsInlineParamDuplicateCount)
	assert.Equal(t, 0, index.GetInlineDuplicateParamCount())

	assert.Equal(t, 2, index.componentsInlineParamUniqueCount)
	assert.Equal(t, 2, index.GetInlineUniqueParamCount())

	assert.Equal(t, 1, len(index.GetAllRequestBodies()))
	assert.NotNil(t, index.GetRootNode())
	assert.NotNil(t, index.GetGlobalTagsNode())
	assert.NotNil(t, index.GetPathsNode())
	assert.NotNil(t, index.GetDiscoveredReferences())
	assert.Equal(t, 1, len(index.GetPolyReferences()))
	assert.NotNil(t, index.GetOperationParameterReferences())
	assert.Equal(t, 3, len(index.GetAllSecuritySchemes()))
	assert.Equal(t, 2, len(index.GetAllParameters()))
	assert.Equal(t, 1, len(index.GetAllResponses()))
	assert.Equal(t, 2, len(index.GetInlineOperationDuplicateParameters()))
	assert.Equal(t, 0, len(index.GetReferencesWithSiblings()))
	assert.Equal(t, mappedRefs, len(index.GetAllReferences()))
	assert.Equal(t, 0, len(index.GetOperationParametersIndexErrors()))
	assert.Equal(t, 5, len(index.GetAllPaths()))
	assert.Equal(t, 5, len(index.GetOperationTags()))
	assert.Equal(t, 3, len(index.GetAllParametersFromOperations()))
}

func TestSpecIndex_GetAllParametersFromOperations(t *testing.T) {
	yml := `openapi: 3.0.0
servers:
  - url: http://localhost:8080
paths:
  /test:
    get:
      parameters:
        - name: action
          in: query
          schema:
            type: string
        - name: action
          in: query
          schema:
            type: string`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())

	assert.Equal(t, 1, len(index.GetAllParametersFromOperations()))
	assert.Equal(t, 1, len(index.GetOperationParametersIndexErrors()))
}

func TestSpecIndex_BurgerShop_AllTheComponents(t *testing.T) {
	burgershop, _ := ioutil.ReadFile("../test_specs/all-the-components.yaml")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(burgershop, &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())

	assert.Equal(t, 1, len(index.GetAllHeaders()))
	assert.Equal(t, 1, len(index.GetAllLinks()))
	assert.Equal(t, 1, len(index.GetAllCallbacks()))
	assert.Equal(t, 1, len(index.GetAllExamples()))
	assert.Equal(t, 1, len(index.GetAllResponses()))
	assert.Equal(t, 2, len(index.GetAllRootServers()))
	assert.Equal(t, 2, len(index.GetAllOperationsServers()))
}

func TestSpecIndex_SwaggerResponses(t *testing.T) {
	yml := `swagger: 2.0
responses:
  niceResponse: 
    description: hi`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())

	assert.Equal(t, 1, len(index.GetAllResponses()))
}

func TestSpecIndex_NoNameParam(t *testing.T) {
	yml := `paths:
  /users/{id}:
    parameters:
    - in: path
      name: id
    - in: query
    get:
      parameters:
        - in: path
          name: id
        - in: query`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())

	assert.Equal(t, 2, len(index.GetOperationParametersIndexErrors()))
}

func TestSpecIndex_NoRoot(t *testing.T) {
	index := NewSpecIndex(nil)
	refs := index.ExtractRefs(nil, nil, nil, 0, false, "")
	docs := index.ExtractExternalDocuments(nil)
	assert.Nil(t, docs)
	assert.Nil(t, refs)
	assert.Nil(t, index.FindComponent("nothing", nil))
	assert.Equal(t, -1, index.GetOperationCount())
	assert.Equal(t, -1, index.GetPathCount())
	assert.Equal(t, -1, index.GetGlobalTagsCount())
	assert.Equal(t, -1, index.GetOperationTagsCount())
	assert.Equal(t, -1, index.GetTotalTagsCount())
	assert.Equal(t, -1, index.GetOperationsParameterCount())
	assert.Equal(t, -1, index.GetComponentParameterCount())
	assert.Equal(t, -1, index.GetComponentSchemaCount())
	assert.Equal(t, -1, index.GetGlobalLinksCount())
}

func TestSpecIndex_BurgerShopMixedRef(t *testing.T) {
	spec, _ := ioutil.ReadFile("../test_specs/mixedref-burgershop.openapi.yaml")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(spec, &rootNode)

	cwd, _ := os.Getwd()

	index := NewSpecIndexWithConfig(&rootNode, &SpecIndexConfig{
		AllowRemoteLookup: true,
		AllowFileLookup:   true,
		BasePath:          cwd,
	})

	assert.Len(t, index.allRefs, 5)
	assert.Len(t, index.allMappedRefs, 5)
	assert.Equal(t, 5, index.GetPathCount())
	assert.Equal(t, 5, index.GetOperationCount())
	assert.Equal(t, 1, index.GetComponentSchemaCount())
	assert.Equal(t, 2, index.GetGlobalTagsCount())
	assert.Equal(t, 3, index.GetTotalTagsCount())
	assert.Equal(t, 2, index.GetOperationTagsCount())
	assert.Equal(t, 0, index.GetGlobalLinksCount())
	assert.Equal(t, 0, index.GetComponentParameterCount())
	assert.Equal(t, 2, index.GetOperationsParameterCount())
	assert.Equal(t, 1, index.GetInlineDuplicateParamCount())
	assert.Equal(t, 1, index.GetInlineUniqueParamCount())
}

func TestSpecIndex_TestEmptyBrokenReferences(t *testing.T) {
	asana, _ := ioutil.ReadFile("../test_specs/badref-burgershop.openapi.yaml")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(asana, &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())
	assert.Equal(t, 5, index.GetPathCount())
	assert.Equal(t, 5, index.GetOperationCount())
	assert.Equal(t, 5, index.GetComponentSchemaCount())
	assert.Equal(t, 2, index.GetGlobalTagsCount())
	assert.Equal(t, 3, index.GetTotalTagsCount())
	assert.Equal(t, 2, index.GetOperationTagsCount())
	assert.Equal(t, 2, index.GetGlobalLinksCount())
	assert.Equal(t, 0, index.GetComponentParameterCount())
	assert.Equal(t, 2, index.GetOperationsParameterCount())
	assert.Equal(t, 1, index.GetInlineDuplicateParamCount())
	assert.Equal(t, 1, index.GetInlineUniqueParamCount())
	assert.Len(t, index.refErrors, 7)
}

func TestTagsNoDescription(t *testing.T) {
	yml := `tags:
  - name: one
  - name: two
  - three: three`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())
	assert.Equal(t, 3, index.GetGlobalTagsCount())
}

func TestGlobalCallbacksNoIndexTest(t *testing.T) {
	idx := new(SpecIndex)
	assert.Equal(t, -1, idx.GetGlobalCallbacksCount())
}

func TestMultipleCallbacksPerOperationVerb(t *testing.T) {
	yml := `components:
  callbacks:  
    callbackA:
      "{$request.query.queryUrl}":
        post:
          description: callbackAPost
        get:
          description: callbackAGet
    callbackB:
      "{$request.query.queryUrl}":
        post:
          description: callbackBPost
        get:
          description: callbackBGet
paths:
  /pb33f/arriving-soon:
    post:
      callbacks:
        callbackA:
          $ref: '#/components/callbacks/CallbackA'
        callbackB:
          $ref: '#/components/callbacks/CallbackB'
    get:
      callbacks:
        callbackB:
          $ref: '#/components/callbacks/CallbackB'
        callbackA:
          $ref: '#/components/callbacks/CallbackA'`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())
	assert.Equal(t, 4, index.GetGlobalCallbacksCount())
}

func TestSpecIndex_ExtractComponentsFromRefs(t *testing.T) {
	yml := `components:
  schemas:
    pizza:
      properties:
        something:
          $ref: '#/components/\schemas/\something'
    something:
      description: something`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())
	assert.Len(t, index.GetReferenceIndexErrors(), 1)
}

func TestSpecIndex_FindComponent_WithACrazyAssPath(t *testing.T) {
	yml := `paths:
  /crazy/ass/references:
    get:
      parameters:
        - name: a param
          schema:
            type: string
      description: Show information about one architecture.
      responses:
        "200":
          content:
            application/xml; charset=utf-8:
              schema:
                example:
                  name: x86_64
          description: OK. The request has succeeded.
        "404":
         content:
            application/xml; charset=utf-8:
              example:
                code: unknown_architecture
                summary: "Architecture does not exist: x999"
              schema:
                 $ref: "#/paths/~1crazy~1ass~1references/get/parameters/0"
        "400":
          content:
            application/xml; charset=utf-8:
              example:
                code: unknown_architecture
                summary: "Architecture does not exist: x999"
              schema:
                $ref: "#/paths/~1crazy~1ass~1references/get/responses/404/content/application~1xml;%20charset=utf-8/schema"
          description: Not Found.`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())
	assert.Equal(t, "#/paths/~1crazy~1ass~1references/get/parameters/0",
		index.FindComponent("#/paths/~1crazy~1ass~1references/get/responses/404/content/application~1xml;%20charset=utf-8/schema", nil).Node.Content[1].Value)

	assert.Equal(t, "a param",
		index.FindComponent("#/paths/~1crazy~1ass~1references/get/parameters/0", nil).Node.Content[1].Value)
}

func TestSpecIndex_FindComponenth(t *testing.T) {
	yml := `components:
  schemas:
    pizza:
      properties:
        something:
          $ref: '#/components/schemas/something'
    something:
      description: something`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())
	assert.Nil(t, index.FindComponent("I-do-not-exist", nil))
}

func TestSpecIndex_TestPathsNodeAsArray(t *testing.T) {
	yml := `components:
  schemas:
    pizza:
      properties:
        something:
          $ref: '#/components/schemas/something'
    something:
      description: something`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())
	assert.Nil(t, index.performExternalLookup(nil, "unknown", nil, nil))
}

func TestSpecIndex_lookupRemoteReference_SeenSourceSimulation_Error(t *testing.T) {
	index := new(SpecIndex)
	index.seenRemoteSources = make(map[string]*yaml.Node)
	index.seenRemoteSources["https://no-hope-for-a-dope.com"] = &yaml.Node{}
	_, _, err := index.lookupRemoteReference("https://no-hope-for-a-dope.com#/$.....#[;]something")
	assert.Error(t, err)
}

func TestSpecIndex_lookupRemoteReference_SeenSourceSimulation_BadFind(t *testing.T) {
	index := new(SpecIndex)
	index.seenRemoteSources = make(map[string]*yaml.Node)
	index.seenRemoteSources["https://no-hope-for-a-dope.com"] = &yaml.Node{}
	a, b, err := index.lookupRemoteReference("https://no-hope-for-a-dope.com#/hey")
	assert.Error(t, err)
	assert.Nil(t, a)
	assert.Nil(t, b)
}

// Discovered in issue https://github.com/pb33f/libopenapi/issues/37
func TestSpecIndex_lookupRemoteReference_NoComponent(t *testing.T) {
	index := new(SpecIndex)
	index.seenRemoteSources = make(map[string]*yaml.Node)
	index.seenRemoteSources["https://api.rest.sh/schemas/ErrorModel.json"] = &yaml.Node{}
	a, b, err := index.lookupRemoteReference("https://api.rest.sh/schemas/ErrorModel.json")
	assert.NoError(t, err)
	assert.NotNil(t, a)
	assert.NotNil(t, b)
}

// Discovered in issue https://github.com/daveshanley/vacuum/issues/225
func TestSpecIndex_lookupFileReference_NoComponent(t *testing.T) {
	cwd, _ := os.Getwd()
	index := new(SpecIndex)
	index.config = &SpecIndexConfig{BasePath: cwd}

	_ = ioutil.WriteFile("coffee-time.yaml", []byte("time: for coffee"), 0o664)
	defer os.Remove("coffee-time.yaml")

	index.seenRemoteSources = make(map[string]*yaml.Node)
	a, b, err := index.lookupFileReference("coffee-time.yaml")
	assert.NoError(t, err)
	assert.NotNil(t, a)
	assert.NotNil(t, b)
}

func TestSpecIndex_CheckBadURLRef(t *testing.T) {
	yml := `openapi: 3.1.0
paths:
  /cakes:
    post:
      parameters:
        - $ref: 'httpsss://badurl'`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())

	assert.Len(t, index.refErrors, 2)
}

func TestSpecIndex_CheckBadURLRefNoRemoteAllowed(t *testing.T) {
	yml := `openapi: 3.1.0
paths:
  /cakes:
    post:
      parameters:
        - $ref: 'httpsss://badurl'`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	c := CreateClosedAPIIndexConfig()
	idx := NewSpecIndexWithConfig(&rootNode, c)

	assert.Len(t, idx.refErrors, 2)
	assert.Equal(t, "remote lookups are not permitted, "+
		"please set AllowRemoteLookup to true in the configuration", idx.refErrors[0].Error())
}

func TestSpecIndex_CheckIndexDiscoversNoComponentLocalFileReference(t *testing.T) {
	_ = ioutil.WriteFile("coffee-time.yaml", []byte("name: time for coffee"), 0o664)
	defer os.Remove("coffee-time.yaml")

	yml := `openapi: 3.0.3
paths:
  /cakes:
    post:
      parameters:
        - $ref: 'coffee-time.yaml'`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())

	assert.NotNil(t, index.GetAllParametersFromOperations()["/cakes"]["post"]["coffee-time.yaml"][0].Node)
}

func TestSpecIndex_lookupRemoteReference_SeenSourceSimulation_BadJSON(t *testing.T) {
	index := NewSpecIndexWithConfig(nil, &SpecIndexConfig{
		AllowRemoteLookup: true,
	})
	index.seenRemoteSources = make(map[string]*yaml.Node)
	a, b, err := index.lookupRemoteReference("https://google.com//logos/doodles/2022/labor-day-2022-6753651837109490.3-l.png#/hey")
	assert.Error(t, err)
	assert.Nil(t, a)
	assert.Nil(t, b)
}

func TestSpecIndex_lookupFileReference_BadFileName(t *testing.T) {
	index := NewSpecIndexWithConfig(nil, CreateOpenAPIIndexConfig())
	_, _, err := index.lookupFileReference("not-a-reference")
	assert.Error(t, err)
}

func TestSpecIndex_lookupFileReference_SeenSourceSimulation_Error(t *testing.T) {
	index := NewSpecIndexWithConfig(nil, CreateOpenAPIIndexConfig())
	index.seenRemoteSources = make(map[string]*yaml.Node)
	index.seenRemoteSources["magic-money-file.json"] = &yaml.Node{}
	_, _, err := index.lookupFileReference("magic-money-file.json#something")
	assert.Error(t, err)
}

func TestSpecIndex_lookupFileReference_BadFile(t *testing.T) {
	index := NewSpecIndexWithConfig(nil, CreateOpenAPIIndexConfig())
	_, _, err := index.lookupFileReference("chickers.json#no-rice")
	assert.Error(t, err)
}

func TestSpecIndex_lookupFileReference_BadFileDataRead(t *testing.T) {
	_ = ioutil.WriteFile("chickers.yaml", []byte("broke: the: thing: [again]"), 0o664)
	defer os.Remove("chickers.yaml")
	var root yaml.Node
	index := NewSpecIndexWithConfig(&root, CreateOpenAPIIndexConfig())
	_, _, err := index.lookupFileReference("chickers.yaml#no-rice")
	assert.Error(t, err)
}

func TestSpecIndex_lookupFileReference_MultiRes(t *testing.T) {
	_ = ioutil.WriteFile("embie.yaml", []byte("naughty:\n - puppy: dog\n - puppy: naughty\npuppy:\n - naughty: puppy"), 0o664)
	defer os.Remove("embie.yaml")

	index := NewSpecIndexWithConfig(nil, CreateOpenAPIIndexConfig())
	index.seenRemoteSources = make(map[string]*yaml.Node)
	k, doc, err := index.lookupFileReference("embie.yaml#/.naughty")
	assert.NoError(t, err)
	assert.NotNil(t, doc)
	assert.Nil(t, k)
}

func TestSpecIndex_lookupFileReference(t *testing.T) {
	_ = ioutil.WriteFile("fox.yaml", []byte("good:\n - puppy: dog\n - puppy: forever-more"), 0o664)
	defer os.Remove("fox.yaml")

	index := NewSpecIndexWithConfig(nil, CreateOpenAPIIndexConfig())
	index.seenRemoteSources = make(map[string]*yaml.Node)
	k, doc, err := index.lookupFileReference("fox.yaml#/good")
	assert.NoError(t, err)
	assert.NotNil(t, doc)
	assert.NotNil(t, k)
}

func TestSpecIndex_parameterReferencesHavePaths(t *testing.T) {
	_ = ioutil.WriteFile("paramour.yaml", []byte(`components:
  parameters:
    param3:
      name: param3
      in: query
      schema:
        type: string`), 0o664)
	defer os.Remove("paramour.yaml")

	yml := `paths:
  /:
    parameters:
      - $ref: '#/components/parameters/param1'
      - $ref: '#/components/parameters/param1'
      - $ref: 'paramour.yaml#/components/parameters/param3'
    get:
      parameters:
        - $ref: '#/components/parameters/param2'
        - $ref: '#/components/parameters/param2'
        - name: test
          in: query
          schema:
            type: string
components:
  parameters:
    param1:
      name: param1
      in: query
      schema:
        type: string
    param2:
      name: param2
      in: query
      schema:
        type: string`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())

	params := index.GetAllParametersFromOperations()

	if assert.Contains(t, params, "/") {
		if assert.Contains(t, params["/"], "top") {
			if assert.Contains(t, params["/"]["top"], "#/components/parameters/param1") {
				assert.Equal(t, "$.components.parameters.param1", params["/"]["top"]["#/components/parameters/param1"][0].Path)
			}
			if assert.Contains(t, params["/"]["top"], "paramour.yaml#/components/parameters/param3") {
				assert.Equal(t, "$.components.parameters.param3", params["/"]["top"]["paramour.yaml#/components/parameters/param3"][0].Path)
			}
		}
		if assert.Contains(t, params["/"], "get") {
			if assert.Contains(t, params["/"]["get"], "#/components/parameters/param2") {
				assert.Equal(t, "$.components.parameters.param2", params["/"]["get"]["#/components/parameters/param2"][0].Path)
			}
			if assert.Contains(t, params["/"]["get"], "test") {
				assert.Equal(t, "$.paths./.get.parameters[2]", params["/"]["get"]["test"][0].Path)
			}
		}
	}
}

func TestSpecIndex_serverReferencesHaveParentNodesAndPaths(t *testing.T) {
	yml := `servers:
  - url: https://api.example.com/v1
paths:
  /:
    servers:
      - url: https://api.example.com/v2
    get:
      servers:
        - url: https://api.example.com/v3`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())

	rootServers := index.GetAllRootServers()

	for i, server := range rootServers {
		assert.NotNil(t, server.ParentNode)
		assert.Equal(t, fmt.Sprintf("$.servers[%d]", i), server.Path)
	}

	opServers := index.GetAllOperationsServers()

	for path, ops := range opServers {
		for op, servers := range ops {
			for i, server := range servers {
				assert.NotNil(t, server.ParentNode)

				opPath := fmt.Sprintf(".%s", op)
				if op == "top" {
					opPath = ""
				}

				assert.Equal(t, fmt.Sprintf("$.paths.%s%s.servers[%d]", path, opPath, i), server.Path)
			}
		}
	}
}

func TestSpecIndex_schemaComponentsHaveParentsAndPaths(t *testing.T) {
	yml := `components:
  schemas:
    Pet:
      type: object
    Dog:
      type: object`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())

	schemas := index.GetAllSchemas()

	for _, schema := range schemas {
		assert.NotNil(t, schema.ParentNode)
		assert.Equal(t, fmt.Sprintf("$.components.schemas.%s", schema.Name), schema.Path)
	}
}

func TestSpecIndex_ParamsWithDuplicateNamesButUniqueInTypes(t *testing.T) {
	yml := `openapi: 3.1.0
info:
 title: Test
 version: 0.0.1
servers:
 - url: http://localhost:35123
paths:
 /example/{action}:
  parameters:
   - name: fastAction
     in: path
     required: true
     schema:
      type: string
   - name: fastAction
     in: query
     required: true
     schema:
      type: string
  get:
     operationId: example
     parameters:
       - name: action
         in: path
         required: true
         schema:
           type: string
       - name: action
         in: query
         required: true
         schema:
           type: string
     responses:
       "200":
         description: OK`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	idx := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())

	assert.Len(t, idx.paramAllRefs, 4)
	assert.Len(t, idx.paramInlineDuplicateNames, 2)
	assert.Len(t, idx.operationParamErrors, 0)
	assert.Len(t, idx.refErrors, 0)
}

func TestSpecIndex_ParamsWithDuplicateNamesAndSameInTypes(t *testing.T) {
	yml := `openapi: 3.1.0
info:
 title: Test
 version: 0.0.1
servers:
 - url: http://localhost:35123
paths:
 /example/{action}:
  parameters:
   - name: fastAction
     in: path
     required: true
     schema:
      type: string
   - name: fastAction
     in: path
     required: true
     schema:
      type: string
  get:
     operationId: example
     parameters:
       - name: action
         in: path
         required: true
         schema:
           type: string
       - name: action
         in: query
         required: true
         schema:
           type: string
     responses:
       "200":
         description: OK`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	idx := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())

	assert.Len(t, idx.paramAllRefs, 3)
	assert.Len(t, idx.paramInlineDuplicateNames, 2)
	assert.Len(t, idx.operationParamErrors, 1)
	assert.Len(t, idx.refErrors, 0)
}

func TestSpecIndex_foundObjectsWithProperties(t *testing.T) {
	yml := `paths:
  /test:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              type: object
              properties:
                test:
                  type: string
components:
  schemas:
    test:
      type: object
      properties:
        test:
          type: string
    test2:
      type: [object, null]
      properties:
        test:
          type: string
    test3:
      type: object
      additionalProperties: true`

	var rootNode yaml.Node
	yaml.Unmarshal([]byte(yml), &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())

	objects := index.GetAllObjectsWithProperties()
	assert.Len(t, objects, 3)
}

// Example of how to load in an OpenAPI Specification and index it.
func ExampleNewSpecIndex() {
	// define a rootNode to hold our raw spec AST.
	var rootNode yaml.Node

	// load in the stripe OpenAPI specification into bytes (it's pretty meaty)
	stripeSpec, _ := os.ReadFile("../test_specs/stripe.yaml")

	// unmarshal spec into our rootNode
	_ = yaml.Unmarshal(stripeSpec, &rootNode)

	// create a new specification index.
	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())

	// print out some statistics
	fmt.Printf("There are %d references\n"+
		"%d paths\n"+
		"%d operations\n"+
		"%d component schemas\n"+
		"%d reference schemas\n"+
		"%d inline schemas\n"+
		"%d inline schemas that are objects or arrays\n"+
		"%d total schemas\n"+
		"%d enums\n"+
		"%d polymorphic references",
		len(index.GetAllCombinedReferences()),
		len(index.GetAllPaths()),
		index.GetOperationCount(),
		len(index.GetAllComponentSchemas()),
		len(index.GetAllReferenceSchemas()),
		len(index.GetAllInlineSchemas()),
		len(index.GetAllInlineSchemaObjects()),
		len(index.GetAllSchemas()),
		len(index.GetAllEnums()),
		len(index.GetPolyOneOfReferences())+len(index.GetPolyAnyOfReferences()))
	// Output: There are 537 references
	// 246 paths
	// 402 operations
	// 537 component schemas
	// 1972 reference schemas
	// 11749 inline schemas
	// 2612 inline schemas that are objects or arrays
	// 14258 total schemas
	// 1516 enums
	// 828 polymorphic references
}

func TestSpecIndex_GetAllPathsHavePathAndParent(t *testing.T) {
	yml := `openapi: 3.1.0
info:
 title: Test
 version: 0.0.1
servers:
 - url: http://localhost:35123
paths:
 /test:
  get:
     responses:
       "200":
         description: OK
  post:
     responses:
       "200":
         description: OK
 /test2:
  delete:
     responses:
       "200":
         description: OK
  put:
     responses:
       "200":
         description: OK`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	idx := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())

	paths := idx.GetAllPaths()

	assert.Equal(t, "$.paths./test.get", paths["/test"]["get"].Path)
	assert.Equal(t, 9, paths["/test"]["get"].ParentNode.Line)
	assert.Equal(t, "$.paths./test.post", paths["/test"]["post"].Path)
	assert.Equal(t, 13, paths["/test"]["post"].ParentNode.Line)
	assert.Equal(t, "$.paths./test2.delete", paths["/test2"]["delete"].Path)
	assert.Equal(t, 18, paths["/test2"]["delete"].ParentNode.Line)
	assert.Equal(t, "$.paths./test2.put", paths["/test2"]["put"].Path)
	assert.Equal(t, 22, paths["/test2"]["put"].ParentNode.Line)
}
