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
	"github.com/pb33f/libopenapi/orderedmap"
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

func TestEnqueueDiscriminatorMappingTargets_StripsFragmentWhenNameMissing(t *testing.T) {
	tmpDir := t.TempDir()

	rootSpec := `openapi: 3.1.0
info:
  title: Enqueue Mapping Fragment
  version: 1.0.0
paths: {}
components:
  schemas:
    Animal:
      type: object
      discriminator:
        propertyName: kind
        mapping:
          cat: './schemas/Cat.yaml#/components/schemas/Cat'`

	catSpec := `components:
  schemas:
    Cat:
      type: object`

	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "root.yaml"), []byte(rootSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "schemas", "Cat.yaml"), []byte(catSpec), 0644))

	specBytes, err := os.ReadFile(filepath.Join(tmpDir, "root.yaml"))
	require.NoError(t, err)

	cfg := datamodel.NewDocumentConfiguration()
	cfg.BasePath = tmpDir
	cfg.AllowFileReferences = true
	cfg.SpecFilePath = "root.yaml"

	doc, err := libopenapi.NewDocumentWithConfiguration(specBytes, cfg)
	require.NoError(t, err)

	v3Doc, err := doc.BuildV3Model()
	require.NoError(t, err)
	require.NotNil(t, v3Doc)

	rolodex := v3Doc.Index.GetRolodex()
	mappings := collectDiscriminatorMappingNodesWithContext(rolodex)
	require.NotEmpty(t, mappings)

	refValue := mappings[0].node.Value
	ref, _ := mappings[0].sourceIdx.SearchIndexForReference(refValue)
	require.NotNil(t, ref)

	// Force name extraction logic by clearing the name.
	ref.Name = ""
	if !strings.Contains(ref.FullDefinition, "#") {
		ref.FullDefinition = ref.FullDefinition + "#/components/schemas/Cat"
	}

	cf := &handleIndexConfig{
		refMap: orderedmap.New[string, *processRef](),
	}

	enqueueDiscriminatorMappingTargets(mappings, cf, rolodex.GetRootIndex())

	pr := cf.refMap.GetOrZero(ref.FullDefinition)
	require.NotNil(t, pr)
	assert.Equal(t, "Cat", pr.name)
}

func TestDeriveNameFromFullDefinition_StripsFragment(t *testing.T) {
	name := deriveNameFromFullDefinition("/tmp/path/Cat.yaml#/components/schemas/Cat")
	assert.Equal(t, "Cat", name)
}

func TestResolveDiscriminatorMappingTarget_NilSourceIdx(t *testing.T) {
	ref, idx := resolveDiscriminatorMappingTarget(nil, "schemas/Cat.yaml")
	assert.Nil(t, ref)
	assert.Nil(t, idx)
}

func TestResolveDiscriminatorMappingTarget_NoRolodex(t *testing.T) {
	var rootNode yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(`openapi: 3.0.0
info:
  title: No Rolodex
  version: 1.0.0
paths: {}`), &rootNode))

	cfg := index.CreateOpenAPIIndexConfig()
	sourceIdx := index.NewSpecIndexWithConfig(&rootNode, cfg)

	ref, idx := resolveDiscriminatorMappingTarget(sourceIdx, "schemas/Cat.yaml")
	assert.Nil(t, ref)
	assert.Nil(t, idx)
}

func TestResolveDiscriminatorMappingTarget_IndexesAndReturnsRef(t *testing.T) {
	tmpDir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "schemas", "Cat.yaml"), []byte("type: object\n"), 0644))

	var rootNode yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(`openapi: 3.0.0
info:
  title: Root
  version: 1.0.0
paths: {}`), &rootNode))

	cfg := index.CreateOpenAPIIndexConfig()
	cfg.BasePath = tmpDir
	cfg.SpecAbsolutePath = ""

	sourceIdx := index.NewSpecIndexWithConfig(&rootNode, cfg)
	rolodex := index.NewRolodex(cfg)

	localFS, err := index.NewLocalFSWithConfig(&index.LocalFSConfig{
		BaseDirectory: tmpDir,
		IndexConfig:   nil,
	})
	require.NoError(t, err)

	rolodex.AddLocalFS(tmpDir, localFS)
	sourceIdx.SetRolodex(rolodex)

	ref, idx := resolveDiscriminatorMappingTarget(sourceIdx, "schemas/Cat.yaml")
	require.NotNil(t, ref)
	_ = idx

	expected, err := filepath.Abs(filepath.Join(tmpDir, "schemas", "Cat.yaml"))
	require.NoError(t, err)

	assert.Equal(t, expected, ref.FullDefinition)
	assert.Equal(t, expected, ref.RemoteLocation)
	assert.Equal(t, filepath.Base(expected), ref.Name)
	require.NotNil(t, ref.Node)
	assert.NotEqual(t, yaml.DocumentNode, ref.Node.Kind)
}

