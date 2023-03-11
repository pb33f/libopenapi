package v3

import (
	"errors"
	"os"
	"sync"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/resolver"
	"github.com/pb33f/libopenapi/utils"
)

// CreateDocument will create a new Document instance from the provided SpecInfo.
//
// Deprecated: Use CreateDocumentFromConfig instead. This function will be removed in a later version, it
// defaults to allowing file and remote references, and does not support relative file references.
func CreateDocument(info *datamodel.SpecInfo) (*Document, []error) {
	config := datamodel.DocumentConfiguration{
		AllowFileReferences:   true,
		AllowRemoteReferences: true,
	}
	return createDocument(info, &config)
}

// CreateDocumentFromConfig Create a new document from the provided SpecInfo and DocumentConfiguration pointer.
func CreateDocumentFromConfig(info *datamodel.SpecInfo, config *datamodel.DocumentConfiguration) (*Document, []error) {
	return createDocument(info, config)
}

func createDocument(info *datamodel.SpecInfo, config *datamodel.DocumentConfiguration) (*Document, []error) {
	_, labelNode, versionNode := utils.FindKeyNodeFull(OpenAPILabel, info.RootNode.Content)
	var version low.NodeReference[string]
	if versionNode == nil {
		return nil, []error{errors.New("no openapi version/tag found, cannot create document")}
	}
	version = low.NodeReference[string]{Value: versionNode.Value, KeyNode: labelNode, ValueNode: versionNode}
	doc := Document{Version: version}

	// get current working directory as a basePath
	cwd, _ := os.Getwd()

	// If basePath is provided override it
	if config.BasePath != "" {
		cwd = config.BasePath
	}
	// build an index
	idx := index.NewSpecIndexWithConfig(info.RootNode, &index.SpecIndexConfig{
		BaseURL:           config.BaseURL,
		BasePath:          cwd,
		AllowFileLookup:   config.AllowFileReferences,
		AllowRemoteLookup: config.AllowRemoteReferences,
	})
	doc.Index = idx

	var errs []error

	errs = idx.GetReferenceIndexErrors()

	// create resolver and check for circular references.
	resolve := resolver.NewResolver(idx)
	resolvingErrors := resolve.CheckForCircularReferences()

	if len(resolvingErrors) > 0 {
		for r := range resolvingErrors {
			errs = append(errs, resolvingErrors[r])
		}
	}

	var wg sync.WaitGroup

	doc.Extensions = low.ExtractExtensions(info.RootNode.Content[0])

	// if set, extract jsonSchemaDialect (3.1)
	_, dialectLabel, dialectNode := utils.FindKeyNodeFull(JSONSchemaDialectLabel, info.RootNode.Content)
	if dialectNode != nil {
		doc.JsonSchemaDialect = low.NodeReference[string]{
			Value: dialectNode.Value, KeyNode: dialectLabel, ValueNode: dialectNode,
		}
	}

	runExtraction := func(info *datamodel.SpecInfo, doc *Document, idx *index.SpecIndex,
		runFunc func(i *datamodel.SpecInfo, d *Document, idx *index.SpecIndex) error,
		ers *[]error,
		wg *sync.WaitGroup,
	) {
		if er := runFunc(info, doc, idx); er != nil {
			*ers = append(*ers, er)
		}
		wg.Done()
	}
	extractionFuncs := []func(i *datamodel.SpecInfo, d *Document, idx *index.SpecIndex) error{
		extractInfo,
		extractServers,
		extractTags,
		extractComponents,
		extractSecurity,
		extractExternalDocs,
		extractPaths,
		extractWebhooks,
	}

	wg.Add(len(extractionFuncs))
	for _, f := range extractionFuncs {
		go runExtraction(info, &doc, idx, f, &errs, &wg)
	}
	wg.Wait()
	return &doc, errs
}

func extractInfo(info *datamodel.SpecInfo, doc *Document, idx *index.SpecIndex) error {
	_, ln, vn := utils.FindKeyNodeFullTop(base.InfoLabel, info.RootNode.Content[0].Content)
	if vn != nil {
		ir := base.Info{}
		_ = low.BuildModel(vn, &ir)
		_ = ir.Build(vn, idx)
		nr := low.NodeReference[*base.Info]{Value: &ir, ValueNode: vn, KeyNode: ln}
		doc.Info = nr
	}
	return nil
}

