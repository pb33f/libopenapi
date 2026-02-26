// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pb33f/libopenapi/arazzo/expression"
	high "github.com/pb33f/libopenapi/datamodel/high/arazzo"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
	low "github.com/pb33f/libopenapi/datamodel/low/arazzo"
	"go.yaml.in/yaml/v4"
)

// lowNodePos extracts line and column from a *yaml.Node, returning (0, 0) if nil.
func lowNodePos(n *yaml.Node) (int, int) {
	if n == nil {
		return 0, 0
	}
	return n.Line, n.Column
}

// rootPos returns line/col from a low-level model's RootNode.
// The getter parameter avoids typed-nil interface issues by only calling the
// getter when the caller has already nil-checked the low-level model pointer.
func rootPos[T any](low *T, getRootNode func(*T) *yaml.Node) (int, int) {
	if low == nil {
		return 0, 0
	}
	return lowNodePos(getRootNode(low))
}

var componentKeyRegex = regexp.MustCompile(`^[a-zA-Z0-9.\-_]+$`)
var sourceDescriptionNameRegex = regexp.MustCompile(`^[A-Za-z0-9_\-]+$`)

// Validate performs structural validation of an Arazzo document.
// Returns nil if the document is valid; callers should nil-check the result
// before accessing Errors or Warnings.
func Validate(doc *high.Arazzo) *ValidationResult {
	v := &validator{
		doc:    doc,
		result: &ValidationResult{},
	}
	if doc == nil {
		v.addError("document", 0, 0, ErrInvalidArazzo)
		return v.result
	}
	v.validate()
	if v.result.HasErrors() || v.result.HasWarnings() {
		return v.result
	}
	return nil
}

type validator struct {
	doc      *high.Arazzo
	result   *ValidationResult
	opLookup *operationResolver
}

func (v *validator) addError(path string, line, col int, cause error) {
	v.result.Errors = append(v.result.Errors, &ValidationError{
		Path:   path,
		Line:   line,
		Column: col,
		Cause:  cause,
	})
}

func (v *validator) addWarning(path string, line, col int, msg string) {
	v.result.Warnings = append(v.result.Warnings, &Warning{
		Path:    path,
		Line:    line,
		Column:  col,
		Message: msg,
	})
}

func (v *validator) validate() {
	// Rule 1: Arazzo version
	v.checkVersion()

	// Rule 2: Required fields
	v.checkRequiredFields()

	if v.doc.Info == nil || len(v.doc.SourceDescriptions) == 0 || len(v.doc.Workflows) == 0 {
		return // Can't validate further without required fields
	}

	// Rule 3: Unique IDs
	v.checkUniqueSourceDescNames()
	v.checkUniqueWorkflowIds()
	v.buildOperationLookupContext()

	// Rules 4-21: Workflow-level validation
	workflowIds := v.buildWorkflowIdSet()
	for i, wf := range v.doc.Workflows {
		v.validateWorkflow(wf, i, workflowIds)
	}

	// Rule 9: Circular dependency detection
	v.checkCircularDependencies()

	// Rule 10: Component key validation
	v.validateComponentKeys()
}

func (v *validator) checkVersion() {
	if v.doc.Arazzo == "" {
		v.addError("arazzo", 0, 0, ErrMissingArazzoField)
		return
	}
	var line, col int
	if l := v.doc.GoLow(); l != nil {
		line, col = lowNodePos(l.Arazzo.ValueNode)
	}
	// Accept 1.0.x versions
	if !strings.HasPrefix(v.doc.Arazzo, "1.0.") {
		v.addError("arazzo", line, col, fmt.Errorf("unsupported arazzo version %q, expected 1.0.x", v.doc.Arazzo))
	}
}

