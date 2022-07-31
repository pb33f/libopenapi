package openapi

import (
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/3.0"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"strconv"
	"sync"
)

const (
	Info    = "info"
	Servers = "servers"
)

func CreateDocument(info *datamodel.SpecInfo) (*v3.Document, error) {

	doc := v3.Document{Version: low.NodeReference[string]{Value: info.Version, ValueNode: info.RootNode}}

	// build an index
	//idx := index.NewSpecIndex(info.RootNode)
	//datamodel.BuildModel(info.RootNode.Content[0], &doc)
	var wg sync.WaitGroup
	var errors []error
	var runExtraction = func(info *datamodel.SpecInfo, doc *v3.Document,
		runFunc func(i *datamodel.SpecInfo, d *v3.Document) error,
		ers *[]error,
		wg *sync.WaitGroup) {

		if er := runFunc(info, doc); er != nil {
			*ers = append(*ers, er)
		}

		wg.Done()
	}

	wg.Add(3)
	go runExtraction(info, &doc, extractInfo, &errors, &wg)
	go runExtraction(info, &doc, extractServers, &errors, &wg)
	go runExtraction(info, &doc, extractTags, &errors, &wg)
	wg.Wait()

	// todo fix this.
	if len(errors) > 0 {
		return &doc, errors[0]
	}

	return &doc, nil
}

func extractInfo(info *datamodel.SpecInfo, doc *v3.Document) error {
	_, ln, vn := utils.FindKeyNodeFull(Info, info.RootNode.Content)
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
	_, ln, vn := utils.FindKeyNodeFull(Servers, info.RootNode.Content)
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

func extractTags(info *datamodel.SpecInfo, doc *v3.Document) error {
	_, ln, vn := utils.FindKeyNodeFull(v3.Tags, info.RootNode.Content)
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

func ExtractExtensions(root *yaml.Node) (map[low.NodeReference[string]]low.NodeReference[any], error) {
	extensions := utils.FindExtensionNodes(root.Content)
	extensionMap := make(map[low.NodeReference[string]]low.NodeReference[any])
	for _, ext := range extensions {
		// this is an object, decode into an unknown map.
		if utils.IsNodeMap(ext.Value) {
			var v interface{}
			err := ext.Value.Decode(&v)
			if err != nil {
				return nil, err
			}
			extensionMap[low.NodeReference[string]{
				Value:     ext.Key.Value,
				KeyNode:   ext.Key,
				ValueNode: ext.Value,
			}] = low.NodeReference[any]{Value: v, KeyNode: ext.Key}
		}
		if utils.IsNodeStringValue(ext.Value) {
			extensionMap[low.NodeReference[string]{
				Value:     ext.Key.Value,
				KeyNode:   ext.Key,
				ValueNode: ext.Value,
			}] = low.NodeReference[any]{Value: ext.Value.Value, ValueNode: ext.Value}
		}
		if utils.IsNodeFloatValue(ext.Value) {
			fv, _ := strconv.ParseFloat(ext.Value.Value, 64)
			extensionMap[low.NodeReference[string]{
				Value:     ext.Key.Value,
				KeyNode:   ext.Key,
				ValueNode: ext.Value,
			}] = low.NodeReference[any]{Value: fv, ValueNode: ext.Value}
		}
		if utils.IsNodeIntValue(ext.Value) {
			iv, _ := strconv.ParseInt(ext.Value.Value, 10, 64)
			extensionMap[low.NodeReference[string]{
				Value:     ext.Key.Value,
				KeyNode:   ext.Key,
				ValueNode: ext.Value,
			}] = low.NodeReference[any]{Value: iv, ValueNode: ext.Value}
		}
		if utils.IsNodeBoolValue(ext.Value) {
			bv, _ := strconv.ParseBool(ext.Value.Value)
			extensionMap[low.NodeReference[string]{
				Value:     ext.Key.Value,
				KeyNode:   ext.Key,
				ValueNode: ext.Value,
			}] = low.NodeReference[any]{Value: bv, ValueNode: ext.Value}
		}
		if utils.IsNodeArray(ext.Value) {
			var v []interface{}
			err := ext.Value.Decode(&v)
			if err != nil {
				return nil, err
			}
			extensionMap[low.NodeReference[string]{
				Value:     ext.Key.Value,
				KeyNode:   ext.Key,
				ValueNode: ext.Value,
			}] = low.NodeReference[any]{Value: v, ValueNode: ext.Value}
		}
	}
	return extensionMap, nil
}
