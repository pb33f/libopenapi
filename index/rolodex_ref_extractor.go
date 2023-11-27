// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"fmt"
	"strings"
)

const (
	Local RefType = iota
	File
	HTTP
)

type RefType int

type ExtractedRef struct {
	Location string
	Type     RefType
}

// GetFile returns the file path of the reference.
func (r *ExtractedRef) GetFile() string {
	switch r.Type {
	case File, HTTP:
		location := strings.Split(r.Location, "#/")
		return location[0]
	default:
		return r.Location
	}
}

// GetReference returns the reference path of the reference.
func (r *ExtractedRef) GetReference() string {
	switch r.Type {
	case File, HTTP:
		location := strings.Split(r.Location, "#/")
		return fmt.Sprintf("#/%s", location[1])
	default:
		return r.Location
	}
}

// ExtractFileType returns the file extension of the reference.
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
