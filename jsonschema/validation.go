// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package jsonschema

import (
	"bytes"
	"encoding/json"
	"sync"

	"github.com/pb33f/go-yaml"
	validator "github.com/pb33f/jsonschema/v6"
	modeljson "github.com/pb33f/libopenapi/json"
)

// The validator ships the draft and its vocabularies. Compiling it needs no network.
var draftMetaSchema = sync.OnceValue(func() *validator.Schema {
	return validator.NewCompiler().MustCompile(Dialect)
})

func validateSource(root *yaml.Node, data []byte) error {
	if root.Kind == yaml.ScalarNode {
		return nil // indexSchema has already checked that this is a boolean.
	}
	var err error
	if !json.Valid(data) {
		data, err = modeljson.YAMLNodeToJSON(root, "")
		if err != nil {
			return err
		}
	}
	value, err := validator.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return err
	}
	return draftMetaSchema().Validate(value)
}
