// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package bundler

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestBundlerComposed(t *testing.T) {
	specBytes, err := os.ReadFile("test/specs/main.yaml")

	doc, err := libopenapi.NewDocumentWithConfiguration(specBytes, &datamodel.DocumentConfiguration{
		BasePath:                "test/specs",
		ExtractRefsSequentially: true,
		Logger:                  slog.Default(),
	})
	if err != nil {
		panic(err)
	}

	v3Doc, errs := doc.BuildV3Model()
	if errs != nil {
		panic(errs)
	}

	var bytes []byte

	bytes, err = BundleDocumentComposed(&v3Doc.Model, &BundleCompositionConfig{Delimiter: "__"})
	if err != nil {
		panic(err)
	}

	// windows needs a different byte count
	if runtime.GOOS != "windows" {
		assert.Len(t, bytes, 9099)
	}

	preBundled, bErr := os.ReadFile("test/specs/bundled.yaml")
	assert.NoError(t, bErr)

	if runtime.GOOS != "windows" {
		assert.Equal(t, len(preBundled), len(bytes)) // windows reads the file with different line endings and changes the byte count.
	}

	// write the bundled spec to a file for inspection
	// uncomment this to rebuild the bundled spec file, if the example spec changes.
	// err = os.WriteFile("test/specs/bundled.yaml", bytes, 0644)

	v3Doc.Model.Components = nil
	err = processReference(&v3Doc.Model, &processRef{}, &handleIndexConfig{compositionConfig: &BundleCompositionConfig{}})
	assert.Error(t, err)
}

func TestCheckFileIteration(t *testing.T) {
	name := calculateCollisionName("bundled", "/test/specs/bundled.yaml", "__", 1)
	assert.Equal(t, "bundled__specs", name)

	name = calculateCollisionName("bundled__specs", "/test/specs/bundled.yaml", "__", 2)
	assert.Equal(t, "bundled__specs__test", name)

	name = calculateCollisionName("bundled-||-specs", "/test/specs/bundled.yaml", "-||-", 2)
	assert.Equal(t, "bundled-||-specs-||-test", name)

	reg := regexp.MustCompile("^bundled__[0-9A-Za-z]{1,4}$")

	name = calculateCollisionName("bundled", "/test/specs/bundled.yaml", "__", 8)
	assert.True(t, reg.MatchString(name))
}

func TestBundleDocumentComposed(t *testing.T) {
	_, err := BundleDocumentComposed(nil, nil)
	assert.Error(t, err)
	assert.Equal(t, "model or rolodex is nil", err.Error())

	_, err = BundleDocumentComposed(nil, &BundleCompositionConfig{Delimiter: ""})
	assert.Error(t, err)
	assert.Equal(t, "model or rolodex is nil", err.Error())

	_, err = BundleDocumentComposed(nil, &BundleCompositionConfig{Delimiter: "#"})
	assert.Error(t, err)
	assert.Equal(t, "composition delimiter cannot contain '#' or '/' characters", err.Error())

	_, err = BundleDocumentComposed(nil, &BundleCompositionConfig{Delimiter: "well hello there"})
	assert.Error(t, err)
	assert.Equal(t, "composition delimiter cannot contain spaces", err.Error())
}

func TestCheckReferenceAndBubbleUp(t *testing.T) {
	err := checkReferenceAndBubbleUp[any]("test", "__",
		&processRef{ref: &index.Reference{Node: &yaml.Node{}}},
		nil, nil,
		func(node *yaml.Node, idx *index.SpecIndex) (any, error) {
			return nil, errors.New("test error")
		})
	assert.Error(t, err)
}

func TestRenameReference(t *testing.T) {
	// test the rename reference function
	assert.Equal(t, "#/_oh_#/_yeah", renameRef(nil, "#/_oh_#/_yeah", nil))
}

func TestBuildSchema(t *testing.T) {
	_, err := buildSchema(nil, nil)
	assert.Error(t, err)
}

func TestBundlerComposed_StrangeRefs(t *testing.T) {
	specBytes, err := os.ReadFile("../test_specs/first.yaml")

	doc, err := libopenapi.NewDocumentWithConfiguration(specBytes, &datamodel.DocumentConfiguration{
		BasePath:                "../test_specs/",
		ExtractRefsSequentially: true,
		Logger:                  slog.Default(),
	})
	if err != nil {
		panic(err)
	}

	v3Doc, errs := doc.BuildV3Model()
	if errs != nil {
		panic(errs)
	}

	var bytes []byte

	bytes, err = BundleDocumentComposed(&v3Doc.Model, &BundleCompositionConfig{Delimiter: "__"})
	if err != nil {
		panic(err)
	}

	// windows needs a different byte count
	if runtime.GOOS != "windows" {
		assert.Len(t, bytes, 3397)
	}
}

