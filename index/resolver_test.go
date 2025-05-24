package index

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/utils"
	"github.com/speakeasy-api/jsonpath/pkg/jsonpath"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestNewResolver(t *testing.T) {
	assert.Nil(t, NewResolver(nil))
}

func TestResolvingError_Error(t *testing.T) {
	errs := []error{
		&ResolvingError{
			Path:     "$.test1",
			ErrorRef: errors.New("test1"),
			Node: &yaml.Node{
				Line:   1,
				Column: 1,
			},
		},
		&ResolvingError{
			Path:     "$.test2",
			ErrorRef: errors.New("test2"),
			Node: &yaml.Node{
				Line:   1,
				Column: 1,
			},
		},
	}

	assert.Equal(t, "test1: $.test1 [1:1]", errs[0].Error())
	assert.Equal(t, "test2: $.test2 [1:1]", errs[1].Error())
}

func TestResolvingError_Error_Index(t *testing.T) {
	errs := []error{
		&ResolvingError{
			ErrorRef: errors.Join(&IndexingError{
				Path: "$.test1",
				Err:  errors.New("test1"),
				Node: &yaml.Node{
					Line:   1,
					Column: 1,
				},
			}),
			Node: &yaml.Node{
				Line:   1,
				Column: 1,
			},
		},
		&ResolvingError{
			ErrorRef: errors.Join(&IndexingError{
				Path: "$.test2",
				Err:  errors.New("test2"),
				Node: &yaml.Node{
					Line:   1,
					Column: 1,
				},
			}),
			Node: &yaml.Node{
				Line:   1,
				Column: 1,
			},
		},
	}

	assert.Equal(t, "test1: $.test1 [1:1]", errs[0].Error())
	assert.Equal(t, "test2: $.test2 [1:1]", errs[1].Error())
}

func Benchmark_ResolveDocumentStripe(b *testing.B) {
	baseDir := "../test_specs/stripe.yaml"
	resolveFile, _ := os.ReadFile(baseDir)
	var rootNode yaml.Node
	_ = yaml.Unmarshal(resolveFile, &rootNode)

	for n := 0; n < b.N; n++ {

		cf := CreateOpenAPIIndexConfig()

		rolo := NewRolodex(cf)
		rolo.SetRootNode(&rootNode)

		indexedErr := rolo.IndexTheRolodex()
		assert.Len(b, utils.UnwrapErrors(indexedErr), 1)

	}
}

func TestResolver_ResolveComponents_CircularSpec(t *testing.T) {
	circular, _ := os.ReadFile("../test_specs/circular-tests.yaml")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(circular, &rootNode)

	cf := CreateClosedAPIIndexConfig()
	cf.AvoidCircularReferenceCheck = true
	rolo := NewRolodex(cf)
	rolo.SetRootNode(&rootNode)

	indexedErr := rolo.IndexTheRolodex()
	assert.NoError(t, indexedErr)

	rolo.Resolve()
	assert.Len(t, rolo.GetCaughtErrors(), 3)

	_, err := yaml.Marshal(rolo.GetRootIndex().GetResolver().resolvedRoot)
	assert.NoError(t, err)
}

func TestResolver_CheckForCircularReferences(t *testing.T) {
	circular, _ := os.ReadFile("../test_specs/circular-tests.yaml")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(circular, &rootNode)

	cf := CreateClosedAPIIndexConfig()

	rolo := NewRolodex(cf)
	rolo.SetRootNode(&rootNode)

	indexedErr := rolo.IndexTheRolodex()
	assert.Error(t, indexedErr)
	assert.Len(t, utils.UnwrapErrors(indexedErr), 3)

	rolo.CheckForCircularReferences()

	assert.Len(t, rolo.GetCaughtErrors(), 3)
	assert.Len(t, rolo.GetRootIndex().GetResolver().GetResolvingErrors(), 3)
	assert.Len(t, rolo.GetRootIndex().GetResolver().GetInfiniteCircularReferences(), 3)
}

