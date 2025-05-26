// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io
// SPDX-License-Identifier: MIT

package bundler

import (
	"errors"
	"slices"
	"strings"
	"sync"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
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

	bundledBytes, e := bundle(&v3Doc.Model)
	return bundledBytes, errors.Join(err, e)
}

// BundleBytesComposed will take a byte slice of an OpenAPI specification and return a composed bundled version of it.
// this is the same as BundleBytes, but it will compose the bundling instead of inline it.
func BundleBytesComposed(bytes []byte, configuration *datamodel.DocumentConfiguration, compositionConfig *BundleCompositionConfig) ([]byte, error) {
	doc, err := libopenapi.NewDocumentWithConfiguration(bytes, configuration)
	if err != nil {
		return nil, err
	}

	v3Doc, errs := doc.BuildV3Model()
	err = errors.Join(errs...)
	if v3Doc == nil || len(errs) > 0 {
		return nil, errors.Join(ErrInvalidModel, err)
	}

	bundledBytes, e := compose(&v3Doc.Model, compositionConfig)
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
	return bundle(model)
}

// BundleCompositionConfig is used to configure the composition of OpenAPI documents when using BundleDocumentComposed.
type BundleCompositionConfig struct {
	Delimiter string // Delimiter is used to separate clashing names. Defaults to `__`.
}

// BundleDocumentComposed will take a v3.Document and return a composed bundled version of it. Composed means
// that every external file will have references lifted out and added to the `components` section of the document.
// Names will be preserved where possible, conflicts will be appended with a number. If the type of the reference cannot
// be determined, it will be added to the `components` section as a `Schema` type, a warning will be logged.
// The document model will be mutated permanently.
//
// Circular references will not be resolved and will be skipped.
func BundleDocumentComposed(model *v3.Document, compositionConfig *BundleCompositionConfig) ([]byte, error) {
	return compose(model, compositionConfig)
}

func compose(model *v3.Document, compositionConfig *BundleCompositionConfig) ([]byte, error) {
	if compositionConfig == nil {
		compositionConfig = &BundleCompositionConfig{
			Delimiter: "__",
		}
	} else {
		if compositionConfig.Delimiter == "" {
			compositionConfig.Delimiter = "__"
		}
		if strings.Contains(compositionConfig.Delimiter, "#") ||
			strings.Contains(compositionConfig.Delimiter, "/") {
			return nil, errors.New("composition delimiter cannot contain '#' or '/' characters")
		}
		if strings.Contains(compositionConfig.Delimiter, " ") {
			return nil, errors.New("composition delimiter cannot contain spaces")
		}
	}

	if model == nil || model.Rolodex == nil {
		return nil, errors.New("model or rolodex is nil")
	}

	rolodex := model.Rolodex
	indexes := rolodex.GetIndexes()

	cf := &handleIndexConfig{
		idx:               rolodex.GetRootIndex(),
		model:             model,
		indexes:           indexes,
		seen:              sync.Map{},
		refMap:            orderedmap.New[string, *processRef](),
		compositionConfig: compositionConfig,
	}
	// recursive function to handle the indexes, we need a different approach to composition vs. inlining.
	handleIndex(cf)

	processedNodes := orderedmap.New[string, *processRef]()
	var errs []error
	for _, ref := range cf.refMap.FromOldest() {
		err := processReference(model, ref, cf)
		errs = append(errs, err)
		processedNodes.Set(ref.ref.FullDefinition, ref)
	}

	rootIndex := rolodex.GetRootIndex()
	remapIndex(rootIndex, processedNodes)

	slices.SortFunc(indexes, func(i, j *index.SpecIndex) int {
		if i.GetSpecAbsolutePath() < j.GetSpecAbsolutePath() {
			return 1
		}
		return 0
	})

	for _, idx := range indexes {
		remapIndex(idx, processedNodes)
	}

	// anything that could not be recomposed and needs inlining
	for _, pr := range cf.inlineRequired {
		pr.seqRef.Node.Content = pr.ref.Node.Content
	}

	b, err := model.Render()
	errs = append(errs, err)

	return b, errors.Join(errs...)
}

func bundle(model *v3.Document) ([]byte, error) {
	rolodex := model.Rolodex
	indexes := rolodex.GetIndexes()
	// indexMap := make(map[string]*index.SpecIndex)
	// compact function.
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
								mr := i.FindComponent(sequenced.Definition)
								if mr != nil {
									// found the component; this is the one we want to use.
									mappedReference = mr
									break
								}
							}
						}
					}
				}

				if refExp[0] == sequenced.Index.GetSpecAbsolutePath() || refExp[0] == "" {
					if root {
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
