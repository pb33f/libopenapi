// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"context"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/speakeasy-api/jsonpath/pkg/jsonpath"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestRolodex_FindNodeOrigin_InRoot(t *testing.T) {
	baseDir := "rolodex_test_data"

	cf := CreateOpenAPIIndexConfig()
	cf.BasePath = baseDir
	cf.AvoidCircularReferenceCheck = true

	fileFS, err := NewLocalFSWithConfig(&LocalFSConfig{
		BaseDirectory: baseDir,
		IndexConfig:   cf,
	})
	if err != nil {
		t.Fatal(err)
	}

	rolo := NewRolodex(cf)
	rolo.AddLocalFS(baseDir, fileFS)

	// open doc2
	f, rerr := rolo.Open("doc2.yaml")
	assert.Nil(t, rerr)
	assert.NotNil(t, f)

	node, _ := f.GetContentAsYAMLNode()

	rolo.SetRootNode(node)

	err = rolo.IndexTheRolodex(context.Background())
	assert.NoError(t, err)
	rolo.Resolve()

	origin := rolo.FindNodeOrigin(node.Content[0])
	assert.NotNil(t, origin)
	assert.Equal(t, rolo.GetRootIndex().specAbsolutePath, origin.AbsoluteLocation)
}

func TestRolodex_FindNodeOrigin_InRoot_InMap(t *testing.T) {
	baseDir := "rolodex_test_data"

	cf := CreateOpenAPIIndexConfig()
	cf.BasePath = baseDir
	cf.AvoidCircularReferenceCheck = true

	fileFS, err := NewLocalFSWithConfig(&LocalFSConfig{
		BaseDirectory: baseDir,
		IndexConfig:   cf,
	})
	if err != nil {
		t.Fatal(err)
	}

	rolo := NewRolodex(cf)
	rolo.AddLocalFS(baseDir, fileFS)

	// open doc2
	f, rerr := rolo.Open("doc2.yaml")
	assert.Nil(t, rerr)
	assert.NotNil(t, f)

	node, _ := f.GetContentAsYAMLNode()

	rolo.SetRootNode(node)

	err = rolo.IndexTheRolodex(context.Background())
	assert.NoError(t, err)
	rolo.Resolve()

	node.Kind = yaml.MappingNode
	node.Tag = "!!map"

	copied := *node

	origin := rolo.FindNodeOrigin(copied.Content[0])
	assert.NotNil(t, origin)
	assert.Equal(t, rolo.GetRootIndex().specAbsolutePath, origin.AbsoluteLocation)
}

func TestRolodex_FindNodeOriginWithValue_NoKey(t *testing.T) {
	baseDir := "rolodex_test_data"

	cf := CreateOpenAPIIndexConfig()
	cf.BasePath = baseDir
	cf.AvoidCircularReferenceCheck = true

	fileFS, err := NewLocalFSWithConfig(&LocalFSConfig{
		BaseDirectory: baseDir,
		IndexConfig:   cf,
	})
	if err != nil {
		t.Fatal(err)
	}

	rolo := NewRolodex(cf)
	rolo.AddLocalFS(baseDir, fileFS)

	origin := rolo.FindNodeOriginWithValue(nil, nil, nil, "")

	assert.Nil(t, origin)
}

func TestRolodex_FindNodeOriginWithValue(t *testing.T) {
	baseDir := "rolodex_test_data"

	cf := CreateOpenAPIIndexConfig()
	cf.BasePath = baseDir
	cf.AvoidCircularReferenceCheck = true

	fileFS, err := NewLocalFSWithConfig(&LocalFSConfig{
		BaseDirectory: baseDir,
		IndexConfig:   cf,
	})
	if err != nil {
		t.Fatal(err)
	}

	rolo := NewRolodex(cf)
	rolo.AddLocalFS(baseDir, fileFS)

	// open doc2
	f, rerr := rolo.Open("doc2.yaml")
	assert.Nil(t, rerr)
	assert.NotNil(t, f)

	node, _ := f.GetContentAsYAMLNode()

	rolo.SetRootNode(node)

	err = rolo.IndexTheRolodex(context.Background())
	assert.NoError(t, err)
	rolo.Resolve()

	origin := rolo.FindNodeOriginWithValue(node.Content[0], node.Content[0], nil, "")

	assert.NotNil(t, origin)
	assert.Equal(t, rolo.GetRootIndex().specAbsolutePath, origin.AbsoluteLocation)
}

