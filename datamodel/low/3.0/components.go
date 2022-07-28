package v3

import "gopkg.in/yaml.v3"

type Components struct {
    Node      *yaml.Node
    Schemas   map[string]Schema
    Responses map[string]Response
}
