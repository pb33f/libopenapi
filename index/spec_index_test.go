// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"bytes"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/pb33f/libopenapi/utils"
	"golang.org/x/sync/syncmap"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestSpecIndex_GetCache(t *testing.T) {
	petstore, _ := os.ReadFile("../test_specs/petstorev3.json")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(petstore, &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())

	extCache := index.GetCache()
	assert.NotNil(t, extCache)
	extCache.Store("test", "test")
	loaded, ok := extCache.Load("test")
	assert.Equal(t, "test", loaded)
	assert.True(t, ok)

	// create a new cache
	newCache := new(syncmap.Map)
	index.SetCache(newCache)

	// check that the cache has been set.
	assert.Equal(t, newCache, index.GetCache())

	// add an item to the new cache and check it exists
	newCache.Store("test2", "test2")
	loaded, ok = newCache.Load("test2")
	assert.Equal(t, "test2", loaded)
	assert.True(t, ok)

	// now check that the new item in the new cache does not exist in the old cache.
	loaded, ok = extCache.Load("test2")
	assert.Nil(t, loaded)
	assert.False(t, ok)
}

func TestSpecIndex_ExtractRefsStripe(t *testing.T) {
	stripe, _ := os.ReadFile("../test_specs/stripe.yaml")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(stripe, &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())

	assert.Equal(t, 626, len(index.allRefs))
	assert.Equal(t, 871, len(index.allMappedRefs))
	combined := index.GetAllCombinedReferences()
	assert.Equal(t, 871, len(combined))

	assert.Equal(t, len(index.rawSequencedRefs), 2712)
	assert.Equal(t, 336, index.pathCount)
	assert.Equal(t, 494, index.operationCount)
	assert.Equal(t, 871, index.schemaCount)
	assert.Equal(t, 0, index.globalTagsCount)
	assert.Equal(t, 0, index.globalLinksCount)
	assert.Equal(t, 0, index.componentParamCount)
	assert.Equal(t, 162, index.operationParamCount)
	assert.Equal(t, 102, index.componentsInlineParamDuplicateCount)
	assert.Equal(t, 60, index.componentsInlineParamUniqueCount)
	assert.Equal(t, 2579, index.enumCount)
	assert.Equal(t, len(index.GetAllEnums()), 2579)
	assert.Len(t, index.GetPolyAllOfReferences(), 0)
	assert.Len(t, index.GetPolyOneOfReferences(), 315)
	assert.Len(t, index.GetPolyAnyOfReferences(), 708)
	assert.Len(t, index.GetAllReferenceSchemas(), 2712)
	assert.NotNil(t, index.GetRootServersNode())
	assert.Len(t, index.GetAllRootServers(), 1)
	assert.Equal(t, "", index.GetSpecAbsolutePath())
	assert.NotNil(t, index.GetLogger())

	// not required, but flip the circular result switch on and off.
	assert.False(t, index.AllowCircularReferenceResolving())
	index.SetAllowCircularReferenceResolving(true)
	assert.True(t, index.AllowCircularReferenceResolving())

	// simulate setting of circular references, also pointless but needed for coverage.
	assert.Nil(t, index.GetCircularReferences())
	index.SetCircularReferences([]*CircularReferenceResult{new(CircularReferenceResult)})
	assert.Len(t, index.GetCircularReferences(), 1)

	assert.Equal(t, 871, len(index.GetRefsByLine()))
	assert.Equal(t, 2712, len(index.GetLinesWithReferences()), 1972)
	assert.Equal(t, 0, len(index.GetAllExternalDocuments()))
}

