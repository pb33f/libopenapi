// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io
// SPDX-License-Identifier: MIT

package bundler

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	highV3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	lowV3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"gopkg.in/yaml.v3"
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
func BundleDocument(model *highV3.Document) ([]byte, error) {
	return bundle(model, BundleOptions{RelativeRefHandling: RefHandlingInline})
}

func bundle(model *highV3.Document, opts BundleOptions) (_ []byte, err error) {
	rolodex := model.Rolodex
	//TODO this is failing on recursion when transversing the references
	idx := rolodex.GetRootIndex()
	mappedReferences := idx.GetMappedReferences()
	sequencedReferences := idx.GetRawReferencesSequenced()

	visitedRefs := make(map[string]bool)

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
			// Recursively process each reference in a depth first traversal.
			if err := processReference(sequenced, mappedReference, model, visitedRefs, opts); err != nil {
				return nil, err
			}
		}
	}
	return model.Render()
}

// processes all the references by traversing the tree and building the components starting with the children in a bottom up fashion
func processReference(ref, mappedRef *index.ReferenceNode, model *highV3.Document, visitedRefs map[string]bool, opts BundleOptions) error {

	idx := ref.Index
	if mappedRef == nil {
		if idx.GetLogger() != nil {
			idx.GetLogger().Warn("[bundler] skipping unresolved reference",
				"ref", ref.FullDefinition)
		}
		return nil
	}

	if visited, exists := visitedRefs[mappedRef.FullDefinition]; exists && visited {
		return nil
	}

	if mappedRef.Circular {
		if idx.GetLogger() != nil {
			idx.GetLogger().Warn("[bundler] skipping circular reference",
				"ref", ref.FullDefinition)
		}
		return nil
	}

	targetIndex := idx
	if targetFile := mappedRef.DefinitionFile(); targetFile != "" {
		targetIndex = idx.GetRolodex().GetFileIndex(targetFile)
	}

	targetMappedReferences := targetIndex.GetMappedReferences()

	childRefs := targetIndex.ExtractRefs(mappedRef.Node, mappedRef.ParentNode, make([]string, 0), 0, false, "")

	for _, childRef := range childRefs {
		childRefTarget := targetMappedReferences[childRef.FullDefinition]
		if err := processReference(childRef, childRefTarget, model, visitedRefs, opts); err != nil {
			return err
		}
	}
	// Build the children references frist then build the parent reference
	if err := buildComponent(mappedRef, model); err != nil {
		return err
	}

	visitedRefs[mappedRef.FullDefinition] = true

	ref.KeyNode.Value = mappedRef.Definition

	return nil

}