// TestBundleBytesComposed_DiscriminatorMapping tests that composed bundling correctly
// updates discriminator mappings when external schemas are moved to components.
func TestBundleBytesComposed_DiscriminatorMapping(t *testing.T) {
	spec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Animal:
      type: object
      discriminator:
        propertyName: type
        mapping:
          cat: './external-cat.yaml#/components/schemas/Cat'
      oneOf:
        - $ref: './external-cat.yaml#/components/schemas/Cat'
    Dog:
      type: object`

	ext := `components:
  schemas:
    Cat:
      type: object
      properties:
        type:
          type: string
        meow:
          type: boolean`

	tmp := t.TempDir()
	write := func(name, src string) {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(src), 0644))
	}
	write("main.yaml", spec)
	write("external-cat.yaml", ext)

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))
	cfg := &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	}

	out, err := BundleBytesComposed(mainBytes, cfg, nil)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(out, &doc))

	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	animal := schemas["Animal"].(map[string]any)

	// discriminator mapping should be updated to point to the new component reference
	mapping := animal["discriminator"].(map[string]any)["mapping"].(map[string]any)
	catMapping := mapping["cat"].(string)
	assert.True(t, strings.HasPrefix(catMapping, "#/components/schemas/"),
		"discriminator mapping should point to component reference, got: %s", catMapping)
	assert.False(t, strings.Contains(catMapping, "./external-cat.yaml"),
		"discriminator mapping should not contain external file path, got: %s", catMapping)

	// oneOf should be updated to point to the new component reference
	oneOf := animal["oneOf"].([]any)[0].(map[string]any)
	oneOfRef := oneOf["$ref"].(string)
	assert.True(t, strings.HasPrefix(oneOfRef, "#/components/schemas/"),
		"oneOf reference should point to component reference, got: %s", oneOfRef)
	assert.False(t, strings.Contains(oneOfRef, "./external-cat.yaml"),
		"oneOf reference should not contain external file path, got: %s", oneOfRef)

	// Cat schema should be moved to components with potentially renamed key
	foundCat := false
	for schemaName := range schemas {
		if schemaName == "Cat" || (schemaName != "Animal" && schemaName != "Dog" && strings.Contains(schemaName, "Cat")) {
			foundCat = true
			break
		}
	}
	assert.True(t, foundCat, "Cat schema should be moved to components")

	runtime.GC()
}

// TestBundleBytesComposed_DiscriminatorMappingMultiple tests that composed bundling
// correctly updates discriminator mappings for multiple external schemas.
func TestBundleBytesComposed_DiscriminatorMappingMultiple(t *testing.T) {
	spec := `openapi: 3.0.0
info:
  title: Vehicles
  version: 1.0.0
paths: {}
components:
  schemas:
    Vehicle:
      type: object
      discriminator:
        propertyName: kind
        mapping:
          car: './vehicles/car.yaml#/components/schemas/Car'
          bike: './vehicles/bike.yaml#/components/schemas/Bike'
      oneOf:
        - $ref: './vehicles/car.yaml#/components/schemas/Car'
        - $ref: './vehicles/bike.yaml#/components/schemas/Bike'`

	car := `components:
  schemas:
    Car:
      type: object
      properties:
        wheels:
          type: integer`
	bike := `components:
  schemas:
    Bike:
      type: object
      properties:
        wheels:
          type: integer`

	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "vehicles"), 0755))
	write := func(name, src string) {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(src), 0644))
	}
	write("main.yaml", spec)
	write("vehicles/car.yaml", car)
	write("vehicles/bike.yaml", bike)

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))
	out, err := BundleBytesComposed(mainBytes, &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	}, nil)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(out, &doc))

	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	vehicle := schemas["Vehicle"].(map[string]any)
	mp := vehicle["discriminator"].(map[string]any)["mapping"].(map[string]any)

	// Both mappings should be updated to component references
	carMapping := mp["car"].(string)
	bikeMapping := mp["bike"].(string)
	assert.True(t, strings.HasPrefix(carMapping, "#/components/schemas/"),
		"car mapping should point to component reference, got: %s", carMapping)
	assert.False(t, strings.Contains(carMapping, "./vehicles/car.yaml"),
		"car mapping should not contain external file path, got: %s", carMapping)
	assert.True(t, strings.HasPrefix(bikeMapping, "#/components/schemas/"),
		"bike mapping should point to component reference, got: %s", bikeMapping)
	assert.False(t, strings.Contains(bikeMapping, "./vehicles/bike.yaml"),
		"bike mapping should not contain external file path, got: %s", bikeMapping)

	// oneOf should be updated
	oneOf := vehicle["oneOf"].([]any)
	carRef := oneOf[0].(map[string]any)["$ref"].(string)
	bikeRef := oneOf[1].(map[string]any)["$ref"].(string)
	assert.True(t, strings.HasPrefix(carRef, "#/components/schemas/"),
		"car oneOf reference should point to component reference, got: %s", carRef)
	assert.False(t, strings.Contains(carRef, "./vehicles/car.yaml"),
		"car oneOf reference should not contain external file path, got: %s", carRef)
	assert.True(t, strings.HasPrefix(bikeRef, "#/components/schemas/"),
		"bike oneOf reference should point to component reference, got: %s", bikeRef)
	assert.False(t, strings.Contains(bikeRef, "./vehicles/bike.yaml"),
		"bike oneOf reference should not contain external file path, got: %s", bikeRef)

	// Both schemas should be moved to components
	foundCar, foundBike := false, false
	for schemaName := range schemas {
		if schemaName == "Car" || (schemaName != "Vehicle" && strings.Contains(schemaName, "Car")) {
			foundCar = true
		}
		if schemaName == "Bike" || (schemaName != "Vehicle" && strings.Contains(schemaName, "Bike")) {
			foundBike = true
		}
	}
	assert.True(t, foundCar, "Car schema should be moved to components")
	assert.True(t, foundBike, "Bike schema should be moved to components")

	runtime.GC()
}

// TestBundleBytesComposed_DiscriminatorMappingPartial tests that composed bundling
// correctly handles discriminator mappings that only reference some of the oneOf alternatives.
func TestBundleBytesComposed_DiscriminatorMappingPartial(t *testing.T) {
	spec := `openapi: 3.0.0
info:
  title: Vehicles
  version: 1.0.0
paths: {}
components:
  schemas:
    Vehicle:
      type: object
      discriminator:
        propertyName: kind
        mapping:
          car: './vehicles/car.yaml#/components/schemas/Car'   # bike missing on purpose
      oneOf:
        - $ref: './vehicles/car.yaml#/components/schemas/Car'
        - $ref: './vehicles/bike.yaml#/components/schemas/Bike'`

	car := `components:
  schemas:
    Car:
      type: object
      properties:
        wheels:
          type: integer`

	bike := `components:
  schemas:
    Bike:
      type: object
      properties:
        wheels:
          type: integer`

	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "vehicles"), 0o755))
	write := func(name, src string) {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(src), 0o644))
	}
	write("main.yaml", spec)
	write("vehicles/car.yaml", car)
	write("vehicles/bike.yaml", bike)

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))

	out, err := BundleBytesComposed(mainBytes, &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	}, nil)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(out, &doc))

	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	vehicle := schemas["Vehicle"].(map[string]any)

	mp := vehicle["discriminator"].(map[string]any)["mapping"].(map[string]any)
	assert.Equal(t, 1, len(mp), "no new mapping rows should have been synthesised")

	carMapping := mp["car"].(string)
	assert.True(t, strings.HasPrefix(carMapping, "#/components/schemas/"),
		"car mapping should point to component reference, got: %s", carMapping)
	assert.False(t, strings.Contains(carMapping, "./vehicles/car.yaml"),
		"car mapping should not contain external file path, got: %s", carMapping)

	// Both oneOf entries should be updated to component references
	oneOf := vehicle["oneOf"].([]any)
	carRef := oneOf[0].(map[string]any)["$ref"].(string)
	bikeRef := oneOf[1].(map[string]any)["$ref"].(string)
	assert.True(t, strings.HasPrefix(carRef, "#/components/schemas/"),
		"car oneOf reference should point to component reference, got: %s", carRef)
	assert.False(t, strings.Contains(carRef, "./vehicles/car.yaml"),
		"car oneOf reference should not contain external file path, got: %s", carRef)
	assert.True(t, strings.HasPrefix(bikeRef, "#/components/schemas/"),
		"bike oneOf reference should point to component reference, got: %s", bikeRef)
	assert.False(t, strings.Contains(bikeRef, "./vehicles/bike.yaml"),
		"bike oneOf reference should not contain external file path, got: %s", bikeRef)

	// Both schemas should be moved to components
	foundCar, foundBike := false, false
	for schemaName := range schemas {
		if schemaName == "Car" || (schemaName != "Vehicle" && strings.Contains(schemaName, "Car")) {
			foundCar = true
		}
		if schemaName == "Bike" || (schemaName != "Vehicle" && strings.Contains(schemaName, "Bike")) {
			foundBike = true
		}
	}
	assert.True(t, foundCar, "Car must be moved to components")
	assert.True(t, foundBike, "Bike must be moved to components")

	runtime.GC()
}

// TestBundleBytesComposed_DiscriminatorMappingAnyOf tests that composed bundling
// correctly handles discriminator mappings with anyOf.
func TestBundleBytesComposed_DiscriminatorMappingAnyOf(t *testing.T) {
	spec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Shape:
      type: object
      discriminator:
        propertyName: type
        mapping:
          circle: './shapes/circle.yaml#/components/schemas/Circle'
      anyOf:
        - $ref: './shapes/circle.yaml#/components/schemas/Circle'
        - type: object
          properties:
            type:
              type: string`

	ext := `components:
  schemas:
    Circle:
      type: object
      properties:
        type:
          type: string
        radius:
          type: number`

	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "shapes"), 0755))
	write := func(name, src string) {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(src), 0644))
	}
	write("main.yaml", spec)
	write("shapes/circle.yaml", ext)

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))
	out, err := BundleBytesComposed(mainBytes, &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	}, nil)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(out, &doc))

	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	shape := schemas["Shape"].(map[string]any)

	mapping := shape["discriminator"].(map[string]any)["mapping"].(map[string]any)
	circleMapping := mapping["circle"].(string)
	assert.True(t, strings.HasPrefix(circleMapping, "#/components/schemas/"),
		"discriminator mapping should point to component reference, got: %s", circleMapping)

	anyOf := shape["anyOf"].([]any)
	circleRef := anyOf[0].(map[string]any)["$ref"].(string)
	assert.True(t, strings.HasPrefix(circleRef, "#/components/schemas/"),
		"anyOf reference should point to component reference, got: %s", circleRef)

	foundCircle := false
	for schemaName := range schemas {
		if schemaName == "Circle" || (schemaName != "Shape" && strings.Contains(schemaName, "Circle")) {
			foundCircle = true
			break
		}
	}
	assert.True(t, foundCircle, "Circle schema should be moved to components")

	runtime.GC()
}

