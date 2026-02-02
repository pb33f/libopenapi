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

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	v3low "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"go.yaml.in/yaml/v4"
)

// extractFragment returns the JSON pointer fragment from a full definition.
// e.g., "file.yaml#/components/schemas/Pet" → "#/components/schemas/Pet"
func extractFragment(fullDef string) string {
	if idx := strings.Index(fullDef, "#/"); idx != -1 {
		return fullDef[idx:]
	}
	return "#/"
}

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
	// preserve original name before collision handling (unless already set)
	if pr != nil && pr.originalName == "" {
		pr.originalName = name
	}

	component, err := buildFunc(pr.ref.Node, idx)
	if err != nil {
		return err
	}

	wasRenamed := false

	// Handle potential collisions and add to the component map
	if v := componentMap.GetOrZero(name); !isZeroOfType(v) {
		uniqueName := handleCollision(name, delimiter, pr, componentMap)
		componentMap.Set(uniqueName, component)
		wasRenamed = true
		name = uniqueName
	} else {
		componentMap.Set(name, component)
	}

	// update final name and renamed flag (preserve existing wasRenamed=true if already set)
	if pr != nil {
		pr.name = name
		// only update wasRenamed if it's being set to true, or if it wasn't already true
		if wasRenamed || !pr.wasRenamed {
			pr.wasRenamed = wasRenamed
		}
	}

	return nil
}

// checkReferenceAndCapture combines reference building and origin tracking.
// eliminates duplication of the check-capture-return pattern used throughout processReference.
func checkReferenceAndCapture[T any](
	name, delimiter, componentType string,
	pr *processRef,
	idx *index.SpecIndex,
	componentMap *orderedmap.Map[string, T],
	buildFunc func(node *yaml.Node, idx *index.SpecIndex) (T, error),
	origins ComponentOriginMap,
) error {
	err := checkReferenceAndBubbleUp(name, delimiter, pr, idx, componentMap, buildFunc)
	if err == nil && origins != nil {
		captureOrigin(pr, componentType, origins)
	}
	return err
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
	pr.wasRenamed = true
	return uniqueName
}

func handleFileImport[T any](pr *processRef, importType, delimiter string, components *orderedmap.Map[string, T]) []string {
	// extract base name from file before collision handling
	// First try pr.ref.Name, then fall back to extracting from FullDefinition
	refName := pr.ref.Name
	if refName == "" {
		// For bare file refs, extract name from FullDefinition path
		refName = pr.ref.FullDefinition
		// Remove any fragment
		if idx := strings.Index(refName, "#"); idx != -1 {
			refName = refName[:idx]
		}
	}
	baseName := filepath.Base(strings.Replace(refName, filepath.Ext(refName), "", 1))

	// preserve original name before any renaming
	if pr.originalName == "" {
		pr.originalName = baseName
	}

	// check for collisions and get final name
	name := checkForCollision(baseName, delimiter, pr, components)

	// detect if renaming occurred
	if name != baseName {
		pr.wasRenamed = true
	}

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

// encodeJSONPointerSegment encodes a string for use in a JSON Pointer per RFC 6901.
// The escape sequence is: ~ → ~0, / → ~1 (order matters: ~ must be escaped first).
func encodeJSONPointerSegment(s string) string {
	if !strings.ContainsAny(s, "~/") {
		return s
	}
	s = strings.ReplaceAll(s, "~", "~0")
	s = strings.ReplaceAll(s, "/", "~1")
	return s
}

// joinLocationAsJSONPointer joins location segments into a JSON Pointer,
// properly encoding each segment per RFC 6901.
func joinLocationAsJSONPointer(location []string) string {
	encoded := make([]string, len(location))
	for i, seg := range location {
		encoded[i] = encodeJSONPointerSegment(seg)
	}
	return strings.Join(encoded, "/")
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
			// check if this single-segment pointer was processed and has a location
			if pr := processedNodes.GetOrZero(def); pr != nil && len(pr.location) > 0 {
				return "#/" + joinLocationAsJSONPointer(pr.location)
			}
			return def
		}
		prefix := strings.Join(segs[:len(segs)-1], "/")

		// reference already renamed during composition
		if pr := processedNodes.GetOrZero(def); pr != nil {
			return fmt.Sprintf("#/%s/%s", prefix, encodeJSONPointerSegment(pr.name))
		}

		if idx != nil {
			if ref, ok := idx.GetMappedReferences()[def]; ok && ref != nil {
				return fmt.Sprintf("#/%s/%s", prefix, encodeJSONPointerSegment(ref.Name))
			}
		}

		// fallback – keep last segment
		return fmt.Sprintf("#/%s/%s", prefix, segs[len(segs)-1])
	}

	// root-file import lifted into components
	if pn := processedNodes.GetOrZero(def); pn != nil && len(pn.location) > 0 {
		return "#/" + joinLocationAsJSONPointer(pn.location)
	}

	return def
}

