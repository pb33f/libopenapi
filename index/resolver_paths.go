// Copyright 2022-2026 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package index

import (
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/pb33f/libopenapi/utils"
	"go.yaml.in/yaml/v4"
)

func (resolver *Resolver) buildDefPath(ref *Reference, l string) string {
	def := ""
	exp := strings.Split(l, "#/")
	if len(exp) == 2 {
		if exp[0] != "" {
			if !strings.HasPrefix(exp[0], "http") {
				if !filepath.IsAbs(exp[0]) {
					if strings.HasPrefix(ref.FullDefinition, "http") {
						u, _ := url.Parse(ref.FullDefinition)
						p, _ := filepath.Abs(utils.CheckPathOverlap(path.Dir(u.Path), exp[0], string(filepath.Separator)))
						u.Path = utils.ReplaceWindowsDriveWithLinuxPath(p)
						def = l
						if len(exp[1]) > 0 {
							def = u.String() + "#/" + exp[1]
						}
					} else {
						z := strings.Split(ref.FullDefinition, "#/")
						if len(z) == 2 {
							if len(z[0]) > 0 {
								abs := resolver.resolveLocalRefPath(filepath.Dir(z[0]), exp[0])
								def = abs + "#/" + exp[1]
							} else {
								abs, _ := filepath.Abs(exp[0])
								def = abs + "#/" + exp[1]
							}
						} else {
							abs := resolver.resolveLocalRefPath(filepath.Dir(ref.FullDefinition), exp[0])
							def = abs + "#/" + exp[1]
						}
					}
				}
			} else if len(exp[1]) > 0 {
				def = l
			} else {
				def = exp[0]
			}
		} else if strings.HasPrefix(ref.FullDefinition, "http") {
			u, _ := url.Parse(ref.FullDefinition)
			u.Fragment = ""
			def = u.String() + "#/" + exp[1]
		} else if strings.HasPrefix(ref.FullDefinition, "#/") {
			def = "#/" + exp[1]
		} else {
			fdexp := strings.Split(ref.FullDefinition, "#/")
			def = fdexp[0] + "#/" + exp[1]
		}
	} else if strings.HasPrefix(l, "http") {
		def = l
	} else if strings.HasPrefix(ref.FullDefinition, "http") {
		u, _ := url.Parse(ref.FullDefinition)
		abs, _ := filepath.Abs(utils.CheckPathOverlap(path.Dir(u.Path), l, string(filepath.Separator)))
		u.Path = utils.ReplaceWindowsDriveWithLinuxPath(abs)
		u.Fragment = ""
		def = u.String()
	} else {
		lookupRef := strings.Split(ref.FullDefinition, "#/")
		def = resolver.resolveLocalRefPath(filepath.Dir(lookupRef[0]), l)
	}

	return def
}

func (resolver *Resolver) resolveLocalRefPath(base, ref string) string {
	if resolver != nil && resolver.specIndex != nil {
		return resolver.specIndex.ResolveRelativeFilePath(base, ref)
	}
	abs, _ := filepath.Abs(utils.CheckPathOverlap(base, ref, string(filepath.Separator)))
	return abs
}

func (resolver *Resolver) buildDefPathWithSchemaBase(ref *Reference, l string, schemaIDBase string) string {
	if schemaIDBase != "" {
		normalized := resolveRefWithSchemaBase(l, schemaIDBase)
		if normalized != l {
			return normalized
		}
	}
	return resolver.buildDefPath(ref, l)
}

func (resolver *Resolver) resolveSchemaIdBase(parentBase string, node *yaml.Node) string {
	if node == nil {
		return parentBase
	}
	idValue := FindSchemaIdInNode(node)
	if idValue == "" {
		return parentBase
	}
	base := parentBase
	if base == "" && resolver.specIndex != nil {
		base = resolver.specIndex.specAbsolutePath
	}
	resolved, err := ResolveSchemaId(idValue, base)
	if err != nil || resolved == "" {
		return idValue
	}
	return resolved
}

func (resolver *Resolver) ResolvePendingNodes() {
	for _, r := range resolver.specIndex.pendingResolve {
		r.ref.Node.Content = r.nodes
	}
}