// TestBundleBytesComposed_DiscriminatorMappingMixed tests that composed bundling
// correctly handles mixed internal and external discriminator mappings.
func TestBundleBytesComposed_DiscriminatorMappingMixed(t *testing.T) {
	spec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Animal:
      type: object
      discriminator:
        propertyName: type
        mapping:
          cat: './external-cat.yaml#/components/schemas/Cat'
          dog: '#/components/schemas/Dog'
          bird: 'Bird'
      oneOf:
        - $ref: './external-cat.yaml#/components/schemas/Cat'
        - $ref: '#/components/schemas/Dog'
    Dog:
      type: object
      properties:
        type:
          type: string
        bark:
          type: boolean`

	ext := `components:
  schemas:
    Cat:
      type: object
      properties:
        type:
          type: string
        meow:
          type: boolean`

	tmp := t.TempDir()
	write := func(name, src string) {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(src), 0644))
	}
	write("main.yaml", spec)
	write("external-cat.yaml", ext)

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))
	out, err := BundleBytesComposed(mainBytes, &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	}, nil)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(out, &doc))

	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	animal := schemas["Animal"].(map[string]any)

	mapping := animal["discriminator"].(map[string]any)["mapping"].(map[string]any)

	catMapping := mapping["cat"].(string)
	assert.True(t, strings.HasPrefix(catMapping, "#/components/schemas/"),
		"external cat mapping should point to component reference, got: %s", catMapping)

	dogMapping := mapping["dog"].(string)
	assert.Equal(t, "#/components/schemas/Dog", dogMapping,
		"internal dog mapping should remain unchanged, got: %s", dogMapping)

	birdMapping := mapping["bird"].(string)
	assert.Equal(t, "Bird", birdMapping,
		"non-reference bird mapping should remain unchanged, got: %s", birdMapping)

	oneOf := animal["oneOf"].([]any)
	catRef := oneOf[0].(map[string]any)["$ref"].(string)
	dogRef := oneOf[1].(map[string]any)["$ref"].(string)
	assert.True(t, strings.HasPrefix(catRef, "#/components/schemas/"),
		"cat oneOf reference should point to component reference, got: %s", catRef)
	assert.Equal(t, "#/components/schemas/Dog", dogRef,
		"dog oneOf reference should remain unchanged, got: %s", dogRef)

	_, dogExists := schemas["Dog"]
	assert.True(t, dogExists, "Dog schema should exist in components")

	foundCat := false
	for schemaName := range schemas {
		if schemaName == "Cat" || (schemaName != "Animal" && schemaName != "Dog" && strings.Contains(schemaName, "Cat")) {
			foundCat = true
			break
		}
	}
	assert.True(t, foundCat, "Cat schema should be moved to components")

	runtime.GC()
}

// TestBundleBytesComposed_DiscriminatorMappingInvalid tests that composed bundling
// gracefully handles invalid discriminator mapping references.
func TestBundleBytesComposed_DiscriminatorMappingInvalid(t *testing.T) {
	spec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Animal:
      type: object
      discriminator:
        propertyName: type
        mapping:
          cat: './nonexistent.yaml#/components/schemas/Cat'
          dog: './external-dog.yaml#/components/schemas/Dog'
      oneOf:
        - $ref: './external-dog.yaml#/components/schemas/Dog'`

	ext := `components:
  schemas:
    Dog:
      type: object
      properties:
        type:
          type: string`

	tmp := t.TempDir()
	write := func(name, src string) {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(src), 0644))
	}
	write("main.yaml", spec)
	write("external-dog.yaml", ext)

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))
	out, err := BundleBytesComposed(mainBytes, &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	}, nil)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(out, &doc))

	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	animal := schemas["Animal"].(map[string]any)

	mapping := animal["discriminator"].(map[string]any)["mapping"].(map[string]any)

	catMapping := mapping["cat"].(string)
	assert.Equal(t, "./nonexistent.yaml#/components/schemas/Cat", catMapping,
		"invalid cat mapping should remain unchanged, got: %s", catMapping)

	dogMapping := mapping["dog"].(string)
	assert.True(t, strings.HasPrefix(dogMapping, "#/components/schemas/"),
		"valid dog mapping should be updated, got: %s", dogMapping)

	runtime.GC()
}