func (v *validator) checkRequiredFields() {
	if v.doc.Info == nil {
		v.addError("info", 0, 0, ErrMissingInfo)
	} else {
		infoLine, infoCol := rootPos(v.doc.Info.GoLow(), (*low.Info).GetRootNode)
		if v.doc.Info.Title == "" {
			v.addError("info.title", infoLine, infoCol, fmt.Errorf("missing required 'title' in info"))
		}
		if v.doc.Info.Version == "" {
			v.addError("info.version", infoLine, infoCol, fmt.Errorf("missing required 'version' in info"))
		}
	}
	if len(v.doc.SourceDescriptions) == 0 {
		v.addError("sourceDescriptions", 0, 0, ErrMissingSourceDescriptions)
	}
	if len(v.doc.Workflows) == 0 {
		v.addError("workflows", 0, 0, ErrMissingWorkflows)
	}
}

func (v *validator) checkUniqueSourceDescNames() {
	seen := make(map[string]bool)
	for i, sd := range v.doc.SourceDescriptions {
		path := fmt.Sprintf("sourceDescriptions[%d]", i)
		line, col := rootPos(sd.GoLow(), (*low.SourceDescription).GetRootNode)
		if sd.Name == "" {
			v.addError(path+".name", line, col, fmt.Errorf("missing required 'name'"))
			continue
		}
		if seen[sd.Name] {
			v.addError(path+".name", line, col, fmt.Errorf("duplicate sourceDescription name %q", sd.Name))
		}
		seen[sd.Name] = true

		// Rule 13: Name format recommendation (warning only)
		if !sourceDescriptionNameRegex.MatchString(sd.Name) {
			v.addWarning(path+".name", line, col, fmt.Sprintf("sourceDescription name %q should match [A-Za-z0-9_-]+", sd.Name))
		}

		// Rule 13a: Type validation
		if sd.Type != "" && sd.Type != "openapi" && sd.Type != "arazzo" {
			v.addError(path+".type", line, col, fmt.Errorf("unknown sourceDescription type %q, must be 'openapi' or 'arazzo'", sd.Type))
		}

		if sd.URL == "" {
			v.addError(path+".url", line, col, fmt.Errorf("missing required 'url'"))
		}
	}
}

func (v *validator) checkUniqueWorkflowIds() {
	seen := make(map[string]bool)
	for i, wf := range v.doc.Workflows {
		line, col := rootPos(wf.GoLow(), (*low.Workflow).GetRootNode)
		if wf.WorkflowId == "" {
			v.addError(fmt.Sprintf("workflows[%d].workflowId", i), line, col, ErrMissingWorkflowId)
			continue
		}
		if seen[wf.WorkflowId] {
			v.addError(fmt.Sprintf("workflows[%d].workflowId", i), line, col, fmt.Errorf("%w: %q", ErrDuplicateWorkflowId, wf.WorkflowId))
		}
		seen[wf.WorkflowId] = true
	}
}

func (v *validator) buildWorkflowIdSet() map[string]bool {
	ids := make(map[string]bool, len(v.doc.Workflows))
	for _, wf := range v.doc.Workflows {
		if wf.WorkflowId != "" {
			ids[wf.WorkflowId] = true
		}
	}
	return ids
}