func TestSpecIndex_Asana(t *testing.T) {
	asana, _ := os.ReadFile("../test_specs/asana.yaml")
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

	location := "https://raw.githubusercontent.com/digitalocean/openapi/main/specification"
	baseURL, _ := url.Parse(location)

	// create a new config that allows remote lookups.
	cf := &SpecIndexConfig{}
	cf.AvoidBuildIndex = true
	cf.AllowRemoteLookup = true
	cf.AvoidCircularReferenceCheck = true
	cf.Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	// setting this baseURL will override the base
	cf.BaseURL = baseURL

	// create a new rolodex
	rolo := NewRolodex(cf)

	// set the rolodex root node to the root node of the spec.
	rolo.SetRootNode(&rootNode)

	// create a new remote fs and set the config for indexing.
	remoteFS, _ := NewRemoteFSWithConfig(cf)

	// create a handler that uses an env variable to capture any GITHUB_TOKEN in the OS ENV
	// and inject it into the request header, so this does not fail when running lots of local tests.
	if os.Getenv("GH_PAT") != "" {
		fmt.Println("GH_PAT found, setting remote handler func")
		client := &http.Client{
			Timeout: time.Second * 120,
		}
		remoteFS.SetRemoteHandlerFunc(func(url string) (*http.Response, error) {
			request, _ := http.NewRequest(http.MethodGet, url, nil)
			request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("GH_PAT")))
			return client.Do(request)
		})
	}

	// add remote filesystem
	rolo.AddRemoteFS(location, remoteFS)

	// index the rolodex.
	indexedErr := rolo.IndexTheRolodex()
	assert.NoError(t, indexedErr)

	// get all the files!
	files := remoteFS.GetFiles()
	fileLen := len(files)
	assert.Equal(t, 1646, fileLen)
	assert.Len(t, remoteFS.GetErrors(), 0)

	// check circular references
	rolo.CheckForCircularReferences()
	assert.Len(t, rolo.GetCaughtErrors(), 0)
	assert.Len(t, rolo.GetIgnoredCircularReferences(), 0)
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
	doLocal, _ := os.ReadFile(spec)

	var rootNode yaml.Node
	_ = yaml.Unmarshal(doLocal, &rootNode)

	basePath := filepath.Join(tmp, "specification")

	// create a new config that allows local and remote to be mixed up.
	cf := CreateOpenAPIIndexConfig()
	cf.AllowRemoteLookup = true
	cf.AvoidCircularReferenceCheck = true
	cf.BasePath = basePath
	cf.Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	// create a new rolodex
	rolo := NewRolodex(cf)

	// set the rolodex root node to the root node of the spec.
	rolo.SetRootNode(&rootNode)

	// configure the local filesystem.
	fsCfg := LocalFSConfig{
		BaseDirectory: cf.BasePath,
		DirFS:         os.DirFS(cf.BasePath),
		Logger:        cf.Logger,
	}

	// create a new local filesystem.
	fileFS, fsErr := NewLocalFSWithConfig(&fsCfg)
	assert.NoError(t, fsErr)

	files := fileFS.GetFiles()
	fileLen := len(files)

	assert.Equal(t, 1699, fileLen)

	rolo.AddLocalFS(basePath, fileFS)

	rErr := rolo.IndexTheRolodex()

	assert.NoError(t, rErr)

	index := rolo.GetRootIndex()

	assert.NotNil(t, index)

	assert.Len(t, index.GetMappedReferencesSequenced(), 301)
	assert.Len(t, index.GetMappedReferences(), 301)
	assert.Len(t, fileFS.GetErrors(), 0)

	// check circular references
	rolo.CheckForCircularReferences()
	assert.Len(t, rolo.GetCaughtErrors(), 0)
	assert.Len(t, rolo.GetIgnoredCircularReferences(), 0)

	assert.Equal(t, "1.27 MB", rolo.RolodexFileSizeAsString())
	assert.Equal(t, 1699, rolo.RolodexTotalFiles())
}