// TestBundleBytesComposed_DiscriminatorMappingDeepRef tests that composed bundling
// correctly handles discriminator mappings that are deeply nested behind $refs.
func TestBundleBytesComposed_DiscriminatorMappingDeepRef(t *testing.T) {
	spec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Animal:
      $ref: './definitions/animal.yaml'`

	animalDef := `type: object
discriminator:
  propertyName: type
  mapping:
    cat: './cat.yaml#/components/schemas/Cat'
oneOf:
  - $ref: './cat.yaml#/components/schemas/Cat'`

	catDef := `components:
  schemas:
    Cat:
      type: object
      properties:
        type:
          type: string
        meow:
          type: boolean`

	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "definitions"), 0755))
	write := func(name, src string) {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(src), 0644))
	}
	write("main.yaml", spec)
	write("definitions/animal.yaml", animalDef)
	write("definitions/cat.yaml", catDef)

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))
	out, err := BundleBytesComposed(mainBytes, &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	}, nil)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(out, &doc))

	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)

	// Find the composed Animal schema (might be renamed)
	var animalSchema map[string]any
	for _, schema := range schemas {
		if s, ok := schema.(map[string]any); ok {
			if _, hasDisc := s["discriminator"]; hasDisc {
				animalSchema = s
				break
			}
		}
	}
	require.NotNil(t, animalSchema, "Animal schema with discriminator should be found")

	// discriminator mapping should be updated to point to the new component reference
	mapping := animalSchema["discriminator"].(map[string]any)["mapping"].(map[string]any)
	catMapping := mapping["cat"].(string)
	assert.True(t, strings.HasPrefix(catMapping, "#/components/schemas/"),
		"discriminator mapping should point to component reference, got: %s", catMapping)
	assert.False(t, strings.Contains(catMapping, "./cat.yaml"),
		"discriminator mapping should not contain external file path, got: %s", catMapping)

	// oneOf should be updated to point to the new component reference
	oneOf := animalSchema["oneOf"].([]any)[0].(map[string]any)
	oneOfRef := oneOf["$ref"].(string)
	assert.True(t, strings.HasPrefix(oneOfRef, "#/components/schemas/"),
		"oneOf reference should point to component reference, got: %s", oneOfRef)
	assert.False(t, strings.Contains(oneOfRef, "./cat.yaml"),
		"oneOf reference should not contain external file path, got: %s", oneOfRef)

	// Cat schema should be moved to components
	foundCat := false
	for schemaName := range schemas {
		if schemaName == "Cat" || strings.Contains(schemaName, "Cat") {
			foundCat = true
			break
		}
	}
	assert.True(t, foundCat, "Cat schema should be moved to components")

	runtime.GC()
}

const emptyDefaultServerSpec = `openapi: 3.0.0
info:
  title: defaults
  version: 1.0.0
servers:
  - url: https://{env}.example.com
    variables:
      env:
        default: ""
        description: environment host
  - url: https://{shard}.example.com
    variables:
      shard:
        description: shard id
        default: ""
  - url: https://{slot}.example.com
    variables:
      slot:
        default: ""
paths: {}`

func TestBundleBytesComposed_PreservesEmptyServerVariableDefaults(t *testing.T) {
	spec := emptyDefaultServerSpec

	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "main.yaml"), []byte(spec), 0o644))

	data, err := os.ReadFile(filepath.Join(tmp, "main.yaml"))
	require.NoError(t, err)

	bundled, err := BundleBytesComposed(data, &datamodel.DocumentConfiguration{
		BasePath: tmp,
	}, nil)
	require.NoError(t, err)

	doc, err := libopenapi.NewDocument(bundled)
	require.NoError(t, err)

	model, err := doc.BuildV3Model()
	require.NoError(t, err)

	var envVar, shardVar *v3.ServerVariable
	for _, srv := range model.Model.Servers {
		if srv == nil || srv.Variables == nil {
			continue
		}
		if candidate := srv.Variables.GetOrZero("env"); candidate != nil {
			envVar = candidate
		}
		if candidate := srv.Variables.GetOrZero("shard"); candidate != nil {
			shardVar = candidate
		}
	}

	require.NotNil(t, envVar, "env variable must exist")
	assert.Equal(t, "", envVar.Default)
	assert.False(t, envVar.GoLow().Default.IsEmpty())
	assert.Equal(t, "environment host", envVar.Description)

	require.NotNil(t, shardVar, "shard variable must exist")
	assert.Equal(t, "", shardVar.Default)
	assert.False(t, shardVar.GoLow().Default.IsEmpty())
	assert.Equal(t, "shard id", shardVar.Description)

	slotVar := model.Model.Servers[2].Variables.GetOrZero("slot")
	require.NotNil(t, slotVar, "slot variable must exist")
	assert.Equal(t, "", slotVar.Default)
	assert.False(t, slotVar.GoLow().Default.IsEmpty())
	assert.Equal(t, "", slotVar.Description)
}

func TestBundleBytes_PreservesEmptyServerVariableDefaults(t *testing.T) {
	spec := []byte(emptyDefaultServerSpec)

	bundled, err := BundleBytes(spec, &datamodel.DocumentConfiguration{})
	require.NoError(t, err)

	doc, err := libopenapi.NewDocument(bundled)
	require.NoError(t, err)

	model, err := doc.BuildV3Model()
	require.NoError(t, err)

	require.Len(t, model.Model.Servers, 3)

	envVar := model.Model.Servers[0].Variables.GetOrZero("env")
	require.NotNil(t, envVar)
	assert.Equal(t, "", envVar.Default)
	assert.False(t, envVar.GoLow().Default.IsEmpty())

	shardVar := model.Model.Servers[1].Variables.GetOrZero("shard")
	require.NotNil(t, shardVar)
	assert.Equal(t, "", shardVar.Default)
	assert.False(t, shardVar.GoLow().Default.IsEmpty())

	slotVar := model.Model.Servers[2].Variables.GetOrZero("slot")
	require.NotNil(t, slotVar)
	assert.Equal(t, "", slotVar.Default)
	assert.False(t, slotVar.GoLow().Default.IsEmpty())
}
