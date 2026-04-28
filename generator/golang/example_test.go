// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"fmt"
	"reflect"
	"strings"

	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
)

func ExampleRenderSchema() {
	properties := orderedmap.New[string, *highbase.SchemaProxy]()
	properties.Set("id", highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}}))

	schema := highbase.CreateSchemaProxy(&highbase.Schema{
		Type:       []string{"object"},
		Required:   []string{"id"},
		Properties: properties,
	})
	source, err := RenderSchema("Pet", schema, WithOptionalFieldsAsPointers(false))
	if err != nil {
		panic(err)
	}

	fmt.Println(strings.Contains(string(source), "type Pet struct"))
	fmt.Println(strings.Contains(string(source), "ID string `json:\"id\"`"))

	// Output:
	// true
	// true
}

type ExampleBillingAddress struct {
	Line1 string `json:"line1"`
}

type ExampleCustomer struct {
	ID      string                `json:"id"`
	Address ExampleBillingAddress `json:"address"`
}

func ExampleGenerator_SchemasFromTypes() {
	generator := NewGenerator()
	set, err := generator.SchemasFromTypes(reflect.TypeOf(ExampleCustomer{}))
	if err != nil {
		panic(err)
	}

	_, hasCustomer := set.Components.Get("ExampleCustomer")
	_, hasAddress := set.Components.Get("ExampleBillingAddress")

	fmt.Println(set.Root.GetReference())
	fmt.Println(hasCustomer, hasAddress)

	// Output:
	// #/components/schemas/ExampleCustomer
	// true true
}