func rewireRef(idx *index.SpecIndex, ref *index.Reference, fullDef string, processedNodes *orderedmap.Map[string, *processRef]) {
	isRef, _, _ := utils.IsNodeRefValue(ref.Node)

	// extract the pr from the processed nodes.
	if pr := processedNodes.GetOrZero(fullDef); pr != nil {
		if kk, _, _ := utils.IsNodeRefValue(pr.ref.Node); kk {
			if pr.refPointer == "" {
				// Use GetRefValueNode to handle OA 3.1 sibling properties correctly
				if refValNode := utils.GetRefValueNode(pr.ref.Node); refValNode != nil {
					pr.refPointer = refValNode.Value
				}
			}
		}
	}

	rename := renameRef(idx, fullDef, processedNodes)
	if isRef {
		// Use GetRefValueNode to find the correct $ref value node
		// This handles OA 3.1 sibling properties where $ref may not be at index 0
		if refValNode := utils.GetRefValueNode(ref.Node); refValNode != nil {
			if refValNode.Value != rename {
				refValNode.Value = rename
			}
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

// captureOrigin records origin information for a processed reference.
// enables navigation from bundled components back to their source files.
func captureOrigin(pr *processRef, componentType string, origins ComponentOriginMap) {
	if pr == nil || pr.ref == nil || pr.idx == nil || origins == nil {
		return
	}

	originalRef := extractFragment(pr.ref.FullDefinition)

	originalName := pr.originalName
	if originalName == "" {
		originalName = pr.name
	}

	// pr.name is updated by checkReferenceAndBubbleUp after collision handling
	bundledRef := "#/components/" + componentType + "/" + pr.name

	origin := &ComponentOrigin{
		OriginalFile:  pr.idx.GetSpecAbsolutePath(),
		OriginalRef:   originalRef,
		OriginalName:  originalName,
		Line:          pr.ref.Node.Line,
		Column:        pr.ref.Node.Column,
		WasRenamed:    pr.wasRenamed,
		BundledRef:    bundledRef,
		ComponentType: componentType,
	}

	origins[bundledRef] = origin
}

// rewriteAllRefs walks an index's document tree and rewrites un-re-written $ref values.
// Must be called as the FINAL step after all other processing.
func rewriteAllRefs(
	idx *index.SpecIndex,
	processedNodes *orderedmap.Map[string, *processRef],
	rolodex *index.Rolodex,
) {
	walkAndRewriteRefs(idx.GetRootNode(), idx, processedNodes, rolodex, false)
}

func walkAndRewriteRefs(
	node *yaml.Node,
	sourceIdx *index.SpecIndex,
	processedNodes *orderedmap.Map[string, *processRef],
	rolodex *index.Rolodex,
	inExtension bool, // Tracks if we're under an x-* key
) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) > 0 {
			walkAndRewriteRefs(node.Content[0], sourceIdx, processedNodes, rolodex, inExtension)
		}
		return
	case yaml.SequenceNode:
		for _, child := range node.Content {
			walkAndRewriteRefs(child, sourceIdx, processedNodes, rolodex, inExtension)
		}
		return
	case yaml.MappingNode:
		// Continue below
	default:
		return
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		// Track extension scope
		childInExtension := inExtension || strings.HasPrefix(keyNode.Value, "x-")

		if keyNode.Value == "$ref" && valueNode.Kind == yaml.ScalarNode && !inExtension {
			newRef := resolveRefToComposed(valueNode.Value, sourceIdx, processedNodes, rolodex)
			if newRef != valueNode.Value {
				valueNode.Value = newRef
			}
		} else {
			walkAndRewriteRefs(valueNode, sourceIdx, processedNodes, rolodex, childInExtension)
		}
	}
}

