// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package high

import "github.com/pb33f/libopenapi/datamodel/low"

type GoesLow[T any] interface {
	GoLow() T
}

func ExtractExtensions(extensions map[low.KeyReference[string]]low.ValueReference[any]) map[string]any {
	extracted := make(map[string]any)
	for k, v := range extensions {
		extracted[k.Value] = v.Value
	}
	return extracted
}
