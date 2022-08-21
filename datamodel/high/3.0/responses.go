// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import low "github.com/pb33f/libopenapi/datamodel/low/3.0"

type Responses struct {
	Codes   map[string]*Response
	Default *Response
	low     *low.Responses
}

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

func (r *Responses) GoLow() *low.Responses {
	return r.low
}
