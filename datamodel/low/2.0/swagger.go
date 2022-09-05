// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/resolver"
	"gopkg.in/yaml.v3"
)

type documentFunction func(root *yaml.Node, doc *Swagger, idx *index.SpecIndex, c chan<- bool, e chan<- error)

type Swagger struct {
	Swagger             low.ValueReference[string]
	Info                low.NodeReference[*base.Info]
	Host                low.NodeReference[string]
	BasePath            low.NodeReference[string]
	Schemes             low.NodeReference[[]low.ValueReference[string]]
	Consumes            low.NodeReference[[]low.ValueReference[string]]
	Produces            low.NodeReference[[]low.ValueReference[string]]
	Paths               low.NodeReference[*Paths]
	Definitions         low.NodeReference[*Definitions]
	SecurityDefinitions low.NodeReference[*SecurityDefinitions]
	Parameters          low.NodeReference[*ParameterDefinitions]
	Responses           low.NodeReference[*ResponsesDefinitions]
	Security            low.NodeReference[[]low.ValueReference[*SecurityRequirement]]
	Tags                low.NodeReference[[]low.ValueReference[*base.Tag]]
	ExternalDocs        low.NodeReference[*base.ExternalDoc]
	Extensions          map[low.KeyReference[string]]low.ValueReference[any]
	Index               *index.SpecIndex
	SpecInfo            *datamodel.SpecInfo
}

func (s *Swagger) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, s.Extensions)
}

func CreateDocument(info *datamodel.SpecInfo) (*Swagger, []error) {

	doc := Swagger{Swagger: low.ValueReference[string]{Value: info.Version, ValueNode: info.RootNode}}
	doc.Extensions = low.ExtractExtensions(info.RootNode.Content[0])

	// build an index
	idx := index.NewSpecIndex(info.RootNode)
	doc.Index = idx
	doc.SpecInfo = info

	var errors []error

	// build out swagger scalar variables.
	bErr := low.BuildModel(info.RootNode, &doc)
	if bErr != nil {
		errors = append(errors, bErr)
	}

	// extract externalDocs
	extDocs, err := low.ExtractObject[*base.ExternalDoc](base.ExternalDocsLabel, info.RootNode, idx)
	if err != nil {
		errors = append(errors, err)
	}

	doc.ExternalDocs = extDocs

	// create resolver and check for circular references.
	resolve := resolver.NewResolver(idx)
	_ = resolve.CheckForCircularReferences()

	extractionFuncs := []documentFunction{
		extractInfo,
		extractPaths,
		extractDefinitions,
		extractParamDefinitions,
		extractResponsesDefinitions,
		extractSecurityDefinitions,
		extractTags,
		extractSecurity,
	}
	doneChan := make(chan bool)
	errChan := make(chan error)
	for i := range extractionFuncs {
		go extractionFuncs[i](info.RootNode, &doc, idx, doneChan, errChan)
	}
	completedExtractions := 0
	for completedExtractions < len(extractionFuncs) {
		select {
		case <-doneChan:
			completedExtractions++
		case e := <-errChan:
			completedExtractions++
			errors = append(errors, e)
		}
	}

	return &doc, errors
}

func extractInfo(root *yaml.Node, doc *Swagger, idx *index.SpecIndex, c chan<- bool, e chan<- error) {
	info, err := low.ExtractObject[*base.Info](base.InfoLabel, root, idx)
	if err != nil {
		e <- err
		return
	}
	doc.Info = info
	c <- true
}

func extractPaths(root *yaml.Node, doc *Swagger, idx *index.SpecIndex, c chan<- bool, e chan<- error) {
	paths, err := low.ExtractObject[*Paths](PathsLabel, root, idx)
	if err != nil {
		e <- err
		return
	}
	doc.Paths = paths
	c <- true
}
func extractDefinitions(root *yaml.Node, doc *Swagger, idx *index.SpecIndex, c chan<- bool, e chan<- error) {
	def, err := low.ExtractObject[*Definitions](DefinitionsLabel, root, idx)
	if err != nil {
		e <- err
		return
	}
	doc.Definitions = def
	c <- true
}
func extractParamDefinitions(root *yaml.Node, doc *Swagger, idx *index.SpecIndex, c chan<- bool, e chan<- error) {
	param, err := low.ExtractObject[*ParameterDefinitions](ParametersLabel, root, idx)
	if err != nil {
		e <- err
		return
	}
	doc.Parameters = param
	c <- true
}

func extractResponsesDefinitions(root *yaml.Node, doc *Swagger, idx *index.SpecIndex, c chan<- bool, e chan<- error) {
	resp, err := low.ExtractObject[*ResponsesDefinitions](ResponsesLabel, root, idx)
	if err != nil {
		e <- err
		return
	}
	doc.Responses = resp
	c <- true
}

func extractSecurityDefinitions(root *yaml.Node, doc *Swagger, idx *index.SpecIndex, c chan<- bool, e chan<- error) {
	sec, err := low.ExtractObject[*SecurityDefinitions](SecurityDefinitionsLabel, root, idx)
	if err != nil {
		e <- err
		return
	}
	doc.SecurityDefinitions = sec
	c <- true
}

func extractTags(root *yaml.Node, doc *Swagger, idx *index.SpecIndex, c chan<- bool, e chan<- error) {
	tags, ln, vn, err := low.ExtractArray[*base.Tag](base.TagsLabel, root, idx)
	if err != nil {
		e <- err
		return
	}
	doc.Tags = low.NodeReference[[]low.ValueReference[*base.Tag]]{
		Value:     tags,
		KeyNode:   ln,
		ValueNode: vn,
	}
	c <- true
}

func extractSecurity(root *yaml.Node, doc *Swagger, idx *index.SpecIndex, c chan<- bool, e chan<- error) {
	sec, ln, vn, err := low.ExtractArray[*SecurityRequirement](SecurityLabel, root, idx)
	if err != nil {
		e <- err
		return
	}
	doc.Security = low.NodeReference[[]low.ValueReference[*SecurityRequirement]]{
		Value:     sec,
		KeyNode:   ln,
		ValueNode: vn,
	}
	c <- true
}