func TestHandleDiscriminatorMappingIndexes_SkipsRootTarget(t *testing.T) {
	rootIdx := &index.SpecIndex{}
	cf := &handleIndexConfig{
		idx:                   rootIdx,
		rootIdx:               rootIdx,
		discriminatorMappings: []*discriminatorMappingWithContext{{targetIdx: rootIdx}},
	}

	err := handleDiscriminatorMappingIndexes(cf, rootIdx, nil)
	require.NoError(t, err)
	assert.Equal(t, rootIdx, cf.idx)
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

// TestBundleBytesComposed_BareFileRef tests that composed bundling correctly
// handles bare file references without JSON pointers (e.g., $ref: child.yaml)
// where the external file contains a named schema map.
//
// CURRENT BEHAVIOR: When a bare file reference points to a map with a named key
// (like {NonRequired: {type: object, ...}}), the bundler cannot determine the
// component type since the root node's keys don't match schema indicators.
// It falls back to inlining the entire content.
func TestBundleBytesComposed_BareFileRef(t *testing.T) {
	rootSpec := `openapi: 3.1.0
paths:
  /nonreq:
    get:
      operationId: getNonReq
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: child.yaml
`

	childSpec := `NonRequired:
  type: object
  properties:
    str:
      type: string
      pattern: ".+"
      nullable: false
`

	tmp := t.TempDir()
	write := func(name, src string) {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(src), 0644))
	}
	write("main.yaml", rootSpec)
	write("child.yaml", childSpec)

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))

	docConfig := datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
		RecomposeRefs:       true,
	}

	bundleConfig := BundleCompositionConfig{
		StrictValidation: true,
	}

	bundled, err := BundleBytesComposed(mainBytes, &docConfig, &bundleConfig)
	require.NoError(t, err)

	t.Logf("Bundled output:\n%s", string(bundled))

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(bundled, &doc))

	// Check what we got
	paths := doc["paths"].(map[string]any)
	nonreq := paths["/nonreq"].(map[string]any)
	get := nonreq["get"].(map[string]any)
	responses := get["responses"].(map[string]any)
	resp200 := responses["200"].(map[string]any)
	content := resp200["content"].(map[string]any)
	appJson := content["application/json"].(map[string]any)
	schema := appJson["schema"].(map[string]any)

	t.Logf("Schema content: %+v", schema)

	// CURRENT BEHAVIOR: The content is inlined because DetectOpenAPIComponentType
	// sees keys like ["NonRequired"] which don't match schema indicators (type, properties, etc.)
	// The schema contains {NonRequired: {type: object, ...}} - the entire file content
	_, hasNonRequiredKey := schema["NonRequired"]
	assert.True(t, hasNonRequiredKey, "Current behavior: content is inlined with NonRequired as a key")
}

// TestBundleBytesComposed_BareFileRefWithJSONPointer shows that single-segment
// JSON pointer references (like child.yaml#/NonRequired) are properly recomposed
// to component references when the referenced content is detected as a schema.
func TestBundleBytesComposed_BareFileRefWithJSONPointer(t *testing.T) {
	rootSpec := `openapi: 3.1.0
paths:
  /nonreq:
    get:
      operationId: getNonReq
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: 'child.yaml#/NonRequired'
`

	childSpec := `NonRequired:
  type: object
  properties:
    str:
      type: string
      pattern: ".+"
      nullable: false
`

	tmp := t.TempDir()
	write := func(name, src string) {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(src), 0644))
	}
	write("main.yaml", rootSpec)
	write("child.yaml", childSpec)

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))

	docConfig := datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
		RecomposeRefs:       true,
	}

	bundleConfig := BundleCompositionConfig{
		StrictValidation: true,
	}

	bundled, err := BundleBytesComposed(mainBytes, &docConfig, &bundleConfig)
	require.NoError(t, err)

	t.Logf("Bundled output:\n%s", string(bundled))

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(bundled, &doc))

	// Check what we got
	paths := doc["paths"].(map[string]any)
	nonreq := paths["/nonreq"].(map[string]any)
	get := nonreq["get"].(map[string]any)
	responses := get["responses"].(map[string]any)
	resp200 := responses["200"].(map[string]any)
	content := resp200["content"].(map[string]any)
	appJson := content["application/json"].(map[string]any)
	schema := appJson["schema"].(map[string]any)

	t.Logf("Schema content: %+v", schema)

	// The single-segment JSON pointer should now be properly recomposed
	// The schema reference should point to #/components/schemas/NonRequired
	ref, hasRef := schema["$ref"].(string)
	require.True(t, hasRef, "Schema should have $ref pointing to component")
	assert.True(t, strings.HasPrefix(ref, "#/components/schemas/"),
		"schema should reference component, got: %s", ref)
	assert.False(t, strings.Contains(ref, "child.yaml"),
		"schema reference should not contain external file path, got: %s", ref)

	// Check that the schema was added to components
	components, ok := doc["components"].(map[string]any)
	require.True(t, ok, "Document should have components section")
	schemas, ok := components["schemas"].(map[string]any)
	require.True(t, ok, "Components should have schemas section")
	t.Logf("Components schemas: %+v", schemas)

	// Find the NonRequired schema in components
	foundNonRequired := false
	for schemaName, schemaVal := range schemas {
		if schemaName == "NonRequired" || strings.Contains(schemaName, "NonRequired") {
			foundNonRequired = true
			schemaMap := schemaVal.(map[string]any)
			assert.Equal(t, "object", schemaMap["type"], "Schema type should be object")
			break
		}
	}
	assert.True(t, foundNonRequired, "NonRequired schema should be added to components")
}

