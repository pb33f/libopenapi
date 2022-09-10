// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import low "github.com/pb33f/libopenapi/datamodel/low/2.0"

type ParameterDefinitions struct {
	Definitions map[string]*Parameter
	low         *low.ParameterDefinitions
}

func NewParametersDefinitions(parametersDefinitions *low.ParameterDefinitions) *ParameterDefinitions {
	pd := new(ParameterDefinitions)
	pd.low = parametersDefinitions
	params := make(map[string]*Parameter)
	var buildParam = func(name string, param *low.Parameter, rChan chan<- asyncResult[*Parameter]) {
		rChan <- asyncResult[*Parameter]{
			key:    name,
			result: NewParameter(param),
		}
	}
	resChan := make(chan asyncResult[*Parameter])
	for k := range parametersDefinitions.Definitions {
		go buildParam(k.Value, parametersDefinitions.Definitions[k].Value, resChan)
	}
	totalParams := len(parametersDefinitions.Definitions)
	completedParams := 0
	for completedParams < totalParams {
		select {
		case r := <-resChan:
			completedParams++
			params[r.key] = r.result
		}
	}
	pd.Definitions = params
	return pd
}

func (p *ParameterDefinitions) GoLow() *low.ParameterDefinitions {
	return p.low
}
