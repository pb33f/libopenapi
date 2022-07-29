package v3

import (
    "fmt"
    "github.com/pb33f/libopenapi/datamodel/low"
    "gopkg.in/yaml.v3"
    "reflect"
)

type Document struct {
    Version      string
    Info         Info
    Servers      []Server
    Paths        Paths
    Components   Components
    Security     []SecurityRequirement
    Tags         []Tag
    ExternalDocs ExternalDoc
    Extensions   map[string]low.ObjectReference
}

func (d Document) Build(node *yaml.Node) {

    doc := Document{
        Version:      "",
        Info:         Info{},
        Servers:      nil,
        Paths:        Paths{},
        Components:   Components{},
        Security:     nil,
        Tags:         nil,
        ExternalDocs: ExternalDoc{},
        Extensions:   nil,
    }

    var j interface{}
    j = doc
    t := reflect.TypeOf(j)
    v := reflect.ValueOf(j)
    k := t.Kind()
    fmt.Println("Type ", t)
    fmt.Println("Value ", v)
    fmt.Println("Kind ", k)
    for i := 0; i < v.NumField(); i++ {
        fmt.Printf("Field:%d type:%T value:%v\n", i, v.Field(i), v.Field(i))
    }

}
