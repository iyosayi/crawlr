package crawler

import (
	"net/url"
	"regexp"
)

var hrefRe = regexp.MustCompile(`(?i)<a\s+[^>]*href=["']([^"']+)["']`)

func extractLinks(body []byte) []string {
	matches := hrefRe.FindAllSubmatch(body, -1)

	links := make([]string, 0, len(matches))

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		links = append(links, string(match[1]))
	}
	return links
}

func resolveLink(baseURL, rawLink string) (string, bool) {
	base, err := url.Parse(baseURL)

	if err != nil {
		return "", false
	}

	ref, err := url.Parse(rawLink)
	if err != nil {
		return "", false
	}

	resolved := base.ResolveReference(ref)

	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return "", false
	}
	return normalizeURL(resolved.String())
}
