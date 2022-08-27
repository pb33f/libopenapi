// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"strings"
	"sync"
)

const (
	ComponentsLabel = "components"
	SchemasLabel    = "schemas"
)

var seenSchemas map[string]*Schema

func init() {
	clearSchemas()
}

func clearSchemas() {
	seenSchemas = make(map[string]*Schema)
}

var seenSchemaLock sync.RWMutex

func addSeenSchema(key string, schema *Schema) {
	defer seenSchemaLock.Unlock()
	seenSchemaLock.Lock()
	if seenSchemas[key] == nil {
		seenSchemas[key] = schema
	}
}
func getSeenSchema(key string) *Schema {
	defer seenSchemaLock.Unlock()
	seenSchemaLock.Lock()
	return seenSchemas[key]
}

type Components struct {
	Schemas         low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*SchemaProxy]]
	Responses       low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Response]]
	Parameters      low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Parameter]]
	Examples        low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Example]]
	RequestBodies   low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*RequestBody]]
	Headers         low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Header]]
	SecuritySchemes low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*SecurityScheme]]
	Links           low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Link]]
	Callbacks       low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Callback]]
	Extensions      map[low.KeyReference[string]]low.ValueReference[any]
}

func (co *Components) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, co.Extensions)
}

func (co *Components) FindSchema(schema string) *low.ValueReference[*SchemaProxy] {
	return low.FindItemInMap[*SchemaProxy](schema, co.Schemas.Value)
}

func (co *Components) FindResponse(response string) *low.ValueReference[*Response] {
	return low.FindItemInMap[*Response](response, co.Responses.Value)
}

func (co *Components) FindParameter(response string) *low.ValueReference[*Parameter] {
	return low.FindItemInMap[*Parameter](response, co.Parameters.Value)
}

func (co *Components) FindSecurityScheme(sScheme string) *low.ValueReference[*SecurityScheme] {
	return low.FindItemInMap[*SecurityScheme](sScheme, co.SecuritySchemes.Value)
}

func (co *Components) FindExample(example string) *low.ValueReference[*Example] {
	return low.FindItemInMap[*Example](example, co.Examples.Value)
}

func (co *Components) FindRequestBody(requestBody string) *low.ValueReference[*RequestBody] {
	return low.FindItemInMap[*RequestBody](requestBody, co.RequestBodies.Value)
}

func (co *Components) FindHeader(header string) *low.ValueReference[*Header] {
	return low.FindItemInMap[*Header](header, co.Headers.Value)
}

func (co *Components) FindLink(link string) *low.ValueReference[*Link] {
	return low.FindItemInMap[*Link](link, co.Links.Value)
}

func (co *Components) FindCallback(callback string) *low.ValueReference[*Callback] {
	return low.FindItemInMap[*Callback](callback, co.Callbacks.Value)
}

