// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestSpecIndex_GetConfig(t *testing.T) {
	idx1 := NewTestSpecIndex()
	c := SpecIndexConfig{}
	idx1.config = &c
	assert.Equal(t, &c, idx1.GetConfig())
}

func Test_MarshalJSON(t *testing.T) {
	rm := &ReferenceMapped{
		OriginalReference: &Reference{
			FullDefinition: "full definition",
			Path:           "path",
			Node: &yaml.Node{
				Line:   1,
				Column: 1,
				Content: []*yaml.Node{
					{
						Line:   9,
						Column: 10,
					},
					{
						Value: "lemon cake",
					},
				},
			},
		},
		Reference: &Reference{
			FullDefinition: "full definition",
			Path:           "path",
			Node: &yaml.Node{
				Line:   2,
				Column: 2,
			},
			KeyNode: &yaml.Node{
				Line:   3,
				Column: 3,
			},
		},
		Definition:     "definition",
		FullDefinition: "full definition",
		IsPolymorphic:  true,
	}

	bytes, _ := json.Marshal(rm)
	assert.Len(t, bytes, 173)
}
