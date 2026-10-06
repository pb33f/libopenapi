// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package expression

import (
	"errors"
	"fmt"
	"strings"
)

// ErrLocalSymbolNotFound identifies a runtime expression that references a
// symbol unavailable in its current Arazzo document scope.
var ErrLocalSymbolNotFound = errors.New("local runtime-expression symbol not found")

// SymbolSet is an allocation-conscious lookup set for local Arazzo names.
type SymbolSet map[string]struct{}

// StepSymbols contains the locally declared outputs for a step.
type StepSymbols struct {
	Outputs SymbolSet
}

// WorkflowSymbols contains the locally declared inputs and outputs for a
// workflow.
type WorkflowSymbols struct {
	Inputs  SymbolSet
	Outputs SymbolSet
}

// LocalSymbols contains the document-local names available to a runtime
// expression. It deliberately contains no document loader or external source
// adapter.
type LocalSymbols struct {
	HasSelf                 bool
	Inputs                  SymbolSet
	Outputs                 SymbolSet
	Steps                   map[string]StepSymbols
	Workflows               map[string]WorkflowSymbols
	SourceDescriptions      SymbolSet
	ComponentParameters     SymbolSet
	ComponentSuccessActions SymbolSet
	ComponentFailureActions SymbolSet
}

// LocalSymbolError reports the narrow local symbol that could not be resolved.
type LocalSymbolError struct {
	Expression string
	Kind       string
	Name       string
}

// Error returns a stable diagnostic for an unresolved local symbol.
func (e *LocalSymbolError) Error() string {
	return fmt.Sprintf("%s: %s %q in %q", ErrLocalSymbolNotFound, e.Kind, e.Name, e.Expression)
}

// Unwrap exposes ErrLocalSymbolNotFound for errors.Is.
func (e *LocalSymbolError) Unwrap() error {
	return ErrLocalSymbolNotFound
}

// ResolveLocal checks the document-local portion of an already parsed runtime
// expression. It never resolves or retrieves an external source document.
func ResolveLocal(expr Expression, symbols *LocalSymbols) error {
	if symbols == nil {
		return fmt.Errorf("nil local runtime-expression symbols")
	}
	switch expr.Type {
	case Self:
		if !symbols.HasSelf {
			return newLocalSymbolError(expr, "self URI", "$self")
		}
	case Inputs:
		return requireSymbol(expr, symbols.Inputs, "input", expr.Name)
	case Outputs:
		return requireSymbol(expr, symbols.Outputs, "output", expr.Name)
	case Steps:
		step, found := symbols.Steps[expr.Name]
		if !found {
			return newLocalSymbolError(expr, "step", expr.Name)
		}
		field, name, ok := splitReferenceTail(expr.Tail)
		if !ok || field != "outputs" {
			return newLocalSymbolError(expr, "step output", expr.Tail)
		}
		return requireSymbol(expr, step.Outputs, "step output", name)
	case Workflows:
		workflow, found := symbols.Workflows[expr.Name]
		if !found {
			return newLocalSymbolError(expr, "workflow", expr.Name)
		}
		field, name, ok := splitReferenceTail(expr.Tail)
		if !ok {
			return newLocalSymbolError(expr, "workflow field", expr.Tail)
		}
		switch field {
		case "inputs":
			return requireSymbol(expr, workflow.Inputs, "workflow input", name)
		case "outputs":
			return requireSymbol(expr, workflow.Outputs, "workflow output", name)
		default:
			return newLocalSymbolError(expr, "workflow field", field)
		}
	case SourceDescriptions:
		return requireSymbol(expr, symbols.SourceDescriptions, "source description", expr.Name)
	case ComponentParameters:
		return requireSymbol(expr, symbols.ComponentParameters, "component parameter", expr.Name)
	case ComponentSuccessActions:
		return requireSymbol(expr, symbols.ComponentSuccessActions, "component success action", expr.Name)
	case ComponentFailureActions:
		return requireSymbol(expr, symbols.ComponentFailureActions, "component failure action", expr.Name)
	}
	return nil
}

// ValidateLocal parses a runtime expression and checks its document-local
// symbols without evaluating runtime values or resolving external documents.
func ValidateLocal(input string, symbols *LocalSymbols) error {
	expr, err := Parse(input)
	if err != nil {
		return err
	}
	return ResolveLocal(expr, symbols)
}

func requireSymbol(expr Expression, symbols SymbolSet, kind, name string) error {
	if hasSymbolOrPrefix(symbols, name) {
		return nil
	}
	return newLocalSymbolError(expr, kind, name)
}

func hasSymbolOrPrefix(symbols SymbolSet, name string) bool {
	if _, found := symbols[name]; found {
		return true
	}
	for separator := strings.LastIndexByte(name, '.'); separator > 0; separator = strings.LastIndexByte(name[:separator], '.') {
		if _, found := symbols[name[:separator]]; found {
			return true
		}
	}
	return false
}

func newLocalSymbolError(expr Expression, kind, name string) error {
	return &LocalSymbolError{Expression: expr.Raw, Kind: kind, Name: name}
}

func splitReferenceTail(tail string) (string, string, bool) {
	separator := strings.IndexByte(tail, '.')
	if separator <= 0 || separator == len(tail)-1 {
		return "", "", false
	}
	return tail[:separator], tail[separator+1:], true
}
