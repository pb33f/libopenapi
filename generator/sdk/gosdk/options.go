// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package gosdk

import (
	modelgen "github.com/pb33f/libopenapi/generator/golang"
	"github.com/pb33f/libopenapi/generator/sdk"
)

// Options configures Go SDK generation.
type Options struct {
	PackageName string
	Prepare     sdk.PrepareOptions
	Workflows   []*sdk.Workflow
	Models      []modelgen.Option
}

func (options Options) packageName() string {
	if options.PackageName == "" {
		return "client"
	}
	return options.PackageName
}
