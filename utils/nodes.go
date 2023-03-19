// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package utils

import "gopkg.in/yaml.v3"

func CreateEmptyMapNode() *yaml.Node {
    n := &yaml.Node{
        Kind: yaml.MappingNode,
        Tag:  "!!map",
    }
    return n
}

func CreateEmptySequenceNode() *yaml.Node {
    n := &yaml.Node{
        Kind: yaml.SequenceNode,
        Tag:  "!!seq",
    }
    return n
}

func CreateStringNode(str string) *yaml.Node {
    n := &yaml.Node{
        Kind:  yaml.ScalarNode,
        Tag:   "!!str",
        Value: str,
    }
    return n
}

func CreateBoolNode(str string) *yaml.Node {
    n := &yaml.Node{
        Kind:  yaml.ScalarNode,
        Tag:   "!!bool",
        Value: str,
    }
    return n
}

func CreateIntNode(str string) *yaml.Node {
    n := &yaml.Node{
        Kind:  yaml.ScalarNode,
        Tag:   "!!int",
        Value: str,
    }
    return n
}

func CreateFloatNode(str string) *yaml.Node {
    n := &yaml.Node{
        Kind:  yaml.ScalarNode,
        Tag:   "!!float",
        Value: str,
    }
    return n
}
