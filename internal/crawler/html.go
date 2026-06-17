package crawler

import (
	"bytes"
	"golang.org/x/net/html"
	"net/url"
)

func extractLinks(body []byte) []string {
	doc, err := html.Parse(bytes.NewReader(body))

	if err != nil {
		return nil
	}

	var links []string

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					links = append(links, attr.Val)
					break
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
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
