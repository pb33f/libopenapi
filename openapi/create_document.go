package openapi

import (
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/3.0"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"sync"
)

func CreateDocument(info *datamodel.SpecInfo) (*v3.Document, error) {

	doc := v3.Document{Version: low.NodeReference[string]{Value: info.Version, ValueNode: info.RootNode}}

	// build an index
	idx := index.NewSpecIndex(info.RootNode)

	var wg sync.WaitGroup
	var errors []error
	var runExtraction = func(info *datamodel.SpecInfo, doc *v3.Document, idx *index.SpecIndex,
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
		extractComponents,
		extractSecurity,
		extractExternalDocs,
	}
	wg.Add(len(extractionFuncs))
	for _, f := range extractionFuncs {
		go runExtraction(info, &doc, idx, f, &errors, &wg)
	}
	wg.Wait()

	// todo fix this.
	if len(errors) > 0 {
		return &doc, errors[0]
	}
	return &doc, nil
}

func extractInfo(info *datamodel.SpecInfo, doc *v3.Document, idx *index.SpecIndex) error {
	_, ln, vn := utils.FindKeyNodeFull(v3.InfoLabel, info.RootNode.Content)
	if vn != nil {
		ir := v3.Info{}
		err := low.BuildModel(vn, &ir)
		if err != nil {
			return err
		}
		err = ir.Build(vn)
		nr := low.NodeReference[*v3.Info]{Value: &ir, ValueNode: vn, KeyNode: ln}
		doc.Info = nr
	}
	return nil
}

func extractSecurity(info *datamodel.SpecInfo, doc *v3.Document, idx *index.SpecIndex) error {
	sec, sErr := low.ExtractObject[*v3.SecurityRequirement](v3.SecurityLabel, info.RootNode, idx)
	if sErr != nil {
		return sErr
	}
	doc.Security = sec
	return nil
}

func extractExternalDocs(info *datamodel.SpecInfo, doc *v3.Document, idx *index.SpecIndex) error {
	extDocs, dErr := low.ExtractObject[*v3.ExternalDoc](v3.ExternalDocsLabel, info.RootNode, idx)
	if dErr != nil {
		return dErr
	}
	doc.ExternalDocs = extDocs
	return nil
}

func extractComponents(info *datamodel.SpecInfo, doc *v3.Document, idx *index.SpecIndex) error {
	_, ln, vn := utils.FindKeyNodeFull(v3.ComponentsLabel, info.RootNode.Content)
	if vn != nil {
		ir := v3.Components{}
		err := low.BuildModel(vn, &ir)
		if err != nil {
			return err
		}
		err = ir.Build(vn, idx)
		nr := low.NodeReference[*v3.Components]{Value: &ir, ValueNode: vn, KeyNode: ln}
		doc.Components = nr
	}
	return nil
}

func extractServers(info *datamodel.SpecInfo, doc *v3.Document, idx *index.SpecIndex) error {
	_, ln, vn := utils.FindKeyNodeFull(v3.ServersLabel, info.RootNode.Content)
	if vn != nil {
		if utils.IsNodeArray(vn) {
			var servers []low.ValueReference[*v3.Server]
			for _, srvN := range vn.Content {
				if utils.IsNodeMap(srvN) {
					srvr := v3.Server{}
					err := low.BuildModel(srvN, &srvr)
					if err != nil {
						return err
					}
					srvr.Build(srvN, idx)
					servers = append(servers, low.ValueReference[*v3.Server]{
						Value:     &srvr,
						ValueNode: srvN,
					})
				}
			}
			doc.Servers = low.NodeReference[[]low.ValueReference[*v3.Server]]{
				Value:     servers,
				KeyNode:   ln,
				ValueNode: vn,
			}
		}
	}
	return nil
}

func extractTags(info *datamodel.SpecInfo, doc *v3.Document, idx *index.SpecIndex) error {
	_, ln, vn := utils.FindKeyNodeFull(v3.TagsLabel, info.RootNode.Content)
	if vn != nil {
		if utils.IsNodeArray(vn) {
			var tags []low.ValueReference[*v3.Tag]
			for _, tagN := range vn.Content {
				if utils.IsNodeMap(tagN) {
					tag := v3.Tag{}
					err := low.BuildModel(tagN, &tag)
					if err != nil {
						return err
					}
					tag.Build(tagN, idx)
					tags = append(tags, low.ValueReference[*v3.Tag]{
						Value:     &tag,
						ValueNode: tagN,
					})
				}
			}
			doc.Tags = low.NodeReference[[]low.ValueReference[*v3.Tag]]{
				Value:     tags,
				KeyNode:   ln,
				ValueNode: vn,
			}
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
