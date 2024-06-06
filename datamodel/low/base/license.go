// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"strings"
)

// License is a low-level representation of a License object as defined by OpenAPI 2 and OpenAPI 3
//
//	v2 - https://swagger.io/specification/v2/#licenseObject
//	v3 - https://spec.openapis.org/oas/v3.1.0#license-object
type License struct {
	Name       low.NodeReference[string]
	URL        low.NodeReference[string]
	Identifier low.NodeReference[string]
	KeyNode    *yaml.Node
	RootNode   *yaml.Node
	*low.Reference
}

// Build out a license, complain if both a URL and identifier are present as they are mutually exclusive
func (l *License) Build(ctx context.Context, keyNode, root *yaml.Node, idx *index.SpecIndex) error {
	l.KeyNode = keyNode
	root = utils.NodeAlias(root)
	l.RootNode = root
	utils.CheckForMergeNodes(root)
	l.Reference = new(low.Reference)
	if l.URL.Value != "" && l.Identifier.Value != "" {
		return fmt.Errorf("license cannot have both a URL and an identifier, they are mutually exclusive")
	}
	return nil
}

// GetRootNode will return the root yaml node of the License object
func (l *License) GetRootNode() *yaml.Node {
	return l.RootNode
}

// GetKeyNode will return the key yaml node of the License object
func (l *License) GetKeyNode() *yaml.Node {
	return l.KeyNode
}

// Hash will return a consistent SHA256 Hash of the License object
func (l *License) Hash() [32]byte {
	var f []string
	if !l.Name.IsEmpty() {
		f = append(f, l.Name.Value)
	}
	if !l.URL.IsEmpty() {
		f = append(f, l.URL.Value)
	}
	if !l.Identifier.IsEmpty() {
		f = append(f, l.Identifier.Value)
	}
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}
