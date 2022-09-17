// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"fmt"
	low "github.com/pb33f/libopenapi/datamodel/low/v3"
)

// Responses represents a high-level OpenAPI 3+ Responses object that is backed by a low-level one.
//
// It's a container for the expected responses of an operation. The container maps a HTTP response code to the
// expected response.
//
// The specification is not necessarily expected to cover all possible HTTP response codes because they may not be
// known in advance. However, documentation is expected to cover a successful operation response and any known errors.
//
// The default MAY be used as a default response object for all HTTP codes that are not covered individually by
// the Responses Object.
//
// The Responses Object MUST contain at least one response code, and if only one response code is provided it SHOULD
// be the response for a successful operation call.
//  - https://spec.openapis.org/oas/v3.1.0#responses-object
type Responses struct {
	Codes   map[string]*Response
	Default *Response
	low     *low.Responses
}

// NewResponses will create a new high-level Responses instance from a low-level one. It operates asynchronously
// internally, as each response may be considerable in complexity.
func NewResponses(response *low.Responses) *Responses {
	r := new(Responses)
	r.low = response
	if !response.Default.IsEmpty() {
		r.Default = NewResponse(response.Default.Value)
	}
	codes := make(map[string]*Response)

	// struct to hold response and code sent over chan.
	type respRes struct {
		code string
		resp *Response
	}

	// build each response async for speed
	rChan := make(chan respRes)
	var buildResponse = func(code string, resp *low.Response, c chan respRes) {
		c <- respRes{code: code, resp: NewResponse(resp)}
	}
	for k, v := range response.Codes {
		go buildResponse(k.Value, v.Value, rChan)
	}
	totalCodes := len(response.Codes)
	codesParsed := 0
	for codesParsed < totalCodes {
		select {
		case re := <-rChan:
			codesParsed++
			codes[re.code] = re.resp
		}
	}
	r.Codes = codes
	return r
}

// FindResponseByCode is a shortcut for looking up code by an integer vs. a string
func (r *Responses) FindResponseByCode(code int) *Response {
	return r.Codes[fmt.Sprintf("%d", code)]
}

// GoLow returns the low-level Response object used to create the high-level one.
func (r *Responses) GoLow() *low.Responses {
	return r.low
}
