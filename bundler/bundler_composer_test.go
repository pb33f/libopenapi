// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package bundler

import (
	"errors"
	"log/slog"
	"os"
	"regexp"
	"runtime"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
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
	if len(errs) > 0 {
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
	if len(errs) > 0 {
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