func resolveRefToComposed(
	refValue string,
	sourceIdx *index.SpecIndex,
	processedNodes *orderedmap.Map[string, *processRef],
	rolodex *index.Rolodex,
) string {
	// Skip external URLs and URNs
	if strings.HasPrefix(refValue, "http://") ||
		strings.HasPrefix(refValue, "https://") ||
		strings.HasPrefix(refValue, "urn:") {
		return refValue
	}

	// Use source index for relative path resolution
	ref, refIdx := sourceIdx.SearchIndexForReference(refValue)
	if ref == nil {
		ref, refIdx = rolodex.GetRootIndex().SearchIndexForReference(refValue)
	}
	if ref == nil {
		for _, idx := range rolodex.GetIndexes() {
			if r, i := idx.SearchIndexForReference(refValue); r != nil {
				ref, refIdx = r, i
				break
			}
		}
	}

	// fallback for unindexed local refs (root cause #4):
	// If the ref starts with #/ but SearchIndexForReference couldn't resolve it,
	// try renameRef directly - it may match a processedNodes entry.
	if ref == nil && strings.HasPrefix(refValue, "#/") {
		// Build absolute key for lookup: sourceIdx path + refValue
		absKey := sourceIdx.GetSpecAbsolutePath() + refValue
		// Gate on processedNodes presence - only rewrite if target was composed
		if processedNodes.GetOrZero(absKey) == nil {
			return refValue
		}
		return renameRef(sourceIdx, absKey, processedNodes)
	}

	if ref == nil || refIdx == nil {
		return refValue
	}

	// SearchIndexForReference returns a Reference with a potentially relative FullDefinition.
	// But processedNodes keys are absolute paths. We need to construct the absolute key
	// using the returned index's path + the fragment from the reference.
	// Format: /abs/path/to/file.yaml#/components/schemas/Name
	absoluteKey := ref.FullDefinition
	fragment := ""
	if idx := strings.Index(ref.FullDefinition, "#"); idx != -1 {
		fragment = ref.FullDefinition[idx:]
	}

	if !filepath.IsAbs(absoluteKey) && refIdx != nil {
		// Build absolute key
		absoluteKey = refIdx.GetSpecAbsolutePath() + fragment
	}

	// If the ref resolves to the ROOT index, and it's a canonical location (#/components/...) ref,
	// we should rewrite it to a local component ref. Root document components are NOT in
	// processedNodes (only external refs are), but they're valid targets.
	rootIdx := rolodex.GetRootIndex()
	if refIdx != nil && refIdx.GetSpecAbsolutePath() == rootIdx.GetSpecAbsolutePath() {
		// This ref points to the root document
		if fragment != "" && strings.HasPrefix(fragment, "#/") {
			// Return the fragment as-is - it's already a valid local ref
			return fragment
		}
	}

	// For non-root refs, gate rewrites on processedNodes presence.
	// Only rewrite if the target was actually composed into the bundled output.
	// This prevents dangling refs when SearchIndexForReference resolves something
	// that never made it into processedNodes.
	if processedNodes.GetOrZero(absoluteKey) == nil {
		return refValue
	}

	// Use renameRef() which handles collision renames
	return renameRef(refIdx, absoluteKey, processedNodes)
}
