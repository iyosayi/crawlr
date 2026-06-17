package crawler

import (
	"context"
	"errors"
	"maps"
	"slices"
	"sort"
	"testing"
)

func resultURLs(results []Result) []string {
	urls := make([]string, 0, len(results))

	for _, result := range results {
		urls = append(urls, result.URL)
	}
	sort.Strings(urls)
	return urls
}

func resultDepths(results []Result) map[string]int {
	depths := make(map[string]int, len(results))

	for _, result := range results {
		depths[result.URL] = result.Depth
	}
	return depths
}

func TestCrawlerVisitSeeds(t *testing.T) {
	c := New(Options{
		Workers: 3,
		MaxURLs: 3,
		Processor: func(ctx context.Context, url string) (Result, error) {
			return Result{
				URL: url,
				Links: []string{
					url + "/a",
					url + "/b",
				},
			}, nil
		},
	})
	seeds := []string{
		"https://example.com",
		"https://golang.org",
		"https://go.dev",
	}

	run, err := c.Run(context.Background(), seeds)

	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	results := run.Results
	if got := len(results); got != len(seeds) {
		t.Fatalf("expected %d visited URLS, got %d", len(seeds), got)
	}

	if got := len(results); got != len(seeds) {
		t.Fatalf("expected %d results, got %d", len(seeds), got)
	}
}

func TestCrawlerDeduplicatesSeeds(t *testing.T) {
	c := New(Options{
		Workers: 3,
		MaxURLs: 2,
		Processor: func(ctx context.Context, url string) (Result, error) {
			return Result{
				URL: url,
				Links: []string{
					url + "/a",
					url + "/b",
				},
			}, nil
		},
	})
	seeds := []string{
		"https://example.com",
		"https://example.com",
		"https://go.dev",
	}

	run, err := c.Run(context.Background(), seeds)
	if err != nil {
		t.Fatalf("Run returne error: %v", err)
	}

	results := run.Results
	if got := len(results); got != 2 {
		t.Fatalf("expected 2 visited URLS, got %d", got)
	}
}

func TestCrawlerSchedulesDiscoveredLinks(t *testing.T) {
	c := New(Options{
		Workers:  3,
		MaxURLs:  10,
		MaxDepth: 10,
		Processor: func(ctx context.Context, url string) (Result, error) {
			return Result{
				URL: url,
				Links: []string{
					url + "/a",
					url + "/b",
				},
			}, nil
		},
	})
	run, err := c.Run(context.Background(), []string{"https://example.com"})

	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	results := run.Results
	if len(results) != 10 {
		t.Fatalf("expected 10 results, got %d", len(results))
	}

}

func TestCrawlerRecordsProcessorErrors(t *testing.T) {
	expectedErr := errors.New("fetch failed")

	c := New(Options{
		Workers: 2,
		MaxURLs: 3,
		Processor: func(ctx context.Context, url string) (Result, error) {
			return Result{}, expectedErr
		},
	})

	run, err := c.Run(context.Background(), []string{
		"https://example.com",
	})

	if err != nil {
		t.Fatalf("Run returned err: %v", err)
	}

	results := run.Results
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Err == nil {
		t.Fatalf("expected result error, got nil")
	}

	if !errors.Is(results[0].Err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, results[0].Err)
	}
}

func TestCrawlerReturnsContextErrorOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := New(Options{
		Workers: 1,
		MaxURLs: 10,
	})

	run, err := c.Run(ctx, []string{"https://example.com"})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	results := run.Results

	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestCrawlerMaxDepthZeroOnlyProcessesSeeds(t *testing.T) {
	c := New(Options{
		Workers:  2,
		MaxURLs:  10,
		MaxDepth: 0,
		Processor: func(ctx context.Context, url string) (Result, error) {
			return Result{
				URL: url,
				Links: []string{
					url + "/a",
					url + "/b",
				},
			}, nil
		},
	})

	run, err := c.Run(context.Background(), []string{"https://example.com"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	results := run.Results
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Depth != 0 {
		t.Fatalf("expected depth 0, got %d", results[0].Depth)
	}
}

func TestCrawlerMaxDepthOneProcessesDiscoveredLinks(t *testing.T) {
	c := New(Options{
		Workers:  2,
		MaxURLs:  10,
		MaxDepth: 1,
		Processor: func(ctx context.Context, url string) (Result, error) {
			return Result{
				URL: url,
				Links: []string{
					url + "/a",
					url + "/b",
				},
			}, nil
		},
	})

	run, err := c.Run(context.Background(), []string{"https://example.com"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	results := run.Results

	got := resultURLs(results)

	want := []string{
		"https://example.com",
		"https://example.com/a",
		"https://example.com/b",
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if !slices.Equal(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestCrawlerAssignsDepths(t *testing.T) {
	c := New(Options{
		Workers:  2,
		MaxURLs:  10,
		MaxDepth: 1,
		Processor: func(ctx context.Context, url string) (Result, error) {
			return Result{
				URL: url,
				Links: []string{
					url + "/a",
					url + "/b",
				},
			}, nil
		},
	})

	run, err := c.Run(context.Background(), []string{"https://example.com"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	results := run.Results
	depths := resultDepths(results)

	want := map[string]int{
		"https://example.com":   0,
		"https://example.com/a": 1,
		"https://example.com/b": 1,
	}

	if !maps.Equal(depths, want) {
		t.Fatalf("expected %v, got %v", want, depths)
	}
}

func TestCrawlerDoesNotProcessLinksBeyondMaxDepth(t *testing.T) {
	c := New(Options{
		Workers:  2,
		MaxURLs:  100,
		MaxDepth: 1,
		Processor: func(ctx context.Context, url string) (Result, error) {
			return Result{
				URL: url,
				Links: []string{
					url + "/a",
				},
			}, nil
		},
	})

	run, err := c.Run(context.Background(), []string{"https://example.com"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	results := run.Results
	got := resultURLs(results)

	want := []string{
		"https://example.com",
		"https://example.com/a",
	}

	if !slices.Equal(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestCrawlerNormalizesSeedURLs(t *testing.T) {
	c := New(Options{
		Workers:  2,
		MaxURLs:  10,
		MaxDepth: 0,
	})

	run, err := c.Run(context.Background(), []string{
		"https://example.com/about#team",
		"https://example.com/about#pricing",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	results := run.Results
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].URL != "https://example.com/about" {
		t.Fatalf("expected normalized URL, got %q", results[0].URL)
	}
}

func TestCrawlerSameHostOnly(t *testing.T) {
	c := New(Options{
		Workers:      2,
		MaxURLs:      10,
		MaxDepth:     1,
		SameHostOnly: true,
		Processor: func(ctx context.Context, url string) (Result, error) {
			return Result{
				URL: url,
				Links: []string{
					"https://example.com/about",
					"https://other.com/page",
				},
			}, nil
		},
	})

	run, err := c.Run(context.Background(), []string{
		"https://example.com",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	results := run.Results

	got := resultURLs(results)

	want := []string{
		"https://example.com",
		"https://example.com/about",
	}

	if !slices.Equal(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestCrawlerRunResultCountsFailures(t *testing.T) {
	expectedErr := errors.New("fetch failed")

	c := New(Options{
		Workers:  2,
		MaxURLs:  2,
		MaxDepth: 0,
		Processor: func(ctx context.Context, url string) (Result, error) {
			if url == "https://bad.com" {
				return Result{}, expectedErr
			}
			return Result{
				URL: url,
			}, nil
		},
	})

	run, err := c.Run(context.Background(), []string{
		"https://good.com",
		"https://bad.com",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if run.FailedCount != 1 {
		t.Fatalf("expected 1 failure, got %d", run.FailedCount)
	}
}

func TestCrawlerRunResultIncludesTiming(t *testing.T) {
	c := New(Options{
		Workers:  1,
		MaxURLs:  1,
		MaxDepth: 0,
	})

	run, err := c.Run(context.Background(), []string{"https://example.com"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if run.StartedAt.IsZero() {
		t.Fatal("expected StartedAt to be set")
	}

	if run.FinishedAt.IsZero() {
		t.Fatal("expected FinishedAt to be set")
	}

	if run.Duration < 0 {
		t.Fatalf("expected non-negative duration, got %v", run.Duration)
	}

	if run.FinishedAt.Before(run.StartedAt) {
		t.Fatal("expected FinishedAt to be after StartedAt")
	}
}

func TestCrawlerCanRunMultipleTimes(t *testing.T) {
	c := New(Options{
		Workers:  2,
		MaxURLs:  10,
		MaxDepth: 0,
	})

	first, err := c.Run(context.Background(), []string{
		"https://example.com",
	})
	if err != nil {
		t.Fatalf("first Run returned error: %v", err)
	}

	second, err := c.Run(context.Background(), []string{
		"https://example.com",
	})

	if err != nil {
		t.Fatalf("Second Run returned error: %v", err)
	}

	if first.VisitedCount != 1 {
		t.Fatalf("expected first run visited count 1, got %d", first.VisitedCount)
	}

	if second.VisitedCount != 1 {
		t.Fatalf("expected second run visited count 1, got %d", second.VisitedCount)
	}
}

func TestCrawlerSameHostOnlyDoesNotLeakAllowedHostsAcrossRuns(t *testing.T) {
	c := New(Options{
		Workers:      2,
		MaxURLs:      10,
		MaxDepth:     1,
		SameHostOnly: true,
		Processor: func(ctx context.Context, url string) (Result, error) {
			return Result{
				URL: url,
				Links: []string{
					"https://example.com/about",
					"https://go.dev/doc",
				},
			}, nil
		},
	})

	first, err := c.Run(context.Background(), []string{
		"https://example.com",
	})
	if err != nil {
		t.Fatalf("first Run returned error: %v", err)
	}

	second, err := c.Run(context.Background(), []string{
		"https://go.dev",
	})
	if err != nil {
		t.Fatalf("second Run returned error: %v", err)
	}

	firstURLs := resultURLs(first.Results)
	firstWant := []string{
		"https://example.com",
		"https://example.com/about",
	}

	if !slices.Equal(firstURLs, firstWant) {
		t.Fatalf("expected first URLs %v, got %v", firstWant, firstURLs)
	}

	secondURLs := resultURLs(second.Results)
	secondWant := []string{
		"https://go.dev",
		"https://go.dev/doc",
	}

	if !slices.Equal(secondURLs, secondWant) {
		t.Fatalf("expected seocnd URLs %v, got %v", secondURLs, secondWant)
	}
}

func TestCrawlerIgnoresInvalidSeeds(t *testing.T) {
	c := New(Options{
		Workers:  2,
		MaxURLs:  10,
		MaxDepth: 0,
	})

	run, err := c.Run(context.Background(), []string{
		"not-a-url",
		"mailto:test@example.com",
		"javascript:void(0)",
		"https://example.com",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	got := resultURLs(run.Results)

	want := []string{
		"https://example.com",
	}

	if !slices.Equal(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}

	if run.VisitedCount != 1 {
		t.Fatalf("expected visited count 1, got %d", run.VisitedCount)
	}
}
