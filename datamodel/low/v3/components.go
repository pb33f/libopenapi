// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"crypto/sha256"
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"sort"
	"strings"
)

// Components represents a low-level OpenAPI 3+ Components Object, that is backed by a low-level one.
//
// Holds a set of reusable objects for different aspects of the OAS. All objects defined within the components object
// will have no effect on the API unless they are explicitly referenced from properties outside the components object.
//  - https://spec.openapis.org/oas/v3.1.0#components-object
type Components struct {
	Schemas         low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*base.SchemaProxy]]
	Responses       low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Response]]
	Parameters      low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Parameter]]
	Examples        low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*base.Example]]
	RequestBodies   low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*RequestBody]]
	Headers         low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Header]]
	SecuritySchemes low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*SecurityScheme]]
	Links           low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Link]]
	Callbacks       low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Callback]]
	Extensions      map[low.KeyReference[string]]low.ValueReference[any]
	*low.Reference
}

// GetExtensions returns all Components extensions and satisfies the low.HasExtensions interface.
func (co *Components) GetExtensions() map[low.KeyReference[string]]low.ValueReference[any] {
	return co.Extensions
}

// Hash will return a consistent SHA256 Hash of the Encoding object
func (co *Components) Hash() [32]byte {
	var f []string
	generateHashForObjectMap(co.Schemas.Value, &f)
	generateHashForObjectMap(co.Responses.Value, &f)
	generateHashForObjectMap(co.Parameters.Value, &f)
	generateHashForObjectMap(co.Examples.Value, &f)
	generateHashForObjectMap(co.RequestBodies.Value, &f)
	generateHashForObjectMap(co.Headers.Value, &f)
	generateHashForObjectMap(co.SecuritySchemes.Value, &f)
	generateHashForObjectMap(co.Links.Value, &f)
	generateHashForObjectMap(co.Callbacks.Value, &f)
	keys := make([]string, len(co.Extensions))
	z := 0
	for k := range co.Extensions {
		keys[z] = fmt.Sprintf("%s-%x", k.Value, sha256.Sum256([]byte(fmt.Sprint(co.Extensions[k].Value))))
		z++
	}
	sort.Strings(keys)
	f = append(f, keys...)
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}

func generateHashForObjectMap[T any](collection map[low.KeyReference[string]]low.ValueReference[T], hash *[]string) {
	if collection == nil {
		return
	}
	l := make([]string, len(collection))
	keys := make(map[string]low.ValueReference[T])
	z := 0
	for k := range collection {
		keys[k.Value] = collection[k]
		l[z] = k.Value
		z++
	}
	sort.Strings(l)
	for k := range l {
		*hash = append(*hash, low.GenerateHashString(keys[l[k]].Value))
	}
}

// FindExtension attempts to locate an extension with the supplied key
func (co *Components) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, co.Extensions)
}

// FindSchema attempts to locate a SchemaProxy from 'schemas' with a specific name
func (co *Components) FindSchema(schema string) *low.ValueReference[*base.SchemaProxy] {
	return low.FindItemInMap[*base.SchemaProxy](schema, co.Schemas.Value)
}

// FindResponse attempts to locate a Response from 'responses' with a specific name
func (co *Components) FindResponse(response string) *low.ValueReference[*Response] {
	return low.FindItemInMap[*Response](response, co.Responses.Value)
}

// FindParameter attempts to locate a Parameter from 'parameters' with a specific name
func (co *Components) FindParameter(response string) *low.ValueReference[*Parameter] {
	return low.FindItemInMap[*Parameter](response, co.Parameters.Value)
}

// FindSecurityScheme attempts to locate a SecurityScheme from 'securitySchemes' with a specific name
func (co *Components) FindSecurityScheme(sScheme string) *low.ValueReference[*SecurityScheme] {
	return low.FindItemInMap[*SecurityScheme](sScheme, co.SecuritySchemes.Value)
}

// FindExample attempts tp
func (co *Components) FindExample(example string) *low.ValueReference[*base.Example] {
	return low.FindItemInMap[*base.Example](example, co.Examples.Value)
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
	co.Reference = new(low.Reference)
	co.Extensions = low.ExtractExtensions(root)

	// build out components asynchronously for speed. There could be some significant weight here.
	skipChan := make(chan bool)
	errorChan := make(chan error)
	paramChan := make(chan low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Parameter]])
	schemaChan := make(chan low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*base.SchemaProxy]])
	responsesChan := make(chan low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Response]])
	examplesChan := make(chan low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*base.Example]])
	requestBodiesChan := make(chan low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*RequestBody]])
	headersChan := make(chan low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Header]])
	securitySchemesChan := make(chan low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*SecurityScheme]])
	linkChan := make(chan low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Link]])
	callbackChan := make(chan low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Callback]])

	go extractComponentValues[*base.SchemaProxy](SchemasLabel, root, skipChan, errorChan, schemaChan, idx)
	go extractComponentValues[*Parameter](ParametersLabel, root, skipChan, errorChan, paramChan, idx)
	go extractComponentValues[*Response](ResponsesLabel, root, skipChan, errorChan, responsesChan, idx)
	go extractComponentValues[*base.Example](base.ExamplesLabel, root, skipChan, errorChan, examplesChan, idx)
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

type componentBuildResult[T any] struct {
	k low.KeyReference[string]
	v low.ValueReference[T]
}

func extractComponentValues[T low.Buildable[N], N any](label string, root *yaml.Node,
	skip chan bool, errorChan chan<- error, resultChan chan<- low.NodeReference[map[low.KeyReference[string]]low.ValueReference[T]], idx *index.SpecIndex) {
	_, nodeLabel, nodeValue := utils.FindKeyNodeFullTop(label, root.Content)
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
	var buildComponent = func(parentLabel string, label *yaml.Node, value *yaml.Node, c chan componentBuildResult[T], ec chan<- error) {
		var n T = new(N)

		// if this is a reference, extract it (although components with references is an antipattern)
		// If you're building components as references... pls... stop, this code should not need to be here.
		// TODO: check circular crazy on this. It may explode
		var err error
		if h, _, _ := utils.IsNodeRefValue(value); h && parentLabel != SchemasLabel {
			value, err = low.LocateRefNode(value, idx)
		}
		if err != nil {
			ec <- err
			return
		}

		// build.
		_ = low.BuildModel(value, n)
		err = n.Build(value, idx)
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
		// only check for lowercase extensions as 'X-' is still valid as a key (annoyingly).
		if strings.HasPrefix(currentLabel.Value, "x-") {
			continue
		}
		totalComponents++
		go buildComponent(label, currentLabel, v, bChan, eChan)
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
