// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"github.com/pb33f/libopenapi/datamodel"
	v2high "github.com/pb33f/libopenapi/datamodel/high/2.0"
	v3high "github.com/pb33f/libopenapi/datamodel/high/3.0"
	v2low "github.com/pb33f/libopenapi/datamodel/low/2.0"
	v3low "github.com/pb33f/libopenapi/datamodel/low/3.0"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

type Document struct {
	version string
	info    *datamodel.SpecInfo
}

type DocumentModel[T v2high.Swagger | v3high.Document] struct {
	Model T
}

func NewDocument(specByteArray []byte) (*Document, error) {
	info, err := datamodel.ExtractSpecInfo(specByteArray)
	if err != nil {
		return nil, err
	}
	d := new(Document)
	d.version = info.Version
	d.info = info
	return d, nil
}

func (d *Document) GetVersion() string {
	return d.version
}

func (d *Document) GetSpecInfo() *datamodel.SpecInfo {
	return d.info
}

func (d *Document) Serialize() ([]byte, error) {
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

func (d *Document) BuildV2Document() (*DocumentModel[v2high.Swagger], []error) {
	var errors []error
	if d.info == nil {
		errors = append(errors, fmt.Errorf("unable to build swagger document, no specification has been loaded"))
		return nil, errors
	}
	if d.info.SpecFormat != datamodel.OAS2 {
		errors = append(errors, fmt.Errorf("unable to build swagger document, "+
			"supplied spec is a different version (%v). Try 'BuildV3Document()'", d.info.SpecFormat))
		return nil, errors
	}
	lowDoc, err := v2low.CreateDocument(d.info)
	if err != nil {
		return nil, err
	}
	highDoc := v2high.NewSwaggerDocument(lowDoc)
	return &DocumentModel[v2high.Swagger]{
		Model: *highDoc,
	}, nil
}

func (d *Document) BuildV3Document() (*DocumentModel[v3high.Document], []error) {
	var errors []error
	if d.info == nil {
		errors = append(errors, fmt.Errorf("unable to build document, no specification has been loaded"))
		return nil, errors
	}
	if d.info.SpecFormat != datamodel.OAS3 {
		errors = append(errors, fmt.Errorf("unable to build openapi document, "+
			"supplied spec is a different version (%v). Try 'BuildV2Document()'", d.info.SpecFormat))
		return nil, errors
	}
	lowDoc, err := v3low.CreateDocument(d.info)
	if err != nil {
		return nil, err
	}
	highDoc := v3high.NewDocument(lowDoc)
	return &DocumentModel[v3high.Document]{
		Model: *highDoc,
	}, nil
}