// TestBundleBytesComposed_BareSchemaFile shows that a bare schema file
// (without a named wrapper) is properly detected and recomposed.
func TestBundleBytesComposed_BareSchemaFile(t *testing.T) {
	rootSpec := `openapi: 3.1.0
paths:
  /nonreq:
    get:
      operationId: getNonReq
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: 'NonRequired.yaml'
`

	// This is a bare schema - no wrapper key
	childSpec := `type: object
properties:
  str:
    type: string
    pattern: ".+"
    nullable: false
`

	tmp := t.TempDir()
	write := func(name, src string) {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(src), 0644))
	}
	write("main.yaml", rootSpec)
	write("NonRequired.yaml", childSpec)

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))

	docConfig := datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
		RecomposeRefs:       true,
	}

	bundleConfig := BundleCompositionConfig{
		StrictValidation: true,
	}

	bundled, err := BundleBytesComposed(mainBytes, &docConfig, &bundleConfig)
	require.NoError(t, err)

	t.Logf("Bundled output:\n%s", string(bundled))

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(bundled, &doc))

	// Check what we got
	paths := doc["paths"].(map[string]any)
	nonreq := paths["/nonreq"].(map[string]any)
	get := nonreq["get"].(map[string]any)
	responses := get["responses"].(map[string]any)
	resp200 := responses["200"].(map[string]any)
	content := resp200["content"].(map[string]any)
	appJson := content["application/json"].(map[string]any)
	schema := appJson["schema"].(map[string]any)

	t.Logf("Schema content: %+v", schema)

	// Check if we have components with schemas
	if components, ok := doc["components"].(map[string]any); ok {
		if schemas, ok := components["schemas"].(map[string]any); ok {
			t.Logf("Components schemas: %+v", schemas)
			// The schema should be added with the filename as the name
			_, hasNonRequired := schemas["NonRequired"]
			assert.True(t, hasNonRequired, "Schema should be added to components with filename as name")
		}
	}

	// With a bare schema file, DetectOpenAPIComponentType should detect it as a schema
	// and the bundler should recompose it using the filename as the component name
	if ref, ok := schema["$ref"].(string); ok {
		t.Logf("Schema has $ref: %s", ref)
		assert.True(t, strings.HasPrefix(ref, "#/components/schemas/"),
			"schema should reference component, got: %s", ref)
	}
}

// TestBundleBytesComposed_SingleSegmentPointerMultipleRefs tests that multiple
// references to the same single-segment pointer are properly deduplicated.
func TestBundleBytesComposed_SingleSegmentPointerMultipleRefs(t *testing.T) {
	rootSpec := `openapi: 3.1.0
paths:
  /pets:
    get:
      operationId: getPets
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: 'schemas.yaml#/Pet'
    post:
      operationId: createPet
      requestBody:
        content:
          application/json:
            schema:
              $ref: 'schemas.yaml#/Pet'
      responses:
        "201":
          description: Created
`

	schemasFile := `Pet:
  type: object
  properties:
    name:
      type: string
    age:
      type: integer
`

	tmp := t.TempDir()
	write := func(name, src string) {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(src), 0644))
	}
	write("main.yaml", rootSpec)
	write("schemas.yaml", schemasFile)

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))

	bundled, err := BundleBytesComposed(mainBytes, &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	}, nil)
	require.NoError(t, err)

	t.Logf("Bundled output:\n%s", string(bundled))

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(bundled, &doc))

	// Both refs should point to the same component
	components := doc["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)

	// There should be exactly one Pet schema (not duplicated)
	petCount := 0
	for schemaName := range schemas {
		if schemaName == "Pet" || strings.Contains(schemaName, "Pet") {
			petCount++
		}
	}
	assert.Equal(t, 1, petCount, "Pet schema should appear exactly once in components")

	// Check the refs in paths
	paths := doc["paths"].(map[string]any)
	petsPath := paths["/pets"].(map[string]any)

	getOp := petsPath["get"].(map[string]any)
	getSchema := getOp["responses"].(map[string]any)["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
	getRef := getSchema["$ref"].(string)
	assert.True(t, strings.HasPrefix(getRef, "#/components/schemas/"),
		"GET response schema should reference component, got: %s", getRef)

	postOp := petsPath["post"].(map[string]any)
	postSchema := postOp["requestBody"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
	postRef := postSchema["$ref"].(string)
	assert.True(t, strings.HasPrefix(postRef, "#/components/schemas/"),
		"POST request body schema should reference component, got: %s", postRef)

	// Both refs should point to the same component
	assert.Equal(t, getRef, postRef, "Both refs should point to the same component")
}

// TestBundleBytesComposed_SingleSegmentPointerMixed tests that mixed reference
// styles (single-segment, full path, and local) all work together correctly.
func TestBundleBytesComposed_SingleSegmentPointerMixed(t *testing.T) {
	rootSpec := `openapi: 3.1.0
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: 'schemas.yaml#/User'
  /pets:
    get:
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: 'external/pet.yaml#/components/schemas/Pet'
components:
  schemas:
    LocalSchema:
      type: string
`

	schemasFile := `User:
  type: object
  properties:
    name:
      type: string
`

	petFile := `components:
  schemas:
    Pet:
      type: object
      properties:
        species:
          type: string
`

	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "external"), 0755))
	write := func(name, src string) {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(src), 0644))
	}
	write("main.yaml", rootSpec)
	write("schemas.yaml", schemasFile)
	write("external/pet.yaml", petFile)

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))

	bundled, err := BundleBytesComposed(mainBytes, &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	}, nil)
	require.NoError(t, err)

	t.Logf("Bundled output:\n%s", string(bundled))

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(bundled, &doc))

	components := doc["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)

	// Should have LocalSchema, User, and Pet
	_, hasLocal := schemas["LocalSchema"]
	assert.True(t, hasLocal, "LocalSchema should still exist")

	foundUser := false
	foundPet := false
	for schemaName := range schemas {
		if schemaName == "User" || strings.Contains(schemaName, "User") {
			foundUser = true
		}
		if schemaName == "Pet" || strings.Contains(schemaName, "Pet") {
			foundPet = true
		}
	}
	assert.True(t, foundUser, "User schema should be added from single-segment pointer")
	assert.True(t, foundPet, "Pet schema should be added from full path pointer")
}

