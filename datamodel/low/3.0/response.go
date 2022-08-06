package v3

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    "gopkg.in/yaml.v3"
)

const (
    LinksLabel = "links"
)

type Responses struct {
    Codes   map[low.KeyReference[string]]low.ValueReference[*Response]
    Default low.NodeReference[*Response]
}

func (r *Responses) Build(root *yaml.Node) error {
    codes, _, _, err := ExtractMapFlat[*Response](ResponsesLabel, root)
    if err != nil {
        return err
    }
    if codes != nil {
        r.Codes = codes
    }
    return nil
}

func (r *Responses) FindResponseByCode(code string) *low.ValueReference[*Response] {
    return FindItemInMap[*Response](code, r.Codes)
}

type Response struct {
    Description low.NodeReference[string]
    Headers     low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Header]]
    Content     low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*MediaType]]
    Extensions  map[low.KeyReference[string]]low.ValueReference[any]
    Links       low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Link]]
}

func (r *Response) FindContent(cType string) *low.ValueReference[*MediaType] {
    return FindItemInMap[*MediaType](cType, r.Content.Value)
}

func (r *Response) FindHeader(hType string) *low.ValueReference[*Header] {
    return FindItemInMap[*Header](hType, r.Headers.Value)
}

func (r *Response) FindLink(hType string) *low.ValueReference[*Link] {
    return FindItemInMap[*Link](hType, r.Links.Value)
}

func (r *Response) Build(root *yaml.Node) error {
    extensionMap, err := ExtractExtensions(root)
    if err != nil {
        return err
    }
    r.Extensions = extensionMap

    // extract headers
    headers, lN, kN, err := ExtractMapFlat[*Header](HeadersLabel, root)
    if err != nil {
        return err
    }
    if headers != nil {
        r.Headers = low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Header]]{
            Value:     headers,
            KeyNode:   lN,
            ValueNode: kN,
        }
    }

    // handle content, if set.
    con, clN, cN, cErr := ExtractMapFlat[*MediaType](ContentLabel, root)
    if cErr != nil {
        return cErr
    }
    if con != nil {
        r.Content = low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*MediaType]]{
            Value:     con,
            KeyNode:   clN,
            ValueNode: cN,
        }
    }

    // handle links if set
    links, linkLabel, linkValue, lErr := ExtractMapFlat[*Link](LinksLabel, root)
    if lErr != nil {
        return lErr
    }
    if links != nil {
        r.Links = low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Link]]{
            Value:     links,
            KeyNode:   linkLabel,
            ValueNode: linkValue,
        }
    }

    return nil
}
