// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

const conflictNameDelimiter = "__"

// NameRegistry allocates deterministic, collision-free Go names.
type NameRegistry struct {
	used map[string]string
}

// NewNameRegistry creates a registry with reserved names already claimed.
func NewNameRegistry(reserved ...string) *NameRegistry {
	registry := &NameRegistry{used: make(map[string]string, len(reserved))}
	for _, name := range reserved {
		registry.used[name] = name
	}
	return registry
}

func newNameRegistry() *NameRegistry {
	return NewNameRegistry()
}

// Resolve returns candidate unless another original value already claimed it.
func (r *NameRegistry) Resolve(original, candidate string) (string, bool) {
	if candidate == "" {
		candidate = "Value"
	}
	if existing, ok := r.used[candidate]; !ok {
		r.used[candidate] = original
		return candidate, false
	} else if existing == original {
		return candidate, false
	}
	for i := 2; ; i++ {
		next := candidate + conflictNameDelimiter + intString(i)
		if _, ok := r.used[next]; !ok {
			r.used[next] = original
			return next, true
		}
	}
}

func (r *NameRegistry) resolve(original, candidate string) (string, bool) {
	return r.Resolve(original, candidate)
}

// Claim reserves a name for a distinct declaration. collisionSuffix is tried
// before the registry's numeric suffix convention.
func (r *NameRegistry) Claim(preferred, collisionSuffix string) string {
	if preferred == "" {
		preferred = "Value"
	}
	if _, exists := r.used[preferred]; !exists {
		r.used[preferred] = preferred
		return preferred
	}
	if collisionSuffix != "" {
		candidate := preferred + collisionSuffix
		if _, exists := r.used[candidate]; !exists {
			r.used[candidate] = candidate
			return candidate
		}
	}
	for suffix := 2; ; suffix++ {
		candidate := preferred + conflictNameDelimiter + intString(suffix)
		if _, exists := r.used[candidate]; !exists {
			r.used[candidate] = candidate
			return candidate
		}
	}
}