// builds the component and registers it in the index
func buildComponent(mappedRef *index.ReferenceNode, model *highV3.Document) error {
	defParts := strings.Split(mappedRef.Definition, "/")
	if len(defParts) != 4 || defParts[1] != lowV3.ComponentsLabel {
		return fmt.Errorf("unsupported component section: %s", mappedRef.Definition)
	}

	// TODO: use constant from low model labels -- these are just fake values added in -- need to update with real stuff
	if len(defParts) != 4 || defParts[1] != lowV3.ComponentsLabel {
		return fmt.Errorf("unsupported component section: %s", mappedRef.Definition)
	}

	lowModel := model.GoLow()
	idx := model.Rolodex.GetRootIndex()
	if idx == nil {
		return fmt.Errorf("no index found")
	}

	switch defParts[2] {
	case "schemas":
		if model.Components.Schemas == nil {
			model.Components.Schemas = orderedmap.New[string, *highbase.SchemaProxy]()
		}
		schemaProxy := new(lowbase.SchemaProxy)
		ctx := context.Background()
		err := schemaProxy.Build(ctx, lowModel.Components.ValueNode, mappedRef.Node, idx)
		if err != nil {
			return fmt.Errorf("failed to build schema proxy for component %s: %w", mappedRef.Definition, err)
		}

		lowSchemaRef := low.NodeReference[*lowbase.SchemaProxy]{
			Value:     schemaProxy,
			ValueNode: mappedRef.Node,
		}

		highSchemaProxy := highbase.NewSchemaProxy(&lowSchemaRef)
		model.Components.Schemas.Set(defParts[3], highSchemaProxy)
		registerComponentInIndex(defParts[2], defParts[3], mappedRef.Node, idx)
	case "responses":
		if model.Components.Responses == nil {
			model.Components.Responses = orderedmap.New[string, *highV3.Response]()
		}

		var response lowV3.Response
		ctx := context.Background()
		err := response.Build(ctx, lowModel.Components.ValueNode, mappedRef.Node, idx)
		if err != nil {
			return fmt.Errorf("failed to build response for component %s: %w", mappedRef.Definition, err)
		}

		highLevelResposne := highV3.NewResponse(&response)
		model.Components.Responses.Set(defParts[3], highLevelResposne)
		registerComponentInIndex(defParts[2], defParts[3], mappedRef.Node, idx)
	case "requestBodies":
		var requestBody lowV3.RequestBody
		ctx := context.Background()
		err := requestBody.Build(ctx, lowModel.Components.ValueNode, mappedRef.Node, idx)
		if err != nil {
			return fmt.Errorf("failed to build response for component %s: %w", mappedRef.Definition, err)
		}
		highLevelRequestBody := highV3.NewRequestBody(&requestBody)
		model.Components.RequestBodies.Set(defParts[3], highLevelRequestBody)
		registerComponentInIndex(defParts[2], defParts[3], mappedRef.Node, idx)

	case "headers":
		var header lowV3.Header
		ctx := context.Background()
		err := header.Build(ctx, lowModel.Components.ValueNode, mappedRef.Node, idx)
		if err != nil {
			return fmt.Errorf("failed to build headers for component %s: %w", mappedRef.Definition, err)
		}
		highLevelHeader := highV3.NewHeader(&header)
		model.Components.Headers.Set(defParts[3], highLevelHeader)
		registerComponentInIndex(defParts[2], defParts[3], mappedRef.Node, idx)

	case "securitySchemes":
		var securityScheme lowV3.SecurityScheme
		ctx := context.Background()
		err := securityScheme.Build(ctx, lowModel.Components.ValueNode, mappedRef.Node, idx)
		if err != nil {
			return fmt.Errorf("failed to build securitySchemes for component %s: %w", mappedRef.Definition, err)
		}
		highLevelSecurityScheme := highV3.NewSecurityScheme(&securityScheme)
		model.Components.SecuritySchemes.Set(defParts[3], highLevelSecurityScheme)
		registerComponentInIndex(defParts[2], defParts[3], mappedRef.Node, idx)

	case "examples":
		var example lowbase.Example
		ctx := context.Background()
		err := example.Build(ctx, lowModel.Components.ValueNode, mappedRef.Node, idx)
		if err != nil {
			return fmt.Errorf("failed to build example for component %s: %w", mappedRef.Definition, err)
		}
		highLevelExample := highbase.NewExample(&example)
		model.Components.Examples.Set(defParts[3], highLevelExample)
		registerComponentInIndex(defParts[2], defParts[3], mappedRef.Node, idx)

	case "links":
		var link lowV3.Link
		ctx := context.Background()
		err := link.Build(ctx, lowModel.Components.ValueNode, mappedRef.Node, idx)
		if err != nil {
			return fmt.Errorf("failed to build response for component %s: %w", mappedRef.Definition, err)
		}
		highLevelLink := highV3.NewLink(&link)
		model.Components.Links.Set(defParts[3], highLevelLink)
		registerComponentInIndex(defParts[2], defParts[3], mappedRef.Node, idx)

	case "callbacks":
		var callback lowV3.Callback
		ctx := context.Background()
		err := callback.Build(ctx, lowModel.Components.ValueNode, mappedRef.Node, idx)
		if err != nil {
			return fmt.Errorf("failed to build response for component %s: %w", mappedRef.Definition, err)
		}
		highLevelCallback := highV3.NewCallback(&callback)
		model.Components.Callbacks.Set(defParts[3], highLevelCallback)
		registerComponentInIndex(defParts[2], defParts[3], mappedRef.Node, idx)

	case "parameters":
		var parameter lowV3.Parameter
		ctx := context.Background()
		err := parameter.Build(ctx, lowModel.Components.ValueNode, mappedRef.Node, idx)
		if err != nil {
			return fmt.Errorf("failed to build parameter for component %s: %w", mappedRef.Definition, err)
		}
		highLevelParameter := highV3.NewParameter(&parameter)
		model.Components.Parameters.Set(defParts[3], highLevelParameter)
		registerComponentInIndex(defParts[2], defParts[3], mappedRef.Node, idx)

	default:
		return fmt.Errorf("unsupported component type: %s", defParts[2])
	}

	return nil
}

