// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package reports

// HasChanges represents a change model that provides a total change count and a breaking change count.
type HasChanges interface {
	// TotalChanges represents number of all changes found
	TotalChanges() int

	// TotalBreakingChanges represents the number of contract breaking changes only.
	TotalBreakingChanges() int
}
