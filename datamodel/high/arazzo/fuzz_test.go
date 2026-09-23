// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"testing"

	"go.yaml.in/yaml/v4"
)

func FuzzParseSpecificationVersionNeverPanics(f *testing.F) {
	for _, version := range []string{"1.0.0", "1.1.0", "1.1.2-rc.1", "", "1..0"} {
		f.Add(version)
	}
	f.Fuzz(func(_ *testing.T, version string) {
		_, _ = ParseSpecificationVersion(version)
	})
}

func FuzzSelectorUnionNeverPanics(f *testing.F) {
	for _, source := range []string{
		"context: $response.body\nselector: $.id\ntype: jsonpath\n",
		"context: $response.body\nselector: /id\ntype:\n  type: jsonpointer\n  version: rfc6901\n",
		"selector: malformed\n",
		"null\n",
	} {
		f.Add([]byte(source))
	}
	f.Fuzz(func(_ *testing.T, source []byte) {
		var document yaml.Node
		if yaml.Unmarshal(source, &document) != nil {
			return
		}
		selectors := selectorsFromNode(&document)
		for _, selector := range selectors {
			if selector != nil {
				_, _ = selector.Render()
			}
		}
	})
}