func TestSpecIndex_DigitalOcean_FullCheckoutLocalResolve_RecursiveLookup(t *testing.T) {
	// this is a full checkout of the digitalocean API repo.
	tmp, _ := os.MkdirTemp("", "openapi")
	cmd := exec.Command("git", "clone", "https://github.com/digitalocean/openapi", tmp)
	defer os.RemoveAll(filepath.Join(tmp, "openapi"))

	err := cmd.Run()
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}

	spec, _ := filepath.Abs(filepath.Join(tmp, "specification", "DigitalOcean-public.v2.yaml"))
	doLocal, _ := os.ReadFile(spec)

	var rootNode yaml.Node
	_ = yaml.Unmarshal(doLocal, &rootNode)

	basePath := filepath.Join(tmp, "specification")

	// create a new config that allows local and remote to be mixed up.
	cf := CreateOpenAPIIndexConfig()
	cf.AllowRemoteLookup = true
	cf.AvoidCircularReferenceCheck = true
	cf.BasePath = basePath
	cf.Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	// create a new rolodex
	rolo := NewRolodex(cf)

	// set the rolodex root node to the root node of the spec.
	rolo.SetRootNode(&rootNode)

	// configure the local filesystem.
	fsCfg := LocalFSConfig{
		BaseDirectory: cf.BasePath,
		IndexConfig:   cf,
		Logger:        cf.Logger,
	}

	// create a new local filesystem.
	fileFS, fsErr := NewLocalFSWithConfig(&fsCfg)
	assert.NoError(t, fsErr)

	rolo.AddLocalFS(basePath, fileFS)

	rErr := rolo.IndexTheRolodex()
	files := fileFS.GetFiles()
	fileLen := len(files)

	assert.Equal(t, 1685, fileLen)

	assert.NoError(t, rErr)

	index := rolo.GetRootIndex()

	assert.NotNil(t, index)

	assert.Len(t, index.GetMappedReferencesSequenced(), 301)
	assert.Len(t, index.GetMappedReferences(), 301)
	assert.Len(t, fileFS.GetErrors(), 0)

	// check circular references
	rolo.CheckForCircularReferences()
	assert.Len(t, rolo.GetCaughtErrors(), 0)
	assert.Len(t, rolo.GetIgnoredCircularReferences(), 0)

	assert.Equal(t, "1.21 MB", rolo.RolodexFileSizeAsString())
	assert.Equal(t, 1685, rolo.RolodexTotalFiles())
}

func TestSpecIndex_DigitalOcean_LookupsNotAllowed(t *testing.T) {
	do, _ := os.ReadFile("../test_specs/digitalocean.yaml")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(do, &rootNode)

	location := "https://raw.githubusercontent.com/digitalocean/openapi/main/specification"
	baseURL, _ := url.Parse(location)

	// create a new config that does not allow remote lookups.
	cf := &SpecIndexConfig{}
	cf.AvoidBuildIndex = true
	cf.AvoidCircularReferenceCheck = true
	var op []byte
	buf := bytes.NewBuffer(op)
	cf.Logger = slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	// setting this baseURL will override the base
	cf.BaseURL = baseURL

	// create a new rolodex
	rolo := NewRolodex(cf)

	// set the rolodex root node to the root node of the spec.
	rolo.SetRootNode(&rootNode)

	// create a new remote fs and set the config for indexing.
	remoteFS, _ := NewRemoteFSWithConfig(cf)

	// add remote filesystem
	rolo.AddRemoteFS(location, remoteFS)

	// index the rolodex.
	indexedErr := rolo.IndexTheRolodex()
	assert.Error(t, indexedErr)
	assert.Len(t, utils.UnwrapErrors(indexedErr), 291)

	index := rolo.GetRootIndex()

	files := remoteFS.GetFiles()
	fileLen := len(files)
	assert.Equal(t, 0, fileLen)
	assert.Len(t, remoteFS.GetErrors(), 0)

	// no lookups allowed, bits have not been set, so there should just be a bunch of errors.
	assert.True(t, len(index.GetReferenceIndexErrors()) > 0)
}

func TestSpecIndex_BaseURLError(t *testing.T) {
	do, _ := os.ReadFile("../test_specs/digitalocean.yaml")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(do, &rootNode)

	location := "https://githerbsandcoffeeandcode.com/fresh/herbs/for/you" // not gonna work bro.
	baseURL, _ := url.Parse(location)

	// create a new config that allows remote lookups.
	cf := &SpecIndexConfig{}
	cf.AvoidBuildIndex = true
	cf.AllowRemoteLookup = true
	cf.AvoidCircularReferenceCheck = true
	var op []byte
	buf := bytes.NewBuffer(op)
	cf.Logger = slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	// setting this baseURL will override the base
	cf.BaseURL = baseURL

	// create a new rolodex
	rolo := NewRolodex(cf)

	// set the rolodex root node to the root node of the spec.
	rolo.SetRootNode(&rootNode)

	// create a new remote fs and set the config for indexing.
	remoteFS, _ := NewRemoteFSWithConfig(cf)

	// add remote filesystem
	rolo.AddRemoteFS(location, remoteFS)

	// index the rolodex.
	indexedErr := rolo.IndexTheRolodex()
	assert.Error(t, indexedErr)
	assert.Len(t, utils.UnwrapErrors(indexedErr), 291)

	files := remoteFS.GetFiles()
	fileLen := len(files)
	assert.Equal(t, 0, fileLen)
	assert.GreaterOrEqual(t, len(remoteFS.GetErrors()), 200)
}

