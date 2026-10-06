// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import "testing"

func TestParameterEffectiveSerialization(t *testing.T) {
	trueValue, falseValue := true, false
	tests := []struct {
		name    string
		param   *Parameter
		style   string
		explode bool
	}{
		{name: "query defaults to exploded form", param: &Parameter{In: "query"}, style: "form", explode: true},
		{name: "cookie defaults to exploded form", param: &Parameter{In: "cookie"}, style: "form", explode: true},
		{name: "path defaults to simple", param: &Parameter{In: "path"}, style: "simple", explode: false},
		{name: "header defaults to simple", param: &Parameter{In: "header"}, style: "simple", explode: false},
		{name: "explicit style controls default explode", param: &Parameter{In: "query", Style: "spaceDelimited"}, style: "spaceDelimited", explode: false},
		{name: "explicit explode true wins", param: &Parameter{In: "path", Explode: &trueValue}, style: "simple", explode: true},
		{name: "explicit explode false wins", param: &Parameter{In: "query", Explode: &falseValue}, style: "form", explode: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if style := test.param.EffectiveStyle(); style != test.style {
				t.Fatalf("EffectiveStyle() = %q, want %q", style, test.style)
			}
			if explode := test.param.EffectiveExplode(); explode != test.explode {
				t.Fatalf("EffectiveExplode() = %t, want %t", explode, test.explode)
			}
		})
	}
}
