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
	for k := range parametersDefinitions.Definitions {
		params[k.Value] = NewParameter(parametersDefinitions.Definitions[k].Value)
	}
	pd.Definitions = params
	return pd
}
