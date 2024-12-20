// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io
// SPDX-License-Identifier: MIT

package bundler

import (
	"errors"
	"fmt"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/index"
)

// ErrInvalidModel is returned when the model is not usable.
var ErrInvalidModel = errors.New("invalid model")

type RefHandling string

const (
	RefHandlingInline  RefHandling = "inline"
	RefHandlingCompose RefHandling = "compose"
)

type BundleOptions struct {
	RelativeRefHandling RefHandling
}

// BundleBytes will take a byte slice of an OpenAPI specification and return a bundled version of it.
// This is useful for when you want to take a specification with external references, and you want to bundle it
// into a single document.
//
// This function will 'resolve' all references in the specification and return a single document. The resulting
// document will be a valid OpenAPI specification, containing no references.
//
// Circular references will not be resolved and will be skipped.
func BundleBytes(bytes []byte, configuration *datamodel.DocumentConfiguration, opts BundleOptions) ([]byte, error) {
	doc, err := libopenapi.NewDocumentWithConfiguration(bytes, configuration)
	if err != nil {
		return nil, err
	}

	v3Doc, errs := doc.BuildV3Model()
	err = errors.Join(errs...)
	if v3Doc == nil {
		return nil, errors.Join(ErrInvalidModel, err)
	}

	// Overwrite bundle options, if deprecated config field is used.
	if configuration.BundleInlineRefs {
		opts.RelativeRefHandling = RefHandlingInline
	}

	bundledBytes, e := bundle(&v3Doc.Model, opts)
	return bundledBytes, errors.Join(err, e)
}

// BundleDocument will take a v3.Document and return a bundled version of it.
// This is useful for when you want to take a document that has been built
// from a specification with external references, and you want to bundle it
// into a single document.
//
// This function will 'resolve' all references in the specification and return a single document. The resulting
// document will be a valid OpenAPI specification, containing no references.
//
// Circular references will not be resolved and will be skipped.
func BundleDocument(model *v3.Document) ([]byte, error) {
	return bundle(model, BundleOptions{RelativeRefHandling: RefHandlingInline})
}

func bundle(model *v3.Document, opts BundleOptions) (_ []byte, err error) {
	rolodex := model.Rolodex

	idx := rolodex.GetRootIndex()
	mappedReferences := idx.GetMappedReferences()
	sequencedReferences := idx.GetRawReferencesSequenced()

	for _, sequenced := range sequencedReferences {
		mappedReference := mappedReferences[sequenced.FullDefinition]
		if mappedReference == nil {
			return nil, fmt.Errorf("no mapped reference found for: %s", sequenced.FullDefinition)
		}

		if mappedReference.DefinitionFile() == idx.GetSpecAbsolutePath() {
			// Don't bundle anything that's in the main file.
			continue
		}

		switch opts.RelativeRefHandling {
		case RefHandlingInline:
			// Just deal with simple inlining.
			sequenced.Node.Content = mappedReference.Node.Content
		case RefHandlingCompose:
			// Recursively collect all reference targets to be bundled into the root
			// file.
			bundledComponents := make(map[string]*index.ReferenceNode)
			if err := bundleRefTarget(sequenced, mappedReference, bundledComponents, opts); err != nil {
				return nil, err
			}
		}
	}

	return model.Render()
}

func bundleRefTarget(ref, mappedRef *index.ReferenceNode, bundledComponents map[string]*index.ReferenceNode, opts BundleOptions) error {
	idx := ref.Index
	if mappedRef == nil {
		if idx.GetLogger() != nil {
			idx.GetLogger().Warn("[bundler] skipping unresolved reference",
				"ref", ref.FullDefinition)
		}
		return nil
	}

	if mappedRef.Circular {
		if idx.GetLogger() != nil {
			idx.GetLogger().Warn("[bundler] skipping circular reference",
				"ref", ref.FullDefinition)
		}
		return nil
	}

	bundledRef, exists := bundledComponents[mappedRef.Definition]
	if exists && bundledRef.FullDefinition != mappedRef.FullDefinition {
		// TODO: we don't want to error here
		return fmt.Errorf("duplicate component definition: %s", mappedRef.Definition)
	} else {
		bundledComponents[mappedRef.Definition] = mappedRef
		ref.KeyNode.Value = mappedRef.Definition
	}

	// When composing, we need to update the ref values to point to a local reference. At the
	// same time we need to track all components referenced by any children of the target, so
	// that we can include them in the final document.
	//
	// One issue we might face is that the name of a target component in any given target
	// document is the same as that of another component in a different target document or
	// even the root document.

	// Obtain the target's file's index because we should find child references using that.
	// Otherwise ExtractRefs will use the ref's index and it's absolute spec path for
	// the FullPath of any extracted ref targets.
	targetIndex := idx
	if targetFile := mappedRef.DefinitionFile(); targetFile != "" {
		targetIndex = idx.GetRolodex().GetFileIndex(targetFile)
	}

	targetMappedReferences := targetIndex.GetMappedReferences()

	childRefs := targetIndex.ExtractRefs(mappedRef.Node, mappedRef.ParentNode, make([]string, 0), 0, false, "")
	for _, childRef := range childRefs {
		childRefTarget := targetMappedReferences[childRef.FullDefinition]
		if err := bundleRefTarget(childRef, childRefTarget, bundledComponents, opts); err != nil {
			return err
		}
	}

	return nil
}