func TestRolodex_FindNodeOriginWithValue_SimulateIsRef(t *testing.T) {
	baseDir := "rolodex_test_data"

	cf := CreateOpenAPIIndexConfig()
	cf.BasePath = baseDir
	cf.AvoidCircularReferenceCheck = true

	fileFS, err := NewLocalFSWithConfig(&LocalFSConfig{
		BaseDirectory: baseDir,
		IndexConfig:   cf,
	})
	if err != nil {
		t.Fatal(err)
	}

	rolo := NewRolodex(cf)
	rolo.AddLocalFS(baseDir, fileFS)

	// open doc2
	f, rerr := rolo.Open("doc2.yaml")
	assert.Nil(t, rerr)
	assert.NotNil(t, f)

	node, _ := f.GetContentAsYAMLNode()

	rolo.SetRootNode(node)

	err = rolo.IndexTheRolodex(context.Background())
	assert.NoError(t, err)
	rolo.Resolve()

	origin := rolo.FindNodeOriginWithValue(node.Content[0], node.Content[0], node.Content[0], "burgers!")

	assert.NotNil(t, origin)
	assert.Equal(t, rolo.GetRootIndex().specAbsolutePath, origin.AbsoluteLocation)
	assert.Equal(t, "openapi", origin.Node.Content[0].Value) // key value.
}

func TestRolodex_FindNodeOriginWithValue_NonRoot(t *testing.T) {
	baseDir := "rolodex_test_data"

	cf := CreateOpenAPIIndexConfig()
	cf.BasePath = baseDir
	cf.AvoidCircularReferenceCheck = true

	fileFS, err := NewLocalFSWithConfig(&LocalFSConfig{
		BaseDirectory: baseDir,
		IndexConfig:   cf,
	})
	if err != nil {
		t.Fatal(err)
	}

	rolo := NewRolodex(cf)
	rolo.AddLocalFS(baseDir, fileFS)

	// open components
	f, rerr := rolo.Open("dir2/components.yaml")
	assert.Nil(t, rerr)
	assert.NotNil(t, f)

	// open doc2
	f2, ferr := rolo.Open("doc2.yaml")
	assert.Nil(t, ferr)
	assert.NotNil(t, f2)

	nodef, _ := f2.GetContentAsYAMLNode()
	node, _ := f.GetContentAsYAMLNode()

	rolo.SetRootNode(nodef)

	err = rolo.IndexTheRolodex(context.Background())
	assert.NoError(t, err)
	rolo.Resolve()

	origin := rolo.FindNodeOriginWithValue(node.Content[0].Content[2], node.Content[0].Content[3], nil, "")

	assert.NotNil(t, origin)

	assert.Equal(t, rolo.GetRootIndex().specAbsolutePath, origin.AbsoluteLocation)
	// should not be equal, root and origin are different
	assert.NotEqual(t, rolo.GetRootIndex().specAbsolutePath, origin.AbsoluteLocationValue)
	assert.Equal(t, 2, origin.Line)
	assert.Equal(t, 1, origin.Column)
	assert.Equal(t, 3, origin.LineValue)
	assert.Equal(t, 3, origin.ColumnValue)
}

