// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

const conflictNameDelimiter = "__"

// NameRegistry allocates deterministic, collision-free Go names.
type NameRegistry struct {
	used      map[string]string
	separator string
}

// NewNameRegistry creates a registry with reserved names already claimed.
// Collisions take a "__N" suffix.
func NewNameRegistry(reserved ...string) *NameRegistry {
	return newRegistry(conflictNameDelimiter, reserved)
}

// NewIdiomaticNameRegistry creates a registry with reserved names already
// claimed. Collisions take a plain numeric suffix, so a second Widget becomes
// Widget2 rather than Widget__2.
func NewIdiomaticNameRegistry(reserved ...string) *NameRegistry {
	return newRegistry("", reserved)
}

func newRegistry(separator string, reserved []string) *NameRegistry {
	registry := &NameRegistry{used: make(map[string]string, len(reserved)), separator: separator}
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
	return r.ResolveFirst(original, candidate)
}

// ResolveFirst returns the first candidate that is free or already claimed by
// original. When every candidate belongs to another original it numbers the
// last candidate and reports the collision.
func (r *NameRegistry) ResolveFirst(original string, candidates ...string) (string, bool) {
	for _, candidate := range candidates {
		if existing, ok := r.used[candidate]; !ok {
			r.used[candidate] = original
			return candidate, false
		} else if existing == original {
			return candidate, false
		}
	}
	next := r.next(candidates[len(candidates)-1])
	r.used[next] = original
	return next, true
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
	next := r.next(preferred)
	r.used[next] = next
	return next
}

// ClaimFirst reserves the first free candidate for a distinct declaration.
// When every candidate is taken it numbers the last one.
func (r *NameRegistry) ClaimFirst(candidates ...string) string {
	for _, candidate := range candidates {
		if _, exists := r.used[candidate]; !exists {
			r.used[candidate] = candidate
			return candidate
		}
	}
	next := r.next(candidates[len(candidates)-1])
	r.used[next] = next
	return next
}

// next returns the first free numbered form of candidate without claiming it.
// Without a separator, a number after a name that ends in a digit would read
// as part of it (Value152), so such names take a letter instead (Value15B).
func (r *NameRegistry) next(candidate string) string {
	if last := candidate[len(candidate)-1]; r.separator == "" && last >= '0' && last <= '9' {
		for letter := 'B'; letter <= 'Z'; letter++ {
			if _, exists := r.used[candidate+string(letter)]; !exists {
				return candidate + string(letter)
			}
		}
		candidate += "Z"
	}
	for suffix := 2; ; suffix++ {
		next := candidate + r.separator + intString(suffix)
		if _, exists := r.used[next]; !exists {
			return next
		}
	}
}
