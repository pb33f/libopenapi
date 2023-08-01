// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

// Components represents a low-level OpenAPI 3+ Components Object, that is backed by a low-level one.
//
// Holds a set of reusable objects for different aspects of the OAS. All objects defined within the components object
// will have no effect on the API unless they are explicitly referenced from properties outside the components object.
//   - https://spec.openapis.org/oas/v3.1.0#components-object
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

// Build converts root YAML node containing components to low level model.
// Process each component in parallel.
func (co *Components) Build(root *yaml.Node, idx *index.SpecIndex) error {
	root = utils.NodeAlias(root)
	utils.CheckForMergeNodes(root)
	co.Reference = new(low.Reference)
	co.Extensions = low.ExtractExtensions(root)

	var reterr error
	var ceMutex sync.Mutex
	var wg sync.WaitGroup
	wg.Add(9)

	captureError := func(err error) {
		ceMutex.Lock()
		defer ceMutex.Unlock()
		if err != nil {
			reterr = err
		}
	}

	go func() {
		schemas, err := extractComponentValues[*base.SchemaProxy](SchemasLabel, root, idx)
		captureError(err)
		co.Schemas = schemas
		wg.Done()
	}()
	go func() {
		parameters, err := extractComponentValues[*Parameter](ParametersLabel, root, idx)
		captureError(err)
		co.Parameters = parameters
		wg.Done()
	}()
	go func() {
		responses, err := extractComponentValues[*Response](ResponsesLabel, root, idx)
		captureError(err)
		co.Responses = responses
		wg.Done()
	}()
	go func() {
		examples, err := extractComponentValues[*base.Example](base.ExamplesLabel, root, idx)
		captureError(err)
		co.Examples = examples
		wg.Done()
	}()
	go func() {
		requestBodies, err := extractComponentValues[*RequestBody](RequestBodiesLabel, root, idx)
		captureError(err)
		co.RequestBodies = requestBodies
		wg.Done()
	}()
	go func() {
		headers, err := extractComponentValues[*Header](HeadersLabel, root, idx)
		captureError(err)
		co.Headers = headers
		wg.Done()
	}()
	go func() {
		securitySchemes, err := extractComponentValues[*SecurityScheme](SecuritySchemesLabel, root, idx)
		captureError(err)
		co.SecuritySchemes = securitySchemes
		wg.Done()
	}()
	go func() {
		links, err := extractComponentValues[*Link](LinksLabel, root, idx)
		captureError(err)
		co.Links = links
		wg.Done()
	}()
	go func() {
		callbacks, err := extractComponentValues[*Callback](CallbacksLabel, root, idx)
		captureError(err)
		co.Callbacks = callbacks
		wg.Done()
	}()

	wg.Wait()
	return reterr
}

type componentBuildResult[T any] struct {
	k low.KeyReference[string]
	v low.ValueReference[T]
}

// extractComponentValues converts all the YAML nodes of a component type to
// low level model.
// Process each node in parallel.
func extractComponentValues[T low.Buildable[N], N any](label string, root *yaml.Node, idx *index.SpecIndex) (retval low.NodeReference[map[low.KeyReference[string]]low.ValueReference[T]], _ error) {
	_, nodeLabel, nodeValue := utils.FindKeyNodeFullTop(label, root.Content)
	if nodeValue == nil {
		return retval, nil
	}
	componentValues := make(map[low.KeyReference[string]]low.ValueReference[T])
	if utils.IsNodeArray(nodeValue) {
		return retval, fmt.Errorf("node is array, cannot be used in components: line %d, column %d", nodeValue.Line, nodeValue.Column)
	}

	type inputValue struct {
		node         *yaml.Node
		currentLabel *yaml.Node
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	in := make(chan inputValue)
	out := make(chan componentBuildResult[T])
	var wg sync.WaitGroup
	wg.Add(2) // input and output goroutines.

	// Send input.
	go func() {
		defer wg.Done()
		var currentLabel *yaml.Node
		for i, node := range nodeValue.Content {
			// always ignore extensions
			if i%2 == 0 {
				currentLabel = node
				continue
			}
			// only check for lowercase extensions as 'X-' is still valid as a key (annoyingly).
			if strings.HasPrefix(currentLabel.Value, "x-") {
				continue
			}

			select {
			case in <- inputValue{
				node:         node,
				currentLabel: currentLabel,
			}:
			case <-ctx.Done():
				return
			}
		}
		close(in)
	}()

	// Collect output.
	go func() {
		for result := range out {
			componentValues[result.k] = result.v
		}
		cancel()
		wg.Done()
	}()

	// Translate.
	translateFunc := func(value inputValue) (retval componentBuildResult[T], _ error) {
		var n T = new(N)
		currentLabel := value.currentLabel
		node := value.node

		// if this is a reference, extract it (although components with references is an antipattern)
		// If you're building components as references... pls... stop, this code should not need to be here.
		// TODO: check circular crazy on this. It may explode
		var err error
		if h, _, _ := utils.IsNodeRefValue(node); h && label != SchemasLabel {
			node, err = low.LocateRefNode(node, idx)
		}
		if err != nil {
			return retval, err
		}

		// build.
		_ = low.BuildModel(node, n)
		err = n.Build(currentLabel, node, idx)
		if err != nil {
			return retval, err
		}
		return componentBuildResult[T]{
			k: low.KeyReference[string]{
				KeyNode: currentLabel,
				Value:   currentLabel.Value,
			},
			v: low.ValueReference[T]{
				Value:     n,
				ValueNode: node,
			},
		}, nil
	}
	err := datamodel.TranslatePipeline[inputValue, componentBuildResult[T]](in, out, translateFunc)
	wg.Wait()
	if err != nil {
		return retval, err
	}

	results := low.NodeReference[map[low.KeyReference[string]]low.ValueReference[T]]{
		KeyNode:   nodeLabel,
		ValueNode: nodeValue,
		Value:     componentValues,
	}
	return results, nil
}
