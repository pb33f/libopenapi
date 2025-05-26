// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package bundler

import (
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestProcessRef_UnknownLocation(t *testing.T) {

	// create an empty doc
	doc, _ := libopenapi.NewDocument([]byte("openapi: 3.1.1"))
	m, _ := doc.BuildV3Model()

	ref := &processRef{
		idx: m.Index,
		ref: &index.Reference{
			FullDefinition: "#/blarp",
		},
		seqRef:   nil,
		name:     "test",
		location: []string{"unknown"},
	}

	config := &handleIndexConfig{
		compositionConfig: &BundleCompositionConfig{
			Delimiter: "__",
		},
		idx: m.Index,
	}

	err := processReference(&m.Model, ref, config)

	assert.NoError(t, err)
	assert.Len(t, config.inlineRequired, 1)

}

func TestProcessRef_UnknownLocation_TwoStep(t *testing.T) {

	// create an empty doc
	doc, _ := libopenapi.NewDocument([]byte("openapi: 3.1.1"))
	m, _ := doc.BuildV3Model()

	ref := &processRef{
		idx: m.Index,
		ref: &index.Reference{
			FullDefinition: "blip.yaml#/blarp/blop",
		},
		seqRef:   nil,
		name:     "test",
		location: []string{"unknown"},
	}

	config := &handleIndexConfig{
		compositionConfig: &BundleCompositionConfig{
			Delimiter: "__",
		},
		idx: m.Index,
	}

	err := processReference(&m.Model, ref, config)

	assert.NoError(t, err)
	assert.Len(t, config.inlineRequired, 1)

}

func TestProcessRef_UnknownLocation_ThreeStep(t *testing.T) {

	// create an empty doc
	doc, _ := libopenapi.NewDocument([]byte("openapi: 3.1.1"))
	m, _ := doc.BuildV3Model()

	ref := &processRef{
		idx: m.Index,
		ref: &index.Reference{
			FullDefinition: "bleep.yaml#/blarp/blop/blurp",
			Definition:     "#/blarp/blop/blurp",
		},
		seqRef:   nil,
		name:     "test",
		location: []string{"unknown"},
	}

	config := &handleIndexConfig{
		compositionConfig: &BundleCompositionConfig{
			Delimiter: "__",
		},
		idx: m.Index,
	}

	err := processReference(&m.Model, ref, config)

	assert.NoError(t, err)
	assert.Len(t, config.inlineRequired, 1)

}