func TestSpecIndex_k8s(t *testing.T) {
	asana, _ := os.ReadFile("../test_specs/k8s.json")
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
	asana, _ := os.ReadFile("../test_specs/petstorev2.json")
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
	xsoar, _ := os.ReadFile("../test_specs/xsoar.json")
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
	petstore, _ := os.ReadFile("../test_specs/petstorev3.json")
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

	index.SetAbsolutePath("/rooty/rootster")
	assert.Equal(t, "/rooty/rootster", index.GetSpecAbsolutePath())
}

var mappedRefs = 15

func TestSpecIndex_BurgerShop(t *testing.T) {
	burgershop, _ := os.ReadFile("../test_specs/burgershop.openapi.yaml")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(burgershop, &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())

	assert.Len(t, index.allRefs, mappedRefs)
	assert.Len(t, index.allMappedRefs, mappedRefs)
	assert.Equal(t, mappedRefs, len(index.GetMappedReferences()))
	assert.Equal(t, mappedRefs+1, len(index.GetMappedReferencesSequenced()))

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
	burgershop, _ := os.ReadFile("../test_specs/all-the-components.yaml")
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
	assert.Nil(t, index.FindComponent("nothing"))
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

func test_buildMixedRefServer() *httptest.Server {
	bs, _ := os.ReadFile("../test_specs/burgershop.openapi.yaml")
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
		_, _ = rw.Write(bs)
	}))
}

func TestSpecIndex_BurgerShopMixedRef(t *testing.T) {
	// create a test server.
	server := test_buildMixedRefServer()
	defer server.Close()

	// create a new config that allows local and remote to be mixed up.
	cf := CreateOpenAPIIndexConfig()
	cf.AvoidBuildIndex = true
	cf.AllowRemoteLookup = true
	cf.AvoidCircularReferenceCheck = true
	cf.BasePath = "../test_specs"
	cf.Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	// setting this baseURL will override the base
	cf.BaseURL, _ = url.Parse(server.URL)

	cFile := "../test_specs/mixedref-burgershop.openapi.yaml"
	yml, _ := os.ReadFile(cFile)
	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	// create a new rolodex
	rolo := NewRolodex(cf)

	// set the rolodex root node to the root node of the spec.
	rolo.SetRootNode(&rootNode)

	// create a new remote fs and set the config for indexing.
	remoteFS, _ := NewRemoteFSWithRootURL(server.URL)
	remoteFS.SetIndexConfig(cf)

	// set our remote handler func

	c := http.Client{}

	remoteFS.RemoteHandlerFunc = c.Get

	// configure the local filesystem.
	fsCfg := LocalFSConfig{
		BaseDirectory: cf.BasePath,
		FileFilters:   []string{"burgershop.openapi.yaml"},
		DirFS:         os.DirFS(cf.BasePath),
	}

	// create a new local filesystem.
	fileFS, err := NewLocalFSWithConfig(&fsCfg)
	assert.NoError(t, err)

	// add file systems to the rolodex
	rolo.AddLocalFS(cf.BasePath, fileFS)
	rolo.AddRemoteFS(server.URL, remoteFS)

	// index the rolodex.
	indexedErr := rolo.IndexTheRolodex()
	rolo.BuildIndexes()

	assert.NoError(t, indexedErr)

	index := rolo.GetRootIndex()
	rolo.CheckForCircularReferences()

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
	assert.Len(t, index.refErrors, 0)
	assert.Len(t, index.GetCircularReferences(), 0)

	// get the size of the rolodex.
	assert.Equal(t, int64(60226), rolo.RolodexFileSize()+int64(len(yml)))
	assert.Equal(t, "50.48 KB", rolo.RolodexFileSizeAsString())
	assert.Equal(t, 3, rolo.RolodexTotalFiles())
}

