// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

// Package libopenapi is a library containing tools for reading and in and manipulating Swagger (OpenAPI 2) and OpenAPI 3+
// specifications into strongly typed documents. These documents have two APIs, a high level (porcelain) and a
// low level (plumbing).
//
// Every single type has a 'GoLow()' method that drops down from the high API to the low API. Once in the low API,
// the entire original document data is available, including all comments, line and column numbers for keys and values.
//
// There are two steps to creating a using Document. First, create a new Document using the NewDocument() method
// and pass in a specification []byte array that contains the OpenAPI Specification. It doesn't matter if YAML or JSON
// are used.
package libopenapi

import (
	"fmt"

	"github.com/pb33f/libopenapi/datamodel"
	v2high "github.com/pb33f/libopenapi/datamodel/high/v2"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
	v2low "github.com/pb33f/libopenapi/datamodel/low/v2"
	v3low "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/resolver"
	"github.com/pb33f/libopenapi/utils"
	what_changed "github.com/pb33f/libopenapi/what-changed"
	"github.com/pb33f/libopenapi/what-changed/model"
	"gopkg.in/yaml.v3"
)

// Document Represents an OpenAPI specification that can then be rendered into a model or serialized back into
// a string document after being manipulated.
type Document interface {

	// GetVersion will return the exact version of the OpenAPI specification set for the document.
	GetVersion() string

	// GetSpecInfo will return the *datamodel.SpecInfo instance that contains all specification information.
	GetSpecInfo() *datamodel.SpecInfo

	// BuildV2Model will build out a Swagger (version 2) model from the specification used to create the document
	// If there are any issues, then no model will be returned, instead a slice of errors will explain all the
	// problems that occurred. This method will only support version 2 specifications and will throw an error for
	// any other types.
	BuildV2Model() (*DocumentModel[v2high.Swagger], []error)

	// BuildV3Model will build out an OpenAPI (version 3+) model from the specification used to create the document
	// If there are any issues, then no model will be returned, instead a slice of errors will explain all the
	// problems that occurred. This method will only support version 3 specifications and will throw an error for
	// any other types.
	BuildV3Model() (*DocumentModel[v3high.Document], []error)

	// Serialize will re-render a Document back into a []byte slice. If any modifications have been made to the
	// underlying data model using low level APIs, then those changes will be reflected in the serialized output.
	//
	// It's important to know that this should not be used if the resolver has been used on a specification to
	// for anything other than checking for circular references. If the resolver is used to resolve the spec, then this
	// method may spin out forever if the specification backing the model has circular references.
	Serialize() ([]byte, error)
}

type document struct {
	version string
	info    *datamodel.SpecInfo
}

// DocumentModel represents either a Swagger document (version 2) or an OpenAPI document (version 3) that is
// built from a parent Document.
type DocumentModel[T v2high.Swagger | v3high.Document] struct {
	Model T
}

// NewDocument will create a new OpenAPI instance from an OpenAPI specification []byte array. If anything goes
// wrong when parsing, reading or processing the OpenAPI specification, there will be no document returned, instead
// a slice of errors will be returned that explain everything that failed.
//
// After creating a Document, the option to build a model becomes available, in either V2 or V3 flavors. The models
// are about 70% different between Swagger and OpenAPI 3, which is why two different models are available.
func NewDocument(specByteArray []byte) (Document, error) {
	info, err := datamodel.ExtractSpecInfo(specByteArray)
	if err != nil {
		return nil, err
	}
	d := new(document)
	d.version = info.Version
	d.info = info
	return d, nil
}

func (d *document) GetVersion() string {
	return d.version
}

func (d *document) GetSpecInfo() *datamodel.SpecInfo {
	return d.info
}

func (d *document) Serialize() ([]byte, error) {
	if d.info == nil {
		return nil, fmt.Errorf("unable to serialize, document has not yet been initialized")
	}
	if d.info.SpecFileType == datamodel.YAMLFileType {
		return yaml.Marshal(d.info.RootNode)
	} else {
		yamlData, _ := yaml.Marshal(d.info.RootNode)
		return utils.ConvertYAMLtoJSON(yamlData)
	}
}

