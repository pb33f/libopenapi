// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io
// SPDX-License-Identifier: MIT

package bundler

import (
	"errors"

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
	return bundle(model, BundleOptions{RelativeRefHandling: RefHandlingIgnore})
}

func bundle(model *v3.Document, opts BundleOptions) ([]byte, error) {
	rolodex := model.Rolodex

	// indexes := rolodex.GetIndexes()
	// for _, idx := range indexes {
	// 	handleRefs(idx, opts)
	// }

	idx := rolodex.GetRootIndex()
	mappedReferences := idx.GetMappedReferences()
	sequencedReferences := idx.GetRawReferencesSequenced()

	for _, sequenced := range sequencedReferences {
		mappedReference := mappedReferences[sequenced.FullDefinition]
		bundleRefTarget(sequenced, mappedReference, opts)
	}

	return model.Render()
}

// TODO: use orderedmap as return value
func bundleRefTarget(ref, refTarget *index.ReferenceNode, opts BundleOptions) (map[string]*index.ReferenceNode, error) {
	idx := ref.Index
	if refTarget == nil {
		if idx.GetLogger() != nil {
			idx.GetLogger().Warn("[bundler] skipping unresolved reference",
				"ref", ref.FullDefinition)
		}
		return nil, nil
	}

	if refTarget.Circular {
		if idx.GetLogger() != nil {
			idx.GetLogger().Warn("[bundler] skipping circular reference",
				"ref", ref.FullDefinition)
		}
		return nil, nil
	}

	switch opts.RelativeRefHandling {
	case RefHandlingInline:
		ref.Node.Content = refTarget.Node.Content
	case RefHandlingCompose:
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
		if targetFile := refTarget.DefinitionFile(); targetFile != "" {
			targetIndex = idx.GetRolodex().GetFileIndex(targetFile)
		}

		targetMappedReferences := targetIndex.GetMappedReferences()

		bundledComponents := map[string]*index.ReferenceNode{
			target
		}

		childRefs := targetIndex.ExtractRefs(refTarget.Node, refTarget.ParentNode, make([]string, 0), 0, false, "")

		for _, childRef := range childRefs {
			childRefTarget := targetMappedReferences[childRef.FullDefinition]
			childComponents, err := bundleRefTarget(childRef, childRefTarget, opts)
			if err != nil {
				return nil, err
			}
		}
	}


	// TODO:
	return nil, nil
}
