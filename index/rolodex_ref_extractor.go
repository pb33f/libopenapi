// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"fmt"
	"regexp"
	"strings"
)

// var refRegex = regexp.MustCompile(`['"]?\$ref['"]?\s*:\s*['"]?([^'"]*?)['"]`)
var refRegex = regexp.MustCompile(`('\$ref'|"\$ref"|\$ref)\s*:\s*('[^']*'|"[^"]*"|\S*)`)

type RefType int

const (
	Local RefType = iota
	File
	HTTP
)

type ExtractedRef struct {
	Location string
	Type     RefType
}

func (r *ExtractedRef) GetFile() string {
	switch r.Type {
	case File, HTTP:
		location := strings.Split(r.Location, "#/")
		return location[0]
	default:
		return r.Location
	}
}

func (r *ExtractedRef) GetReference() string {
	switch r.Type {
	case File, HTTP:
		location := strings.Split(r.Location, "#/")
		return fmt.Sprintf("#/%s", location[1])
	default:
		return r.Location
	}
}

func ExtractFileType(ref string) FileExtension {
	if strings.HasSuffix(ref, ".yaml") {
		return YAML
	}
	if strings.HasSuffix(ref, ".yml") {
		return YAML
	}
	if strings.HasSuffix(ref, ".json") {
		return JSON
	}
	return UNSUPPORTED
}
