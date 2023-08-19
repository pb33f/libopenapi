package libopenapi

import (
	"net/http"
	"net/url"

	"github.com/pb33f/libopenapi/datamodel"
)

// Configuration is used to configure the document creation process. It was added in v0.6.0 to allow
// for more fine-grained control over controls and new features.
//
// The default configuration will set AllowFileReferences to false and AllowRemoteReferences to false, which means
// any non-local (local being the specification, not the file system) references, will be ignored.
type Configuration struct {
	// The BaseURL will be the root from which relative references will be resolved from if they can't be found locally.
	// Schema must be set to "http/https".
	BaseURL *url.URL

	// RemoteURLHandler is a function that will be used to retrieve remote documents. If not set, the default
	// remote document getter will be used.
	// Resolves [#132]: https://github.com/pb33f/libopenapi/issues/132
	RemoteURLHandler func(url string) (*http.Response, error)

	// If resolving locally, the BasePath will be the root from which relative references will be resolved from.
	// It's usually the location of the root specification.
	BasePath string // set the Base Path for resolving relative references if the spec is exploded.

	// AllowFileReferences will allow the index to locate relative file references. This is disabled by default.
	AllowFileReferences bool

	// AllowRemoteReferences will allow the index to lookup remote references. This is disabled by default.
	AllowRemoteReferences bool

	// AvoidIndexBuild will avoid building the index. This is disabled by default, only use if you are sure you don't need it.
	// This is useful for developers building out models that should be indexed later on.
	AvoidIndexBuild bool

	// BypassDocumentCheck will bypass the document check. This is disabled by default. This will allow any document to
	// passed in and used. Only enable this when parsing non openapi documents.
	BypassDocumentCheck bool

	// AllowCircularReferences will allow circular references to be resolved. This is disabled by default.
	// Will return an error in case of circular references.
	AllowCircularReferenceResolving bool
}

func (c *Configuration) toModelConfig() *datamodel.DocumentConfiguration {
	return &datamodel.DocumentConfiguration{
		BaseURL:               c.BaseURL,
		RemoteURLHandler:      c.RemoteURLHandler,
		BasePath:              c.BasePath,
		AllowFileReferences:   c.AllowFileReferences,
		AllowRemoteReferences: c.AllowRemoteReferences,
		AvoidIndexBuild:       c.AvoidIndexBuild,
		BypassDocumentCheck:   c.BypassDocumentCheck,
	}
}

type ConfigurationOption func(*Configuration) error

func WithBaseURL(baseUrl *url.URL) ConfigurationOption {
	return func(o *Configuration) error {
		o.BaseURL = baseUrl
		return nil
	}
}

func WithRemoteURLHandler(handler func(url string) (*http.Response, error)) ConfigurationOption {
	return func(o *Configuration) error {
		o.RemoteURLHandler = handler
		return nil

	}
}

func WithBasePath(basePath string) ConfigurationOption {
	return func(o *Configuration) error {
		o.BasePath = basePath
		return nil
	}
}

func WithAllowFileReferences(allow bool) ConfigurationOption {
	return func(o *Configuration) error {
		o.AllowFileReferences = allow
		return nil
	}
}

func WithAllowRemoteReferences(allow bool) ConfigurationOption {
	return func(o *Configuration) error {
		o.AllowRemoteReferences = allow
		return nil
	}
}

func WithAvoidIndexBuild(avoid bool) ConfigurationOption {
	return func(o *Configuration) error {
		o.AvoidIndexBuild = avoid
		return nil
	}
}

func WithBypassDocumentCheck(bypass bool) ConfigurationOption {
	return func(o *Configuration) error {
		o.BypassDocumentCheck = bypass
		return nil
	}
}