// TestBundleBytesComposed_SingleSegmentRootKeySkipped tests that references to
// OpenAPI root-level keys (like #/paths or #/info) are NOT recomposed as components
// but instead inlined (as they cannot be component types).
func TestBundleBytesComposed_SingleSegmentRootKeySkipped(t *testing.T) {
	// This is a contrived example - in practice, you wouldn't reference #/paths
	// But we test that the bundler handles this gracefully
	rootSpec := `openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: 'external.yaml#/paths'
`

	externalFile := `paths:
  /external:
    get:
      responses:
        "200":
          description: OK
`

	tmp := t.TempDir()
	write := func(name, src string) {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(src), 0644))
	}
	write("main.yaml", rootSpec)
	write("external.yaml", externalFile)

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))

	bundled, err := BundleBytesComposed(mainBytes, &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	}, nil)
	require.NoError(t, err)

	t.Logf("Bundled output:\n%s", string(bundled))

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(bundled, &doc))

	// The schema should be inlined, not recomposed as a component
	// because "paths" is a root-level OpenAPI key
	paths := doc["paths"].(map[string]any)
	testPath := paths["/test"].(map[string]any)
	getOp := testPath["get"].(map[string]any)
	responses := getOp["responses"].(map[string]any)
	resp200 := responses["200"].(map[string]any)
	content := resp200["content"].(map[string]any)
	appJson := content["application/json"].(map[string]any)
	schema := appJson["schema"].(map[string]any)

	// The content should be inlined (contain the paths structure directly)
	// or kept as-is if inlining isn't performed
	_, hasRef := schema["$ref"]
	_, hasInlinedPath := schema["/external"]

	// Either the ref was kept (because it couldn't be resolved as a component)
	// or the content was inlined
	assert.True(t, hasRef || hasInlinedPath,
		"Root key reference should either be kept as $ref or inlined, not moved to components")
}

// TestBundleBytesComposed_JSONPointerEscapeRoundTrip tests that single-segment
// pointers with escaped characters (~ and /) are properly handled end-to-end.
// The component name "Foo/Bar" must be escaped as "Foo~1Bar" in the output reference.
func TestBundleBytesComposed_JSONPointerEscapeRoundTrip(t *testing.T) {
	// The reference uses ~1 to represent / in the component name
	rootSpec := `openapi: 3.1.0
paths:
  /test:
    get:
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: 'schemas.yaml#/Foo~1Bar'
`
	// The actual key in YAML is "Foo/Bar" (the / is literal in the key)
	schemasFile := `"Foo/Bar":
  type: object
  properties:
    name:
      type: string
`

	tmp := t.TempDir()
	write := func(name, src string) {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(src), 0644))
	}
	write("main.yaml", rootSpec)
	write("schemas.yaml", schemasFile)

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))

	bundled, err := BundleBytesComposed(mainBytes, &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	}, nil)
	require.NoError(t, err)

	t.Logf("Bundled output:\n%s", string(bundled))

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(bundled, &doc))

	// Check the reference is properly escaped
	paths := doc["paths"].(map[string]any)
	testPath := paths["/test"].(map[string]any)
	getOp := testPath["get"].(map[string]any)
	responses := getOp["responses"].(map[string]any)
	resp200 := responses["200"].(map[string]any)
	content := resp200["content"].(map[string]any)
	appJson := content["application/json"].(map[string]any)
	schema := appJson["schema"].(map[string]any)

	ref, hasRef := schema["$ref"].(string)
	require.True(t, hasRef, "Schema should have $ref")

	// The reference must use ~1 to escape the / in the component name
	assert.Equal(t, "#/components/schemas/Foo~1Bar", ref,
		"Reference must escape / as ~1 in component name")

	// Verify the schema was added to components with the literal key "Foo/Bar"
	components := doc["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	_, hasSchema := schemas["Foo/Bar"]
	assert.True(t, hasSchema, "Schema with key 'Foo/Bar' should exist in components")
}