func (v *validator) buildOperationLookupContext() {
	attachedDocs := v.doc.GetOpenAPISourceDocuments()
	if len(attachedDocs) == 0 {
		return
	}

	uniqueDocs := make([]*v3high.Document, 0, len(attachedDocs))
	seenDocs := make(map[*v3high.Document]struct{}, len(attachedDocs))
	for _, doc := range attachedDocs {
		if doc == nil {
			continue
		}
		if _, seen := seenDocs[doc]; seen {
			continue
		}
		seenDocs[doc] = struct{}{}
		uniqueDocs = append(uniqueDocs, doc)
	}
	if len(uniqueDocs) == 0 {
		return
	}

	resolver := &operationResolver{
		searchDocs: uniqueDocs,
		sourceDocs: make(map[string]*v3high.Document),
	}

	type sourceCandidate struct {
		Index int
		Name  string
		URL   string
	}

	openAPISources := make([]sourceCandidate, 0, len(v.doc.SourceDescriptions))
	for i, source := range v.doc.SourceDescriptions {
		if source == nil || !isOpenAPISourceType(source.Type) {
			continue
		}
		openAPISources = append(openAPISources, sourceCandidate{
			Index: i,
			Name:  source.Name,
			URL:   source.URL,
		})
	}

	if len(openAPISources) == 0 {
		v.addWarning("sourceDescriptions", 0, 0,
			fmt.Sprintf("%v: no OpenAPI sourceDescriptions available to map attached OpenAPI documents",
				ErrOperationSourceMapping))
		return
	}

	remainingDocs := make(map[int]struct{}, len(uniqueDocs))
	docIDs := make([]string, len(uniqueDocs))
	for i, doc := range uniqueDocs {
		remainingDocs[i] = struct{}{}
		docIDs[i] = openAPIDocumentIdentity(doc)
	}

	// First pass: match by normalized URL identity.
	matchedSources := make(map[int]struct{}, len(openAPISources))
	for _, source := range openAPISources {
		sourceID := normalizeLookupLocation(source.URL)
		if sourceID == "" {
			continue
		}
		for i, docID := range docIDs {
			if _, ok := remainingDocs[i]; !ok || docID == "" {
				continue
			}
			if sourceID == docID {
				resolver.sourceDocs[source.Name] = uniqueDocs[i]
				resolver.sourceOrder = append(resolver.sourceOrder, source.Name)
				matchedSources[source.Index] = struct{}{}
				delete(remainingDocs, i)
				break
			}
		}
	}

	// Second pass: deterministic order fallback for remaining unmapped sources/documents.
	remainingSourceIndices := make([]int, 0, len(openAPISources))
	for _, source := range openAPISources {
		if _, matched := matchedSources[source.Index]; !matched {
			remainingSourceIndices = append(remainingSourceIndices, source.Index)
		}
	}

	remainingDocIndices := make([]int, 0, len(remainingDocs))
	for i := range uniqueDocs {
		if _, ok := remainingDocs[i]; ok {
			remainingDocIndices = append(remainingDocIndices, i)
		}
	}

	for i, sourceIndex := range remainingSourceIndices {
		if i >= len(remainingDocIndices) {
			break
		}
		docIndex := remainingDocIndices[i]
		source := v.doc.SourceDescriptions[sourceIndex]
		resolver.sourceDocs[source.Name] = uniqueDocs[docIndex]
		resolver.sourceOrder = append(resolver.sourceOrder, source.Name)
		delete(remainingDocs, docIndex)
	}

	v.opLookup = resolver

	// Warning mode: report incomplete mappings, do not hard-fail validation.
	for _, source := range openAPISources {
		if _, ok := resolver.sourceDocs[source.Name]; ok {
			continue
		}
		line, col := rootPos(v.doc.SourceDescriptions[source.Index].GoLow(), (*low.SourceDescription).GetRootNode)
		v.addWarning(fmt.Sprintf("sourceDescriptions[%d]", source.Index), line, col,
			fmt.Sprintf("%v: sourceDescription %q is not mapped to an attached OpenAPI document",
				ErrOperationSourceMapping, source.Name))
	}
}

