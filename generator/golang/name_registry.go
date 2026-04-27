// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

type nameRegistry struct {
	used map[string]string
}

func newNameRegistry() *nameRegistry {
	return &nameRegistry{used: make(map[string]string)}
}

func (r *nameRegistry) resolve(original, candidate string) (string, bool) {
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
		next := candidate + intString(i)
		if _, ok := r.used[next]; !ok {
			r.used[next] = original
			return next, true
		}
	}
}
