// Copyright 2022-2033 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package index

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"go.yaml.in/yaml/v4"
)

const (
	// theoreticalRoot is the name of the theoretical spec file used when a root spec file does not exist
	theoreticalRoot = "root.yaml"
)

func NewSpecIndexWithConfigAndContext(ctx context.Context, rootNode *yaml.Node, config *SpecIndexConfig) *SpecIndex {
	index := new(SpecIndex)
	boostrapIndexCollections(index)
	index.InitHighCache()
	index.config = config
	index.rolodex = config.Rolodex
	index.uri = config.uri
	index.specAbsolutePath = config.SpecAbsolutePath
	if config.Logger != nil {
		index.logger = config.Logger
	} else {
		index.logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	}
	if rootNode == nil || len(rootNode.Content) <= 0 {
		return index
	}
	index.root = rootNode
	return createNewIndex(ctx, rootNode, index, config.AvoidBuildIndex)
}

func NewSpecIndexWithConfig(rootNode *yaml.Node, config *SpecIndexConfig) *SpecIndex {
	return NewSpecIndexWithConfigAndContext(context.Background(), rootNode, config)
}

func NewSpecIndex(rootNode *yaml.Node) *SpecIndex {
	index := new(SpecIndex)
	index.InitHighCache()
	index.config = CreateOpenAPIIndexConfig()
	index.root = rootNode
	boostrapIndexCollections(index)
	return createNewIndex(context.Background(), rootNode, index, false)
}

func createNewIndex(ctx context.Context, rootNode *yaml.Node, index *SpecIndex, avoidBuildOut bool) *SpecIndex {
	if rootNode == nil {
		return index
	}
	index.nodeMapCompleted = make(chan struct{})
	index.nodeMap = make(map[int]map[int]*yaml.Node)
	go index.MapNodes(rootNode)

	index.cache = new(sync.Map)
	results := index.ExtractRefs(ctx, index.root.Content[0], index.root, []string{}, 0, false, "")

	dd := make(map[string]struct{})
	var dedupedResults []*Reference
	for _, ref := range results {
		if _, ok := dd[ref.FullDefinition]; !ok {
			dd[ref.FullDefinition] = struct{}{}
			dedupedResults = append(dedupedResults, ref)
		}
	}

	polyKeys := make([]string, 0, len(index.polymorphicRefs))
	for k := range index.polymorphicRefs {
		polyKeys = append(polyKeys, k)
	}
	sort.Strings(polyKeys)
	poly := make([]*Reference, len(index.polymorphicRefs))
	for i, k := range polyKeys {
		poly[i] = index.polymorphicRefs[k]
	}

	if len(dedupedResults) > 0 {
		index.ExtractComponentsFromRefs(ctx, dedupedResults)
	}
	if len(poly) > 0 {
		index.ExtractComponentsFromRefs(ctx, poly)
	}

	index.ExtractExternalDocuments(index.root)
	index.GetPathCount()

	if !avoidBuildOut {
		index.BuildIndex()
	}
	<-index.nodeMapCompleted
	return index
}

func (index *SpecIndex) BuildIndex() {
	if index.built {
		return
	}
	countFuncs := []func() int{
		index.GetOperationCount,
		index.GetComponentSchemaCount,
		index.GetGlobalTagsCount,
		index.GetComponentParameterCount,
		index.GetOperationsParameterCount,
	}

	var wg sync.WaitGroup
	wg.Add(len(countFuncs))
	runIndexFunction(countFuncs, &wg)
	wg.Wait()

	countFuncs = []func() int{
		index.GetInlineUniqueParamCount,
		index.GetOperationTagsCount,
		index.GetGlobalLinksCount,
		index.GetGlobalCallbacksCount,
	}
	wg.Add(len(countFuncs))
	runIndexFunction(countFuncs, &wg)
	wg.Wait()

	index.GetInlineDuplicateParamCount()
	index.GetAllDescriptionsCount()
	index.GetTotalTagsCount()
	index.built = true
}

func (index *SpecIndex) GetLogger() *slog.Logger {
	return index.logger
}

func (index *SpecIndex) GetRootNode() *yaml.Node {
	return index.root
}

func (index *SpecIndex) SetRootNode(node *yaml.Node) {
	index.root = node
}

func (index *SpecIndex) GetRolodex() *Rolodex {
	return index.rolodex
}

func (index *SpecIndex) SetRolodex(rolodex *Rolodex) {
	index.rolodex = rolodex
}

func (index *SpecIndex) GetSpecFileName() string {
	if index == nil || index.rolodex == nil || index.rolodex.indexConfig == nil || index.rolodex.indexConfig.SpecFilePath == "" {
		return theoreticalRoot
	}
	return filepath.Base(index.rolodex.indexConfig.SpecFilePath)
}

func (index *SpecIndex) GetGlobalTagsNode() *yaml.Node {
	return index.tagsNode
}
