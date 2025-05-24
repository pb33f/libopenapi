// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package bundler

import (
	"context"
	"errors"
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	v3low "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

func calculateCollisionName(name, pointer string) string {
	jsonPointer := strings.Split(pointer, "#/")
	if len(jsonPointer) == 2 {

		// TODO: make delimiter configurable.
		// count the number of collisions by splitting the name by the __ delimiter.
		nameSegments := strings.Split(name, "__")
		if len(nameSegments) > 1 {

			if len(nameSegments) == 2 {
				return fmt.Sprintf("%s__%s", name, "1")
			}
			if len(nameSegments) == 3 {
				count, err := strconv.Atoi(nameSegments[2])
				if err != nil {
					return fmt.Sprintf("%s__%s", name, "X")
				}
				count++
				nameSegments[2] = strconv.Itoa(count)
				return strings.Join(nameSegments, "__")
			}

		} else {

			// the first collision attempt will be to use the last segment of the location as a postfix.
			// this will be the last segment of the path.
			uri := jsonPointer[0]
			b := filepath.Base(uri)
			fileName := fmt.Sprintf("%s__%s", name, strings.Replace(b, filepath.Ext(b), "", 1))
			return fileName

		}

	}

	// TODO: handle full file imports.
	return name
}

func checkReferenceAndBubbleUp[T any](
	name string,
	pr *processRef,
	idx *index.SpecIndex,
	componentMap *orderedmap.Map[string, T],
	buildFunc func(node *yaml.Node, idx *index.SpecIndex) (T, error),
) error {
	// Build the component
	component, err := buildFunc(pr.ref.Node, idx)
	if err != nil {
		idx.GetLogger().Error("[bundler] unable to build component", "error", err)
		return err
	}

	// Handle potential collisions and add to the component map
	if v := componentMap.GetOrZero(name); !isZeroOfType(v) {
		uniqueName := handleCollision(name, pr, componentMap)
		componentMap.Set(uniqueName, component)
	} else {
		componentMap.Set(name, component)
	}

	return nil
}

func isZeroOfType[T any](v T) bool {
	isZero := reflect.ValueOf(v).IsZero()
	return isZero
}

func handleCollision[T any](schemaName string, pr *processRef, componentsItem *orderedmap.Map[string, T]) string {
	foundUnique := false
	uniqueName := schemaName
	for !foundUnique {
		uniqueName = calculateCollisionName(uniqueName, pr.ref.FullDefinition)
		if v := componentsItem.GetOrZero(uniqueName); isZeroOfType(v) {
			foundUnique = true
		}

	}
	pr.name = uniqueName
	return uniqueName
}

func remapIndex(idx *index.SpecIndex, processedNodes *orderedmap.Map[string, *processRef]) {

	seq := idx.GetRawReferencesSequenced()
	for _, sequenced := range seq {
		rewireRef(sequenced, sequenced.FullDefinition, processedNodes)
	}
	mapped := idx.GetMappedReferences()
	reMapped := make(map[string]*index.Reference)
	for _, mRef := range mapped {
		rewireRef(mRef, mRef.FullDefinition, processedNodes)
		reMapped[mRef.FullDefinition] = mRef
	}
	idx.SetMappedReferences(reMapped)

}

func renameRef(def string, processedNodes *orderedmap.Map[string, *processRef]) string {

	defSplit := strings.Split(def, "#/")
	if len(defSplit) != 2 {
		return def
	}

	split := strings.Split(defSplit[1], "/")
	return fmt.Sprintf("#/%s/%s", strings.Join(split[:len(split)-1], "/"), processedNodes.GetOrZero(def).name)
}

func rewireRef(ref *index.Reference, fullDef string, processedNodes *orderedmap.Map[string, *processRef]) {

	isRef, _, _ := utils.IsNodeRefValue(ref.Node)
	rename := renameRef(fullDef, processedNodes)
	if isRef {
		ref.Node.Content[1].Value = rename
		ref.FullDefinition = ref.Node.Content[1].Value
	} else {
		ref.FullDefinition = rename
	}
}

func buildComponents(idx *index.SpecIndex) (*v3.Components, error) {
	comp := v3low.Components{}
	_ = low.BuildModel(&yaml.Node{}, &comp)
	ctx := context.Background()
	err := comp.Build(ctx, &yaml.Node{}, idx)
	return v3.NewComponents(&comp), err
}

func buildSchema(node *yaml.Node, idx *index.SpecIndex) (*base.SchemaProxy, error) {
	if node == nil {
		return nil, errors.New("node is nil")
	}
	schema := lowbase.Schema{}
	_ = low.BuildModel(node, &schema)
	ctx := context.Background()
	err := schema.Build(ctx, node, idx)

	var sch lowbase.SchemaProxy
	err = sch.Build(ctx, &yaml.Node{}, node, idx)
	r := &low.NodeReference[*lowbase.SchemaProxy]{Value: &sch}
	highSchemaProxy := base.NewSchemaProxy(r)
	return highSchemaProxy, err
}

func buildResponse(node *yaml.Node, idx *index.SpecIndex) (*v3.Response, error) {
	resp := v3low.Response{}
	_ = low.BuildModel(node, &resp)
	ctx := context.Background()
	err := resp.Build(ctx, &yaml.Node{}, node, idx)
	return v3.NewResponse(&resp), err
}

func buildParameter(node *yaml.Node, idx *index.SpecIndex) (*v3.Parameter, error) {
	param := v3low.Parameter{}
	_ = low.BuildModel(node, &param)
	ctx := context.Background()
	err := param.Build(ctx, &yaml.Node{}, node, idx)
	return v3.NewParameter(&param), err
}

func buildHeader(node *yaml.Node, idx *index.SpecIndex) (*v3.Header, error) {
	header := v3low.Header{}
	_ = low.BuildModel(node, &header)
	ctx := context.Background()
	err := header.Build(ctx, &yaml.Node{}, node, idx)
	return v3.NewHeader(&header), err
}

func buildRequestBody(node *yaml.Node, idx *index.SpecIndex) (*v3.RequestBody, error) {
	requestBody := v3low.RequestBody{}
	_ = low.BuildModel(node, &requestBody)
	ctx := context.Background()
	err := requestBody.Build(ctx, &yaml.Node{}, node, idx)
	return v3.NewRequestBody(&requestBody), err
}

func buildExample(node *yaml.Node, idx *index.SpecIndex) (*base.Example, error) {
	example := lowbase.Example{}
	_ = low.BuildModel(node, &example)
	ctx := context.Background()
	err := example.Build(ctx, &yaml.Node{}, node, idx)
	return base.NewExample(&example), err
}

func buildLink(node *yaml.Node, idx *index.SpecIndex) (*v3.Link, error) {
	link := v3low.Link{}
	_ = low.BuildModel(node, &link)
	ctx := context.Background()
	err := link.Build(ctx, &yaml.Node{}, node, idx)
	return v3.NewLink(&link), err
}

func buildCallback(node *yaml.Node, idx *index.SpecIndex) (*v3.Callback, error) {
	callback := v3low.Callback{}
	_ = low.BuildModel(node, &callback)
	ctx := context.Background()
	err := callback.Build(ctx, &yaml.Node{}, node, idx)
	return v3.NewCallback(&callback), err
}

func buildPathItem(node *yaml.Node, idx *index.SpecIndex) (*v3.PathItem, error) {
	pathItem := v3low.PathItem{}
	_ = low.BuildModel(node, &pathItem)
	ctx := context.Background()
	err := pathItem.Build(ctx, &yaml.Node{}, node, idx)
	return v3.NewPathItem(&pathItem), err
}
