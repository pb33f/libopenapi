package v3

import (
    "github.com/pb33f/libopenapi/datamodel/low"
)

type Contact struct {
    Name  low.NodeReference[string]
    URL   low.NodeReference[string]
    Email low.NodeReference[string]
}
