// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	high "github.com/pb33f/libopenapi/datamodel/high/arazzo"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/datamodel/low/arazzo"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
)

var resolveFilepathAbs = filepath.Abs

// OpenAPIDocumentFactory creates a parsed OpenAPI document from raw bytes.
// The sourceURL provides location context for relative reference resolution.
type OpenAPIDocumentFactory func(sourceURL string, bytes []byte) (*v3high.Document, error)

// ArazzoDocumentFactory creates a parsed Arazzo document from raw bytes.
// The sourceURL provides location context for relative reference resolution.
type ArazzoDocumentFactory func(sourceURL string, bytes []byte) (*high.Arazzo, error)

// ResolveConfig configures how source descriptions are resolved.
type ResolveConfig struct {
	OpenAPIFactory OpenAPIDocumentFactory // Creates *v3high.Document from bytes
	ArazzoFactory  ArazzoDocumentFactory  // Creates *high.Arazzo from bytes
	BaseURL        string
	HTTPHandler    func(url string) ([]byte, error)
	HTTPClient     *http.Client
	FSRoots        []string

	Timeout        time.Duration // Per-source fetch timeout (default: 30s)
	MaxBodySize    int64         // Max response body in bytes (default: 10MB)
	AllowedSchemes []string      // URL scheme allowlist (default: ["https", "http", "file"])
	AllowedHosts   []string      // Host allowlist (nil = allow all)
	MaxSources     int           // Max source descriptions to resolve (default: 50)
}

// ResolvedSource represents a successfully resolved source description.
type ResolvedSource struct {
	Name            string           // SourceDescription name
	URL             string           // Resolved URL
	Type            string           // "openapi" or "arazzo"
	OpenAPIDocument *v3high.Document // Non-nil when Type == "openapi"
	ArazzoDocument  *high.Arazzo     // Non-nil when Type == "arazzo"
}

// ResolveSources resolves all source descriptions in an Arazzo document.
func ResolveSources(doc *high.Arazzo, config *ResolveConfig) ([]*ResolvedSource, error) {
	if doc == nil {
		return nil, fmt.Errorf("nil arazzo document")
	}

	if config == nil {
		config = &ResolveConfig{}
	}

	// Apply defaults
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxBodySize == 0 {
		config.MaxBodySize = 10 * 1024 * 1024 // 10MB
	}
	if config.MaxSources == 0 {
		config.MaxSources = 50
	}
	if len(config.AllowedSchemes) == 0 {
		config.AllowedSchemes = []string{"https", "http", "file"}
	}
	if config.HTTPClient == nil && config.HTTPHandler == nil {
		config.HTTPClient = &http.Client{Timeout: config.Timeout}
	}

	if len(doc.SourceDescriptions) > config.MaxSources {
		return nil, fmt.Errorf("too many source descriptions: %d (max %d)", len(doc.SourceDescriptions), config.MaxSources)
	}

	resolved := make([]*ResolvedSource, 0, len(doc.SourceDescriptions))
	for _, sd := range doc.SourceDescriptions {
		if sd == nil {
			return nil, fmt.Errorf("%w: source description is nil", ErrSourceDescLoadFailed)
		}

		rs := &ResolvedSource{Name: sd.Name}

		parsedURL, err := parseAndResolveSourceURL(sd.URL, config.BaseURL)
		if err != nil {
			return nil, fmt.Errorf("%w (%q): %v", ErrSourceDescLoadFailed, sd.Name, err)
		}

		if err := validateSourceURL(parsedURL, config); err != nil {
			return nil, fmt.Errorf("%w (%q): %v", ErrSourceDescLoadFailed, sd.Name, err)
		}

		docBytes, resolvedURL, err := fetchSourceBytes(parsedURL, config)
		if err != nil {
			return nil, fmt.Errorf("%w (%q): %v", ErrSourceDescLoadFailed, sd.Name, err)
		}

		rs.URL = resolvedURL
		rs.Type = strings.ToLower(sd.Type)
		if rs.Type == "" {
			rs.Type = "openapi" // Default per spec
		}

		switch rs.Type {
		case v3.OpenAPILabel:
			if config.OpenAPIFactory == nil {
				return nil, fmt.Errorf("%w (%q): no OpenAPIFactory configured", ErrSourceDescLoadFailed, sd.Name)
			}
			openDoc, factoryErr := config.OpenAPIFactory(resolvedURL, docBytes)
			if factoryErr != nil {
				return nil, fmt.Errorf("%w (%q): %v", ErrSourceDescLoadFailed, sd.Name, factoryErr)
			}
			rs.OpenAPIDocument = openDoc
		case arazzo.ArazzoLabel:
			if config.ArazzoFactory == nil {
				return nil, fmt.Errorf("%w (%q): no ArazzoFactory configured", ErrSourceDescLoadFailed, sd.Name)
			}
			arazzoDoc, factoryErr := config.ArazzoFactory(resolvedURL, docBytes)
			if factoryErr != nil {
				return nil, fmt.Errorf("%w (%q): %v", ErrSourceDescLoadFailed, sd.Name, factoryErr)
			}
			rs.ArazzoDocument = arazzoDoc
		default:
			return nil, fmt.Errorf("%w (%q): unknown source type %q", ErrSourceDescLoadFailed, sd.Name, rs.Type)
		}

		resolved = append(resolved, rs)
	}

	// Auto-attach OpenAPI source documents to the Arazzo model so that
	// validation and the engine can resolve operation references without
	// the caller needing to wire this up manually.
	for _, rs := range resolved {
		if rs.OpenAPIDocument != nil {
			doc.AddOpenAPISourceDocument(rs.OpenAPIDocument)
		}
	}

	return resolved, nil
}

func parseAndResolveSourceURL(rawURL, base string) (*url.URL, error) {
	if rawURL == "" {
		return nil, fmt.Errorf("missing source URL")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid source URL %q: %w", rawURL, err)
	}

	// Detect Windows absolute paths (e.g. "C:\Users\..." or "D:/foo/bar").
	// url.Parse misinterprets the drive letter as a URL scheme ("c:", "d:").
	// A single-letter scheme is always a Windows drive letter; real URL schemes
	// are at least two characters. Use strings.ReplaceAll instead of
	// filepath.ToSlash so backslashes are normalized on all platforms.
	if len(parsed.Scheme) == 1 {
		parsed = &url.URL{Scheme: "file", Path: strings.ReplaceAll(rawURL, `\`, "/")}
	}

	// Resolve relative URLs against BaseURL when provided.
	if parsed.Scheme == "" && base != "" {
		baseURL, err := url.Parse(base)
		if err != nil {
			return nil, fmt.Errorf("invalid base URL %q: %w", base, err)
		}
		parsed = baseURL.ResolveReference(parsed)
	}

	if parsed.Scheme == "" {
		parsed = &url.URL{Scheme: "file", Path: parsed.Path}
	}

	return parsed, nil
}

func validateSourceURL(sourceURL *url.URL, config *ResolveConfig) error {
	if !containsFold(config.AllowedSchemes, sourceURL.Scheme) {
		return fmt.Errorf("scheme %q is not allowed", sourceURL.Scheme)
	}

	if len(config.AllowedHosts) > 0 && sourceURL.Scheme != "file" {
		host := sourceURL.Hostname()
		if !containsFold(config.AllowedHosts, host) {
			return fmt.Errorf("host %q is not allowed", host)
		}
	}

	return nil
}

func fetchSourceBytes(sourceURL *url.URL, config *ResolveConfig) ([]byte, string, error) {
	switch sourceURL.Scheme {
	case "http", "https":
		b, err := fetchHTTPSourceBytes(sourceURL.String(), config)
		if err != nil {
			return nil, "", err
		}
		return b, sourceURL.String(), nil
	case "file":
		filePath := sourceURL.Path
		// On Windows, file URLs without a leading slash (e.g. "file://C:/path")
		// cause url.Parse to place the drive letter in Host ("C:") and strip it
		// from Path ("/path"). Reconstruct the full path.
		if len(sourceURL.Host) == 2 && sourceURL.Host[1] == ':' {
			filePath = sourceURL.Host + filePath
		}
		path, err := resolveFilePath(filePath, config.FSRoots)
		if err != nil {
			return nil, "", err
		}

		b, err := readFileWithLimit(path, config.MaxBodySize)
		if err != nil {
			return nil, "", err
		}

		return b, (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String(), nil
	default:
		return nil, "", fmt.Errorf("unsupported source scheme %q", sourceURL.Scheme)
	}
}

func fetchHTTPSourceBytes(sourceURL string, config *ResolveConfig) ([]byte, error) {
	if config.HTTPHandler != nil {
		b, err := config.HTTPHandler(sourceURL)
		if err != nil {
			return nil, err
		}
		if int64(len(b)) > config.MaxBodySize {
			return nil, fmt.Errorf("response body exceeds max size of %d bytes", config.MaxBodySize)
		}
		return b, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return nil, err
	}

	client := getResolveHTTPClient(config)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, config.MaxBodySize+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}

	if int64(len(body)) > config.MaxBodySize {
		return nil, fmt.Errorf("response body exceeds max size of %d bytes", config.MaxBodySize)
	}

	return body, nil
}

func getResolveHTTPClient(config *ResolveConfig) *http.Client {
	if config != nil && config.HTTPClient != nil {
		return config.HTTPClient
	}
	timeout := 30 * time.Second
	if config != nil && config.Timeout > 0 {
		timeout = config.Timeout
	}
	return &http.Client{Timeout: timeout}
}

func readFileWithLimit(path string, maxBytes int64) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > maxBytes {
		return nil, fmt.Errorf("file exceeds max size of %d bytes", maxBytes)
	}
	return os.ReadFile(path)
}

func resolveFilePath(path string, roots []string) (string, error) {
	unescapedPath, err := url.PathUnescape(path)
	if err != nil {
		return "", fmt.Errorf("failed to decode file path %q: %w", path, err)
	}

	cleaned := filepath.Clean(unescapedPath)

	// If no roots are configured, resolve relative paths from the current working directory.
	if len(roots) == 0 {
		if filepath.IsAbs(cleaned) {
			return cleaned, nil
		}
		return filepath.Abs(cleaned)
	}
	absRoots := make([]string, 0, len(roots))
	for _, root := range roots {
		absRoot, err := resolveFilepathAbs(root)
		if err != nil {
			continue
		}
		absRoots = append(absRoots, absRoot)
	}
	canonicalRoots := canonicalizeRoots(absRoots)

	// Absolute paths must be inside one of the configured roots.
	// Canonicalize the cleaned path for comparison only (resolves Windows 8.3
	// short names and macOS /var -> /private/var symlinks) so that the path
	// matches canonicalRoots. The original cleaned path is returned to callers.
	if filepath.IsAbs(cleaned) {
		canonical := cleaned
		if resolved, err := filepath.EvalSymlinks(cleaned); err == nil {
			canonical = resolved
		}
		if !isPathWithinRoots(canonical, canonicalRoots) {
			return "", fmt.Errorf("file path %q is outside configured roots", cleaned)
		}
		if err := ensureResolvedPathWithinRoots(cleaned, canonicalRoots); err != nil {
			return "", err
		}
		return cleaned, nil
	}

	// Relative paths are resolved against each root in order.
	// Use absRoots for building candidates (preserves original paths) but
	// canonicalRoots for security checks.
	for _, root := range absRoots {
		candidate := filepath.Join(root, cleaned)
		if !isPathWithinRoots(candidate, []string{root}) {
			continue
		}
		if _, lstatErr := os.Lstat(candidate); lstatErr == nil {
			if err := ensureResolvedPathWithinRoots(candidate, canonicalRoots); err != nil {
				return "", err
			}
			return candidate, nil
		} else if !errors.Is(lstatErr, os.ErrNotExist) {
			return "", lstatErr
		}
	}

	return "", fmt.Errorf("file path %q not found within configured roots", cleaned)
}

// isPathWithinRoots checks whether path falls inside at least one of the given roots.
// Both path and roots must be absolute paths; no filepath.Abs calls are made here
// since callers already guarantee absolute inputs.
func isPathWithinRoots(path string, roots []string) bool {
	for _, root := range roots {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			continue
		}
		if rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..") {
			return true
		}
	}
	return false
}

func canonicalizeRoots(roots []string) []string {
	canonicalRoots := make([]string, 0, len(roots))
	for _, root := range roots {
		absRoot, err := resolveFilepathAbs(root)
		if err != nil {
			continue
		}
		resolvedRoot, err := filepath.EvalSymlinks(absRoot)
		if err == nil {
			canonicalRoots = append(canonicalRoots, resolvedRoot)
			continue
		}
		if !errors.Is(err, os.ErrNotExist) {
			continue
		}
		canonicalRoots = append(canonicalRoots, absRoot)
	}
	return canonicalRoots
}

func ensureResolvedPathWithinRoots(path string, roots []string) error {
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if !isPathWithinRoots(resolvedPath, roots) {
		return fmt.Errorf("file path %q is outside configured roots", path)
	}
	return nil
}


func containsFold(values []string, value string) bool {
	for _, v := range values {
		if strings.EqualFold(v, value) {
			return true
		}
	}
	return false
}