func (d *document) BuildV2Model() (*DocumentModel[v2high.Swagger], []error) {
	var errors []error
	if d.info == nil {
		errors = append(errors, fmt.Errorf("unable to build swagger document, no specification has been loaded"))
		return nil, errors
	}
	if d.info.SpecFormat != datamodel.OAS2 {
		errors = append(errors, fmt.Errorf("unable to build swagger document, "+
			"supplied spec is a different version (%v). Try 'BuildV3Model()'", d.info.SpecFormat))
		return nil, errors
	}
	lowDoc, errs := v2low.CreateDocument(d.info)
	// Do not shortcircuit on circular reference errors, so the client
	// has the option of ignoring them.
	for _, err := range errs {
		if refErr, ok := err.(*resolver.ResolvingError); ok {
			if refErr.CircularReference == nil {
				return nil, errs
			}
		} else {
			return nil, errs
		}
	}
	highDoc := v2high.NewSwaggerDocument(lowDoc)
	return &DocumentModel[v2high.Swagger]{
		Model: *highDoc,
	}, errs
}

func (d *document) BuildV3Model() (*DocumentModel[v3high.Document], []error) {
	var errors []error
	if d.info == nil {
		errors = append(errors, fmt.Errorf("unable to build document, no specification has been loaded"))
		return nil, errors
	}
	if d.info.SpecFormat != datamodel.OAS3 {
		errors = append(errors, fmt.Errorf("unable to build openapi document, "+
			"supplied spec is a different version (%v). Try 'BuildV2Model()'", d.info.SpecFormat))
		return nil, errors
	}
	lowDoc, errs := v3low.CreateDocument(d.info)
	// Do not shortcircuit on circular reference errors, so the client
	// has the option of ignoring them.
	for _, err := range errs {
		if refErr, ok := err.(*resolver.ResolvingError); ok {
			if refErr.CircularReference == nil {
				return nil, errs
			}
		} else {
			return nil, errs
		}
	}
	highDoc := v3high.NewDocument(lowDoc)
	return &DocumentModel[v3high.Document]{
		Model: *highDoc,
	}, errs
}

// CompareDocuments will accept a left and right Document implementing struct, build a model for the correct
// version and then compare model documents for changes.
//
// If there are any errors when building the models, those errors are returned with a nil pointer for the
// model.DocumentChanges. If there are any changes found however between either Document, then a pointer to
// model.DocumentChanges is returned containing every single change, broken down, model by model.
func CompareDocuments(original, updated Document) (*model.DocumentChanges, []error) {
	var errors []error
	if original.GetSpecInfo().SpecType == utils.OpenApi3 && updated.GetSpecInfo().SpecType == utils.OpenApi3 {
		v3ModelLeft, errs := original.BuildV3Model()
		if len(errs) > 0 {
			errors = errs
		}
		v3ModelRight, errs := updated.BuildV3Model()
		if len(errs) > 0 {
			errors = errs
		}
		if len(errors) > 0 {
			return nil, errors
		}
		return what_changed.CompareOpenAPIDocuments(v3ModelLeft.Model.GoLow(), v3ModelRight.Model.GoLow()), nil
	}
	if original.GetSpecInfo().SpecType == utils.OpenApi2 && updated.GetSpecInfo().SpecType == utils.OpenApi2 {
		v2ModelLeft, errs := original.BuildV2Model()
		if len(errs) > 0 {
			errors = errs
		}
		v2ModelRight, errs := updated.BuildV2Model()
		if len(errs) > 0 {
			errors = errs
		}
		if len(errors) > 0 {
			return nil, errors
		}
		return what_changed.CompareSwaggerDocuments(v2ModelLeft.Model.GoLow(), v2ModelRight.Model.GoLow()), nil
	}
	return nil, nil
}