func TestRolodex_FindNodeOriginWithValue_BadKeyAndValue(t *testing.T) {
	baseDir := "rolodex_test_data"

	cf := CreateOpenAPIIndexConfig()
	cf.BasePath = baseDir
	cf.AvoidCircularReferenceCheck = true

	fileFS, err := NewLocalFSWithConfig(&LocalFSConfig{
		BaseDirectory: baseDir,
		IndexConfig:   cf,
	})
	if err != nil {
		t.Fatal(err)
	}

	rolo := NewRolodex(cf)
	rolo.AddLocalFS(baseDir, fileFS)

	// open components
	f, rerr := rolo.Open("dir2/components.yaml")
	assert.Nil(t, rerr)
	assert.NotNil(t, f)

	// open doc2
	f2, ferr := rolo.Open("doc2.yaml")
	assert.Nil(t, ferr)
	assert.NotNil(t, f2)

	nodef, _ := f2.GetContentAsYAMLNode()

	rolo.SetRootNode(nodef)
	err = rolo.IndexTheRolodex(context.Background())
	rolo.Resolve()

	origin := rolo.FindNodeOriginWithValue(&yaml.Node{
		Kind:   yaml.ScalarNode,
		Value:  "burgers!",
		Line:   9999,
		Column: 9999,
	}, &yaml.Node{
		Kind:   yaml.ScalarNode,
		Value:  "fries and beer!",
		Line:   22222,
		Column: 232323,
	}, nil, "")

	assert.Nil(t, origin)
}

func TestRolodex_FindNodeOriginWithValue_BadValue(t *testing.T) {
	baseDir := "rolodex_test_data"

	cf := CreateOpenAPIIndexConfig()
	cf.BasePath = baseDir
	cf.AvoidCircularReferenceCheck = true

	fileFS, err := NewLocalFSWithConfig(&LocalFSConfig{
		BaseDirectory: baseDir,
		IndexConfig:   cf,
	})
	if err != nil {
		t.Fatal(err)
	}

	rolo := NewRolodex(cf)
	rolo.AddLocalFS(baseDir, fileFS)

	// open components
	f, rerr := rolo.Open("dir2/components.yaml")
	assert.Nil(t, rerr)
	assert.NotNil(t, f)

	// open doc2
	f2, ferr := rolo.Open("doc2.yaml")
	assert.Nil(t, ferr)
	assert.NotNil(t, f2)

	node, _ := f2.GetContentAsYAMLNode()

	rolo.SetRootNode(node)
	err = rolo.IndexTheRolodex(context.Background())
	rolo.Resolve()

	origin := rolo.FindNodeOriginWithValue(node.Content[0], &yaml.Node{
		Kind:   yaml.ScalarNode,
		Value:  "fries and beer!",
		Line:   22222,
		Column: 232323,
	}, nil, "")

	assert.Nil(t, origin)
}

func TestRolodex_FindNodeOrigin(t *testing.T) {
	baseDir := "rolodex_test_data"

	cf := CreateOpenAPIIndexConfig()
	cf.BasePath = baseDir
	cf.AvoidCircularReferenceCheck = true

	fileFS, err := NewLocalFSWithConfig(&LocalFSConfig{
		BaseDirectory: baseDir,
		IndexConfig:   cf,
	})
	if err != nil {
		t.Fatal(err)
	}

	rolo := NewRolodex(cf)
	rolo.AddLocalFS(baseDir, fileFS)

	// open doc2
	f, rerr := rolo.Open("doc2.yaml")
	assert.Nil(t, rerr)
	assert.NotNil(t, f)

	node, _ := f.GetContentAsYAMLNode()

	rolo.SetRootNode(node)

	err = rolo.IndexTheRolodex(context.Background())
	assert.NoError(t, err)
	rolo.Resolve()

	assert.Len(t, rolo.indexes, 4)

	// extract something that can only exist after resolution
	path := "$.paths['/nested/files3'].get.responses['200'].content['application/json'].schema.properties['message'].properties['utilMessage'].properties['message'].description"
	yp, _ := jsonpath.NewPath(path)
	results := yp.Query(node)

	assert.NotNil(t, results)
	assert.Len(t, results, 1)
	assert.Equal(t, "I am pointless dir2 utility, I am multiple levels deep.", results[0].Value)

	// now for the truth, where did this come from?
	origin := rolo.FindNodeOrigin(results[0])

	assert.NotNil(t, origin)
	sep := string(filepath.Separator)
	assert.True(t, strings.HasSuffix(origin.AbsoluteLocation, "index"+sep+
		"rolodex_test_data"+sep+"dir2"+sep+"utils"+sep+"utils.yaml"))

	// should be identical to the original node
	assert.Equal(t, results[0], origin.Node)

	// look for something that cannot exist
	origin = rolo.FindNodeOrigin(nil)
	assert.Nil(t, origin)

	// modify the node and try again
	m := *results[0]
	m.Value = "I am a new message"
	origin = rolo.FindNodeOrigin(&m)
	assert.Nil(t, origin)

	// extract the doc root
	origin = rolo.FindNodeOrigin(node)
	assert.Nil(t, origin)
}