func TestCalcSizeAsString(t *testing.T) {
	assert.Equal(t, "345 B", HumanFileSize(345))
	assert.Equal(t, "1 KB", HumanFileSize(1024))
	assert.Equal(t, "1 KB", HumanFileSize(1025))
	assert.Equal(t, "1.98 KB", HumanFileSize(2025))
	assert.Equal(t, "1 MB", HumanFileSize(1025*1024))
	assert.Equal(t, "1 GB", HumanFileSize(1025*1025*1025))
}

func TestSpecIndex_TestEmptyBrokenReferences(t *testing.T) {
	badref, _ := os.ReadFile("../test_specs/badref-burgershop.openapi.yaml")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(badref, &rootNode)

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
	assert.Len(t, index.refErrors, 6)
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
		index.FindComponent("#/paths/~1crazy~1ass~1references/get/responses/404/content/application~1xml;%20charset=utf-8/schema").Node.Content[1].Value)

	assert.Equal(t, "a param",
		index.FindComponent("#/paths/~1crazy~1ass~1references/get/parameters/0").Node.Content[1].Value)
}

func TestSpecIndex_FindComponent(t *testing.T) {
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
	assert.Nil(t, index.FindComponent("I-do-not-exist"))
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
	assert.Nil(t, index.lookupRolodex(nil))
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

	assert.Len(t, idx.refErrors, 1)
}

func TestSpecIndex_CheckIndexDiscoversNoComponentLocalFileReference(t *testing.T) {
	c := []byte("name: time for coffee")

	_ = os.WriteFile("coffee-time.yaml", c, 0o664)
	defer os.Remove("coffee-time.yaml")

	// create a new config that allows local and remote to be mixed up.
	cf := CreateOpenAPIIndexConfig()
	cf.AvoidCircularReferenceCheck = true
	cf.BasePath = "."

	// create a new rolodex
	rolo := NewRolodex(cf)

	// configure the local filesystem.
	fsCfg := LocalFSConfig{
		BaseDirectory: cf.BasePath,
		FileFilters:   []string{"coffee-time.yaml"},
		DirFS:         os.DirFS(cf.BasePath),
	}

	// create a new local filesystem.
	fileFS, err := NewLocalFSWithConfig(&fsCfg)
	assert.NoError(t, err)

	yml := `openapi: 3.0.3
paths:
  /cakes:
    post:
      parameters:
        - $ref: 'coffee-time.yaml'`

	var coffee yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &coffee)

	// set the rolodex root node to the root node of the spec.
	rolo.SetRootNode(&coffee)

	rolo.AddLocalFS(cf.BasePath, fileFS)
	rErr := rolo.IndexTheRolodex()

	assert.NoError(t, rErr)

	index := rolo.GetRootIndex()

	assert.NotNil(t, index.GetAllParametersFromOperations()["/cakes"]["post"]["coffee-time.yaml"][0].Node)
}

func TestSpecIndex_lookupFileReference_MultiRes(t *testing.T) {
	embie := []byte("naughty:\n - puppy: dog\n - puppy: naughty\npuppy:\n - naughty: puppy")

	_ = os.WriteFile("embie.yaml", embie, 0o664)
	defer os.Remove("embie.yaml")

	// create a new config that allows local and remote to be mixed up.
	cf := CreateOpenAPIIndexConfig()
	cf.AvoidBuildIndex = true
	cf.AvoidCircularReferenceCheck = true
	cf.BasePath = "."

	// create a new rolodex
	rolo := NewRolodex(cf)

	var myPuppy yaml.Node
	_ = yaml.Unmarshal(embie, &myPuppy)

	// set the rolodex root node to the root node of the spec.
	rolo.SetRootNode(&myPuppy)

	// configure the local filesystem.
	fsCfg := LocalFSConfig{
		BaseDirectory: cf.BasePath,
		FileFilters:   []string{"embie.yaml"},
		DirFS:         os.DirFS(cf.BasePath),
	}

	// create a new local filesystem.
	fileFS, err := NewLocalFSWithConfig(&fsCfg)
	assert.NoError(t, err)

	rolo.AddLocalFS(cf.BasePath, fileFS)
	rErr := rolo.IndexTheRolodex()

	assert.NoError(t, rErr)

	embieRoloFile, fErr := rolo.Open("embie.yaml")

	assert.NoError(t, fErr)
	assert.NotNil(t, embieRoloFile)

	index := rolo.GetRootIndex()
	//index.seenRemoteSources = make(map[string]*yaml.Node)
	absoluteRef, _ := filepath.Abs("embie.yaml#/naughty")
	fRef, _ := index.SearchIndexForReference(absoluteRef)
	assert.NotNil(t, fRef)
}

