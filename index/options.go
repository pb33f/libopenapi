package index

import (
	"path"
	"strings"
)

// Options that affect the behaviour of SpecIndex
type Options struct {
	// Source path from which the spec is loaded, typically a URL or a local file path.
	specPath string

	// Base path (URL or local path) relative to which references are going to be resolved.
	referenceBasePath string
}

func NewOptions() *Options {
	return &Options{}
}

// Sets the path to the source file from which the spec is loaded.
func (options *Options) SetSpecPath(specPath string) {
	options.specPath = specPath

	partAfterLastSlash := path.Base(options.specPath)
	options.referenceBasePath = strings.TrimSuffix(specPath, partAfterLastSlash)
}

func (options *Options) GetReferenceBasePath() string {
	return options.referenceBasePath
}
