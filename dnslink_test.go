package dnslink

import (
	"testing"
)

func TestBuildPath(t *testing.T) {
	tests := []struct {
		name         string
		namespace    string
		identifier   string
		replacement  string // empty means use prefix (namespace)
		originalPath string
		expected     string
	}{
		{
			name:         "swarm with bzz replacement, root path",
			namespace:    "swarm",
			identifier:   "abc123",
			replacement:  "/bzz",
			originalPath: "/",
			expected:     "/bzz/abc123/",
		},
		{
			name:         "swarm with bzz replacement, subpath",
			namespace:    "swarm",
			identifier:   "abc123",
			replacement:  "/bzz",
			originalPath: "/index.html",
			expected:     "/bzz/abc123/index.html",
		},
		{
			name:         "swarm with bzz replacement, nested subpath",
			namespace:    "swarm",
			identifier:   "abc123",
			replacement:  "/bzz",
			originalPath: "/assets/style.css",
			expected:     "/bzz/abc123/assets/style.css",
		},
		{
			name:         "ipfs without replacement, root path",
			namespace:    "ipfs",
			identifier:   "QmXyz789",
			replacement:  "",
			originalPath: "/",
			expected:     "/ipfs/QmXyz789/",
		},
		{
			name:         "ipfs without replacement, subpath",
			namespace:    "ipfs",
			identifier:   "QmXyz789",
			replacement:  "",
			originalPath: "/readme.md",
			expected:     "/ipfs/QmXyz789/readme.md",
		},
		{
			name:         "identifier with nested path",
			namespace:    "swarm",
			identifier:   "abc123/subdir",
			replacement:  "/bzz",
			originalPath: "/file.txt",
			expected:     "/bzz/abc123/subdir/file.txt",
		},
		{
			name:         "empty original path",
			namespace:    "swarm",
			identifier:   "abc123",
			replacement:  "/bzz",
			originalPath: "",
			expected:     "/bzz/abc123/",
		},
		{
			name:         "identifier already has trailing slash",
			namespace:    "swarm",
			identifier:   "abc123/",
			replacement:  "/bzz",
			originalPath: "/index.html",
			expected:     "/bzz/abc123/index.html", // correctly handles trailing slash
		},
		{
			name:         "original path without leading slash",
			namespace:    "swarm",
			identifier:   "abc123",
			replacement:  "/bzz",
			originalPath: "file.txt", // no leading slash
			expected:     "/bzz/abc123/file.txt",
		},
		{
			name:         "real swarm hash",
			namespace:    "swarm",
			identifier:   "d1e8b6f5a3c2b9e4d7f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2",
			replacement:  "/bzz",
			originalPath: "/",
			expected:     "/bzz/d1e8b6f5a3c2b9e4d7f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2/",
		},
		{
			name:         "replacement without leading slash",
			namespace:    "swarm",
			identifier:   "abc123",
			replacement:  "bzz",
			originalPath: "/index.html",
			expected:     "bzz/abc123/index.html",
		},
		{
			name:         "replacement with trailing slash",
			namespace:    "swarm",
			identifier:   "abc123",
			replacement:  "/bzz/",
			originalPath: "/index.html",
			expected:     "/bzz/abc123/index.html",
		},
		{
			name:         "double slash in original path",
			namespace:    "swarm",
			identifier:   "abc123",
			replacement:  "/bzz",
			originalPath: "//index.html",
			expected:     "/bzz/abc123//index.html", // preserves double slash, don't want to assume it's not intentional.
		},
		{
			name:         "path with query-like string (not actual query)",
			namespace:    "swarm",
			identifier:   "abc123",
			replacement:  "/bzz",
			originalPath: "/page?foo=bar",
			expected:     "/bzz/abc123/page?foo=bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildPath(tt.namespace, tt.identifier, tt.replacement, tt.originalPath)
			if result != tt.expected {
				t.Errorf("buildPath() = %q, want %q", result, tt.expected)
			}
		})
	}
}
