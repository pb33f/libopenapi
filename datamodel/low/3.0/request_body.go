package v3

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    "gopkg.in/yaml.v3"
)

type RequestBody struct {
    Description low.NodeReference[string]
    Content     map[low.KeyReference[string]]map[low.KeyReference[string]]low.ValueReference[*MediaType]
    Required    low.NodeReference[bool]
    Extensions  map[low.KeyReference[string]]low.ValueReference[any]
}

func (rb *RequestBody) Build(root *yaml.Node) error {
    // extract extensions
    extensionMap, err := ExtractExtensions(root)
    if err != nil {
        return err
    }
    if extensionMap != nil {
        rb.Extensions = extensionMap
    }

    // handle content, if set.
    con, cErr := ExtractMap[*MediaType](ContentLabel, root)
    if cErr != nil {
        return cErr
    }
    if con != nil {
        rb.Content = con
    }
    return nil
}
