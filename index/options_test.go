package index

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptions_SetSpecPath(t *testing.T) {
	var tests = []struct {
		specPath                  string
		expectedReferenceBasePath string
	}{
		{"relative.yaml", ""},
		{"/some/absolute/path/spec.yaml", "/some/absolute/path/"},
		{"http://example.com/path/spec.yaml", "http://example.com/path/"},
		{"https://example.com/path/spec.yaml", "https://example.com/path/"},
		{"http://example.com/path/spec.yaml?param=value", "http://example.com/path/"},
		{"https://example.com/path/spec.yaml?param=value", "https://example.com/path/"},
	}

	for _, testCase := range tests {
		options := NewOptions()
		options.SetSpecPath(testCase.specPath)

		assert.Equal(t, testCase.expectedReferenceBasePath, options.GetReferenceBasePath())
	}
}
