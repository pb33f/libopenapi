// Copyright 2022-2025 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package overlay

import (
	"fmt"
	"regexp"

	"github.com/pb33f/go-yaml"
	highoverlay "github.com/pb33f/libopenapi/datamodel/high/overlay"
)

var overlayVersion = regexp.MustCompile(`^1\.[012]\.[0-9]+$`)

// validateOverlay checks that the overlay has all required fields.
func validateOverlay(overlay *highoverlay.Overlay) error {
	if overlay.Overlay == "" {
		return ErrMissingOverlayField
	}
	if !overlayVersion.MatchString(overlay.Overlay) {
		return ErrUnsupportedVersion
	}
	if overlay.Info == nil {
		return ErrMissingInfo
	}
	if info := overlay.Info.GoLow(); info != nil &&
		((info.Title.IsEmpty() && overlay.Info.Title == "") || (info.Version.IsEmpty() && overlay.Info.Version == "")) {
		return ErrInvalidInfo
	}
	if _, err := overlay.ResolveExtends(""); err != nil {
		return err
	}
	if len(overlay.Actions) == 0 {
		return ErrEmptyActions
	}
	if overlay.Components != nil {
		for pair := overlay.Components.Actions.First(); pair != nil; pair = pair.Next() {
			if pair.Value() == nil {
				return ErrInvalidReusableAction
			}
			fields := pair.Value().Fields
			if fields != nil && (fields.Target != "" || fields.Ref != "" ||
				(fields.GoLow() != nil && (fields.GoLow().Target.KeyNode != nil || fields.GoLow().Ref.KeyNode != nil))) {
				return ErrInvalidReusableAction
			}
		}
	}
	return nil
}

// validateTargets checks the JSON type category, not the individual primitive type.
func validateTargets(nodes []*yaml.Node) error {
	for _, node := range nodes {
		if node.Kind != yaml.MappingNode && node.Kind != yaml.SequenceNode && node.Kind != yaml.ScalarNode {
			return ErrIncompatibleUpdate
		}
		if node.Kind != nodes[0].Kind {
			return fmt.Errorf("%w: mixed target types", ErrIncompatibleUpdate)
		}
	}
	return nil
}
