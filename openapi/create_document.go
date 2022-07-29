package openapi

import (
    "github.com/pb33f/libopenapi/datamodel"
    v3 "github.com/pb33f/libopenapi/datamodel/low/3.0"
)

func CreateDocument(spec []byte) (*v3.Document, error) {

    // extract details from spec
    info, err := datamodel.ExtractSpecInfo(spec)
    if err != nil {
        return nil, err
    }

    doc := &v3.Document{}
    doc.Build(info.RootNode.Content[0])
    return doc, nil
}