func (co *Components) Build(root *yaml.Node, idx *index.SpecIndex) error {
	co.Extensions = low.ExtractExtensions(root)

	// build out components asynchronously for speed. There could be some significant weight here.
	skipChan := make(chan bool)
	errorChan := make(chan error)
	paramChan := make(chan low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Parameter]])
	schemaChan := make(chan low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*SchemaProxy]])
	responsesChan := make(chan low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Response]])
	examplesChan := make(chan low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Example]])
	requestBodiesChan := make(chan low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*RequestBody]])
	headersChan := make(chan low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Header]])
	securitySchemesChan := make(chan low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*SecurityScheme]])
	linkChan := make(chan low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Link]])
	callbackChan := make(chan low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Callback]])

	go extractComponentValues[*SchemaProxy](SchemasLabel, root, skipChan, errorChan, schemaChan, idx)
	go extractComponentValues[*Parameter](ParametersLabel, root, skipChan, errorChan, paramChan, idx)
	go extractComponentValues[*Response](ResponsesLabel, root, skipChan, errorChan, responsesChan, idx)
	go extractComponentValues[*Example](ExamplesLabel, root, skipChan, errorChan, examplesChan, idx)
	go extractComponentValues[*RequestBody](RequestBodiesLabel, root, skipChan, errorChan, requestBodiesChan, idx)
	go extractComponentValues[*Header](HeadersLabel, root, skipChan, errorChan, headersChan, idx)
	go extractComponentValues[*SecurityScheme](SecuritySchemesLabel, root, skipChan, errorChan, securitySchemesChan, idx)
	go extractComponentValues[*Link](LinksLabel, root, skipChan, errorChan, linkChan, idx)
	go extractComponentValues[*Callback](CallbacksLabel, root, skipChan, errorChan, callbackChan, idx)

	n := 0
	total := 9

	for n < total {
		select {
		case buildError := <-errorChan:
			return buildError
		case <-skipChan:
			n++
		case params := <-paramChan:
			co.Parameters = params
			n++
		case schemas := <-schemaChan:
			co.Schemas = schemas
			n++
		case responses := <-responsesChan:
			co.Responses = responses
			n++
		case examples := <-examplesChan:
			co.Examples = examples
			n++
		case reqBody := <-requestBodiesChan:
			co.RequestBodies = reqBody
			n++
		case headers := <-headersChan:
			co.Headers = headers
			n++
		case sScheme := <-securitySchemesChan:
			co.SecuritySchemes = sScheme
			n++
		case links := <-linkChan:
			co.Links = links
			n++
		case callbacks := <-callbackChan:
			co.Callbacks = callbacks
			n++
		}
	}
	return nil
}

func cacheSchemas(sch map[low.KeyReference[string]]low.ValueReference[*Schema]) {
	for _, v := range sch {
		addSeenSchema(v.GenerateMapKey(), v.Value)
	}
}

type componentBuildResult[T any] struct {
	k low.KeyReference[string]
	v low.ValueReference[T]
}

func extractComponentValues[T low.Buildable[N], N any](label string, root *yaml.Node,
	skip chan bool, errorChan chan<- error, resultChan chan<- low.NodeReference[map[low.KeyReference[string]]low.ValueReference[T]], idx *index.SpecIndex) {
	_, nodeLabel, nodeValue := utils.FindKeyNodeFull(label, root.Content)
	if nodeValue == nil {
		skip <- true
		return
	}
	var currentLabel *yaml.Node
	componentValues := make(map[low.KeyReference[string]]low.ValueReference[T])
	if utils.IsNodeArray(nodeValue) {
		errorChan <- fmt.Errorf("node is array, cannot be used in components: line %d, column %d", nodeValue.Line, nodeValue.Column)
		return
	}

	// for every component, build in a new thread!
	bChan := make(chan componentBuildResult[T])
	eChan := make(chan error)
	var buildComponent = func(label *yaml.Node, value *yaml.Node, c chan componentBuildResult[T], ec chan<- error) {
		var n T = new(N)
		_ = low.BuildModel(value, n)
		err := n.Build(value, idx)
		if err != nil {
			ec <- err
			return
		}
		c <- componentBuildResult[T]{
			k: low.KeyReference[string]{
				KeyNode: label,
				Value:   label.Value,
			},
			v: low.ValueReference[T]{
				Value:     n,
				ValueNode: value,
			},
		}
	}
	totalComponents := 0
	for i, v := range nodeValue.Content {
		// always ignore extensions
		if i%2 == 0 {
			currentLabel = v
			continue
		}
		if strings.HasPrefix(strings.ToLower(currentLabel.Value), "x-") {
			continue
		}
		totalComponents++
		go buildComponent(currentLabel, v, bChan, eChan)
	}

	completedComponents := 0
	for completedComponents < totalComponents {
		select {
		case e := <-eChan:
			errorChan <- e
		case r := <-bChan:
			componentValues[r.k] = r.v
			completedComponents++
		}
	}

	results := low.NodeReference[map[low.KeyReference[string]]low.ValueReference[T]]{
		KeyNode:   nodeLabel,
		ValueNode: nodeValue,
		Value:     componentValues,
	}
	resultChan <- results
}
