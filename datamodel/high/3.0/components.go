// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	low "github.com/pb33f/libopenapi/datamodel/low/3.0"
	"sync"
)

var seenSchemas map[string]*Schema

func init() {
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
	Schemas         map[string]*Schema
	Responses       map[string]*Response
	Parameters      map[string]*Parameter
	Examples        map[string]*Example
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
	c.Extensions = high.ExtractExtensions(comp.Extensions)
	callbacks := make(map[string]*Callback)
	links := make(map[string]*Link)
	schemas := make(map[string]*Schema)

	for k, v := range comp.Callbacks.Value {
		callbacks[k.Value] = NewCallback(v.Value)
	}
	c.Callbacks = callbacks
	for k, v := range comp.Links.Value {
		links[k.Value] = NewLink(v.Value)
	}
	c.Links = links

	sLock := sync.RWMutex{}
	buildOutSchema := func(k lowmodel.KeyReference[string],
		schema lowmodel.ValueReference[*low.Schema], doneChan chan bool, schemas map[string]*Schema) {
		var sch *Schema
		if ss := getSeenSchema(schema.GenerateMapKey()); ss != nil {
			sch = ss
		} else {
			sch = NewSchema(schema.Value)
		}
		defer sLock.Unlock()
		sLock.Lock()
		schemas[k.Value] = sch
		addSeenSchema(schema.GenerateMapKey(), sch)
		doneChan <- true
	}

	doneChan := make(chan bool)
	for k, v := range comp.Schemas.Value {
		go buildOutSchema(k, v, doneChan, schemas)
	}
	k := 0
	for k < len(comp.Schemas.Value) {
		select {
		case <-doneChan:
			k++
		}
	}
	c.Schemas = schemas
	return c
}

func (c *Components) GoLow() *low.Components {
	return c.low
}
