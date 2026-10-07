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
	"BIC": "BIC", "CVC": "CVC", "CVV": "CVV", "GUID": "GUID", "HTML": "HTML", "HTTP": "HTTP",
	"HTTPS": "HTTPS", "IBAN": "IBAN", "ID": "ID", "IP": "IP", "JSON": "JSON", "JWT": "JWT",
	"MD5": "MD5", "QPS": "QPS", "RAM": "RAM", "RPC": "RPC", "SHA1": "SHA1", "SHA256": "SHA256",
	"SHA512": "SHA512", "SLA": "SLA", "SMTP": "SMTP",
	"SQL": "SQL", "SSH": "SSH", "TCP": "TCP", "TLS": "TLS", "TTL": "TTL", "UDP": "UDP",
	"UI": "UI", "UID": "UID", "URI": "URI", "URL": "URL", "UTF8": "UTF8", "UUID": "UUID",
	"VM": "VM", "XML": "XML", "XMPP": "XMPP", "XSRF": "XSRF", "XSS": "XSS",
}

// symbolWords names the leading symbols that distinguish JSON property names
// such as "_id" and "@type" from their plain spellings.
var symbolWords = map[rune]string{
	'_': "Underscore", '$': "Dollar", '@': "At", '#': "Hash",
	'+': "Plus", '-': "Minus", '.': "Dot", '~': "Tilde",
}

// irregularPlurals holds plurals that the suffix rules in singular would get
// wrong. An empty value marks a word that has no distinct singular.
var irregularPlurals = map[string]string{
	"aliases": "alias", "analyses": "analysis", "analytics": "", "caches": "cache",
	"children": "child", "cookies": "cookie", "criteria": "criterion", "data": "",
	"indices": "index", "matrices": "matrix", "media": "", "men": "man", "movies": "movie",
	"news": "", "people": "person", "series": "", "species": "", "statuses": "status",
	"vertices": "vertex", "women": "woman",
}

func (g *Generator) publicName(name string) string {
	if g.typeNameResolver != nil {
		if resolved := g.typeNameResolver(name); resolved != "" {
			return resolved
		}
	}
	if g.nameResolver != nil {
		if resolved := g.nameResolver(name); resolved != "" {
			return resolved
		}
	}
	return toPublicName(name)
}

func (g *Generator) fieldName(name string) string {
	if g.fieldNameResolver != nil {
		if resolved := g.fieldNameResolver(name); resolved != "" {
			return resolved
		}
	}
	if g.nameResolver != nil {
		if resolved := g.nameResolver(name); resolved != "" {
			return resolved
		}
	}
	return toPublicName(name)
}

func (g *Generator) enumValueName(name string) string {
	if g.enumValueNameResolver != nil {
		if resolved := g.enumValueNameResolver(name); resolved != "" {
			return resolved
		}
	}
	if g.nameResolver != nil {
		if resolved := g.nameResolver(name); resolved != "" {
			return resolved
		}
	}
	return toPublicName(enumNameSeed(name))
}

func (g *Generator) componentTypeName(name string) string {
	if g.componentTypeNames != nil {
		if resolved := g.componentTypeNames[name]; resolved != "" {
			return resolved
		}
	}
	return g.publicName(name)
}

func (g *Generator) nestedTypeName(parent, child string) string {
	childName := g.publicName(child)
	if parent == "" {
		return childName
	}
	return parent + g.nestedTypeNameDelimiter + childName
}

func enumNameSeed(value string) string {
	value = strings.Trim(strings.ReplaceAll(strings.ReplaceAll(value, "-", "_"), " ", "_"), "_")
	if value == "" {
		return "empty"
	}
	return value
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
		// A plural initialism such as IDS reaches here only with its S removed.
		if v, ok := initialisms[strings.TrimSuffix(upper, "S")]; ok {
			b.WriteString(v)
			b.WriteByte('s')
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

// PublicName converts an OpenAPI name to the exported Go identifier used by
// the model generator. SDK emitters use it to keep method and model references
// consistent.
func PublicName(name string) string {
	return toPublicName(name)
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
		lowerToUpper := (unicode.IsLower(prev) || unicode.IsDigit(prev)) && unicode.IsUpper(cur)
		acronymToWord := unicode.IsUpper(prev) && unicode.IsUpper(cur) && next != 0 && unicode.IsLower(next) &&
			!pluralAcronym(rs, i+1)
		if lowerToUpper || acronymToWord {
			parts = append(parts, string(rs[start:i]))
			start = i
		}
	}
	parts = append(parts, string(rs[start:]))
	return parts
}

// joinTypeName qualifies leaf with owner. Words that end owner and also begin
// leaf are written once, so Contact and ContactEmail join as ContactEmail.
func joinTypeName(owner, leaf string) string {
	ownerWords, leafWords := splitCamel(owner), splitCamel(leaf)
	for overlap := min(len(ownerWords), len(leafWords)-1); overlap > 0; overlap-- {
		if strings.Join(ownerWords[len(ownerWords)-overlap:], "") == strings.Join(leafWords[:overlap], "") {
			return owner + strings.Join(leafWords[overlap:], "")
		}
	}
	return owner + leaf
}

// itemName names the element type of an array from the array's Go name.
// A plural final word becomes singular (FundingEvents is FundingEvent);
// otherwise Item is appended (EmploymentHistory is EmploymentHistoryItem).
func itemName(name string) string {
	words := splitCamel(name)
	last := words[len(words)-1]
	if singular, ok := singular(last); ok {
		return strings.Join(words[:len(words)-1], "") + singular
	}
	return name + "Item"
}

// singular returns the singular Go word for an English plural, or false when
// word is not a plural it can confidently reduce.
func singular(word string) (string, bool) {
	if initialism, ok := initialisms[strings.ToUpper(strings.TrimSuffix(word, "s"))]; ok && strings.HasSuffix(word, "s") {
		return initialism, true
	}
	lower := strings.ToLower(word)
	if irregular, ok := irregularPlurals[lower]; ok {
		return toPublicName(irregular), irregular != ""
	}
	switch {
	case strings.HasSuffix(lower, "ies") && len(lower) > 4:
		lower = strings.TrimSuffix(lower, "ies") + "y"
	case strings.HasSuffix(lower, "sses"), strings.HasSuffix(lower, "xes"),
		strings.HasSuffix(lower, "ches"), strings.HasSuffix(lower, "shes"), strings.HasSuffix(lower, "zzes"):
		lower = strings.TrimSuffix(lower, "es")
	case strings.HasSuffix(lower, "ss"), strings.HasSuffix(lower, "us"), strings.HasSuffix(lower, "is"):
		return "", false
	case strings.HasSuffix(lower, "s") && len(lower) > 3:
		lower = strings.TrimSuffix(lower, "s")
	default:
		return "", false
	}
	return toPublicName(lower), true
}

// symbolPrefixedName spells out the leading symbols of a JSON property name
// before its base Go name, so "_id" becomes UnderscoreID. The result is false
// when the name has no leading symbol with a word.
func symbolPrefixedName(source, base string) (string, bool) {
	var prefix strings.Builder
	for _, r := range source {
		word, ok := symbolWords[r]
		if !ok {
			break
		}
		prefix.WriteString(word)
	}
	if prefix.Len() == 0 {
		return "", false
	}
	return prefix.String() + base, true
}

// pluralAcronym reports whether the lowercase s at rs[at] pluralizes the
// acronym before it, as in IDs or URLsFor, rather than starting a word.
func pluralAcronym(rs []rune, at int) bool {
	return rs[at] == 's' && (at+1 == len(rs) || !unicode.IsLower(rs[at+1]))
}

// RefName returns the RFC 6901-decoded final path segment of ref.
func RefName(ref string) string {
	if ref == "" {
		return ""
	}
	segment := ref
	i := strings.LastIndex(ref, "/")
	if i >= 0 && i < len(ref)-1 {
		segment = ref[i+1:]
	}
	return unescapeJSONPointer(segment)
}

func refName(ref string) string {
	return RefName(ref)
}

func unescapeJSONPointer(value string) string {
	value = strings.ReplaceAll(value, "~1", "/")
	return strings.ReplaceAll(value, "~0", "~")
}

func (g *Generator) refTypeName(ref string) string {
	if ref == "" {
		return ""
	}
	if strings.HasPrefix(ref, "#/") {
		return g.componentTypeName(refName(ref))
	}
	if g.externalRefResolver != nil {
		if resolved := g.externalRefResolver(ref); resolved != "" {
			return resolved
		}
	}
	return g.publicName(refName(ref))
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
		name := base + conflictNameDelimiter + intString(i)
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
