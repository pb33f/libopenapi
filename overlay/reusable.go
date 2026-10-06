// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package overlay

import (
	"fmt"
	"net/url"
	"strings"

	highoverlay "github.com/pb33f/libopenapi/datamodel/high/overlay"
)

func resolveAction(overlay *highoverlay.Overlay, action *highoverlay.Action) (*highoverlay.Action, error) {
	if action == nil || action.Target == "" {
		return nil, ErrMissingTarget
	}
	if action.Ref == "" && (action.GoLow() == nil || action.GoLow().Ref.KeyNode == nil) {
		return action, nil
	}
	key, err := actionReferenceKey(overlay, action.Ref)
	if err != nil {
		return nil, err
	}
	if overlay.Components == nil || overlay.Components.Actions == nil {
		return nil, ErrInvalidActionReference
	}
	reusable, ok := overlay.Components.Actions.Get(key)
	if !ok || reusable == nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidActionReference, action.Ref)
	}
	effective := new(highoverlay.Action)
	if reusable.Fields != nil {
		fields := reusable.Fields
		*effective = *fields
	}
	effective.Target, effective.Ref = action.Target, action.Ref
	if action.Description != "" || (action.GoLow() != nil && action.GoLow().Description.KeyNode != nil) {
		effective.Description = action.Description
	}
	if action.Update != nil {
		effective.Update = action.Update
	}
	if action.Copy != "" || (action.GoLow() != nil && action.GoLow().Copy.KeyNode != nil) {
		effective.Copy = action.Copy
	}
	if action.HasRemove() {
		effective.SetRemove(action.Remove)
	}
	return effective, nil
}

func actionReferenceKey(overlay *highoverlay.Overlay, ref string) (string, error) {
	if strings.HasPrefix(ref, "#") {
		return reusableKey(ref)
	}
	uri, err := url.Parse(ref)
	if err != nil || uri.IsAbs() || strings.ContainsAny(ref, " \t\r\n") {
		return "", ErrInvalidActionReference
	}
	self, err := url.Parse(overlay.Self)
	if err != nil || !self.IsAbs() || strings.ContainsAny(overlay.Self, "# \t\r\n") {
		return "", ErrInvalidActionReference
	}
	resolved := self.ResolveReference(uri)
	fragment := resolved.EscapedFragment()
	resolved.Fragment, resolved.RawFragment = "", ""
	if resolved.String() != self.String() {
		return "", ErrInvalidActionReference
	}
	return reusableKey("#" + fragment)
}

func reusableKey(ref string) (string, error) {
	if !strings.HasPrefix(ref, "#") || strings.ContainsAny(ref[1:], "# \t\r\n") {
		return "", ErrInvalidActionReference
	}
	pointer, err := url.PathUnescape(ref[1:])
	const prefix = "/components/actions/"
	if err != nil || !strings.HasPrefix(pointer, prefix) {
		return "", ErrInvalidActionReference
	}
	token := strings.TrimPrefix(pointer, prefix)
	if strings.Contains(token, "/") {
		return "", ErrInvalidActionReference
	}
	for i := 0; i < len(token); i++ {
		if token[i] == '~' {
			if i+1 == len(token) || (token[i+1] != '0' && token[i+1] != '1') {
				return "", ErrInvalidActionReference
			}
			i++
		}
	}
	return strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~"), nil
}
