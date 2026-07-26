// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// FeatureFamily identifies the patch-insensitive Arazzo feature set.
type FeatureFamily string

const (
	// FeatureFamily10 identifies the Arazzo 1.0 feature set.
	FeatureFamily10 FeatureFamily = "1.0"
	// FeatureFamily11 identifies the Arazzo 1.1 feature set.
	FeatureFamily11 FeatureFamily = "1.1"
)

// ErrUnsupportedVersion indicates a malformed or unsupported Arazzo specification version.
var ErrUnsupportedVersion = errors.New("unsupported Arazzo specification version")

// SpecificationVersion is the parsed Arazzo specification version, distinct from info.version.
type SpecificationVersion struct {
	Authored string
	Major    int
	Minor    int
	Patch    int
	Family   FeatureFamily
}

// ParseSpecificationVersion parses supported 1.0.x and 1.1.x specification versions.
func ParseSpecificationVersion(authored string) (*SpecificationVersion, error) {
	numericVersion, suffix, hasSuffix := strings.Cut(authored, "-")
	if hasSuffix && suffix == "" {
		return nil, fmt.Errorf("%w %q: empty prerelease suffix", ErrUnsupportedVersion, authored)
	}
	parts := strings.Split(numericVersion, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("%w %q: expected major.minor.patch", ErrUnsupportedVersion, authored)
	}
	values := [3]int{}
	for i, part := range parts {
		if part == "" {
			return nil, fmt.Errorf("%w %q: expected major.minor.patch", ErrUnsupportedVersion, authored)
		}
		value, err := strconv.Atoi(part)
		if err != nil || value < 0 {
			return nil, fmt.Errorf("%w %q: expected non-negative numeric components", ErrUnsupportedVersion, authored)
		}
		values[i] = value
	}
	var family FeatureFamily
	switch {
	case values[0] == 1 && values[1] == 0:
		family = FeatureFamily10
	case values[0] == 1 && values[1] == 1:
		family = FeatureFamily11
	default:
		return nil, fmt.Errorf("%w %q", ErrUnsupportedVersion, authored)
	}
	return &SpecificationVersion{
		Authored: authored,
		Major:    values[0],
		Minor:    values[1],
		Patch:    values[2],
		Family:   family,
	}, nil
}

// GetSpecificationVersion returns patch-insensitive feature metadata for the authored arazzo field.
func (a *Arazzo) GetSpecificationVersion() (*SpecificationVersion, error) {
	if a == nil {
		return nil, fmt.Errorf("%w: nil document", ErrUnsupportedVersion)
	}
	return ParseSpecificationVersion(a.Arazzo)
}
