// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
    "fmt"
    lowmodel "github.com/pb33f/libopenapi/datamodel/low"
    lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
    "gopkg.in/yaml.v3"
)

func ExampleNewXML() {

    // create an example schema object
    // this can be either JSON or YAML.
    yml := `
namespace: https://pb33f.io/schema
prefix: sample`

    // unmarshal raw bytes
    var node yaml.Node
    _ = yaml.Unmarshal([]byte(yml), &node)

    // build out the low-level model
    var lowXML lowbase.XML
    _ = lowmodel.BuildModel(&node, &lowXML)
    _ = lowXML.Build(node.Content[0], nil)

    // build the high level tag
    highXML := NewXML(&lowXML)

    // print out the XML namespace
    fmt.Print(highXML.Namespace)
    // Output: https://pb33f.io/schema
}