// register the new component in the index so when we build the parents the index will have visibility of the child references
func registerComponentInIndex(componentType string, componentName string, node *yaml.Node, idx *index.SpecIndex) error {
	refPath := fmt.Sprintf("#/components/%s/%s", componentType, componentName)

	mappedReferences := idx.GetMappedReferences()

	newRef := &index.ReferenceNode{
		Definition:     refPath,
		FullDefinition: refPath,
		Node:           node,
		Index:          idx,
		Seen:           true,
		Resolved:       true,
	}

	mappedReferences[refPath] = newRef

	switch componentType {
	case "schemas":
		schemaRefs := idx.GetAllComponentSchemas()
		schemaRefs[refPath] = newRef

	case "responses":
		responseRefs := idx.GetAllResponses()
		responseRefs[refPath] = newRef

	case "parameters":
		parameterRefs := idx.GetAllParameters()
		parameterRefs[refPath] = newRef

	case "headers":
		headerRefs := idx.GetAllHeaders()
		headerRefs[refPath] = newRef

	case "securitySchemes":
		securitySchemesRefs := idx.GetAllSecuritySchemes()
		securitySchemesRefs[refPath] = newRef

	case "examples":
		exampleRefs := idx.GetAllExamples()
		exampleRefs[refPath] = newRef

	case "requestBodies":
		requestBodyRefs := idx.GetAllRequestBodies()
		requestBodyRefs[refPath] = newRef

	case "links":
		linkRefs := idx.GetAllLinks()
		linkRefs[refPath] = newRef

	case "callbacks":
		callbackRefs := idx.GetAllCallbacks()
		callbackRefs[refPath] = newRef
	}

	return nil
}

//TODO delete this and clean it up
// func bundleRefTarget(ref, mappedRef *index.ReferenceNode, bundledComponents map[string]*index.ReferenceNode, opts BundleOptions) error {
// 	//TOODO this is failing on recursion when transversing the references
// 	idx := ref.Index
// 	if mappedRef == nil {
// 		if idx.GetLogger() != nil {
// 			idx.GetLogger().Warn("[bundler] skipping unresolved reference",
// 				"ref", ref.FullDefinition)
// 		}
// 		return nil
// 	}

// 	if mappedRef.Circular {
// 		if idx.GetLogger() != nil {
// 			idx.GetLogger().Warn("[bundler] skipping circular reference",
// 				"ref", ref.FullDefinition)
// 		}
// 		return nil
// 	}

// 	bundledRef, exists := bundledComponents[mappedRef.Definition]
// 	if exists && bundledRef.FullDefinition != mappedRef.FullDefinition {
// 		// TODO: we don't want to error here
// 		return fmt.Errorf("duplicate component definition: %s", mappedRef.Definition)
// 	} else {
// 		bundledComponents[mappedRef.Definition] = mappedRef
// 		ref.KeyNode.Value = mappedRef.Definition
// 	}

// 	// When composing, we need to update the ref values to point to a local reference. At the
// 	// same time we need to track all components referenced by any children of the target, so
// 	// that we can include them in the final document.
// 	//
// 	// One issue we might face is that the name of a target component in any given target
// 	// document is the same as that of another component in a different target document or
// 	// even the root document.

// 	// Obtain the target's file's index because we should find child references using that.
// 	// Otherwise ExtractRefs will use the ref's index and it's absolute spec path for
// 	// the FullPath of any extracted ref targets.
// 	targetIndex := idx
// 	if targetFile := mappedRef.DefinitionFile(); targetFile != "" {
// 		targetIndex = idx.GetRolodex().GetFileIndex(targetFile)
// 	}

// 	targetMappedReferences := targetIndex.GetMappedReferences()

// 	childRefs := targetIndex.ExtractRefs(mappedRef.Node, mappedRef.ParentNode, make([]string, 0), 0, false, "")
// 	for _, childRef := range childRefs {
// 		childRefTarget := targetMappedReferences[childRef.FullDefinition]
// 		if err := bundleRefTarget(childRef, childRefTarget, bundledComponents, opts); err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }
