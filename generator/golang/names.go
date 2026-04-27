// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"go/token"
	"reflect"
	"strings"
	"unicode"
)

var initialisms = map[string]string{
	"API": "API", "ASCII": "ASCII", "CPU": "CPU", "CSS": "CSS", "DNS": "DNS", "EOF": "EOF",
	"GUID": "GUID", "HTML": "HTML", "HTTP": "HTTP", "HTTPS": "HTTPS", "ID": "ID", "IP": "IP",
	"JSON": "JSON", "QPS": "QPS", "RAM": "RAM", "RPC": "RPC", "SLA": "SLA", "SMTP": "SMTP",
	"SQL": "SQL", "SSH": "SSH", "TCP": "TCP", "TLS": "TLS", "TTL": "TTL", "UDP": "UDP",
	"UI": "UI", "UID": "UID", "URI": "URI", "URL": "URL", "UTF8": "UTF8", "UUID": "UUID",
	"VM": "VM", "XML": "XML", "XMPP": "XMPP", "XSRF": "XSRF", "XSS": "XSS",
}

func (g *Generator) publicName(name string) string {
	if g.nameResolver != nil {
		if resolved := g.nameResolver(name); resolved != "" {
			return resolved
		}
	}
	return toPublicName(name)
}

func toPublicName(name string) string {
	parts := splitIdentifier(name)
	if len(parts) == 0 {
		return "Value"
	}
	var b strings.Builder
	for _, p := range parts {
		upper := strings.ToUpper(p)
		if v, ok := initialisms[upper]; ok {
			b.WriteString(v)
			continue
		}
		rs := []rune(strings.ToLower(p))
		rs[0] = unicode.ToUpper(rs[0])
		b.WriteString(string(rs))
	}
	out := b.String()
	first := []rune(out)[0]
	if unicode.IsDigit(first) {
		return "Value" + out
	}
	return out
}

func toPrivateName(name string) string {
	pub := toPublicName(name)
	parts := splitCamel(pub)
	parts[0] = strings.ToLower(parts[0])
	return strings.Join(parts, "")
}

func splitIdentifier(name string) []string {
	var raw []string
	var b strings.Builder
	flush := func() {
		if b.Len() > 0 {
			raw = append(raw, b.String())
			b.Reset()
		}
	}
	for _, r := range name {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		default:
			flush()
		}
	}
	flush()
	var parts []string
	for _, part := range raw {
		parts = append(parts, splitCamel(part)...)
	}
	return parts
}

func splitCamel(value string) []string {
	rs := []rune(value)
	if len(rs) == 0 {
		return nil
	}
	var parts []string
	start := 0
	for i := 1; i < len(rs); i++ {
		prev := rs[i-1]
		cur := rs[i]
		var next rune
		if i+1 < len(rs) {
			next = rs[i+1]
		}
		lowerToUpper := unicode.IsLower(prev) && unicode.IsUpper(cur)
		acronymToWord := unicode.IsUpper(prev) && unicode.IsUpper(cur) && next != 0 && unicode.IsLower(next)
		if lowerToUpper || acronymToWord {
			parts = append(parts, string(rs[start:i]))
			start = i
		}
	}
	parts = append(parts, string(rs[start:]))
	return parts
}

func refName(ref string) string {
	if ref == "" {
		return ""
	}
	i := strings.LastIndex(ref, "/")
	if i < 0 || i == len(ref)-1 {
		return ref
	}
	return ref[i+1:]
}

func uniqueName(base string, used map[string]struct{}) string {
	if base == "" {
		base = "Value"
	}
	if _, ok := used[base]; !ok {
		used[base] = struct{}{}
		return base
	}
	for i := 2; ; i++ {
		name := base + intString(i)
		if _, ok := used[name]; !ok {
			used[name] = struct{}{}
			return name
		}
	}
}

func intString(v int) string {
	const digits = "0123456789"
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = digits[v%10]
		v /= 10
	}
	return string(buf[i:])
}

func validatePackageName(name string) error {
	if name == "" || !token.IsIdentifier(name) || token.Lookup(name).IsKeyword() {
		return wrapPath(ErrInvalidPackageName, name)
	}
	return nil
}

func derefType(t reflect.Type) reflect.Type {
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

func typeName(t reflect.Type) string {
	if t == nil {
		return ""
	}
	if name := t.Name(); name != "" {
		return name
	}
	return toPublicName(t.Kind().String())
}

func interfaceKey(target any) reflect.Type {
	if target == nil {
		return nil
	}
	t := reflect.TypeOf(target)
	if t.Kind() == reflect.Pointer && t.Elem().Kind() == reflect.Interface {
		return t.Elem()
	}
	return nil
}