func (v *validator) validateWorkflow(wf *high.Workflow, idx int, workflowIds map[string]bool) {
	prefix := fmt.Sprintf("workflows[%d]", idx)
	wfLine, wfCol := rootPos(wf.GoLow(), (*low.Workflow).GetRootNode)

	if len(wf.Steps) == 0 {
		v.addError(prefix+".steps", wfLine, wfCol, ErrEmptySteps)
		return
	}

	// Rule 8: DependsOn validation
	for j, dep := range wf.DependsOn {
		if !workflowIds[dep] {
			v.addError(fmt.Sprintf("%s.dependsOn[%d]", prefix, j), wfLine, wfCol, fmt.Errorf("%w: %q", ErrUnresolvedWorkflowRef, dep))
		}
	}

	// Build step ID set for this workflow
	stepIds := make(map[string]bool, len(wf.Steps))
	for i, step := range wf.Steps {
		stepPath := fmt.Sprintf("%s.steps[%d]", prefix, i)
		stepLine, stepCol := rootPos(step.GoLow(), (*low.Step).GetRootNode)
		if step.StepId == "" {
			v.addError(stepPath+".stepId", stepLine, stepCol, ErrMissingStepId)
			continue
		}
		if stepIds[step.StepId] {
			v.addError(stepPath+".stepId", stepLine, stepCol, fmt.Errorf("%w: %q", ErrDuplicateStepId, step.StepId))
		}
		stepIds[step.StepId] = true
	}

	// Validate steps
	for i, step := range wf.Steps {
		stepPath := fmt.Sprintf("%s.steps[%d]", prefix, i)
		v.validateStep(step, stepPath, stepIds, workflowIds)
	}

	// Validate workflow-level success/failure actions
	v.validateSuccessActions(wf.SuccessActions, prefix+".successActions", stepIds, workflowIds)
	v.validateFailureActions(wf.FailureActions, prefix+".failureActions", stepIds, workflowIds)

	// Rule 14: Output key validation
	if wf.Outputs != nil {
		for k, _ := range wf.Outputs.FromOldest() {
			if !componentKeyRegex.MatchString(k) {
				v.addError(fmt.Sprintf("%s.outputs.%s", prefix, k), wfLine, wfCol, fmt.Errorf("output key %q must match [a-zA-Z0-9.\\-_]+", k))
			}
		}
	}
}

func (v *validator) validateStep(step *high.Step, path string, stepIds, workflowIds map[string]bool) {
	stepLine, stepCol := rootPos(step.GoLow(), (*low.Step).GetRootNode)

	// Rule 4: Step mutual exclusivity
	count := 0
	if step.OperationId != "" {
		count++
	}
	if step.OperationPath != "" {
		count++
	}
	if step.WorkflowId != "" {
		count++
	}
	if count != 1 {
		v.addError(path, stepLine, stepCol, ErrStepMutualExclusion)
	}
	if step.WorkflowId != "" && !workflowIds[step.WorkflowId] {
		v.addError(path+".workflowId", stepLine, stepCol, fmt.Errorf("%w: %q", ErrUnresolvedWorkflowRef, step.WorkflowId))
	}
	if count == 1 && v.opLookup != nil {
		v.validateStepOperationLookup(step, path, stepLine, stepCol)
	}

	// Validate parameters
	v.validateParameters(step.Parameters, path+".parameters")

	// Validate success criteria
	for i, c := range step.SuccessCriteria {
		v.validateCriterion(c, fmt.Sprintf("%s.successCriteria[%d]", path, i))
	}

	// Validate onSuccess/onFailure
	v.validateSuccessActions(step.OnSuccess, path+".onSuccess", stepIds, workflowIds)
	v.validateFailureActions(step.OnFailure, path+".onFailure", stepIds, workflowIds)

	// Rule 14: Output key validation
	if step.Outputs != nil {
		for k, _ := range step.Outputs.FromOldest() {
			if !componentKeyRegex.MatchString(k) {
				v.addError(fmt.Sprintf("%s.outputs.%s", path, k), stepLine, stepCol, fmt.Errorf("output key %q must match [a-zA-Z0-9.\\-_]+", k))
			}
		}
	}
}

