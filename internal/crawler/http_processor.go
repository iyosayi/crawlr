package crawler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type HTTPProcessor struct {
	client       *http.Client
	maxBodyBytes int64
}

func NewHTTPProcessor(client *http.Client) *HTTPProcessor {
	if client == nil {
		client = &http.Client{
			Timeout: 5 * time.Second,
		}
	}

	return &HTTPProcessor{
		client:       client,
		maxBodyBytes: 1 << 20, //1MB
	}
}

func (p *HTTPProcessor) Process(ctx context.Context, url string) (Result, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Result{}, fmt.Errorf("create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("fetch url: %w", err)
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, p.maxBodyBytes+1)

	body, err := io.ReadAll(limited)
	if err != nil {
		return Result{}, fmt.Errorf("read body: %w", err)
	}
	links := []string(nil)
	if strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
		rawLinks := extractLinks(body)

		for _, rawLink := range rawLinks {
			link, ok := resolveLink(url, rawLink)
			if !ok {
				continue
			}
			links = append(links, link)
		}
	}

	bodyBytes := int64(len(body))

	if bodyBytes > p.maxBodyBytes {
		body = body[:p.maxBodyBytes]
		bodyBytes = p.maxBodyBytes
	}

	return Result{
		URL:         url,
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		BodyBytes:   bodyBytes,
		Body:        body,
		Links:       links,
	}, nil
}
