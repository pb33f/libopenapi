// Copyright 2023-2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io
// SPDX-License-Identifier: MIT

package bundler

// ComponentOrigin tracks the original location of a component that was lifted
// into the bundled document's components section.
type ComponentOrigin struct {
	// OriginalFile is the absolute path to the file containing the original definition.
	// e.g., "/path/to/models/User.yaml"
	OriginalFile string `json:"originalFile" yaml:"originalFile"`

	// OriginalRef is the JSON Pointer within the original file.
	// e.g., "#/components/schemas/User" or "#/User"
	OriginalRef string `json:"originalRef" yaml:"originalRef"`

	// OriginalName is the component name before any collision renaming.
	// e.g., "User" (even if bundled as "User__2")
	OriginalName string `json:"originalName" yaml:"originalName"`

	// Line is the 1-based line number in the original file.
	Line int `json:"line" yaml:"line"`

	// Column is the 1-based column number in the original file.
	Column int `json:"column" yaml:"column"`

	// WasRenamed indicates if the component was renamed due to collision.
	WasRenamed bool `json:"wasRenamed" yaml:"wasRenamed"`

	// BundledRef is the final JSON Pointer in the bundled document.
	// e.g., "#/components/schemas/User__2"
	BundledRef string `json:"bundledRef" yaml:"bundledRef"`

	// ComponentType is the type of component (schemas, responses, parameters, etc.)
	ComponentType string `json:"componentType" yaml:"componentType"`
}

// ComponentOriginMap maps bundled refs to their original locations.
// Key is the bundled JSON Pointer (e.g., "#/components/schemas/User").
type ComponentOriginMap map[string]*ComponentOrigin

// BundleResult contains the bundled bytes and origin tracking information.
type BundleResult struct {
	// Bytes is the bundled YAML output.
	Bytes []byte

	// Origins maps bundled JSON Pointer paths to their original locations.
	// This enables navigation from bundled components back to source files.
	Origins ComponentOriginMap
}

// NewBundleResult creates a new BundleResult with initialized maps.
func NewBundleResult() *BundleResult {
	return &BundleResult{
		Origins: make(ComponentOriginMap),
	}
}

// AddOrigin adds a component origin to the result.
func (r *BundleResult) AddOrigin(bundledRef string, origin *ComponentOrigin) {
	if r.Origins == nil {
		r.Origins = make(ComponentOriginMap)
	}
	origin.BundledRef = bundledRef
	r.Origins[bundledRef] = origin
}

// GetOrigin retrieves the origin for a bundled reference.
func (r *BundleResult) GetOrigin(bundledRef string) *ComponentOrigin {
	if r.Origins == nil {
		return nil
	}
	return r.Origins[bundledRef]
}

// OriginCount returns the number of tracked origins.
func (r *BundleResult) OriginCount() int {
	if r.Origins == nil {
		return 0
	}
	return len(r.Origins)
}
