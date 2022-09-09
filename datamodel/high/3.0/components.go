// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	low "github.com/pb33f/libopenapi/datamodel/low/3.0"
	"github.com/pb33f/libopenapi/datamodel/low/base"
)

const (
	responses = iota
	parameters
	examples
	requestBodies
	headers
	securitySchemes
	links
	callbacks
)

type Components struct {
	Schemas         map[string]*highbase.SchemaProxy
	Responses       map[string]*Response
	Parameters      map[string]*Parameter
	Examples        map[string]*highbase.Example
	RequestBodies   map[string]*RequestBody
	Headers         map[string]*Header
	SecuritySchemes map[string]*SecurityScheme
	Links           map[string]*Link
	Callbacks       map[string]*Callback
	Extensions      map[string]any
	low             *low.Components
}

func NewComponents(comp *low.Components) *Components {
	c := new(Components)
	c.low = comp
	if len(comp.Extensions) > 0 {
		c.Extensions = high.ExtractExtensions(comp.Extensions)
	}
	cbMap := make(map[string]*Callback)
	linkMap := make(map[string]*Link)
	responseMap := make(map[string]*Response)
	parameterMap := make(map[string]*Parameter)
	exampleMap := make(map[string]*highbase.Example)
	requestBodyMap := make(map[string]*RequestBody)
	headerMap := make(map[string]*Header)
	securitySchemeMap := make(map[string]*SecurityScheme)
	schemas := make(map[string]*highbase.SchemaProxy)
	schemaChan := make(chan componentResult[*highbase.SchemaProxy])
	cbChan := make(chan componentResult[*Callback])
	linkChan := make(chan componentResult[*Link])
	responseChan := make(chan componentResult[*Response])
	paramChan := make(chan componentResult[*Parameter])
	exampleChan := make(chan componentResult[*highbase.Example])
	requestBodyChan := make(chan componentResult[*RequestBody])
	headerChan := make(chan componentResult[*Header])
	securitySchemeChan := make(chan componentResult[*SecurityScheme])

	// build all components asynchronously.
	for k, v := range comp.Callbacks.Value {
		go buildComponent[*Callback, *low.Callback](callbacks, k.Value, v.Value, cbChan, NewCallback)
	}
	for k, v := range comp.Links.Value {
		go buildComponent[*Link, *low.Link](links, k.Value, v.Value, linkChan, NewLink)
	}
	for k, v := range comp.Responses.Value {
		go buildComponent[*Response, *low.Response](responses, k.Value, v.Value, responseChan, NewResponse)
	}
	for k, v := range comp.Parameters.Value {
		go buildComponent[*Parameter, *low.Parameter](parameters, k.Value, v.Value, paramChan, NewParameter)
	}
	for k, v := range comp.Examples.Value {
		go buildComponent[*highbase.Example, *base.Example](parameters, k.Value, v.Value, exampleChan, highbase.NewExample)
	}
	for k, v := range comp.RequestBodies.Value {
		go buildComponent[*RequestBody, *low.RequestBody](requestBodies, k.Value, v.Value,
			requestBodyChan, NewRequestBody)
	}
	for k, v := range comp.Headers.Value {
		go buildComponent[*Header, *low.Header](headers, k.Value, v.Value, headerChan, NewHeader)
	}
	for k, v := range comp.SecuritySchemes.Value {
		go buildComponent[*SecurityScheme, *low.SecurityScheme](securitySchemes, k.Value, v.Value,
			securitySchemeChan, NewSecurityScheme)
	}
	for k, v := range comp.Schemas.Value {
		go buildSchema(k, v, schemaChan)
	}

	totalComponents := len(comp.Callbacks.Value) + len(comp.Links.Value) + len(comp.Responses.Value) +
		len(comp.Parameters.Value) + len(comp.Examples.Value) + len(comp.RequestBodies.Value) +
		len(comp.Headers.Value) + len(comp.SecuritySchemes.Value) + len(comp.Schemas.Value)

	processedComponents := 0
	for processedComponents < totalComponents {
		select {
		case sRes := <-schemaChan:
			processedComponents++
			schemas[sRes.key] = sRes.res
		case cbRes := <-cbChan:
			processedComponents++
			cbMap[cbRes.key] = cbRes.res
		case lRes := <-linkChan:
			processedComponents++
			linkMap[lRes.key] = lRes.res
		case respRes := <-responseChan:
			processedComponents++
			responseMap[respRes.key] = respRes.res
		case pRes := <-paramChan:
			processedComponents++
			parameterMap[pRes.key] = pRes.res
		case eRes := <-exampleChan:
			processedComponents++
			exampleMap[eRes.key] = eRes.res
		case rbRes := <-requestBodyChan:
			processedComponents++
			requestBodyMap[rbRes.key] = rbRes.res
		case hRes := <-headerChan:
			processedComponents++
			headerMap[hRes.key] = hRes.res
		case ssRes := <-securitySchemeChan:
			processedComponents++
			securitySchemeMap[ssRes.key] = ssRes.res
		}
	}
	c.Schemas = schemas
	c.Callbacks = cbMap
	c.Links = linkMap
	c.Parameters = parameterMap
	c.Headers = headerMap
	c.Responses = responseMap
	c.RequestBodies = requestBodyMap
	c.Examples = exampleMap
	c.SecuritySchemes = securitySchemeMap
	return c
}

type componentResult[T any] struct {
	res  T
	key  string
	comp int
}

func buildComponent[N any, O any](comp int, key string, orig O, c chan componentResult[N], f func(O) N) {
	c <- componentResult[N]{comp: comp, res: f(orig), key: key}
}

func buildSchema(key lowmodel.KeyReference[string], orig lowmodel.ValueReference[*base.SchemaProxy], c chan componentResult[*highbase.SchemaProxy]) {
	var sch *highbase.SchemaProxy
	sch = highbase.NewSchemaProxy(&lowmodel.NodeReference[*base.SchemaProxy]{
		Value:     orig.Value,
		ValueNode: orig.ValueNode,
	})
	c <- componentResult[*highbase.SchemaProxy]{res: sch, key: key.Value}
}

func (c *Components) GoLow() *low.Components {
	return c.low
}