func TestRolodex_FindNodeOrigin_Concurrent(t *testing.T) {
	var wg sync.WaitGroup
	failures := atomic.Int32{}
	iterations := 1000 // run this like mad.
	wg.Add(iterations)

	// Create a channel to collect detailed error information
	errorChan := make(chan string, iterations)

	for i := 0; i < iterations; i++ {
		go func(iteration int) {
			defer wg.Done()

			baseDir := "rolodex_test_data"

			cf := CreateOpenAPIIndexConfig()
			cf.BasePath = baseDir
			cf.AvoidCircularReferenceCheck = true

			fileFS, err := NewLocalFSWithConfig(&LocalFSConfig{
				BaseDirectory: baseDir,
				IndexConfig:   cf,
			})
			if err != nil {
				errorChan <- fmt.Sprintf("Iteration %d: Failed to create LocalFS: %v", iteration, err)
				failures.Add(1)
				return
			}

			rolo := NewRolodex(cf)
			rolo.AddLocalFS(baseDir, fileFS)

			// open doc2
			f, rerr := rolo.Open("doc2.yaml")
			if rerr != nil {
				errorChan <- fmt.Sprintf("Iteration %d: Failed to open doc2.yaml: %v", iteration, rerr)
				failures.Add(1)
				return
			}
			if f == nil {
				errorChan <- fmt.Sprintf("Iteration %d: File is nil", iteration)
				failures.Add(1)
				return
			}

			node, nodeErr := f.GetContentAsYAMLNode()
			if nodeErr != nil {
				errorChan <- fmt.Sprintf("Iteration %d: Failed to get YAML node: %v", iteration, nodeErr)
				failures.Add(1)
				return
			}

			rolo.SetRootNode(node)

			err = rolo.IndexTheRolodex(context.Background())
			if err != nil {
				errorChan <- fmt.Sprintf("Iteration %d: Failed to index rolodex: %v", iteration, err)
				failures.Add(1)
				return
			}

			rolo.Resolve()

			if len(rolo.indexes) != 4 {
				errorChan <- fmt.Sprintf("Iteration %d: Expected 4 indexes, got %d", iteration, len(rolo.indexes))
				failures.Add(1)
				return
			}

			// extract something that can only exist after resolution
			path := "$.paths['/nested/files3'].get.responses['200'].content['application/json'].schema.properties['message'].properties['utilMessage'].properties['message'].description"
			yp, _ := jsonpath.NewPath(path)
			results := yp.Query(node)

			if len(results) != 1 {
				errorChan <- fmt.Sprintf("Iteration %d: Expected 1 result, got %d", iteration, len(results))
				failures.Add(1)
				return
			}

			if results[0].Value != "I am pointless dir2 utility, I am multiple levels deep." {
				errorChan <- fmt.Sprintf("Iteration %d: Unexpected value: %s", iteration, results[0].Value)
				failures.Add(1)
				return
			}

			// now for the truth, where did this come from?
			origin := rolo.FindNodeOrigin(results[0])

			if origin == nil {
				errorChan <- fmt.Sprintf("Iteration %d: Origin is nil", iteration)
				failures.Add(1)
				return
			}

			sep := string(filepath.Separator)
			expectedSuffix := "index" + sep + "rolodex_test_data" + sep + "dir2" + sep + "utils" + sep + "utils.yaml"
			if !strings.HasSuffix(origin.AbsoluteLocation, expectedSuffix) {
				errorChan <- fmt.Sprintf("Iteration %d: Wrong suffix. Expected %s, got %s",
					iteration, expectedSuffix, origin.AbsoluteLocation)
				failures.Add(1)
				return
			}

		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errorChan)

	// Collect and report all errors
	var errorMessages []string
	for msg := range errorChan {
		errorMessages = append(errorMessages, msg)
	}

	if failures.Load() > 0 {
		t.Errorf("Test failed in %d/%d iterations with following errors:\n%s",
			failures.Load(), iterations, strings.Join(errorMessages, "\n"))
	}
}

func TestRolodex_FindNodeOrigin_ModifyLookup(t *testing.T) {
	baseDir := "rolodex_test_data"

	cf := CreateOpenAPIIndexConfig()
	cf.BasePath = baseDir
	cf.AvoidCircularReferenceCheck = true

	fileFS, err := NewLocalFSWithConfig(&LocalFSConfig{
		BaseDirectory: baseDir,
		IndexConfig:   cf,
	})
	if err != nil {
		t.Fatal(err)
	}

	rolo := NewRolodex(cf)
	rolo.AddLocalFS(baseDir, fileFS)

	// open doc2
	f, rerr := rolo.Open("doc2.yaml")
	assert.Nil(t, rerr)
	assert.NotNil(t, f)

	node, _ := f.GetContentAsYAMLNode()

	rolo.SetRootNode(node)

	err = rolo.IndexTheRolodex(context.Background())
	assert.NoError(t, err)
	rolo.Resolve()

	assert.Len(t, rolo.indexes, 4)

	path := "$.paths['/nested/files3'].get.responses['200'].content['application/json'].schema"
	yp, _ := jsonpath.NewPath(path)
	results := yp.Query(node)

	// copy, modify, and try again
	o := *results[0]
	o.Content = []*yaml.Node{
		{Value: "beer"}, {Value: "wine"}, {Value: "cake"}, {Value: "burgers"}, {Value: "herbs"}, {Value: "spices"},
	}
	origin := rolo.FindNodeOrigin(&o)
	assert.Nil(t, origin)
}

func TestSpecIndex_TestPathsAsRefWithFiles(t *testing.T) {
	// We're TDD'ing some code that previously had a race condition.
	// This test is to ensure that we don't regress.
	wg := sync.WaitGroup{}
	wg.Add(1000)
	for i := 0; i < 1000; i++ {
		func(i int) {

			yml := `paths:
  /test:
    $ref: 'rolodex_test_data/paths/paths.yaml#/~1some~1path'
  /test-2:
    $ref: './rolodex_test_data/paths/paths.yaml#/~1some~1path'
`

			baseDir := "."

			cf := CreateOpenAPIIndexConfig()
			cf.BasePath = baseDir
			cf.AvoidCircularReferenceCheck = true

			fileFS, err := NewLocalFSWithConfig(&LocalFSConfig{
				BaseDirectory: baseDir,
				IndexConfig:   cf,
			})
			if err != nil {
				t.Fatal(err)
			}

			rolo := NewRolodex(cf)
			rolo.AddLocalFS(baseDir, fileFS)

			var rootNode yaml.Node
			_ = yaml.Unmarshal([]byte(yml), &rootNode)

			rolo.SetRootNode(&rootNode)

			err = rolo.IndexTheRolodex(context.Background())

			assert.NoError(t, err)
			require.Len(t, rolo.indexes, 2)
			rolo.Resolve()

			require.Len(t, rolo.GetCaughtErrors(), 0)

			params := rolo.rootIndex.GetAllParametersFromOperations()
			require.Len(t, params, 2)
			lookupPath, ok := params["/test"]
			require.True(t, ok)
			lookupOperation, ok := lookupPath["get"]
			require.True(t, ok)
			require.Len(t, lookupOperation, 1)
			lookupRef, ok := lookupOperation["../components.yaml#/components/parameters/SomeParam"]
			require.True(t, ok)
			require.Len(t, lookupRef, 1)
			require.Equal(t, lookupRef[0].Name, "SomeParam")
			wg.Done()
		}(i)
	}
	wg.Wait()
}

func TestRolodex_FindNodeOrigin_NonRootToNonRootLookup(t *testing.T) {
	baseDir := "rolodex_test_data"

	cf := CreateOpenAPIIndexConfig()
	cf.BasePath = baseDir
	cf.AvoidCircularReferenceCheck = true

	fileFS, err := NewLocalFSWithConfig(&LocalFSConfig{
		BaseDirectory: baseDir,
		IndexConfig:   cf,
	})
	if err != nil {
		t.Fatal(err)
	}

	rolo := NewRolodex(cf)
	rolo.AddLocalFS(baseDir, fileFS)

	// open doc2
	f, rerr := rolo.Open("doc2.yaml")
	assert.Nil(t, rerr)
	assert.NotNil(t, f)

	node, _ := f.GetContentAsYAMLNode()

	// open dir2 components
	comp, cerr := rolo.Open("dir2/components.yaml")
	assert.Nil(t, cerr)
	assert.NotNil(t, comp)

	nodeComponents, _ := comp.GetContentAsYAMLNode()
	assert.NotNil(t, nodeComponents)

	// open utils
	utils, uerr := rolo.Open("dir2/utils/utils.yaml")
	assert.Nil(t, uerr)
	assert.NotNil(t, utils)

	nodeUtils, _ := utils.GetContentAsYAMLNode()
	assert.NotNil(t, nodeUtils)

	rolo.SetRootNode(node)
	// create a spec info
	b := []byte(f.GetContent())
	specInfo := &datamodel.SpecInfo{
		SpecBytes: &b,
	}
	cf.SpecInfo = specInfo

	key := nodeComponents.Content[0].Content[5].Content[1].Content[4]
	keyRef := nodeComponents.Content[0].Content[5].Content[1].Content[5]
	value := nodeUtils.Content[0]

	assert.NotNil(t, key)
	assert.NotNil(t, value)

	err = rolo.IndexTheRolodex(context.Background())

	origin := rolo.FindNodeOriginWithValue(key, value, nil, "")
	assert.NotNil(t, origin)
	assert.NoError(t, err)
	assert.Equal(t, 20, origin.Line)
	assert.Equal(t, 5, origin.Column)
	assert.Equal(t, 1, origin.LineValue)
	assert.Equal(t, 1, origin.ColumnValue)
	assert.NotEmpty(t, origin.AbsoluteLocation)
	assert.NotEmpty(t, origin.AbsoluteLocationValue)
	assert.NotEqual(t, origin.AbsoluteLocationValue, origin.AbsoluteLocation)

	// pretend that we have a reference.
	origin = rolo.FindNodeOriginWithValue(key, value, keyRef, "#/burgers")
	assert.NotNil(t, origin)
	assert.NoError(t, err)
	assert.Equal(t, 20, origin.Line)
	assert.Equal(t, 5, origin.Column)
	assert.Equal(t, 0, origin.LineValue)
	assert.Equal(t, 0, origin.ColumnValue)
	assert.NotEmpty(t, origin.AbsoluteLocation)
	assert.Empty(t, origin.AbsoluteLocationValue)

	origin = rolo.FindNodeOrigin(nodeUtils.Content[0].Content[0])
	assert.NotNil(t, origin)
	assert.Equal(t, 1, origin.Line)
	assert.Equal(t, 1, origin.Column)
	// get full line count.
	assert.Equal(t, int64(100), rolo.GetFullLineCount())
}
