package crawler

import (
	"net/url"
)

func normalizeURL(rawURL string) (string, bool) {
	u, err := url.Parse(rawURL)

	if err != nil {
		return "", false
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return "", false
	}

	u.Fragment = ""
	return u.String(), true
}
