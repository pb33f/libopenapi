// Copyright 2022-2025 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package overlay

import (
	"testing"

	"github.com/pb33f/go-yaml"
	highoverlay "github.com/pb33f/libopenapi/datamodel/high/overlay"
	"github.com/pb33f/testify/assert"
)

func TestValidateOverlay_Valid(t *testing.T) {
	overlay := &highoverlay.Overlay{
		Overlay: "1.0.0",
		Info: &highoverlay.Info{
			Title:   "Test",
			Version: "1.0.0",
		},
		Actions: []*highoverlay.Action{
			{Target: "$.info"},
		},
	}

	err := validateOverlay(overlay)
	assert.NoError(t, err)
}

func TestValidateOverlay_MissingOverlayField(t *testing.T) {
	overlay := &highoverlay.Overlay{
		Info: &highoverlay.Info{
			Title:   "Test",
			Version: "1.0.0",
		},
		Actions: []*highoverlay.Action{
			{Target: "$.info"},
		},
	}

	err := validateOverlay(overlay)
	assert.ErrorIs(t, err, ErrMissingOverlayField)
}

func TestValidateOverlay_MissingInfo(t *testing.T) {
	overlay := &highoverlay.Overlay{
		Overlay: "1.0.0",
		Actions: []*highoverlay.Action{
			{Target: "$.info"},
		},
	}

	err := validateOverlay(overlay)
	assert.ErrorIs(t, err, ErrMissingInfo)
}

func TestValidateOverlay_EmptyActions(t *testing.T) {
	overlay := &highoverlay.Overlay{
		Overlay: "1.0.0",
		Info: &highoverlay.Info{
			Title:   "Test",
			Version: "1.0.0",
		},
		Actions: []*highoverlay.Action{},
	}

	err := validateOverlay(overlay)
	assert.ErrorIs(t, err, ErrEmptyActions)
}

func TestValidateTarget_Scalar(t *testing.T) {
	node := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "test",
	}

	err := validateTargets([]*yaml.Node{node})
	assert.NoError(t, err)
}

func TestValidateTarget_Mapping(t *testing.T) {
	node := &yaml.Node{
		Kind: yaml.MappingNode,
	}

	err := validateTargets([]*yaml.Node{node})
	assert.NoError(t, err)
}

func TestValidateTarget_Sequence(t *testing.T) {
	node := &yaml.Node{
		Kind: yaml.SequenceNode,
	}

	err := validateTargets([]*yaml.Node{node})
	assert.NoError(t, err)
}

func TestValidateTarget_Document(t *testing.T) {
	node := &yaml.Node{
		Kind: yaml.DocumentNode,
	}

	err := validateTargets([]*yaml.Node{node})
	assert.ErrorIs(t, err, ErrIncompatibleUpdate)
}