func extractSecurity(info *datamodel.SpecInfo, doc *Document, idx *index.SpecIndex) error {
	sec, ln, vn, err := low.ExtractArray[*base.SecurityRequirement](SecurityLabel, info.RootNode.Content[0], idx)
	if err != nil {
		return err
	}
	if vn != nil && ln != nil {
		doc.Security = low.NodeReference[[]low.ValueReference[*base.SecurityRequirement]]{
			Value:     sec,
			KeyNode:   ln,
			ValueNode: vn,
		}
	}
	return nil
}

func extractExternalDocs(info *datamodel.SpecInfo, doc *Document, idx *index.SpecIndex) error {
	extDocs, dErr := low.ExtractObject[*base.ExternalDoc](base.ExternalDocsLabel, info.RootNode.Content[0], idx)
	if dErr != nil {
		return dErr
	}
	doc.ExternalDocs = extDocs
	return nil
}

func extractComponents(info *datamodel.SpecInfo, doc *Document, idx *index.SpecIndex) error {
	_, ln, vn := utils.FindKeyNodeFullTop(ComponentsLabel, info.RootNode.Content[0].Content)
	if vn != nil {
		ir := Components{}
		_ = low.BuildModel(vn, &ir)
		err := ir.Build(vn, idx)
		if err != nil {
			return err
		}
		nr := low.NodeReference[*Components]{Value: &ir, ValueNode: vn, KeyNode: ln}
		doc.Components = nr
	}
	return nil
}

func extractServers(info *datamodel.SpecInfo, doc *Document, idx *index.SpecIndex) error {
	_, ln, vn := utils.FindKeyNodeFull(ServersLabel, info.RootNode.Content[0].Content)
	if vn != nil {
		if utils.IsNodeArray(vn) {
			var servers []low.ValueReference[*Server]
			for _, srvN := range vn.Content {
				if utils.IsNodeMap(srvN) {
					srvr := Server{}
					_ = low.BuildModel(srvN, &srvr)
					_ = srvr.Build(srvN, idx)
					servers = append(servers, low.ValueReference[*Server]{
						Value:     &srvr,
						ValueNode: srvN,
					})
				}
			}
			doc.Servers = low.NodeReference[[]low.ValueReference[*Server]]{
				Value:     servers,
				KeyNode:   ln,
				ValueNode: vn,
			}
		}
	}
	return nil
}

func extractTags(info *datamodel.SpecInfo, doc *Document, idx *index.SpecIndex) error {
	_, ln, vn := utils.FindKeyNodeFull(base.TagsLabel, info.RootNode.Content[0].Content)
	if vn != nil {
		if utils.IsNodeArray(vn) {
			var tags []low.ValueReference[*base.Tag]
			for _, tagN := range vn.Content {
				if utils.IsNodeMap(tagN) {
					tag := base.Tag{}
					_ = low.BuildModel(tagN, &tag)
					if err := tag.Build(tagN, idx); err != nil {
						return err
					}
					tags = append(tags, low.ValueReference[*base.Tag]{
						Value:     &tag,
						ValueNode: tagN,
					})
				}
			}
			doc.Tags = low.NodeReference[[]low.ValueReference[*base.Tag]]{
				Value:     tags,
				KeyNode:   ln,
				ValueNode: vn,
			}
		}
	}
	return nil
}

func extractPaths(info *datamodel.SpecInfo, doc *Document, idx *index.SpecIndex) error {
	_, ln, vn := utils.FindKeyNodeFull(PathsLabel, info.RootNode.Content[0].Content)
	if vn != nil {
		ir := Paths{}
		err := ir.Build(vn, idx)
		if err != nil {
			return err
		}
		nr := low.NodeReference[*Paths]{Value: &ir, ValueNode: vn, KeyNode: ln}
		doc.Paths = nr
	}
	return nil
}

func extractWebhooks(info *datamodel.SpecInfo, doc *Document, idx *index.SpecIndex) error {
	hooks, hooksL, hooksN, eErr := low.ExtractMap[*PathItem](WebhooksLabel, info.RootNode, idx)
	if eErr != nil {
		return eErr
	}
	if hooks != nil {
		doc.Webhooks = low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*PathItem]]{
			Value:     hooks,
			KeyNode:   hooksL,
			ValueNode: hooksN,
		}
	}
	return nil
}
