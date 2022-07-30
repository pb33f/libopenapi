package openapi

import (
    "github.com/pb33f/libopenapi/datamodel"
    "github.com/pb33f/libopenapi/datamodel/low"
    v3 "github.com/pb33f/libopenapi/datamodel/low/3.0"
    "github.com/pb33f/libopenapi/utils"
)

func CreateDocument(spec []byte) (*v3.Document, error) {

    // extract details from spec
    info, err := datamodel.ExtractSpecInfo(spec)
    if err != nil {
        return nil, err
    }

    doc := v3.Document{Version: low.NodeReference[string]{Value: info.Version, ValueNode: info.RootNode}}

    // build an index
    //idx := index.NewSpecIndex(info.RootNode)
    datamodel.BuildModel(info.RootNode.Content[0], &doc)

    // extract info
    extractErr := extractInfo(info, &doc)
    if extractErr != nil {
        return nil, extractErr
    }

    // extract servers
    extractErr = extractServers(info, &doc)
    if extractErr != nil {
        return nil, extractErr
    }

    return &doc, nil
}

func extractInfo(info *datamodel.SpecInfo, doc *v3.Document) error {
    _, ln, vn := utils.FindKeyNodeFull("info", info.RootNode.Content)
    if vn != nil {
        ir := v3.Info{}
        err := datamodel.BuildModel(vn, &ir)
        if err != nil {
            return err
        }
        err = ir.Build(vn)
        nr := low.NodeReference[*v3.Info]{Value: &ir, ValueNode: vn, KeyNode: ln}
        doc.Info = nr
    }
    return nil
}

func extractServers(info *datamodel.SpecInfo, doc *v3.Document) error {
    _, ln, vn := utils.FindKeyNodeFull("servers", info.RootNode.Content)
    if vn != nil {
        if utils.IsNodeArray(vn) {
            var servers []low.NodeReference[*v3.Server]
            for _, srvN := range vn.Content {
                if utils.IsNodeMap(srvN) {
                    srvr := v3.Server{}
                    err := datamodel.BuildModel(srvN, &srvr)
                    if err != nil {
                        return err
                    }
                    srvr.Build(srvN)
                    servers = append(servers, low.NodeReference[*v3.Server]{
                        Value:     &srvr,
                        ValueNode: srvN,
                        KeyNode:   ln,
                    })
                }
            }
            doc.Servers = servers
        }
    }
    return nil
}