func (v *validator) validateStepOperationLookup(step *high.Step, path string, line, col int) {
	if step == nil {
		return
	}

	if step.OperationId != "" {
		if len(v.opLookup.searchDocs) == 0 {
			v.addWarning(path+".operationId", line, col,
				fmt.Sprintf("%v: no attached OpenAPI source documents available for operation lookup",
					ErrOperationSourceMapping))
		} else if !v.opLookup.findOperationByID(step.OperationId) {
			v.addError(path+".operationId", line, col, fmt.Errorf("%w: %q", ErrUnresolvedOperationRef, step.OperationId))
		}
	}

	if step.OperationPath == "" {
		return
	}

	var lookupDoc *v3high.Document
	if sourceName, found := extractSourceNameFromOperationPath(step.OperationPath); found {
		lookupDoc = v.opLookup.docForSource(sourceName)
		if lookupDoc == nil {
			v.addWarning(path+".operationPath", line, col,
				fmt.Sprintf("%v: sourceDescription %q is not mapped to an attached OpenAPI document",
					ErrOperationSourceMapping, sourceName))
			return
		}
	} else {
		lookupDoc = v.opLookup.defaultDoc()
		if lookupDoc == nil {
			v.addWarning(path+".operationPath", line, col,
				fmt.Sprintf("%v: no attached OpenAPI source documents available for operation lookup",
					ErrOperationSourceMapping))
			return
		}
	}

	exists, checkable := operationPathExistsInDoc(lookupDoc, step.OperationPath)
	if !checkable {
		v.addWarning(path+".operationPath", line, col,
			fmt.Sprintf("%v: operationPath %q is not a supported OpenAPI pointer (expected '#/paths/{path}/{method}')",
				ErrOperationSourceMapping, step.OperationPath))
		return
	}
	if !exists {
		v.addError(path+".operationPath", line, col,
			fmt.Errorf("%w: %q", ErrUnresolvedOperationRef, step.OperationPath))
	}
}

func isOpenAPISourceType(sourceType string) bool {
	normalized := strings.ToLower(strings.TrimSpace(sourceType))
	return normalized == "" || normalized == "openapi"
}

func openAPIDocumentIdentity(doc *v3high.Document) string {
	if doc == nil {
		return ""
	}
	if idx := doc.GetIndex(); idx != nil {
		if path := strings.TrimSpace(idx.GetSpecAbsolutePath()); path != "" {
			return normalizeLookupLocation(path)
		}
	}
	return ""
}

func normalizeLookupLocation(location string) string {
	trimmed := strings.TrimSpace(location)
	if trimmed == "" {
		return ""
	}
	if parsed, err := url.Parse(trimmed); err == nil && parsed.Scheme != "" {
		parsed.Fragment = ""
		parsed.Path = filepath.ToSlash(filepath.Clean(parsed.Path))
		if parsed.Path == "." {
			parsed.Path = ""
		}
		return strings.TrimSuffix(parsed.String(), "/")
	}
	if abs, err := filepath.Abs(trimmed); err == nil {
		trimmed = abs
	}
	trimmed = filepath.ToSlash(filepath.Clean(trimmed))
	if trimmed == "." {
		trimmed = ""
	}
	return strings.TrimSuffix(trimmed, "/")
}

func operationIDExistsInDocs(docs []*v3high.Document, operationID string) bool {
	for _, doc := range docs {
		if operationIDExistsInDoc(doc, operationID) {
			return true
		}
	}
	return false
}

func operationIDExistsInDoc(doc *v3high.Document, operationID string) bool {
	if doc == nil || doc.Paths == nil || doc.Paths.PathItems == nil {
		return false
	}
	for _, pathItem := range doc.Paths.PathItems.FromOldest() {
		if pathItem == nil {
			continue
		}
		operations := pathItem.GetOperations()
		if operations == nil {
			continue
		}
		for _, operation := range operations.FromOldest() {
			if operation != nil && operation.OperationId == operationID {
				return true
			}
		}
	}
	return false
}

func operationPathExistsInDoc(doc *v3high.Document, operationPath string) (exists bool, checkable bool) {
	pathKey, method, ok := parseOperationPathPointer(operationPath)
	if !ok {
		return false, false
	}
	if doc == nil || doc.Paths == nil || doc.Paths.PathItems == nil {
		return false, true
	}
	pathItem := doc.Paths.PathItems.GetOrZero(pathKey)
	if pathItem == nil {
		return false, true
	}
	operations := pathItem.GetOperations()
	if operations == nil {
		return false, true
	}
	return operations.GetOrZero(method) != nil, true
}

