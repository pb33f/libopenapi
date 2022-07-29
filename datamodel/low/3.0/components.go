package v3

import (
    "gopkg.in/yaml.v3"
    "net/http"
)

type Components struct {
    Node            *yaml.Node
    Schemas         map[string]Schema
    Responses       map[string]Response
    Parameters      map[string]Parameter
    Examples        map[string]Example
    RequestBodies   map[string]RequestBody
    Headers         map[string]http.Header
    SecuritySchemes map[string]SecurityScheme
    Links           map[string]Link
    Callbacks       map[string]Callback
}
