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
	"strings"
)

// ErrInvalidModel is returned when the model is not usable.
var ErrInvalidModel = errors.New("invalid model")

// BundleBytes will take a byte slice of an OpenAPI specification and return a bundled version of it.
// This is useful for when you want to take a specification with external references, and you want to bundle it
// into a single document.
//
// This function will 'resolve' all references in the specification and return a single document. The resulting
// document will be a valid OpenAPI specification, containing no references.
//
// Circular references will not be resolved and will be skipped.
func BundleBytes(bytes []byte, configuration *datamodel.DocumentConfiguration) ([]byte, error) {
	doc, err := libopenapi.NewDocumentWithConfiguration(bytes, configuration)
	if err != nil {
		return nil, err
	}

	v3Doc, errs := doc.BuildV3Model()
	err = errors.Join(errs...)
	if v3Doc == nil {
		return nil, errors.Join(ErrInvalidModel, err)
	}

	bundledBytes, e := bundle(&v3Doc.Model, configuration.BundleInlineRefs)
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
	return bundle(model, false)
}

func bundle(model *v3.Document, inline bool) ([]byte, error) {
	rolodex := model.Rolodex
	indexes := rolodex.GetIndexes()
	compact := func(idx *index.SpecIndex, root bool) {
		mappedReferences := idx.GetMappedReferences()
		sequencedReferences := idx.GetRawReferencesSequenced()
		for _, sequenced := range sequencedReferences {
			mappedReference := mappedReferences[sequenced.FullDefinition]

			// if we're in the root document, don't bundle anything.
			refExp := strings.Split(sequenced.FullDefinition, "#/")
			if len(refExp) == 2 {

				// make sure to use the correct index.
				// https://github.com/pb33f/libopenapi/issues/397
				if root {
					for _, i := range indexes {
						if i.GetSpecAbsolutePath() == refExp[0] {
							if mappedReference != nil && !mappedReference.Circular {
								mr := i.FindComponent(refExp[1])
								if mr != nil {
									// found the component, this is the one we want to use.
									mappedReference = mr
									break
								}
							}
						}
					}
				}

				if refExp[0] == sequenced.Index.GetSpecAbsolutePath() || refExp[0] == "" {
					if root && !inline {
						idx.GetLogger().Debug("[bundler] skipping local root reference",
							"ref", sequenced.Definition)
						continue
					}
				}
			}

			if mappedReference != nil && !mappedReference.Circular {
				sequenced.Node.Content = mappedReference.Node.Content
				continue
			}

			if mappedReference != nil && mappedReference.Circular {
				if idx.GetLogger() != nil {
					idx.GetLogger().Warn("[bundler] skipping circular reference",
						"ref", sequenced.FullDefinition)
				}
			}
		}
	}

	for _, idx := range indexes {
		compact(idx, false)
	}
	compact(rolodex.GetRootIndex(), true)
	return model.Render()
}