func parseOperationPathPointer(operationPath string) (path string, method string, ok bool) {
	const marker = "#/paths/"
	idx := strings.Index(operationPath, marker)
	if idx < 0 {
		return "", "", false
	}
	fragment := operationPath[idx:]
	if cut := strings.IndexAny(fragment, " \t\r\n"); cut >= 0 {
		fragment = fragment[:cut]
	}
	parts := strings.Split(strings.TrimPrefix(fragment, "#/"), "/")
	if len(parts) < 3 || parts[0] != "paths" {
		return "", "", false
	}
	pathToken := expression.UnescapeJSONPointer(parts[1])
	methodToken := strings.ToLower(expression.UnescapeJSONPointer(parts[2]))
	if pathToken == "" || methodToken == "" {
		return "", "", false
	}
	return pathToken, methodToken, true
}

func extractSourceNameFromOperationPath(operationPath string) (string, bool) {
	const exprPrefix = "$sourceDescriptions."
	if idx := strings.Index(operationPath, exprPrefix); idx >= 0 {
		start := idx + len(exprPrefix)
		end := start
		for end < len(operationPath) {
			c := operationPath[end]
			if c == '.' || c == '}' || c == '/' || c == '#' {
				break
			}
			end++
		}
		if end > start {
			return operationPath[start:end], true
		}
	}
	return "", false
}

func (v *validator) validateParameters(params []*high.Parameter, path string) {
	seen := make(map[string]bool)
	for i, p := range params {
		paramPath := fmt.Sprintf("%s[%d]", path, i)
		pLine, pCol := rootPos(p.GoLow(), (*low.Parameter).GetRootNode)

		if p.IsReusable() {
			// Reusable parameter - validate reference resolves
			v.validateComponentReference(p.Reference, paramPath+".reference", "parameters")
			continue
		}

		// Rule 5: Parameter validation
		if p.Name == "" {
			v.addError(paramPath+".name", pLine, pCol, ErrMissingParameterName)
		}
		if p.Value == nil {
			v.addError(paramPath+".value", pLine, pCol, ErrMissingParameterValue)
		}

		// Rule 5: Parameter `in` validation
		if p.In == "" {
			v.addError(paramPath+".in", pLine, pCol, ErrMissingParameterIn)
		} else {
			switch p.In {
			case "path", "query", "header", "cookie":
				// valid
			default:
				v.addError(paramPath+".in", pLine, pCol, ErrInvalidParameterIn)
			}
		}

		// Rule 16: Duplicate parameters (name+in)
		key := p.Name + ":" + p.In
		if seen[key] {
			v.addError(paramPath, pLine, pCol, fmt.Errorf("duplicate parameter (name=%q, in=%q)", p.Name, p.In))
		}
		seen[key] = true
	}
}

func (v *validator) validateSuccessActions(actions []*high.SuccessAction, path string, stepIds, workflowIds map[string]bool) {
	seen := make(map[string]bool)
	for i, a := range actions {
		actionPath := fmt.Sprintf("%s[%d]", path, i)
		aLine, aCol := rootPos(a.GoLow(), (*low.SuccessAction).GetRootNode)

		if a.IsReusable() {
			v.validateComponentReference(a.Reference, actionPath+".reference", "successActions")
			continue
		}

		if a.Type != "" && a.Type != "end" && a.Type != "goto" {
			v.addError(actionPath+".type", aLine, aCol, ErrInvalidSuccessType)
		}

		v.validateActionCommon(a.Name, a.Type, a.WorkflowId, a.StepId, actionPath, aLine, aCol, stepIds, workflowIds, seen)
	}
}