// TestBundleBytesComposed_CaseSensitiveRootKeyGuard tests that the root key
// guard is case-sensitive, allowing component names like "Paths" to be recomposed.
func TestBundleBytesComposed_CaseSensitiveRootKeyGuard(t *testing.T) {
	// "Paths" (capital P) should be treated as a valid component name, not a root key
	rootSpec := `openapi: 3.1.0
paths:
  /test:
    get:
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: 'schemas.yaml#/Paths'
`
	schemasFile := `Paths:
  type: object
  description: This is a schema named Paths, not the OpenAPI paths object
  properties:
    route:
      type: string
`

	tmp := t.TempDir()
	write := func(name, src string) {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(src), 0644))
	}
	write("main.yaml", rootSpec)
	write("schemas.yaml", schemasFile)

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))

	bundled, err := BundleBytesComposed(mainBytes, &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	}, nil)
	require.NoError(t, err)

	t.Logf("Bundled output:\n%s", string(bundled))

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(bundled, &doc))

	// Check the reference - "Paths" should be recomposed as a component
	paths := doc["paths"].(map[string]any)
	testPath := paths["/test"].(map[string]any)
	getOp := testPath["get"].(map[string]any)
	responses := getOp["responses"].(map[string]any)
	resp200 := responses["200"].(map[string]any)
	content := resp200["content"].(map[string]any)
	appJson := content["application/json"].(map[string]any)
	schema := appJson["schema"].(map[string]any)

	ref, hasRef := schema["$ref"].(string)
	require.True(t, hasRef, "Schema should have $ref pointing to component")

	// "Paths" (capital P) should be recomposed, not inlined
	assert.Equal(t, "#/components/schemas/Paths", ref,
		"'Paths' (capital P) should be recomposed as a component, not treated as root key")

	// Verify the schema was added to components
	components := doc["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	_, hasPathsSchema := schemas["Paths"]
	assert.True(t, hasPathsSchema, "Schema named 'Paths' should exist in components")
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

// TestBundleBytesComposed_SingleSegmentResponse tests that single-segment JSON pointer
// references to response objects are properly recomposed to component references.
func TestBundleBytesComposed_SingleSegmentResponse(t *testing.T) {
	rootSpec := `openapi: 3.1.0
paths:
  /test:
    get:
      operationId: getTest
      responses:
        "200":
          $ref: 'responses.yaml#/OkResponse'
`

	// Response: must have content/headers/links and NOT required
	responsesFile := `OkResponse:
  description: Success response
  content:
    application/json:
      schema:
        type: string
`

	tmp := t.TempDir()
	write := func(name, src string) {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(src), 0644))
	}
	write("main.yaml", rootSpec)
	write("responses.yaml", responsesFile)

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))

	bundled, err := BundleBytesComposed(mainBytes, &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	}, nil)
	require.NoError(t, err)

	t.Logf("Bundled output:\n%s", string(bundled))

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(bundled, &doc))

	// Check the reference was recomposed
	paths := doc["paths"].(map[string]any)
	testPath := paths["/test"].(map[string]any)
	getOp := testPath["get"].(map[string]any)
	responses := getOp["responses"].(map[string]any)
	resp200 := responses["200"].(map[string]any)

	ref, hasRef := resp200["$ref"].(string)
	require.True(t, hasRef, "Response should have $ref pointing to component")
	assert.True(t, strings.HasPrefix(ref, "#/components/responses/"),
		"response should reference component, got: %s", ref)
	assert.False(t, strings.Contains(ref, "responses.yaml"),
		"response reference should not contain external file path, got: %s", ref)

	// Check that the response was added to components
	components := doc["components"].(map[string]any)
	responsesComp, ok := components["responses"].(map[string]any)
	require.True(t, ok, "Components should have responses section")

	foundOkResponse := false
	for responseName := range responsesComp {
		if responseName == "OkResponse" || strings.Contains(responseName, "OkResponse") {
			foundOkResponse = true
			break
		}
	}
	assert.True(t, foundOkResponse, "OkResponse should be added to components")
}

