package resolver

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestNewResolver(t *testing.T) {
	assert.Nil(t, NewResolver(nil))
}

func Benchmark_ResolveDocumentStripe(b *testing.B) {
	stripe, _ := os.ReadFile("../test_specs/stripe.yaml")
	for n := 0; n < b.N; n++ {
		var rootNode yaml.Node
		_ = yaml.Unmarshal(stripe, &rootNode)
		idx := index.NewSpecIndexWithConfig(&rootNode, index.CreateClosedAPIIndexConfig())
		resolver := NewResolver(idx)
		resolver.Resolve()
	}
}

func TestResolver_ResolveComponents_CircularSpec(t *testing.T) {
	circular, _ := os.ReadFile("../test_specs/circular-tests.yaml")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(circular, &rootNode)

	idx := index.NewSpecIndexWithConfig(&rootNode, index.CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	circ := resolver.Resolve()
	assert.Len(t, circ, 3)

	_, err := yaml.Marshal(resolver.resolvedRoot)
	assert.NoError(t, err)
}

func TestResolver_CheckForCircularReferences(t *testing.T) {
	circular, _ := os.ReadFile("../test_specs/circular-tests.yaml")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(circular, &rootNode)

	idx := index.NewSpecIndexWithConfig(&rootNode, index.CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	circ := resolver.CheckForCircularReferences()
	assert.Len(t, circ, 3)
	assert.Len(t, resolver.GetResolvingErrors(), 3)
	assert.Len(t, resolver.GetCircularErrors(), 3)

	_, err := yaml.Marshal(resolver.resolvedRoot)
	assert.NoError(t, err)
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

	idx := index.NewSpecIndexWithConfig(&rootNode, index.CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	circ := resolver.CheckForCircularReferences()
	assert.Len(t, circ, 1)
	assert.Len(t, resolver.GetResolvingErrors(), 1) // infinite loop is a resolving error.
	assert.Len(t, resolver.GetCircularErrors(), 1)
	assert.True(t, resolver.GetCircularErrors()[0].IsArrayResult)

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

	idx := index.NewSpecIndexWithConfig(&rootNode, index.CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	resolver.IgnoreArrayCircularReferences()

	circ := resolver.CheckForCircularReferences()
	assert.Len(t, circ, 0)
	assert.Len(t, resolver.GetResolvingErrors(), 0)
	assert.Len(t, resolver.GetCircularErrors(), 0)

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

	idx := index.NewSpecIndexWithConfig(&rootNode, index.CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	resolver.IgnorePolymorphicCircularReferences()

	circ := resolver.CheckForCircularReferences()
	assert.Len(t, circ, 0)
	assert.Len(t, resolver.GetResolvingErrors(), 0)
	assert.Len(t, resolver.GetCircularErrors(), 0)

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

	idx := index.NewSpecIndexWithConfig(&rootNode, index.CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	resolver.IgnorePolymorphicCircularReferences()

	circ := resolver.CheckForCircularReferences()
	assert.Len(t, circ, 0)
	assert.Len(t, resolver.GetResolvingErrors(), 0)
	assert.Len(t, resolver.GetCircularErrors(), 0)

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

	idx := index.NewSpecIndexWithConfig(&rootNode, index.CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	resolver.IgnorePolymorphicCircularReferences()

	circ := resolver.CheckForCircularReferences()
	assert.Len(t, circ, 0)
	assert.Len(t, resolver.GetResolvingErrors(), 0)
	assert.Len(t, resolver.GetCircularErrors(), 0)

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

	idx := index.NewSpecIndexWithConfig(&rootNode, index.CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	circ := resolver.CheckForCircularReferences()
	assert.Len(t, circ, 0)
	assert.Len(t, resolver.GetResolvingErrors(), 0) // not an infinite loop if poly.
	assert.Len(t, resolver.GetCircularErrors(), 1)
	assert.Equal(t, "anyOf", resolver.GetCircularErrors()[0].PolymorphicType)
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

	idx := index.NewSpecIndexWithConfig(&rootNode, index.CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	circ := resolver.CheckForCircularReferences()
	assert.Len(t, circ, 0)
	assert.Len(t, resolver.GetResolvingErrors(), 0) // not an infinite loop if poly.
	assert.Len(t, resolver.GetCircularErrors(), 1)
	assert.Equal(t, "allOf", resolver.GetCircularErrors()[0].PolymorphicType)
	assert.True(t, resolver.GetCircularErrors()[0].IsPolymorphicResult)
	_, err := yaml.Marshal(resolver.resolvedRoot)
	assert.NoError(t, err)
}

func TestResolver_CheckForCircularReferences_DigitalOcean(t *testing.T) {
	circular, _ := os.ReadFile("../test_specs/digitalocean.yaml")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(circular, &rootNode)

	baseURL, _ := url.Parse("https://raw.githubusercontent.com/digitalocean/openapi/main/specification")

	idx := index.NewSpecIndexWithConfig(&rootNode, &index.SpecIndexConfig{
		AllowRemoteLookup: true,
		AllowFileLookup:   true,
		BaseURL:           baseURL,
	})

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	circ := resolver.CheckForCircularReferences()
	assert.Len(t, circ, 0)
	assert.Len(t, resolver.GetResolvingErrors(), 0)
	assert.Len(t, resolver.GetCircularErrors(), 0)

	_, err := yaml.Marshal(resolver.resolvedRoot)
	assert.NoError(t, err)
}

func TestResolver_CircularReferencesRequiredValid(t *testing.T) {
	circular, _ := os.ReadFile("../test_specs/swagger-valid-recursive-model.yaml")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(circular, &rootNode)

	idx := index.NewSpecIndexWithConfig(&rootNode, index.CreateClosedAPIIndexConfig())

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

	idx := index.NewSpecIndexWithConfig(&rootNode, index.CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	circ := resolver.CheckForCircularReferences()
	assert.Len(t, circ, 2)

	_, err := yaml.Marshal(resolver.resolvedRoot)
	assert.NoError(t, err)
}

func TestResolver_DeepJourney(t *testing.T) {
	var journey []*index.Reference
	for f := 0; f < 200; f++ {
		journey = append(journey, nil)
	}
	idx := index.NewSpecIndexWithConfig(nil, nil)
	resolver := NewResolver(idx)
	assert.Nil(t, resolver.extractRelatives(nil, nil, nil, journey, false))
}

func TestResolver_ResolveComponents_Stripe(t *testing.T) {
	stripe, _ := os.ReadFile("../test_specs/stripe.yaml")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(stripe, &rootNode)

	idx := index.NewSpecIndexWithConfig(&rootNode, index.CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	circ := resolver.Resolve()
	assert.Len(t, circ, 3)

	assert.Len(t, resolver.GetNonPolymorphicCircularErrors(), 3)
	assert.Len(t, resolver.GetPolymorphicCircularErrors(), 0)
}

func TestResolver_ResolveComponents_BurgerShop(t *testing.T) {
	mixedref, _ := os.ReadFile("../test_specs/burgershop.openapi.yaml")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(mixedref, &rootNode)

	idx := index.NewSpecIndexWithConfig(&rootNode, index.CreateClosedAPIIndexConfig())

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

	idx := index.NewSpecIndexWithConfig(&rootNode, index.CreateClosedAPIIndexConfig())

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

	idx := index.NewSpecIndexWithConfig(&rootNode, index.CreateClosedAPIIndexConfig())

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

	idx := index.NewSpecIndexWithConfig(&rootNode, index.CreateClosedAPIIndexConfig())

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	err := resolver.Resolve()
	assert.Len(t, err, 1)
	assert.Equal(t, "cannot resolve reference `go home, I am drunk`, it's missing: $go home, I am drunk [18:11]", err[0].Error())
}

func TestResolver_ResolveComponents_MixedRef(t *testing.T) {
	mixedref, _ := os.ReadFile("../test_specs/mixedref-burgershop.openapi.yaml")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(mixedref, &rootNode)

	b := index.CreateOpenAPIIndexConfig()
	idx := index.NewSpecIndexWithConfig(&rootNode, b)

	resolver := NewResolver(idx)
	assert.NotNil(t, resolver)

	circ := resolver.Resolve()
	assert.Len(t, circ, 0)
	assert.Equal(t, 5, resolver.GetIndexesVisited())

	// in v0.8.2 a new check was added when indexing, to prevent re-indexing the same file multiple times.
	assert.Equal(t, 191, resolver.GetRelativesSeen())
	assert.Equal(t, 35, resolver.GetJourneysTaken())
	assert.Equal(t, 62, resolver.GetReferenceVisited())
}

func TestResolver_ResolveComponents_k8s(t *testing.T) {
	k8s, _ := os.ReadFile("../test_specs/k8s.json")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(k8s, &rootNode)

	idx := index.NewSpecIndexWithConfig(&rootNode, index.CreateClosedAPIIndexConfig())

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
	indexConfig := index.CreateClosedAPIIndexConfig()
	idx := index.NewSpecIndexWithConfig(&rootNode, indexConfig)

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
		CircularReference: &index.CircularReferenceResult{},
	}

	fmt.Printf("%s", re.Error())
	// Output: Je suis une erreur: #/definitions/JeSuisUneErreur [5:21]
}
