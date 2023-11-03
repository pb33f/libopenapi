package index

import (
	"errors"
	"fmt"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/utils"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestNewResolver(t *testing.T) {
	assert.Nil(t, NewResolver(nil))
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
		assert.Len(b, utils.UnwrapErrors(indexedErr), 3)

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
	assert.Nil(t, resolver.extractRelatives(nil, nil, nil, nil, journey, false))
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
	assert.Len(t, circ, 3)

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

	assert.Len(t, rolo.GetCaughtErrors(), 3)
	assert.Len(t, rolo.GetRootIndex().GetResolver().GetNonPolymorphicCircularErrors(), 3)
	assert.Len(t, rolo.GetRootIndex().GetResolver().GetPolymorphicCircularErrors(), 0)

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
	assert.Equal(t, "cannot resolve reference `go home, I am drunk`, it's missing: $go home, I am drunk [18:11]", err[0].Error())
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
	server := test_buildMixedRefServer()
	defer server.Close()

	// create a new config that allows local and remote to be mixed up.
	cf := CreateOpenAPIIndexConfig()
	cf.AvoidBuildIndex = true
	cf.AllowRemoteLookup = true
	cf.AvoidCircularReferenceCheck = true
	cf.BasePath = "../test_specs"

	// setting this baseURL will override the base
	cf.BaseURL, _ = url.Parse(server.URL)

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

	assert.NoError(t, indexedErr)

	rolo.Resolve()
	index := rolo.GetRootIndex
	resolver := index().GetResolver()

	assert.Len(t, resolver.GetCircularReferences(), 0)
	assert.Equal(t, 2, resolver.GetIndexesVisited())

	// in v0.8.2 a new check was added when indexing, to prevent re-indexing the same file multiple times.
	assert.Equal(t, 6, resolver.GetRelativesSeen())
	assert.Equal(t, 5, resolver.GetJourneysTaken())
	assert.Equal(t, 7, resolver.GetReferenceVisited())
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
	fmt.Printf("There are %d circular reference errors, %d of them are polymorphic errors, %d are not",
		len(circularErrors), len(resolver.GetPolymorphicCircularErrors()), len(resolver.GetNonPolymorphicCircularErrors()))
	// Output: There are 3 circular reference errors, 0 of them are polymorphic errors, 3 are not
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

	var d = `openapi: 3.1.0
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

	var d = `openapi: 3.1.0
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

	var d = `openapi: 3.1.0
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

/*


 */