// TestBundleBytesComposed_SingleSegmentParameter tests that single-segment JSON pointer
// references to parameter objects are properly recomposed to component references.
func TestBundleBytesComposed_SingleSegmentParameter(t *testing.T) {
	rootSpec := `openapi: 3.1.0
paths:
  /test:
    get:
      operationId: getTest
      parameters:
        - $ref: 'params.yaml#/IdParam'
      responses:
        "200":
          description: OK
`

	// Parameter: must have name or in
	paramsFile := `IdParam:
  name: id
  in: query
  schema:
    type: string
`

	tmp := t.TempDir()
	write := func(name, src string) {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(src), 0644))
	}
	write("main.yaml", rootSpec)
	write("params.yaml", paramsFile)

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))

	bundled, err := BundleBytesComposed(mainBytes, &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	}, nil)
	require.NoError(t, err)

	t.Logf("Bundled output:\n%s", string(bundled))

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(bundled, &doc))

	// Check the reference was recomposed
	paths := doc["paths"].(map[string]any)
	testPath := paths["/test"].(map[string]any)
	getOp := testPath["get"].(map[string]any)
	params := getOp["parameters"].([]any)
	param := params[0].(map[string]any)

	ref, hasRef := param["$ref"].(string)
	require.True(t, hasRef, "Parameter should have $ref pointing to component")
	assert.True(t, strings.HasPrefix(ref, "#/components/parameters/"),
		"parameter should reference component, got: %s", ref)
	assert.False(t, strings.Contains(ref, "params.yaml"),
		"parameter reference should not contain external file path, got: %s", ref)

	// Check that the parameter was added to components
	components := doc["components"].(map[string]any)
	paramsComp, ok := components["parameters"].(map[string]any)
	require.True(t, ok, "Components should have parameters section")

	foundIdParam := false
	for paramName := range paramsComp {
		if paramName == "IdParam" || strings.Contains(paramName, "IdParam") {
			foundIdParam = true
			break
		}
	}
	assert.True(t, foundIdParam, "IdParam should be added to components")
}

// TestBundleBytesComposed_SingleSegmentHeader tests that single-segment JSON pointer
// references to header objects are properly recomposed to component references.
func TestBundleBytesComposed_SingleSegmentHeader(t *testing.T) {
	rootSpec := `openapi: 3.1.0
paths:
  /test:
    get:
      operationId: getTest
      responses:
        "200":
          description: OK
          headers:
            X-Rate-Limit:
              $ref: 'headers.yaml#/RateLimitHeader'
`

	// Header: must have schema or content, but NOT in and NOT name
	headersFile := `RateLimitHeader:
  description: Rate limit header
  schema:
    type: integer
`

	tmp := t.TempDir()
	write := func(name, src string) {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(src), 0644))
	}
	write("main.yaml", rootSpec)
	write("headers.yaml", headersFile)

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))

	bundled, err := BundleBytesComposed(mainBytes, &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	}, nil)
	require.NoError(t, err)

	t.Logf("Bundled output:\n%s", string(bundled))

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(bundled, &doc))

	// Check the reference was recomposed
	paths := doc["paths"].(map[string]any)
	testPath := paths["/test"].(map[string]any)
	getOp := testPath["get"].(map[string]any)
	responses := getOp["responses"].(map[string]any)
	resp200 := responses["200"].(map[string]any)
	headers := resp200["headers"].(map[string]any)
	rateLimitHeader := headers["X-Rate-Limit"].(map[string]any)

	ref, hasRef := rateLimitHeader["$ref"].(string)
	require.True(t, hasRef, "Header should have $ref pointing to component")
	assert.True(t, strings.HasPrefix(ref, "#/components/headers/"),
		"header should reference component, got: %s", ref)
	assert.False(t, strings.Contains(ref, "headers.yaml"),
		"header reference should not contain external file path, got: %s", ref)

	// Check that the header was added to components
	components := doc["components"].(map[string]any)
	headersComp, ok := components["headers"].(map[string]any)
	require.True(t, ok, "Components should have headers section")

	foundRateLimitHeader := false
	for headerName := range headersComp {
		if headerName == "RateLimitHeader" || strings.Contains(headerName, "RateLimitHeader") {
			foundRateLimitHeader = true
			break
		}
	}
	assert.True(t, foundRateLimitHeader, "RateLimitHeader should be added to components")
}

