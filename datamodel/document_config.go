// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package datamodel

import "net/url"

// DocumentConfiguration is used to configure the document creation process. It was added in v0.6.0 to allow
// for more fine-grained control over controls and new features.
//
// The default configuration will set AllowFileReferences to false and AllowRemoteReferences to false, which means
// any non-local (local being the specification, not the file system) references, will be ignored.
type DocumentConfiguration struct {
	// The BaseURL will be the root from which relative references will be resolved from if they can't be found locally.
	// Schema must be set to "http/https".
	BaseURL *url.URL

	// If resolving locally, the BasePath will be the root from which relative references will be resolved from.
	// It's usually the location of the root specification.
	BasePath string // set the Base Path for resolving relative references if the spec is exploded.

	// AllowFileReferences will allow the index to locate relative file references. This is disabled by default.
	AllowFileReferences bool

	// AllowRemoteReferences will allow the index to lookup remote references. This is disabled by default.
	AllowRemoteReferences bool
}

func NewOpenDocumentConfiguration() *DocumentConfiguration {
	return &DocumentConfiguration{
		AllowFileReferences:   true,
		AllowRemoteReferences: true,
	}
}

func NewClosedDocumentConfiguration() *DocumentConfiguration {
	return &DocumentConfiguration{
		AllowFileReferences:   false,
		AllowRemoteReferences: false,
	}
}
