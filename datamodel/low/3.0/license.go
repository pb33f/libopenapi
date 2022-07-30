package v3

import (
    "github.com/pb33f/libopenapi/datamodel/low"
)

type License struct {
    Name low.NodeReference[string]
    URL  low.NodeReference[string]
}
