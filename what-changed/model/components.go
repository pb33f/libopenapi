// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

type ComponentsChanges struct {
	PropertyChanges
	SchemaChanges         map[string]*SchemaChanges
	ResponsesChanges      map[string]*SchemaChanges
	ParameterChanges      map[string]*ParameterChanges
	ExamplesChanges       map[string]*ExamplesChanges
	RequestBodyChanges    map[string]*RequestBodyChanges
	HeaderChanges         map[string]*HeaderChanges
	SecuritySchemeChanges map[string]*SecuritySchemeChanges
	LinkChanges           map[string]*LinkChanges
	// todo:
	//CallbackChanges map[string]*CallbackChanges
	ExtensionChanges *ExtensionChanges
}
