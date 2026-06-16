package crawler

import (
	"slices"
	"testing"
)

func TestExtractLinks(t *testing.T) {
	body := []byte(`
		<html>
			<body>
				<a href="https://example.com/about">About</a>
				<a href='https://example.com/contact'>Contact</a>
			</body>
		</html>`)

	got := extractLinks(body)

	want := []string{
		"https://example.com/about",
		"https://example.com/contact",
	}

	if !slices.Equal(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestResolveLink(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		rawLink string
		want    string
		ok      bool
	}{
		{
			name:    "absolute url",
			baseURL: "https://example.com/docs/",
			rawLink: "https://example.com/about",
			want:    "https://example.com/about",
			ok:      true,
		},
		{
			name:    "absolute path",
			baseURL: "https://example.com/docs/page",
			rawLink: "/about",
			want:    "https://example.com/about",
			ok:      true,
		},
		{
			name:    "relative path",
			baseURL: "https://example.com/docs/page",
			rawLink: "next",
			want:    "https://example.com/docs/next",
			ok:      true,
		},
		{
			name:    "reject mailto",
			baseURL: "https://example.com/docs/page",
			rawLink: "mailto:test@example.com",
			ok:      false,
		},
		{
			name:    "reject javascript",
			baseURL: "https://example.com/docs/page",
			rawLink: "javascript:void(0)",
			ok:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := resolveLink(tt.baseURL, tt.rawLink)

			if ok != tt.ok {
				t.Fatalf("expected ok %v, got %v", tt.ok, ok)
			}

			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name   string
		rawURL string
		want   string
		ok     bool
	}{
		{
			name:   "removes fragment",
			rawURL: "https://example.com/about#team",
			want:   "https://example.com/about",
			ok:     true,
		},
		{
			name:   "keeps query string",
			rawURL: "https://example.com/search?q=go#top",
			want:   "https://example.com/search?q=go",
			ok:     true,
		},
		{
			name:   "rejects mailto",
			rawURL: "mailto:test@example.com",
			ok:     false,
		},
		{
			name:   "rejects javascript",
			rawURL: "javascript:void(0)",
			ok:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := normalizeURL(tt.rawURL)

			if ok != tt.ok {
				t.Fatalf("expected ok %v, got %v", tt.ok, ok)
			}

			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