func (v *validator) validateFailureActions(actions []*high.FailureAction, path string, stepIds, workflowIds map[string]bool) {
	seen := make(map[string]bool)
	for i, a := range actions {
		actionPath := fmt.Sprintf("%s[%d]", path, i)
		aLine, aCol := rootPos(a.GoLow(), (*low.FailureAction).GetRootNode)

		if a.IsReusable() {
			v.validateComponentReference(a.Reference, actionPath+".reference", "failureActions")
			continue
		}

		if a.Type != "" && a.Type != "end" && a.Type != "retry" && a.Type != "goto" {
			v.addError(actionPath+".type", aLine, aCol, ErrInvalidFailureType)
		}

		v.validateActionCommon(a.Name, a.Type, a.WorkflowId, a.StepId, actionPath, aLine, aCol, stepIds, workflowIds, seen)

		if a.RetryAfter != nil && *a.RetryAfter < 0 {
			v.addError(actionPath+".retryAfter", aLine, aCol, fmt.Errorf("retryAfter must be non-negative, got %f", *a.RetryAfter))
		}
		if a.RetryLimit != nil && *a.RetryLimit < 0 {
			v.addError(actionPath+".retryLimit", aLine, aCol, fmt.Errorf("retryLimit must be non-negative, got %d", *a.RetryLimit))
		}
	}
}

// validateActionCommon validates fields shared between success and failure actions:
// name, type, target mutual exclusion, goto target, workflow/step references, duplicate names.
func (v *validator) validateActionCommon(name, actionType, workflowId, stepId, actionPath string, line, col int, stepIds, workflowIds map[string]bool, seen map[string]bool) {
	if name == "" {
		v.addError(actionPath+".name", line, col, ErrMissingActionName)
	}
	if actionType == "" {
		v.addError(actionPath+".type", line, col, ErrMissingActionType)
	}

	if workflowId != "" && stepId != "" {
		v.addError(actionPath, line, col, ErrActionMutualExclusion)
	}
	if actionType == "goto" && workflowId == "" && stepId == "" {
		v.addError(actionPath, line, col, ErrGotoRequiresTarget)
	}
	if workflowId != "" && !workflowIds[workflowId] {
		v.addError(actionPath+".workflowId", line, col, fmt.Errorf("%w: %q", ErrUnresolvedWorkflowRef, workflowId))
	}
	if stepId != "" && !stepIds[stepId] {
		v.addError(actionPath+".stepId", line, col, fmt.Errorf("%w: %q", ErrStepIdNotInWorkflow, stepId))
	}

	if name != "" {
		if seen[name] {
			v.addError(actionPath+".name", line, col, fmt.Errorf("duplicate action name %q", name))
		}
		seen[name] = true
	}
}

func (v *validator) validateCriterion(c *high.Criterion, path string) {
	cLine, cCol := rootPos(c.GoLow(), (*low.Criterion).GetRootNode)

	if c.Condition == "" {
		v.addError(path+".condition", cLine, cCol, ErrMissingCondition)
	}

	// Rule 15a: Context required when type is specified
	effectiveType := c.GetEffectiveType()
	if effectiveType != "simple" && c.Context == "" {
		v.addError(path+".context", cLine, cCol, fmt.Errorf("context is required when type is %q", effectiveType))
	}

	// Rule 15: CriterionExpressionType validation
	if c.ExpressionType != nil {
		v.validateCriterionExpressionType(c.ExpressionType, path+".type")
	}

	// Validate context as runtime expression if present
	if c.Context != "" {
		if err := expression.Validate(c.Context); err != nil {
			v.addError(path+".context", cLine, cCol, fmt.Errorf("%w: %v", ErrInvalidExpression, err))
		}
	}
}

func (v *validator) validateCriterionExpressionType(cet *high.CriterionExpressionType, path string) {
	if cet.Type == "" {
		v.addError(path+".type", 0, 0, fmt.Errorf("missing required 'type' in criterion expression type"))
		return
	}

	switch cet.Type {
	case "jsonpath":
		if cet.Version != "" && cet.Version != "draft-goessner-dispatch-jsonpath-00" {
			v.addError(path+".version", 0, 0, fmt.Errorf("unknown jsonpath version %q", cet.Version))
		}
	case "xpath":
		validVersions := map[string]bool{"xpath-30": true, "xpath-20": true, "xpath-10": true}
		if cet.Version != "" && !validVersions[cet.Version] {
			v.addError(path+".version", 0, 0, fmt.Errorf("unknown xpath version %q", cet.Version))
		}
	}
}

