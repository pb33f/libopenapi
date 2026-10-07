// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package gosdk

type clientView struct {
	PackageName     string
	DefaultServer   string
	SecuritySchemes []securitySchemeView
	Resources       []resourceView
	Workflows       []workflowView
	HasQueryParams  bool
	// Uses names the runtime helpers that generated code calls, so the client
	// declares only those.
	Uses map[string]bool
}

type securitySchemeView struct {
	Name          string
	Type          string
	In            string
	ParameterName string
	Scheme        string
}

type resourceView struct {
	Name       string
	TypeName   string
	Operations []operationView
}

type operationView struct {
	ID               string
	ResourceName     string
	ResourceType     string
	MethodName       string
	Comment          string
	ParamsType       string
	HTTPMethod       string
	Path             string
	Parameters       []parameterView
	Fields           []fieldView
	HasParameters    bool
	HasQueryParams   bool
	HasPathParams    bool
	Body             *bodyView
	ResponseType     string
	SuccessStatuses  []string
	SuccessCondition string
	DecodeCondition  string
	DecodeAll        bool
	ErrorCases       []errorResponseView
	ErrorSwitch      string
	ErrorDoc         []string
	Security         [][]securityRequirementView
}

type securityRequirementView struct {
	Name   string
	Scopes []string
}

type workflowView struct {
	ID               string
	MethodName       string
	Comment          string
	InputType        string
	ResourceName     string
	OperationMethod  string
	OperationParams  string
	HasParameters    bool
	ParameterFields  []workflowAssignmentView
	BodyType         string
	BodyFields       []workflowAssignmentView
	ResponseType     string
	SuccessCondition string
}

type workflowAssignmentView struct {
	Target        string
	Source        string
	Input         string
	ExpectedType  string
	ExpectedModel string
	ExpectedField string
}

type parameterView struct {
	Name        string
	FieldName   string
	Type        string
	In          string
	Required    bool
	Explode     bool
	Description string
	Encoder     string
	Array       bool
}

// fieldView is one field of an operation's parameters struct. Comment holds
// its rendered doc comment, and Spaced puts a blank line before it.
type fieldView struct {
	Name    string
	Type    string
	Doc     string
	Comment string
	Spaced  bool
}

type bodyView struct {
	Type        string
	FieldType   string
	Required    bool
	ContentType string
	Description string
}

// errorResponseView is one case of the switch that decodes typed error
// responses. Case is its condition; Default marks the default response.
type errorResponseView struct {
	Statuses []string
	Type     string
	Case     string
	Default  bool
}
