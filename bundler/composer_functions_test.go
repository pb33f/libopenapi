// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package bundler

import (
	"testing"

	"github.com/pb33f/libopenapi/orderedmap"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
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

// A component key that contains a dot (“asdf.zxcv”) must *not* be shortened to
// “asdf” when we re-wire references.
func TestRenameRef_KeepsDotInComponentName(t *testing.T) {
	spec := []byte(`
openapi: 3.1.0
info:
  title: Test
  version: 0.0.0
components:
  schemas:
    "asdf.zxcv":
      type: object
    Foo:
      allOf:
        - $ref: '#/components/schemas/asdf.zxcv'
`)

	doc, err := libopenapi.NewDocument(spec)
	assert.NoError(t, err)

	v3doc, errs := doc.BuildV3Model()
	assert.NoError(t, errs)

	idx := v3doc.Model.Index

	processed := orderedmap.New[string, *processRef]()

	got := renameRef(idx, "#/components/schemas/asdf.zxcv", processed)

	assert.Equal(t,
		"#/components/schemas/asdf.zxcv",
		got,
		"renameRef must not strip the .zxcv part from the component key")
}

// A reference that really *is* a filename + JSON-pointer must still have the
// extension stripped
func TestRenameRef_FilePointer_Extensions(t *testing.T) {
	exts := []string{".yaml", ".yml", ".json"}

	for _, ext := range exts {
		def := "schemas/pet" + ext + "#/components/schemas/Pet"
		got := renameRef(nil, def, orderedmap.New[string, *processRef]())
		assert.Equal(t, "#/components/schemas/Pet", got,
			"extension %s should not affect the pointer rewrite", ext)
	}
}

// If a component name has already been changed during composition,
// renameRef must return that new name.
func TestRenameRef_RespectsAlreadyRenamedComponent(t *testing.T) {
	ps := orderedmap.New[string, *processRef]()
	ps.Set("#/components/schemas/asdf.zxcv",
		&processRef{name: "asdf__1", location: []string{}})

	got := renameRef(nil,
		"#/components/schemas/asdf.zxcv",
		ps)

	assert.Equal(t,
		"#/components/schemas/asdf__1",
		got,
		"renameRef should use the name stored in processedNodes")
}

func TestRenameRef_RootFileImport(t *testing.T) {
	processed := orderedmap.New[string, *processRef]()
	processed.Set("schemas/pet.yaml",
		&processRef{location: []string{"components", "schemas", "Pet"}})

	got := renameRef(nil, "schemas/pet.yaml", processed)

	assert.Equal(t, "#/components/schemas/Pet", got)
}

// A JSON-pointer that has only one segment (e.g. "#/Foo") must be returned
// unchanged
func TestRenameRef_ShortPointerIsReturnedUnchanged(t *testing.T) {
	got := renameRef(nil, "#/Foo", orderedmap.New[string, *processRef]())
	assert.Equal(t, "#/Foo", got)
}
