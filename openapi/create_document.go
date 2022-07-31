package openapi

import (
	"fmt"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/3.0"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/resolver"
	"github.com/pb33f/libopenapi/utils"
	"sync"
)

func CreateDocument(info *datamodel.SpecInfo) (*v3.Document, error) {

	doc := v3.Document{Version: low.NodeReference[string]{Value: info.Version, ValueNode: info.RootNode}}

	// build an index
	idx := index.NewSpecIndex(info.RootNode)
	rsolvr := resolver.NewResolver(idx)

	// todo handle errors
	rsolvr.Resolve()

	var wg sync.WaitGroup
	var errors []error
	var runExtraction = func(info *datamodel.SpecInfo, doc *v3.Document,
		runFunc func(i *datamodel.SpecInfo, d *v3.Document, idx *index.SpecIndex) error,
		ers *[]error,
		wg *sync.WaitGroup) {

		if er := runFunc(info, doc, idx); er != nil {
			*ers = append(*ers, er)
		}

		wg.Done()
	}

	extractionFuncs := []func(i *datamodel.SpecInfo, d *v3.Document, idx *index.SpecIndex) error{
		extractInfo,
		extractServers,
		extractTags,
		extractPaths,
	}
	wg.Add(len(extractionFuncs))
	for _, f := range extractionFuncs {
		go runExtraction(info, &doc, f, &errors, &wg)
	}
	wg.Wait()

	// todo fix this.
	if len(errors) > 0 {
		return &doc, errors[0]
	}
	fmt.Sprint(idx)
	return &doc, nil
}

func extractInfo(info *datamodel.SpecInfo, doc *v3.Document, idx *index.SpecIndex) error {
	_, ln, vn := utils.FindKeyNodeFull(v3.InfoLabel, info.RootNode.Content)
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

func extractServers(info *datamodel.SpecInfo, doc *v3.Document, idx *index.SpecIndex) error {
	_, ln, vn := utils.FindKeyNodeFull(v3.ServersLabel, info.RootNode.Content)
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

func extractTags(info *datamodel.SpecInfo, doc *v3.Document, idx *index.SpecIndex) error {
	_, ln, vn := utils.FindKeyNodeFull(v3.TagsLabel, info.RootNode.Content)
	if vn != nil {
		if utils.IsNodeArray(vn) {
			var tags []low.NodeReference[*v3.Tag]
			for _, tagN := range vn.Content {
				if utils.IsNodeMap(tagN) {
					tag := v3.Tag{}
					err := datamodel.BuildModel(tagN, &tag)
					if err != nil {
						return err
					}
					tag.Build(tagN)
					tags = append(tags, low.NodeReference[*v3.Tag]{
						Value:     &tag,
						ValueNode: tagN,
						KeyNode:   ln,
					})
				}
			}
			doc.Tags = tags
		}
	}
	return nil
}

func extractPaths(info *datamodel.SpecInfo, doc *v3.Document, idx *index.SpecIndex) error {
	_, ln, vn := utils.FindKeyNodeFull(v3.PathsLabel, info.RootNode.Content)
	if vn != nil {
		ir := v3.Paths{}
		err := ir.Build(vn, idx)
		if err != nil {
			return err
		}
		nr := low.NodeReference[*v3.Paths]{Value: &ir, ValueNode: vn, KeyNode: ln}
		doc.Paths = nr
	}
	return nil
}
