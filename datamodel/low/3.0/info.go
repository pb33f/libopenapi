package v3

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    "gopkg.in/yaml.v3"
)

type Info struct {
    Node           *yaml.Node
    Title          low.NodeReference[string]
    Description    low.NodeReference[string]
    TermsOfService low.NodeReference[string]
    Contact        Contact
    License        License
    Version        low.NodeReference[string]
}
