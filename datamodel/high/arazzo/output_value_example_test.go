// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo_test

import (
	"fmt"

	arazzo "github.com/pb33f/libopenapi/datamodel/high/arazzo"
	"github.com/pb33f/libopenapi/orderedmap"
)

func ExampleOutputValue() {
	outputs := orderedmap.New[string, *arazzo.OutputValue]()
	outputs.Set("createdPetId", arazzo.NewExpressionOutputValue("$response.body#/id"))
	outputs.Set("email", arazzo.NewSelectorOutputValue(&arazzo.Selector{
		Context:  "$response.body",
		Selector: "$.profile.email",
		Type:     "jsonpath",
	}))

	expressionOutput, _ := outputs.Get("createdPetId")
	expression, _ := expressionOutput.GetExpression()
	selectorOutput, _ := outputs.Get("email")
	selector, _ := selectorOutput.GetSelector()

	fmt.Println(expression)
	fmt.Println(selector.Selector)
	// Output:
	// $response.body#/id
	// $.profile.email
}
