// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/v2"
)

// Responses is a high-level representation of a Swagger / OpenAPI 2 Responses object, backed by a low level one.
type Responses struct {
	Codes      map[string]*Response
	Default    *Response
	Extensions map[string]any
	low        *low.Responses
}

// NewResponses will create a new high-level instance of Responses from a low-level one.
func NewResponses(responses *low.Responses) *Responses {
	r := new(Responses)
	r.low = responses
	r.Extensions = high.ExtractExtensions(responses.Extensions)

	// async function.
	var buildPath = func(code string, pi *low.Response, rChan chan<- asyncResult[*Response]) {
		rChan <- asyncResult[*Response]{
			key:    code,
			result: NewResponse(pi),
		}
	}

	if !responses.Default.IsEmpty() {
		r.Default = NewResponse(responses.Default.Value)
	}

	// run everything async. lots of responses with lots of data are possible.
	if len(responses.Codes) > 0 {
		resultChan := make(chan asyncResult[*Response])
		for k := range responses.Codes {
			go buildPath(k.Value, responses.Codes[k].Value, resultChan)
		}
		resp := make(map[string]*Response)
		totalResponses := len(responses.Codes)
		completedResponses := 0
		for completedResponses < totalResponses {
			select {
			case res := <-resultChan:
				completedResponses++
				resp[res.key] = res.result
			}
		}
		r.Codes = resp
	}
	return r
}

// GoLow will return the low-level object used to create the high-level one.
func (r *Responses) GoLow() *low.Responses {
	return r.low
}