func (v *validator) validateComponentReference(ref, path, componentType string) {
	if v.doc.Components == nil {
		v.addError(path, 0, 0, fmt.Errorf("%w: no components defined", ErrUnresolvedComponent))
		return
	}

	// Reference format: $components.{type}.{name}
	expectedPrefix := "$components." + componentType + "."
	if !strings.HasPrefix(ref, expectedPrefix) {
		v.addError(path, 0, 0, fmt.Errorf("reference %q must start with %q", ref, expectedPrefix))
		return
	}

	name := ref[len(expectedPrefix):]
	if name == "" {
		v.addError(path, 0, 0, fmt.Errorf("empty component name in reference %q", ref))
		return
	}

	// Check component exists
	switch componentType {
	case "parameters":
		if v.doc.Components.Parameters == nil {
			v.addError(path, 0, 0, fmt.Errorf("%w: %q", ErrUnresolvedComponent, ref))
			return
		}
		if _, ok := v.doc.Components.Parameters.Get(name); !ok {
			v.addError(path, 0, 0, fmt.Errorf("%w: %q", ErrUnresolvedComponent, ref))
		}
	case "successActions":
		if v.doc.Components.SuccessActions == nil {
			v.addError(path, 0, 0, fmt.Errorf("%w: %q", ErrUnresolvedComponent, ref))
			return
		}
		if _, ok := v.doc.Components.SuccessActions.Get(name); !ok {
			v.addError(path, 0, 0, fmt.Errorf("%w: %q", ErrUnresolvedComponent, ref))
		}
	case "failureActions":
		if v.doc.Components.FailureActions == nil {
			v.addError(path, 0, 0, fmt.Errorf("%w: %q", ErrUnresolvedComponent, ref))
			return
		}
		if _, ok := v.doc.Components.FailureActions.Get(name); !ok {
			v.addError(path, 0, 0, fmt.Errorf("%w: %q", ErrUnresolvedComponent, ref))
		}
	}
}

func (v *validator) checkCircularDependencies() {
	// Build adjacency map
	adj := make(map[string][]string)
	for _, wf := range v.doc.Workflows {
		if wf.WorkflowId != "" {
			adj[wf.WorkflowId] = wf.DependsOn
		}
	}

	// DFS with recursion stack
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(id string, path []string) bool
	dfs = func(id string, path []string) bool {
		visited[id] = true
		recStack[id] = true
		path = append(path, id)

		for _, dep := range adj[id] {
			if !visited[dep] {
				if dfs(dep, path) {
					return true
				}
			} else if recStack[dep] {
				v.addError("workflows", 0, 0, fmt.Errorf("%w: %s", ErrCircularDependency, strings.Join(append(path, dep), " -> ")))
				return true
			}
		}

		recStack[id] = false
		return false
	}

	for id := range adj {
		if !visited[id] {
			dfs(id, nil)
		}
	}
}

func (v *validator) validateComponentKeys() {
	if v.doc.Components == nil {
		return
	}
	if v.doc.Components.Parameters != nil {
		for k, _ := range v.doc.Components.Parameters.FromOldest() {
			v.validateComponentKey(k, "parameters")
		}
	}
	if v.doc.Components.SuccessActions != nil {
		for k, _ := range v.doc.Components.SuccessActions.FromOldest() {
			v.validateComponentKey(k, "successActions")
		}
	}
	if v.doc.Components.FailureActions != nil {
		for k, _ := range v.doc.Components.FailureActions.FromOldest() {
			v.validateComponentKey(k, "failureActions")
		}
	}
	if v.doc.Components.Inputs != nil {
		for k, _ := range v.doc.Components.Inputs.FromOldest() {
			v.validateComponentKey(k, "inputs")
		}
	}
}

func (v *validator) validateComponentKey(key, componentType string) {
	if !componentKeyRegex.MatchString(key) {
		v.addError(fmt.Sprintf("components.%s.%s", componentType, key), 0, 0, fmt.Errorf("component key %q must match [a-zA-Z0-9.\\-_]+", key))
	}
}
