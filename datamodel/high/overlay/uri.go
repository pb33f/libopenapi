// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package overlay

import (
	"fmt"
	"net/url"
	"strings"
)

// ResolveExtends resolves extends against $self, then baseURI. baseURI is the
// retrieval URI, encapsulating entity URI, or application default supplied by
// the caller. With no absolute base, extends is returned unchanged. It performs no I/O.
func (o *Overlay) ResolveExtends(baseURI string) (string, error) {
	base, err := documentURI(baseURI)
	if err != nil {
		return "", err
	}
	if o.Self != "" {
		self, err := documentURI(o.Self)
		if err != nil {
			return "", err
		}
		if self.IsAbs() || !base.IsAbs() {
			base = self
		} else {
			base = base.ResolveReference(self)
		}
	}
	if o.Extends == "" && (o.low == nil || o.low.Extends.KeyNode == nil) {
		return "", nil
	}
	extends, err := documentURI(o.Extends)
	if err != nil {
		return "", err
	}
	if extends.IsAbs() || !base.IsAbs() {
		return extends.String(), nil
	}
	return base.ResolveReference(extends).String(), nil
}

func documentURI(value string) (*url.URL, error) {
	uri, err := url.Parse(value)
	if err != nil || strings.Contains(value, "#") || strings.ContainsAny(value, " \t\r\n") {
		return nil, fmt.Errorf("invalid overlay document URI %q: fragments and whitespace are not allowed", value)
	}
	return uri, nil
}