func TestSpecIndex_lookupFileReference(t *testing.T) {
	pup := []byte("good:\n - puppy: dog\n - puppy: forever-more")

	var myPuppy yaml.Node
	_ = yaml.Unmarshal(pup, &myPuppy)

	_ = os.WriteFile("fox.yaml", pup, 0o664)
	defer os.Remove("fox.yaml")

	// create a new config that allows local and remote to be mixed up.
	cf := CreateOpenAPIIndexConfig()
	cf.AvoidBuildIndex = true
	cf.AvoidCircularReferenceCheck = true
	cf.BasePath = "."

	// create a new rolodex
	rolo := NewRolodex(cf)

	// set the rolodex root node to the root node of the spec.
	rolo.SetRootNode(&myPuppy)

	// configure the local filesystem.
	fsCfg := LocalFSConfig{
		BaseDirectory: cf.BasePath,
		FileFilters:   []string{"fox.yaml"},
		DirFS:         os.DirFS(cf.BasePath),
	}

	// create a new local filesystem.
	fileFS, err := NewLocalFSWithConfig(&fsCfg)
	assert.NoError(t, err)

	rolo.AddLocalFS(cf.BasePath, fileFS)
	rErr := rolo.IndexTheRolodex()

	assert.NoError(t, rErr)

	fox, fErr := rolo.Open("fox.yaml")
	assert.NoError(t, fErr)
	assert.Equal(t, "fox.yaml", fox.Name())
	assert.Equal(t, "good:\n - puppy: dog\n - puppy: forever-more", string(fox.GetContent()))
}

func TestSpecIndex_parameterReferencesHavePaths(t *testing.T) {
	_ = os.WriteFile("paramour.yaml", []byte(`components:
  parameters:
    param3:
      name: param3
      in: query
      schema:
        type: string`), 0o664)
	defer os.Remove("paramour.yaml")

	// create a new config that allows local and remote to be mixed up.
	cf := CreateOpenAPIIndexConfig()
	cf.AvoidBuildIndex = true
	cf.AllowRemoteLookup = true
	cf.AvoidCircularReferenceCheck = true
	cf.BasePath = "."

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

	// create a new rolodex
	rolo := NewRolodex(cf)

	// set the rolodex root node to the root node of the spec.
	rolo.SetRootNode(&rootNode)

	// configure the local filesystem.
	fsCfg := LocalFSConfig{
		BaseDirectory: cf.BasePath,
		FileFilters:   []string{"paramour.yaml"},
		DirFS:         os.DirFS(cf.BasePath),
	}

	// create a new local filesystem.
	fileFS, err := NewLocalFSWithConfig(&fsCfg)
	assert.NoError(t, err)

	// add file system
	rolo.AddLocalFS(cf.BasePath, fileFS)

	// index the rolodex.
	indexedErr := rolo.IndexTheRolodex()
	assert.NoError(t, indexedErr)
	rolo.BuildIndexes()

	index := rolo.GetRootIndex()

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
	// Output: There are 871 references
	// 336 paths
	// 494 operations
	// 871 component schemas
	// 2712 reference schemas
	// 15928 inline schemas
	// 3857 inline schemas that are objects or arrays
	// 19511 total schemas
	// 2579 enums
	// 1023 polymorphic references
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
