package crawler

import (
	"context"
	"testing"
)

func TestCrawlerVisitSeeds(t *testing.T) {
	c := New(3)
	c.SetMaxURLs(3)
	seeds := []string{
		"https://example.com",
		"https://golang.org",
		"https://go.dev",
	}

	results, err := c.Run(context.Background(), seeds)

	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	for _, seed := range seeds {
		if !c.Visited(seed) {
			t.Fatalf("expected %q to be visited", seed)
		}
	}

	if got := c.VisitedCount(); got != len(seeds) {
		t.Fatalf("expected %d visited URLS, got %d", len(seeds), got)
	}

	if got := len(results); got != len(seeds) {
		t.Fatalf("expected %d results, got %d", len(seeds), got)
	}
}

func TestCrawlerDeduplicatesSeeds(t *testing.T) {
	c := New(3)
	c.SetMaxURLs(2)
	seeds := []string{
		"https://example.com",
		"https://example.com",
		"https://go.dev",
	}

	results, err := c.Run(context.Background(), seeds)
	if err != nil {
		t.Fatalf("Run returne error: %v", err)
	}

	if got := len(results); got != 2 {
		t.Fatalf("expected 2 visited URLS, got %d", got)
	}
}

func TestCrawlerSchedulesDiscoveredLinks(t *testing.T) {
	c := New(3)
	c.SetMaxURLs(10)

	results, err := c.Run(context.Background(), []string{"https://example.com"})

	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if len(results) != 10 {
		t.Fatalf("expected 10 results, got %d", len(results))
	}

	if c.VisitedCount() != 10 {
		t.Fatalf("expected 10 visited URLs, got %d", c.VisitedCount())
	}
}