// TestBundleBytesComposed_SingleSegmentRequestBody tests that single-segment JSON pointer
// references to requestBody objects are properly recomposed to component references.
func TestBundleBytesComposed_SingleSegmentRequestBody(t *testing.T) {
	rootSpec := `openapi: 3.1.0
paths:
  /test:
    post:
      operationId: createTest
      requestBody:
        $ref: 'bodies.yaml#/CreateRequest'
      responses:
        "201":
          description: Created
`

	// RequestBody: must have content AND required (required helps distinguish from Response)
	bodiesFile := `CreateRequest:
  description: Create request body
  required: true
  content:
    application/json:
      schema:
        type: object
`

	tmp := t.TempDir()
	write := func(name, src string) {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(src), 0644))
	}
	write("main.yaml", rootSpec)
	write("bodies.yaml", bodiesFile)

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))

	bundled, err := BundleBytesComposed(mainBytes, &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	}, nil)
	require.NoError(t, err)

	t.Logf("Bundled output:\n%s", string(bundled))

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(bundled, &doc))

	// Check the reference was recomposed
	paths := doc["paths"].(map[string]any)
	testPath := paths["/test"].(map[string]any)
	postOp := testPath["post"].(map[string]any)
	reqBody := postOp["requestBody"].(map[string]any)

	ref, hasRef := reqBody["$ref"].(string)
	require.True(t, hasRef, "RequestBody should have $ref pointing to component")
	assert.True(t, strings.HasPrefix(ref, "#/components/requestBodies/"),
		"requestBody should reference component, got: %s", ref)
	assert.False(t, strings.Contains(ref, "bodies.yaml"),
		"requestBody reference should not contain external file path, got: %s", ref)

	// Check that the requestBody was added to components
	components := doc["components"].(map[string]any)
	bodiesComp, ok := components["requestBodies"].(map[string]any)
	require.True(t, ok, "Components should have requestBodies section")

	foundCreateRequest := false
	for bodyName := range bodiesComp {
		if bodyName == "CreateRequest" || strings.Contains(bodyName, "CreateRequest") {
			foundCreateRequest = true
			break
		}
	}
	assert.True(t, foundCreateRequest, "CreateRequest should be added to components")
}

// TestBundleBytesComposed_SingleSegmentExample tests that single-segment JSON pointer
// references to example objects are properly recomposed to component references.
func TestBundleBytesComposed_SingleSegmentExample(t *testing.T) {
	rootSpec := `openapi: 3.1.0
paths:
  /test:
    get:
      operationId: getTest
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                type: object
              examples:
                sample:
                  $ref: 'examples.yaml#/SampleExample'
`

	// Example: must have value or externalValue
	examplesFile := `SampleExample:
  summary: A sample example
  value:
    name: test
    id: 123
`

	tmp := t.TempDir()
	write := func(name, src string) {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(src), 0644))
	}
	write("main.yaml", rootSpec)
	write("examples.yaml", examplesFile)

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))

	bundled, err := BundleBytesComposed(mainBytes, &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	}, nil)
	require.NoError(t, err)

	t.Logf("Bundled output:\n%s", string(bundled))

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(bundled, &doc))

	// Check the reference was recomposed
	paths := doc["paths"].(map[string]any)
	testPath := paths["/test"].(map[string]any)
	getOp := testPath["get"].(map[string]any)
	responses := getOp["responses"].(map[string]any)
	resp200 := responses["200"].(map[string]any)
	content := resp200["content"].(map[string]any)
	appJson := content["application/json"].(map[string]any)
	examples := appJson["examples"].(map[string]any)
	sampleExample := examples["sample"].(map[string]any)

	ref, hasRef := sampleExample["$ref"].(string)
	require.True(t, hasRef, "Example should have $ref pointing to component")
	assert.True(t, strings.HasPrefix(ref, "#/components/examples/"),
		"example should reference component, got: %s", ref)
	assert.False(t, strings.Contains(ref, "examples.yaml"),
		"example reference should not contain external file path, got: %s", ref)

	// Check that the example was added to components
	components := doc["components"].(map[string]any)
	examplesComp, ok := components["examples"].(map[string]any)
	require.True(t, ok, "Components should have examples section")

	foundSampleExample := false
	for exampleName := range examplesComp {
		if exampleName == "SampleExample" || strings.Contains(exampleName, "SampleExample") {
			foundSampleExample = true
			break
		}
	}
	assert.True(t, foundSampleExample, "SampleExample should be added to components")
}

// TestBundleBytesComposed_SingleSegmentLink tests that single-segment JSON pointer
// references to link objects are properly recomposed to component references.
func TestBundleBytesComposed_SingleSegmentLink(t *testing.T) {
	rootSpec := `openapi: 3.1.0
paths:
  /test:
    get:
      operationId: getTest
      responses:
        "200":
          description: OK
          links:
            GetNext:
              $ref: 'links.yaml#/NextPageLink'
`

	// Link: must have operationRef or operationId
	linksFile := `NextPageLink:
  operationId: getNextPage
  parameters:
    page: $response.body#/nextPage
`

	tmp := t.TempDir()
	write := func(name, src string) {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(src), 0644))
	}
	write("main.yaml", rootSpec)
	write("links.yaml", linksFile)

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))

	bundled, err := BundleBytesComposed(mainBytes, &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	}, nil)
	require.NoError(t, err)

	t.Logf("Bundled output:\n%s", string(bundled))

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(bundled, &doc))

	// Check the reference was recomposed
	paths := doc["paths"].(map[string]any)
	testPath := paths["/test"].(map[string]any)
	getOp := testPath["get"].(map[string]any)
	responses := getOp["responses"].(map[string]any)
	resp200 := responses["200"].(map[string]any)
	links := resp200["links"].(map[string]any)
	getNextLink := links["GetNext"].(map[string]any)

	ref, hasRef := getNextLink["$ref"].(string)
	require.True(t, hasRef, "Link should have $ref pointing to component")
	assert.True(t, strings.HasPrefix(ref, "#/components/links/"),
		"link should reference component, got: %s", ref)
	assert.False(t, strings.Contains(ref, "links.yaml"),
		"link reference should not contain external file path, got: %s", ref)

	// Check that the link was added to components
	components := doc["components"].(map[string]any)
	linksComp, ok := components["links"].(map[string]any)
	require.True(t, ok, "Components should have links section")

	foundNextPageLink := false
	for linkName := range linksComp {
		if linkName == "NextPageLink" || strings.Contains(linkName, "NextPageLink") {
			foundNextPageLink = true
			break
		}
	}
	assert.True(t, foundNextPageLink, "NextPageLink should be added to components")
}

