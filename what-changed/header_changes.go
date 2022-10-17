// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

type HeaderChanges struct {
	PropertyChanges
	SchemaChanges    *SchemaChanges
	ExampleChanges   map[string]*ExampleChanges
	ContentChanges   map[string]*MediaTypeChanges
	ExtensionChanges *ExtensionChanges
}

func (h *HeaderChanges) TotalChanges() int {
	c := len(h.Changes)
	for k := range h.ExampleChanges {
		c += h.ExampleChanges[k].TotalChanges()
	}
	for k := range h.ContentChanges {
		c += h.ContentChanges[k].TotalChanges()
	}
	if h.ExtensionChanges != nil {
		c += h.ExtensionChanges.TotalChanges()
	}
	return c
}

func (h *HeaderChanges) TotalBreakingChanges() int {
	c := len(h.Changes)
	for k := range h.ExampleChanges {
		c += h.ExampleChanges[k].TotalChanges()
	}
	for k := range h.ContentChanges {
		c += h.ContentChanges[k].TotalChanges()
	}
	if h.ExtensionChanges != nil {
		c += h.ExtensionChanges.TotalChanges()
	}
	return c
}
