// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package bundler

import (
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"os"
	"testing"
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

	bytes, err := BundleDocumentComposed(&v3Doc.Model)
	if err != nil {
		panic(err)
	}
	assert.Len(t, bytes, 5069)

	// write the bundled spec to a file for inspection
	// uncomment this to rebuild the bundled spec file, if the example spec changes.
	//err = os.WriteFile("test/specs/bundled.yaml", bytes, 0644)

	preBundled, bErr := os.ReadFile("test/specs/bundled.yaml")
	assert.NoError(t, bErr)
	assert.Equal(t, string(preBundled), string(bytes))
}
