// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package bundler

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/pkg-base/libopenapi/datamodel/high/base"
	v3 "github.com/pkg-base/libopenapi/datamodel/high/v3"
	"github.com/pkg-base/libopenapi/datamodel/low"
	lowbase "github.com/pkg-base/libopenapi/datamodel/low/base"
	v3low "github.com/pkg-base/libopenapi/datamodel/low/v3"
	"github.com/pkg-base/libopenapi/index"
	"github.com/pkg-base/libopenapi/orderedmap"
	"github.com/pkg-base/libopenapi/utils"
	"github.com/pkg-base/yaml"
)

func calculateCollisionName(name, pointer, delimiter string, iteration int) string {
	jsonPointer := strings.Split(pointer, "#/")
	if len(jsonPointer) == 2 {

		// count the number of collisions by splitting the name by the __ delimiter.
		nameSegments := strings.Split(name, delimiter)
		if len(nameSegments) > 1 {

			if len(nameSegments) == 2 {
				return fmt.Sprintf("%s%s%s", name, delimiter, "1")
			}
			if len(nameSegments) == 3 {
				count, _ := strconv.Atoi(nameSegments[2])
				count++
				nameSegments[2] = strconv.Itoa(count)
				return strings.Join(nameSegments, delimiter)
			}

		} else {

			// the first collision attempt will be to use the last segment of the location as a postfix.
			// this will be the last segment of the path.
			uri := jsonPointer[0]
			b := filepath.Base(uri)
			fileName := fmt.Sprintf("%s%s%s", name, delimiter, strings.Replace(b, filepath.Ext(b), "", 1))
			return fileName

		}
	}

	// split a path into segments and then create a new name by appending the iteration count.
	segments := strings.Split(utils.ReplaceWindowsDriveWithLinuxPath(filepath.Dir(pointer)), "/")
	if len(segments) > 0 {
		if iteration < len(segments) {

			lastSegment := segments[len(segments)-(iteration)]

			// split the name by the delimiter and append the last segment of the path
			nameSegments := strings.Split(name, delimiter)
			if len(nameSegments) > 1 {
				if len(nameSegments) <= iteration {
					name = fmt.Sprintf("%s%s%s", name, delimiter, lastSegment)
				}
			} else {
				name = fmt.Sprintf("%s%s%s", name, delimiter, lastSegment)
			}
		} else {
			name = fmt.Sprintf("%s%s%s", name, delimiter, utils.GenerateAlphanumericString(4))
		}
	}
	return name
}

func checkReferenceAndBubbleUp[T any](
	name, delimiter string,
	pr *processRef,
	idx *index.SpecIndex,
	componentMap *orderedmap.Map[string, T],
	buildFunc func(node *yaml.Node, idx *index.SpecIndex) (T, error),
) error {
	// Build the component
	component, err := buildFunc(pr.ref.Node, idx)
	if err != nil {
		return err
	}

	// Handle potential collisions and add to the component map
	if v := componentMap.GetOrZero(name); !isZeroOfType(v) {
		uniqueName := handleCollision(name, delimiter, pr, componentMap)
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

func handleCollision[T any](schemaName, delimiter string, pr *processRef, componentsItem *orderedmap.Map[string, T]) string {
	foundUnique := false
	uniqueName := schemaName
	iterations := 0
	for !foundUnique {
		iterations++
		uniqueName = calculateCollisionName(uniqueName, pr.ref.FullDefinition, delimiter, iterations)
		if v := componentsItem.GetOrZero(uniqueName); isZeroOfType(v) {
			foundUnique = true
		}

	}
	pr.name = uniqueName
	return uniqueName
}

func handleFileImport[T any](pr *processRef, importType, delimiter string, components *orderedmap.Map[string, T]) []string {
	name := checkForCollision(filepath.Base(strings.Replace(pr.ref.Name, filepath.Ext(pr.ref.Name), "", 1)), delimiter, pr, components)
	pr.name = name
	pr.ref.Name = name
	pr.seqRef.Name = name
	return []string{v3low.ComponentsLabel, importType, name}
}

func checkForCollision[T any](schemaName, delimiter string, pr *processRef, componentsItem *orderedmap.Map[string, T]) string {
	if v := componentsItem.GetOrZero(schemaName); !isZeroOfType(v) {
		return handleCollision(schemaName, delimiter, pr, componentsItem)
	}
	return schemaName
}

func remapIndex(idx *index.SpecIndex, processedNodes *orderedmap.Map[string, *processRef]) {
	seq := idx.GetRawReferencesSequenced()
	for _, sequenced := range seq {
		rewireRef(idx, sequenced, sequenced.FullDefinition, processedNodes)
	}

	mapped := idx.GetMappedReferences()

	for _, mRef := range mapped {
		origDef := mRef.FullDefinition
		rewireRef(idx, mRef, mRef.FullDefinition, processedNodes)
		mapped[mRef.FullDefinition] = mRef
		mapped[origDef] = mRef
	}
}

func renameRef(idx *index.SpecIndex, def string, processedNodes *orderedmap.Map[string, *processRef]) string {
	if strings.Contains(def, "#/") {
		defSplit := strings.Split(def, "#/")
		if len(defSplit) != 2 {
			return def
		}
		ptr := defSplit[1]
		segs := strings.Split(ptr, "/")
		if len(segs) < 2 {
			return def
		}
		prefix := strings.Join(segs[:len(segs)-1], "/")

		// reference already renamed during composition
		if pr := processedNodes.GetOrZero(def); pr != nil {
			return fmt.Sprintf("#/%s/%s", prefix, pr.name)
		}

		if idx != nil {
			if ref, ok := idx.GetMappedReferences()[def]; ok && ref != nil {
				return fmt.Sprintf("#/%s/%s", prefix, ref.Name)
			}
		}

		// fallback – keep last segment
		return fmt.Sprintf("#/%s/%s", prefix, segs[len(segs)-1])
	}

	// root-file import lifted into components
	if pn := processedNodes.GetOrZero(def); pn != nil && len(pn.location) > 0 {
		return "#/" + strings.Join(pn.location, "/")
	}

	return def
}

func rewireRef(idx *index.SpecIndex, ref *index.Reference, fullDef string, processedNodes *orderedmap.Map[string, *processRef]) {
	isRef, _, _ := utils.IsNodeRefValue(ref.Node)

	// extract the pr from the processed nodes.
	if pr := processedNodes.GetOrZero(fullDef); pr != nil {
		if kk, _, _ := utils.IsNodeRefValue(pr.ref.Node); kk {
			if pr.refPointer == "" {
				pr.refPointer = pr.ref.Node.Content[1].Value
			}
		}
	}

	rename := renameRef(idx, fullDef, processedNodes)
	if isRef {

		if ref.Node.Content[1].Value != rename {
			ref.Node.Content[1].Value = rename
		}
		ref.FullDefinition = rename
		ref.Definition = rename
	} else {
		ref.FullDefinition = rename
		ref.Definition = rename
	}
}

func buildComponents(idx *index.SpecIndex) (*v3.Components, error) {
	if idx == nil {
		return nil, errors.New("index is nil")
	}
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