func TestResolver_CheckForCircularReferences_CatchArray(t *testing.T) {
	circular := []byte(`openapi: 3.0.0
components:
  schemas:
    ProductCategory:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: "array"
          items:
            $ref: "#/components/schemas/ProductCategory"
          description: "Array of sub-categories in the same format."
      required:
        - "name"
        - "children"`)
	var rootNode yaml.Node
	_ = yaml.Unmarshal(circular, &rootNode)

	idx := NewSpecIndexWithConfig(&rootNode, CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	circ := resolver.CheckForCircularReferences()
	assert.Len(t, circ, 1)
	assert.Len(t, resolver.GetResolvingErrors(), 1) // infinite loop is a resolving error.
	assert.Len(t, resolver.GetInfiniteCircularReferences(), 1)
	assert.True(t, resolver.GetInfiniteCircularReferences()[0].IsArrayResult)

	_, err := yaml.Marshal(resolver.resolvedRoot)
	assert.NoError(t, err)
}

func TestResolver_CheckForCircularReferences_IgnoreArray(t *testing.T) {
	circular := []byte(`openapi: 3.0.0
components:
  schemas:
    ProductCategory:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: "array"
          items:
            $ref: "#/components/schemas/ProductCategory"
          description: "Array of sub-categories in the same format."
      required:
        - "name"
        - "children"`)
	var rootNode yaml.Node
	_ = yaml.Unmarshal(circular, &rootNode)

	idx := NewSpecIndexWithConfig(&rootNode, CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	resolver.IgnoreArrayCircularReferences()

	circ := resolver.CheckForCircularReferences()
	assert.Len(t, circ, 0)
	assert.Len(t, resolver.GetResolvingErrors(), 0)
	assert.Len(t, resolver.GetCircularReferences(), 0)

	_, err := yaml.Marshal(resolver.resolvedRoot)
	assert.NoError(t, err)
}

func TestResolver_CheckForCircularReferences_IgnorePoly_Any(t *testing.T) {
	circular := []byte(`openapi: 3.1.0
paths:
  /one:
    get:
      responses:
        '200':
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/One'
components:
  schemas:
    One:
      properties:
        thing:
          oneOf:
            - "$ref": "#/components/schemas/Two"
            - "$ref": "#/components/schemas/Three"
      required:
        - thing
    Two:
      description: "test two"
      properties:
        testThing:
          "$ref": "#/components/schemas/One"
    Three:
      description: "test three"
      properties:
        testThing:
          "$ref": "#/components/schemas/One"`)
	var rootNode yaml.Node
	_ = yaml.Unmarshal(circular, &rootNode)

	idx := NewSpecIndexWithConfig(&rootNode, CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	resolver.IgnorePolymorphicCircularReferences()

	circ := resolver.CheckForCircularReferences()
	assert.Len(t, circ, 0)
	assert.Len(t, resolver.GetResolvingErrors(), 0)
	assert.Len(t, resolver.GetCircularReferences(), 0)

	_, err := yaml.Marshal(resolver.resolvedRoot)
	assert.NoError(t, err)
}

func TestResolver_CheckForCircularReferences_IgnorePoly_All(t *testing.T) {
	circular := []byte(`openapi: 3.0.0
components:
  schemas:
    ProductCategory:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: "object"
          allOf:
            - $ref: "#/components/schemas/ProductCategory"
          description: "Array of sub-categories in the same format."
      required:
        - "name"
        - "children"`)
	var rootNode yaml.Node
	_ = yaml.Unmarshal(circular, &rootNode)

	idx := NewSpecIndexWithConfig(&rootNode, CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	resolver.IgnorePolymorphicCircularReferences()

	circ := resolver.CheckForCircularReferences()
	assert.Len(t, circ, 0)
	assert.Len(t, resolver.GetResolvingErrors(), 0)
	assert.Len(t, resolver.GetCircularReferences(), 0)

	_, err := yaml.Marshal(resolver.resolvedRoot)
	assert.NoError(t, err)
}

func TestResolver_CheckForCircularReferences_IgnorePoly_One(t *testing.T) {
	circular := []byte(`openapi: 3.0.0
components:
  schemas:
    ProductCategory:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: "object"
          oneOf:
            - $ref: "#/components/schemas/ProductCategory"
          description: "Array of sub-categories in the same format."
      required:
        - "name"
        - "children"`)
	var rootNode yaml.Node
	_ = yaml.Unmarshal(circular, &rootNode)

	idx := NewSpecIndexWithConfig(&rootNode, CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	resolver.IgnorePolymorphicCircularReferences()

	circ := resolver.CheckForCircularReferences()
	assert.Len(t, circ, 0)
	assert.Len(t, resolver.GetResolvingErrors(), 0)
	assert.Len(t, resolver.GetCircularReferences(), 0)

	_, err := yaml.Marshal(resolver.resolvedRoot)
	assert.NoError(t, err)
}

func TestResolver_CheckForCircularReferences_CatchPoly_Any(t *testing.T) {
	circular := []byte(`openapi: 3.0.0
components:
  schemas:
    ProductCategory:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: "object"
          anyOf:
            - $ref: "#/components/schemas/ProductCategory"
          description: "Array of sub-categories in the same format."
      required:
        - "name"
        - "children"`)
	var rootNode yaml.Node
	_ = yaml.Unmarshal(circular, &rootNode)

	idx := NewSpecIndexWithConfig(&rootNode, CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	circ := resolver.CheckForCircularReferences()
	assert.Len(t, circ, 0)
	assert.Len(t, resolver.GetResolvingErrors(), 0) // not an infinite loop if poly.
	assert.Len(t, resolver.GetCircularReferences(), 1)
	assert.Equal(t, "anyOf", resolver.GetCircularReferences()[0].PolymorphicType)
	_, err := yaml.Marshal(resolver.resolvedRoot)
	assert.NoError(t, err)
}

func TestResolver_CheckForCircularReferences_CatchPoly_All(t *testing.T) {
	circular := []byte(`openapi: 3.0.0
components:
  schemas:
    ProductCategory:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: "object"
          allOf:
            - $ref: "#/components/schemas/ProductCategory"
          description: "Array of sub-categories in the same format."
      required:
        - "name"
        - "children"`)
	var rootNode yaml.Node
	_ = yaml.Unmarshal(circular, &rootNode)

	idx := NewSpecIndexWithConfig(&rootNode, CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	circ := resolver.CheckForCircularReferences()
	assert.Len(t, circ, 0)
	assert.Len(t, resolver.GetResolvingErrors(), 0) // not an infinite loop if poly.
	assert.Len(t, resolver.GetCircularReferences(), 1)
	assert.Equal(t, "allOf", resolver.GetCircularReferences()[0].PolymorphicType)
	assert.True(t, resolver.GetCircularReferences()[0].IsPolymorphicResult)
	_, err := yaml.Marshal(resolver.resolvedRoot)
	assert.NoError(t, err)
}

func TestResolver_CircularReferencesRequiredValid(t *testing.T) {
	circular, _ := os.ReadFile("../test_specs/swagger-valid-recursive-model.yaml")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(circular, &rootNode)

	idx := NewSpecIndexWithConfig(&rootNode, CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	circ := resolver.CheckForCircularReferences()
	assert.Len(t, circ, 0)

	_, err := yaml.Marshal(resolver.resolvedRoot)
	assert.NoError(t, err)
}

func TestResolver_CircularReferencesRequiredInvalid(t *testing.T) {
	circular, _ := os.ReadFile("../test_specs/swagger-invalid-recursive-model.yaml")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(circular, &rootNode)

	idx := NewSpecIndexWithConfig(&rootNode, CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	circ := resolver.CheckForCircularReferences()
	assert.Len(t, circ, 2)

	_, err := yaml.Marshal(resolver.resolvedRoot)
	assert.NoError(t, err)
}

func TestResolver_DeepJourney(t *testing.T) {
	var journey []*Reference
	for f := 0; f < 200; f++ {
		journey = append(journey, nil)
	}
	idx := NewSpecIndexWithConfig(nil, CreateClosedAPIIndexConfig())
	resolver := NewResolver(idx)
	assert.Nil(t, resolver.extractRelatives(nil, nil, nil, nil, journey, nil, false, 0))
}

func TestResolver_DeepDepth(t *testing.T) {
	var refA, refB *yaml.Node

	refA = &yaml.Node{
		Value: "A",
		Tag:   "!!seq",
	}

	refB = &yaml.Node{
		Value: "B",
		Tag:   "!!seq",
	}

	refA.Content = append(refA.Content, refB)
	refB.Content = append(refB.Content, refA)

	idx := NewSpecIndexWithConfig(nil, CreateClosedAPIIndexConfig())
	resolver := NewResolver(idx)

	// add a logger
	var log []byte
	buf := bytes.NewBuffer(log)
	logger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	idx.logger = logger

	ref := &Reference{
		FullDefinition: "#/components/schemas/A",
	}
	found := resolver.extractRelatives(ref, refA, nil, nil, nil, nil, false, 0)

	assert.Nil(t, found)
	assert.Contains(t, buf.String(), "libopenapi resolver: relative depth exceeded 100 levels")
}

func TestResolver_ResolveComponents_Stripe_NoRolodex(t *testing.T) {
	baseDir := "../test_specs/stripe.yaml"

	resolveFile, _ := os.ReadFile(baseDir)

	var stripeRoot yaml.Node
	_ = yaml.Unmarshal(resolveFile, &stripeRoot)

	info, _ := datamodel.ExtractSpecInfoWithDocumentCheck(resolveFile, true)

	cf := CreateOpenAPIIndexConfig()
	cf.SpecInfo = info

	idx := NewSpecIndexWithConfig(&stripeRoot, cf)

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	circ := resolver.CheckForCircularReferences()
	assert.Len(t, circ, 1)

	_, err := yaml.Marshal(resolver.resolvedRoot)
	assert.NoError(t, err)
}

func TestResolver_ResolveComponents_Stripe(t *testing.T) {
	baseDir := "../test_specs/stripe.yaml"

	resolveFile, _ := os.ReadFile(baseDir)

	var stripeRoot yaml.Node
	_ = yaml.Unmarshal(resolveFile, &stripeRoot)

	info, _ := datamodel.ExtractSpecInfoWithDocumentCheck(resolveFile, true)

	cf := CreateOpenAPIIndexConfig()
	cf.SpecInfo = info
	cf.AvoidCircularReferenceCheck = true

	rolo := NewRolodex(cf)
	rolo.SetRootNode(&stripeRoot)

	indexedErr := rolo.IndexTheRolodex()
	assert.NoError(t, indexedErr)

	// after resolving, the rolodex will have errors.
	rolo.Resolve()

	assert.Len(t, rolo.GetCaughtErrors(), 1)
	assert.Len(t, rolo.GetRootIndex().GetResolver().GetNonPolymorphicCircularErrors(), 1)
	assert.Len(t, rolo.GetRootIndex().GetResolver().GetPolymorphicCircularErrors(), 0)
	assert.Len(t, rolo.GetRootIndex().GetResolver().GetSafeCircularReferences(), 25)
}

func TestResolver_ResolveComponents_BurgerShop(t *testing.T) {
	mixedref, _ := os.ReadFile("../test_specs/burgershop.openapi.yaml")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(mixedref, &rootNode)

	idx := NewSpecIndexWithConfig(&rootNode, CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	circ := resolver.Resolve()
	assert.Len(t, circ, 0)
}

func TestResolver_ResolveComponents_PolyNonCircRef(t *testing.T) {
	yml := `paths:
  /hey:
    get:
      responses:
        "200":
          $ref: '#/components/schemas/crackers'
components:
  schemas:
    cheese:
      description: cheese
      anyOf:
        items:
          $ref: '#/components/schemas/crackers' 
    crackers:
      description: crackers
      allOf:
       - $ref: '#/components/schemas/tea'
    tea:
      description: tea`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	idx := NewSpecIndexWithConfig(&rootNode, CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	circ := resolver.CheckForCircularReferences()
	assert.Len(t, circ, 0)
}

func TestResolver_ResolveComponents_PolyCircRef(t *testing.T) {
	yml := `openapi: 3.1.0
components:
  schemas:
    cheese:
      description: cheese
      anyOf:
        - $ref: '#/components/schemas/crackers' 
    crackers:
      description: crackers
      anyOf:
       - $ref: '#/components/schemas/cheese'
    tea:
      description: tea`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	idx := NewSpecIndexWithConfig(&rootNode, CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	_ = resolver.CheckForCircularReferences()
	resolver.circularReferences[0].IsInfiniteLoop = true // override
	assert.Len(t, idx.GetCircularReferences(), 1)
	assert.Len(t, resolver.GetPolymorphicCircularErrors(), 1)
	assert.Equal(t, 2, idx.GetCircularReferences()[0].LoopIndex)
}

func TestResolver_ResolveComponents_Missing(t *testing.T) {
	yml := `paths:
  /hey:
    get:
      responses:
        "200":
          $ref: '#/components/schemas/crackers'
components:
  schemas:
    cheese:
      description: cheese
      properties:
        thang:
          $ref: '#/components/schemas/crackers' 
    crackers:
      description: crackers
      properties:
        butter:
          $ref: 'go home, I am drunk'`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	idx := NewSpecIndexWithConfig(&rootNode, CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	err := resolver.Resolve()
	assert.Len(t, err, 2)
	assert.Equal(t, "cannot resolve reference `go home, I am drunk`, it's missing: $.['go home, I am drunk'] [18:11]", err[0].Error())
}

func TestResolver_ResolveThroughPaths(t *testing.T) {
	yml := `paths:
  /pizza/{cake}/{pizza}/pie:
    parameters:
      - name: juicy
  /companies/{companyId}/data/payments/{paymentId}:
    get:
      tags:
        - Accounts receivable
      parameters:
        - $ref: '#/paths/~1pizza~1%7Bcake%7D~1%7Bpizza%7D~1pie/parameters/0'`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	idx := NewSpecIndexWithConfig(&rootNode, CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	err := resolver.Resolve()
	assert.Len(t, err, 0)
}

func TestResolver_ResolveComponents_MixedRef(t *testing.T) {
	mixedref, _ := os.ReadFile("../test_specs/mixedref-burgershop.openapi.yaml")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(mixedref, &rootNode)

	// create a test server.
	// server := test_buildMixedRefServer()
	// defer server.Close()

	// create a new config that allows local and remote to be mixed up.
	cf := CreateOpenAPIIndexConfig()
	cf.AvoidBuildIndex = true
	cf.AllowRemoteLookup = true
	cf.AvoidCircularReferenceCheck = true
	cf.BasePath = "../test_specs"
	cf.SpecAbsolutePath, _ = filepath.Abs("../test_specs/mixedref-burgershop.openapi.yaml")
	cf.ExtractRefsSequentially = true

	// setting this baseURL will override the base
	cf.BaseURL, _ = url.Parse("https://raw.githubusercontent.com/daveshanley/vacuum/main/model/test_files/")

	// create a new rolodex
	rolo := NewRolodex(cf)

	// set the rolodex root node to the root node of the spec.
	rolo.SetRootNode(&rootNode)

	// create a new remote fs and set the config for indexing.
	remoteFS, _ := NewRemoteFSWithConfig(cf)
	remoteFS.SetIndexConfig(cf)

	// set our remote handler func

	c := http.Client{}

	remoteFS.RemoteHandlerFunc = c.Get

	// configure the local filesystem.
	fsCfg := LocalFSConfig{
		BaseDirectory: cf.BasePath,
		IndexConfig:   cf,
	}

	// create a new local filesystem.
	fileFS, err := NewLocalFSWithConfig(&fsCfg)
	assert.NoError(t, err)

	// add file systems to the rolodex
	rolo.AddLocalFS(cf.BasePath, fileFS)
	rolo.AddRemoteFS("https://raw.githubusercontent.com/daveshanley/vacuum/main/model/test_files/", remoteFS)

	// index the rolodex.
	indexedErr := rolo.IndexTheRolodex()

	assert.NoError(t, indexedErr)

	rolo.Resolve()
	index := rolo.GetRootIndex
	resolver := index().GetResolver()

	assert.Len(t, resolver.GetCircularReferences(), 0)
	assert.Equal(t, 9, resolver.GetIndexesVisited())

	// in v0.8.2 a new check was added when indexing, to prevent re-indexing the same file multiple times.
	assert.Equal(t, 6, resolver.GetRelativesSeen())
	assert.Equal(t, 15, resolver.GetJourneysTaken())
	assert.Equal(t, 17, resolver.GetReferenceVisited())
}

func TestResolver_ResolveComponents_k8s(t *testing.T) {
	k8s, _ := os.ReadFile("../test_specs/k8s.json")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(k8s, &rootNode)

	idx := NewSpecIndexWithConfig(&rootNode, CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	circ := resolver.Resolve()
	assert.Len(t, circ, 0)
}

// Example of how to resolve the Stripe OpenAPI specification, and check for circular reference errors
func ExampleNewResolver() {
	// create a yaml.Node reference as a root node.
	var rootNode yaml.Node

	//  load in the Stripe OpenAPI spec (lots of polymorphic complexity in here)
	stripeBytes, _ := os.ReadFile("../test_specs/stripe.yaml")

	// unmarshal bytes into our rootNode.
	_ = yaml.Unmarshal(stripeBytes, &rootNode)

	// create a new spec index (resolver depends on it)
	indexConfig := CreateClosedAPIIndexConfig()
	idx := NewSpecIndexWithConfig(&rootNode, indexConfig)

	// create a new resolver using the index.
	resolver := NewResolver(idx)

	// resolve the document, if there are circular reference errors, they are returned/
	// WARNING: this is a destructive action and the rootNode will be PERMANENTLY altered and cannot be unresolved
	circularErrors := resolver.Resolve()

	// The Stripe API has a bunch of circular reference problems, mainly from polymorphism.
	// So let's print them out.
	//
	fmt.Printf("There is %d circular reference error, %d of them are polymorphic errors, %d are not\n"+
		"with a total pf %d safe circular references.\n",
		len(circularErrors), len(resolver.GetPolymorphicCircularErrors()), len(resolver.GetNonPolymorphicCircularErrors()),
		len(resolver.GetSafeCircularReferences()))
	// Output: There is 1 circular reference error, 0 of them are polymorphic errors, 1 are not
	// with a total pf 25 safe circular references.
}

func ExampleResolvingError() {
	re := ResolvingError{
		ErrorRef: errors.New("je suis une erreur"),
		Node: &yaml.Node{
			Line:   5,
			Column: 21,
		},
		Path:              "#/definitions/JeSuisUneErreur",
		CircularReference: &CircularReferenceResult{},
	}

	fmt.Printf("%s", re.Error())
	// Output: je suis une erreur: #/definitions/JeSuisUneErreur [5:21]
}

func TestDocument_IgnoreArrayCircularReferences(t *testing.T) {
	d := `openapi: 3.1.0
components:
  schemas:
    ProductCategory:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: "array"
          items:
            $ref: "#/components/schemas/ProductCategory"
          description: "Array of sub-categories in the same format."
      required:
        - "name"
        - "children"`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)

	idx := NewSpecIndexWithConfig(&rootNode, CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	resolver.IgnoreArrayCircularReferences()
	assert.NotNil(t, resolver)

	circ := resolver.Resolve()
	assert.Len(t, circ, 0)
	assert.Len(t, resolver.GetIgnoredCircularArrayReferences(), 1)
}

func TestDocument_IgnorePolyCircularReferences(t *testing.T) {
	d := `openapi: 3.1.0
components:
  schemas:
    ProductCategory:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: "object"
          anyOf:
            - $ref: "#/components/schemas/ProductCategory"
          description: "Array of sub-categories in the same format."
      required:
        - "name"
        - "children"`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)

	idx := NewSpecIndexWithConfig(&rootNode, CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	resolver.IgnorePolymorphicCircularReferences()
	assert.NotNil(t, resolver)

	circ := resolver.Resolve()
	assert.Len(t, circ, 0)
	assert.Len(t, resolver.GetIgnoredCircularPolyReferences(), 1)
}

func TestDocument_IgnorePolyCircularReferences_NoArrayForRef(t *testing.T) {
	d := `openapi: 3.1.0
components:
  schemas:
    bingo:
       type: object
       properties:
         bango:
           $ref: "#/components/schemas/ProductCategory"
    ProductCategory:
      type: "object"
      properties:
        name:
          type: "string"
        children:
          type: "object"
          items:
            anyOf:
              items:
                $ref: "#/components/schemas/ProductCategory"
          description: "Array of sub-categories in the same format."
      required:
        - "name"
        - "children"`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)

	idx := NewSpecIndexWithConfig(&rootNode, CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	resolver.IgnorePolymorphicCircularReferences()
	assert.NotNil(t, resolver)

	circ := resolver.Resolve()
	assert.Len(t, circ, 0)
	assert.Len(t, resolver.GetIgnoredCircularPolyReferences(), 1)
}

func TestResolver_isInfiniteCircularDep_NoRef(t *testing.T) {
	resolver := NewResolver(nil)
	a, b := resolver.isInfiniteCircularDependency(nil, nil, nil)
	assert.False(t, a)
	assert.Nil(t, b)
}

func TestResolver_AllowedCircle(t *testing.T) {
	d := `openapi: 3.1.0
paths:
  /test:
    get:
      responses:
        '200':
          description: OK
components:
  schemas:
    Obj:
      type: object
      properties:
        other:
          $ref: '#/components/schemas/Obj2'
    Obj2:
      type: object
      properties:
        other:
          $ref: '#/components/schemas/Obj'
      required:
        - other`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)

	idx := NewSpecIndexWithConfig(&rootNode, CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	circ := resolver.Resolve()
	assert.Len(t, circ, 0)
	assert.Len(t, resolver.GetInfiniteCircularReferences(), 0)
	assert.Len(t, resolver.GetSafeCircularReferences(), 1)
}

func TestResolver_AllowedCircle_Array(t *testing.T) {
	d := `openapi: 3.1.0
components:
  schemas:
    Obj:
      type: object
      properties:
        other:
          $ref: '#/components/schemas/Obj2'
      required:
        - other
    Obj2:
      type: object
      properties:
        children:
          type: array
          items:
            $ref: '#/components/schemas/Obj'
      required:
        - children`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)

	cf := CreateClosedAPIIndexConfig()
	cf.IgnoreArrayCircularReferences = true

	idx := NewSpecIndexWithConfig(&rootNode, cf)

	resolver := NewResolver(idx)
	resolver.IgnoreArrayCircularReferences()
	assert.NotNil(t, resolver)

	circ := resolver.Resolve()
	assert.Len(t, circ, 0)
	assert.Len(t, resolver.GetInfiniteCircularReferences(), 0)
	assert.Len(t, resolver.GetSafeCircularReferences(), 0)
	assert.Len(t, resolver.GetIgnoredCircularArrayReferences(), 1)
}

func TestResolver_CatchInvalidMapPolyCircle(t *testing.T) {
	d := `openapi: 3.1.0
paths:
  /test:
    get:
      responses:
        '200':
          description: OK
components:
  schemas:
    ObjectWithOneOf:
      type: object
      properties:
        child:
          oneOf:
            $ref: '#/components/schemas/ObjectWithOneOf'
      required:
        - child
`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)

	cf := CreateClosedAPIIndexConfig()
	idx := NewSpecIndexWithConfig(&rootNode, cf)

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	circ := resolver.Resolve()
	assert.Len(t, circ, 0)
	assert.Len(t, resolver.GetInfiniteCircularReferences(), 0)
	assert.Len(t, resolver.GetSafeCircularReferences(), 1)
	assert.Len(t, resolver.GetIgnoredCircularPolyReferences(), 0)
}

func TestResolver_CatchInvalidMapPolyCircle_Ignored(t *testing.T) {
	d := `openapi: 3.1.0
paths:
  /test:
    get:
      responses:
        '200':
          description: OK
components:
  schemas:
    ObjectWithOneOf:
      type: object
      properties:
        child:
          oneOf:
            $ref: '#/components/schemas/ObjectWithOneOf'
      required:
        - child
`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)

	cf := CreateClosedAPIIndexConfig()
	cf.IgnorePolymorphicCircularReferences = true

	idx := NewSpecIndexWithConfig(&rootNode, cf)

	resolver := NewResolver(idx)
	resolver.IgnorePolymorphicCircularReferences()
	assert.NotNil(t, resolver)

	circ := resolver.Resolve()
	assert.Len(t, circ, 0)
	assert.Len(t, resolver.GetInfiniteCircularReferences(), 0)
	assert.Len(t, resolver.GetSafeCircularReferences(), 0)
	assert.Len(t, resolver.GetIgnoredCircularPolyReferences(), 1)
}

func TestResolver_CatchInvalidMapPoly(t *testing.T) {
	d := `openapi: 3.1.0
paths:
  /test:
    get:
      responses:
        '200':
          description: OK
components:
  schemas:
    Anything:
      type: object
    ObjectWithOneOf:
      type: object
      properties:
        child:
          oneOf:
            $ref: '#/components/schemas/Anything'
      required:
        - child
`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)

	cf := CreateClosedAPIIndexConfig()
	cf.IgnorePolymorphicCircularReferences = true

	idx := NewSpecIndexWithConfig(&rootNode, cf)

	resolver := NewResolver(idx)
	resolver.IgnorePolymorphicCircularReferences()
	assert.NotNil(t, resolver)

	circ := resolver.Resolve()
	assert.Len(t, circ, 0)
	assert.Len(t, resolver.GetInfiniteCircularReferences(), 0)
	assert.Len(t, resolver.GetSafeCircularReferences(), 0)
	assert.Len(t, resolver.GetIgnoredCircularPolyReferences(), 0)
}

func TestResolver_NotAllowedDeepCircle(t *testing.T) {
	d := `openapi: 3.0
components:
  schemas:
    Three:
      description: "test three"
      properties:
        bester:
          "$ref": "#/components/schemas/Seven"
      required:
       - bester
    Seven:
      properties:
        wow:
          "$ref": "#/components/schemas/Three"
      required:
        - wow`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)

	idx := NewSpecIndexWithConfig(&rootNode, CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	circ := resolver.Resolve()
	assert.Len(t, circ, 1)
	assert.Len(t, resolver.GetInfiniteCircularReferences(), 1)
	assert.Len(t, resolver.GetSafeCircularReferences(), 0)
}

func TestLocateRefEnd_WithResolve(t *testing.T) {
	yml, _ := os.ReadFile("../../test_specs/first.yaml")
	var bsn yaml.Node
	_ = yaml.Unmarshal(yml, &bsn)

	cf := CreateOpenAPIIndexConfig()
	cf.BasePath = "../test_specs"

	localFSConfig := &LocalFSConfig{
		BaseDirectory: cf.BasePath,
		FileFilters:   []string{"first.yaml", "second.yaml", "third.yaml", "fourth.yaml"},
		DirFS:         os.DirFS(cf.BasePath),
	}
	localFs, _ := NewLocalFSWithConfig(localFSConfig)
	rolo := NewRolodex(cf)
	rolo.AddLocalFS(cf.BasePath, localFs)
	rolo.SetRootNode(&bsn)
	rolo.IndexTheRolodex()

	wd, _ := os.Getwd()
	cp, _ := filepath.Abs(filepath.Join(wd, "../test_specs/third.yaml"))
	third := localFs.GetFiles()[cp]
	refs := third.GetIndex().GetMappedReferences()
	fullDef := fmt.Sprintf("%s#/properties/property/properties/statistics", cp)
	ref := refs[fullDef]

	assert.Equal(t, "statistics", ref.Name)
	isRef, _, _ := utils.IsNodeRefValue(ref.Node)
	assert.True(t, isRef)

	// resolve the stack, it should convert the ref to a node.
	rolo.Resolve()

	isRef, _, _ = utils.IsNodeRefValue(ref.Node)
	assert.False(t, isRef)
}

func TestResolveDoc_Issue195(t *testing.T) {
	spec := `openapi: 3.0.1
info:
  title: Some Example!
paths:
  "/pet/findByStatus":
    get:
      responses:
        default:
          content:
            application/json:
              schema:
                "$ref": https://raw.githubusercontent.com/pb33f/openapi-specification/main/examples/v3.0/petstore.yaml#/components/schemas/Error`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(spec), &rootNode)

	// create an index config
	config := CreateOpenAPIIndexConfig()

	// the rolodex will automatically try and check for circular references, you don't want to do this
	// if you're resolving the spec, as the node tree is marked as 'seen' and you won't be able to resolve
	// correctly.
	config.AvoidCircularReferenceCheck = true

	// new in 0.13+ is the ability to add remote and local file systems to the index
	// requires a new part, the rolodex. It holds all the indexes and knows where to find
	// every reference across local and remote files.
	rolodex := NewRolodex(config)

	// add a new remote file system.
	remoteFS, _ := NewRemoteFSWithConfig(config)

	// add the remote file system to the rolodex
	rolodex.AddRemoteFS("", remoteFS)

	// set the root node of the rolodex, this is your spec.
	rolodex.SetRootNode(&rootNode)

	// index the rolodex
	indexingError := rolodex.IndexTheRolodex()
	if indexingError != nil {
		panic(indexingError)
	}

	// resolve the rolodex
	rolodex.Resolve()

	// there should be no errors at this point
	resolvingErrors := rolodex.GetCaughtErrors()
	if resolvingErrors != nil {
		panic(resolvingErrors)
	}

	// perform some lookups.
	var nodes []*yaml.Node

	// pull out schema type
	path, _ := jsonpath.NewPath("$.paths['/pet/findByStatus'].get.responses['default'].content['application/json'].schema.type")
	nodes = path.Query(&rootNode)
	assert.Equal(t, nodes[0].Value, "object")

	// pull out required array
	path, _ = jsonpath.NewPath("$.paths['/pet/findByStatus'].get.responses['default'].content['application/json'].schema.required")
	nodes = path.Query(&rootNode)
	assert.Equal(t, nodes[0].Content[0].Value, "code")
	assert.Equal(t, nodes[0].Content[1].Value, "message")
}

func TestDocument_LoopThroughAnArray(t *testing.T) {
	d := `openapi: "3.0.1"
components:
  schemas:
    B:
      type: object
      properties:
        children:
          type: array
          items:
            $ref: '#/components/schemas/B'`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)

	config := CreateClosedAPIIndexConfig()
	config.IgnoreArrayCircularReferences = true
	idx := NewSpecIndexWithConfig(&rootNode, config)

	resolver := NewResolver(idx)
	resolver.IgnoreArrayCircularReferences()
	assert.NotNil(t, resolver)

	circ := resolver.Resolve()
	assert.Len(t, circ, 0)
	assert.Len(t, resolver.GetIgnoredCircularArrayReferences(), 1)
}

func TestDocument_ObjectWithPolyAndArray(t *testing.T) {
	d := `openapi: "3.0.1"
components:
  schemas:
    A:
      type: object
      properties: {}
    B:
      type: object
      allOf:
        - $ref: '#/components/schemas/A' 
      properties:
        children:
          type: array
          items:
            $ref: '#/components/schemas/B'`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)

	config := CreateClosedAPIIndexConfig()
	config.IgnoreArrayCircularReferences = true
	idx := NewSpecIndexWithConfig(&rootNode, config)

	resolver := NewResolver(idx)
	resolver.IgnoreArrayCircularReferences()
	assert.NotNil(t, resolver)

	circ := resolver.Resolve()
	assert.Len(t, circ, 0)
	assert.Len(t, resolver.GetIgnoredCircularArrayReferences(), 1)
}

func TestDocument_ObjectWithMultiPolyAndArray(t *testing.T) {
	d := `openapi: "3.0.1"
components:
  schemas:
    A:
      type: object
      properties: {}
    B:
      type: object
      allOf:
        - $ref: '#/components/schemas/A'
      oneOf:
        - $ref: '#/components/schemas/B'
      properties:
        children:
          type: array
          items:
            $ref: '#/components/schemas/B'`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)

	config := CreateClosedAPIIndexConfig()
	config.IgnoreArrayCircularReferences = true
	idx := NewSpecIndexWithConfig(&rootNode, config)

	resolver := NewResolver(idx)
	resolver.IgnoreArrayCircularReferences()
	assert.NotNil(t, resolver)

	circ := resolver.Resolve()
	assert.Len(t, circ, 0)
	assert.Len(t, resolver.GetSafeCircularReferences(), 1)
	assert.Len(t, resolver.GetIgnoredCircularArrayReferences(), 0)
}

func TestIssue_259(t *testing.T) {
	d := `openapi: 3.0.3
info:
  description: test
  title: test
  version: test
paths: {}
components:
  schemas:
    test:
      type: object
      properties:
        Reference:
          $ref: '#/components/schemas/ReferenceType'
    ReferenceType:
      type: object
      required: [$ref]
      properties: 
        $ref:
          type: string`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)

	config := CreateClosedAPIIndexConfig()
	config.IgnoreArrayCircularReferences = true
	idx := NewSpecIndexWithConfig(&rootNode, config)

	resolver := NewResolver(idx)
	resolver.IgnoreArrayCircularReferences()
	assert.NotNil(t, resolver)

	errs := resolver.Resolve()
	assert.Len(t, errs, 0)
}

func TestIssue_481(t *testing.T) {
	d := `openapi: 3.0.1
components:
  schemas:
    PetPot:
      type: object
      properties:
        value:
          oneOf:
            - type: array
              items:
                type: object
                required:
                  - $ref
                  - value`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)

	config := CreateClosedAPIIndexConfig()
	config.IgnoreArrayCircularReferences = true
	idx := NewSpecIndexWithConfig(&rootNode, config)

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	errs := resolver.Resolve()
	assert.Len(t, errs, 0)
}

func TestVisitReference_Nil(t *testing.T) {
	d := `banana`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)

	config := CreateClosedAPIIndexConfig()
	idx := NewSpecIndexWithConfig(&rootNode, config)

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	errs := resolver.Resolve()
	assert.Len(t, errs, 0)
	n := resolver.VisitReference(nil, nil, nil, false)
	assert.Nil(t, n)
}

// func (resolver *Resolver) VisitReference(ref *Reference, seen map[string]bool, journey []*Reference, resolve bool) []*yaml.Node {
