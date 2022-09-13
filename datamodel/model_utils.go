// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package datamodel

import (
	_ "embed"
)

// Constants used by utilities to determine the version of OpenAPI that we're referring to.
const (
	OAS2  = "oas2"
	OAS3  = "oas3"
	OAS31 = "oas3_1"
)

// OpenAPI3SchemaData is an embedded version of the OpenAPI 3 Schema
//go:embed schemas/oas3-schema.json
var OpenAPI3SchemaData string // embedded OAS3 schema

// OpenAPI2SchemaData is an embedded version of the OpenAPI 2 (Swagger) Schema
//go:embed schemas/swagger2-schema.json
var OpenAPI2SchemaData string // embedded OAS3 schema

// OAS3_1Format defines documents that can only be version 3.1
var OAS3_1Format = []string{OAS31}

// OAS3Format defines documents that can only be version 3.0
var OAS3Format = []string{OAS3}

// OAS3AllFormat defines documents that compose all 3+ versions
var OAS3AllFormat = []string{OAS3, OAS31}

// OAS2Format defines documents that compose swagger documnets (version 2.0)
var OAS2Format = []string{OAS2}

// AllFormats defines all versions of OpenAPI
var AllFormats = []string{OAS3, OAS31, OAS2}
