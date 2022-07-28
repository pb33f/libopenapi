package v3

import "github.com/pb33f/libopenapi/datamodel/low"

type Server struct {
    URL         low.NodeReference[string]
    Description low.NodeReference[string]
    Variables   map[string]ServerVariable
}