// TestBundleBytesComposed_SingleSegmentCallback tests that single-segment JSON pointer
// references to callback objects are properly recomposed to component references.
func TestBundleBytesComposed_SingleSegmentCallback(t *testing.T) {
	rootSpec := `openapi: 3.1.0
paths:
  /test:
    post:
      operationId: createTest
      callbacks:
        onEvent:
          $ref: 'callbacks.yaml#/EventCallback'
      responses:
        "201":
          description: Created
`

	// Callback: is a map with keys containing {$
	callbacksFile := `EventCallback:
  '{$request.body#/callbackUrl}':
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
      responses:
        "200":
          description: OK
`

	tmp := t.TempDir()
	write := func(name, src string) {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(src), 0644))
	}
	write("main.yaml", rootSpec)
	write("callbacks.yaml", callbacksFile)

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))

	bundled, err := BundleBytesComposed(mainBytes, &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	}, nil)
	require.NoError(t, err)

	t.Logf("Bundled output:\n%s", string(bundled))

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(bundled, &doc))

	// Check the reference was recomposed
	paths := doc["paths"].(map[string]any)
	testPath := paths["/test"].(map[string]any)
	postOp := testPath["post"].(map[string]any)
	callbacks := postOp["callbacks"].(map[string]any)
	onEventCallback := callbacks["onEvent"].(map[string]any)

	ref, hasRef := onEventCallback["$ref"].(string)
	require.True(t, hasRef, "Callback should have $ref pointing to component")
	assert.True(t, strings.HasPrefix(ref, "#/components/callbacks/"),
		"callback should reference component, got: %s", ref)
	assert.False(t, strings.Contains(ref, "callbacks.yaml"),
		"callback reference should not contain external file path, got: %s", ref)

	// Check that the callback was added to components
	components := doc["components"].(map[string]any)
	callbacksComp, ok := components["callbacks"].(map[string]any)
	require.True(t, ok, "Components should have callbacks section")

	foundEventCallback := false
	for callbackName := range callbacksComp {
		if callbackName == "EventCallback" || strings.Contains(callbackName, "EventCallback") {
			foundEventCallback = true
			break
		}
	}
	assert.True(t, foundEventCallback, "EventCallback should be added to components")
}

// TestBundleBytesComposed_SingleSegmentPathItem tests that single-segment JSON pointer
// references to pathItem objects are properly recomposed to component references.
func TestBundleBytesComposed_SingleSegmentPathItem(t *testing.T) {
	rootSpec := `openapi: 3.1.0
paths:
  /test:
    $ref: 'pathitems.yaml#/TestPath'
`

	// PathItem: has HTTP methods (get, post, etc.) or parameters
	pathitemsFile := `TestPath:
  get:
    operationId: getTest
    responses:
      "200":
        description: OK
  post:
    operationId: createTest
    responses:
      "201":
        description: Created
`

	tmp := t.TempDir()
	write := func(name, src string) {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(src), 0644))
	}
	write("main.yaml", rootSpec)
	write("pathitems.yaml", pathitemsFile)

	mainBytes, _ := os.ReadFile(filepath.Join(tmp, "main.yaml"))

	bundled, err := BundleBytesComposed(mainBytes, &datamodel.DocumentConfiguration{
		BasePath:            tmp,
		AllowFileReferences: true,
	}, nil)
	require.NoError(t, err)

	t.Logf("Bundled output:\n%s", string(bundled))

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(bundled, &doc))

	// Check the reference was recomposed
	paths := doc["paths"].(map[string]any)
	testPath := paths["/test"].(map[string]any)

	ref, hasRef := testPath["$ref"].(string)
	require.True(t, hasRef, "PathItem should have $ref pointing to component")
	assert.True(t, strings.HasPrefix(ref, "#/components/pathItems/"),
		"pathItem should reference component, got: %s", ref)
	assert.False(t, strings.Contains(ref, "pathitems.yaml"),
		"pathItem reference should not contain external file path, got: %s", ref)

	// Check that the pathItem was added to components
	components := doc["components"].(map[string]any)
	pathItemsComp, ok := components["pathItems"].(map[string]any)
	require.True(t, ok, "Components should have pathItems section")

	foundTestPath := false
	for pathItemName := range pathItemsComp {
		if pathItemName == "TestPath" || strings.Contains(pathItemName, "TestPath") {
			foundTestPath = true
			break
		}
	}
	assert.True(t, foundTestPath, "TestPath should be added to components")
}
