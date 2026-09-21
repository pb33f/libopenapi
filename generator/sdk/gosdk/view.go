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
	MethodName       string
	ParamsType       string
	Summary          string
	HTTPMethod       string
	Path             string
	Parameters       []parameterView
	HasParameters    bool
	HasQueryParams   bool
	Body             *bodyView
	ResponseType     string
	SuccessStatuses  []string
	SuccessCondition string
	ErrorResponses   []errorResponseView
	Security         [][]securityRequirementView
}

type securityRequirementView struct {
	Name   string
	Scopes []string
}

type workflowView struct {
	ID               string
	MethodName       string
	InputType        string
	Summary          string
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

type bodyView struct {
	Type        string
	FieldType   string
	Required    bool
	ContentType string
	Description string
}

type errorResponseView struct {
	Status    string
	Type      string
	Condition string
}
