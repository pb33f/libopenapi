// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package index

import (
	"sync"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

func TestLocalFileIndexPreservesPreviouslyReadNodes(t *testing.T) {
	for _, source := range []string{"type: object\nproperties:\n  name:\n    type: string\n", `{"type":"object","properties":{"name":{"type":"string"}}}`} {
		file := &LocalFile{data: []byte(source), fullPath: "schema.yaml"}
		before, err := file.GetContentAsYAMLNode()
		require.NoError(t, err)
		config := CreateOpenAPIIndexConfig()
		config.AvoidBuildIndex = true
		indexed, err := file.Index(config)
		require.NoError(t, err)
		after, err := file.GetContentAsYAMLNode()
		require.NoError(t, err)
		require.Same(t, before, indexed.GetRootNode())
		require.Same(t, before, after)
		require.Same(t, before, config.SpecInfo.RootNode)
	}
}

func TestLocalFileConcurrentReadAndIndexShareNodes(t *testing.T) {
	file := &LocalFile{data: []byte("type: object\nproperties:\n  name:\n    type: string\n"), fullPath: "schema.yaml"}
	start := make(chan struct{})
	nodes := make(chan *yaml.Node, 16)
	failures := make(chan error, 16)
	var workers sync.WaitGroup
	for i := 0; i < 16; i++ {
		workers.Add(1)
		go func(index bool) {
			defer workers.Done()
			<-start
			if index {
				config := CreateOpenAPIIndexConfig()
				config.AvoidBuildIndex = true
				idx, err := file.Index(config)
				if err != nil {
					failures <- err
					return
				}
				nodes <- idx.GetRootNode()
			} else {
				node, err := file.GetContentAsYAMLNode()
				if err != nil {
					failures <- err
					return
				}
				nodes <- node
			}
		}(i%2 == 0)
	}
	close(start)
	workers.Wait()
	close(nodes)
	close(failures)
	for err := range failures {
		require.NoError(t, err)
	}
	for node := range nodes {
		require.Same(t, file.GetIndex().GetRootNode(), node)
	}
}

func TestLocalFileIndexRecoversJSONAfterEarlyParseFailure(t *testing.T) {
	for _, test := range []struct{ source, value string }{
		{`{"type":"string","description":"a\/b"}`, "a/b"},
		{`{"type":"string","description":"\uD83D\uDE00"}`, "😀"},
	} {
		file := &LocalFile{data: []byte(test.source), fullPath: "schema.json"}
		early, err := file.GetContentAsYAMLNode()
		require.Error(t, err)
		require.Equal(t, yaml.ScalarNode, early.Content[0].Kind)
		config := CreateOpenAPIIndexConfig()
		config.AvoidBuildIndex = true
		indexed, err := file.Index(config)
		require.NoError(t, err)
		root := indexed.GetRootNode()
		require.Equal(t, yaml.MappingNode, root.Content[0].Kind)
		var schema map[string]any
		require.NoError(t, root.Decode(&schema))
		require.Equal(t, test.value, schema["description"])
		after, err := file.GetContentAsYAMLNode()
		require.NoError(t, err)
		require.Same(t, root, after)
	}
}
