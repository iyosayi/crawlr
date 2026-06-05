package crawler

import (
	"context"
	"sync"
)

type Crawler struct {
	workers int
	mu      sync.Mutex
	visited map[string]struct{}
	maxURLs int
}

type Result struct {
	URL   string
	Links []string
}

func New(workers int) *Crawler {
	if workers <= 0 {
		workers = 1
	}
	return &Crawler{
		workers: workers,
		visited: make(map[string]struct{}),
		maxURLs: 10,
	}
}

// Key ownership rule:
// workers send results
// the coordinator closes results after all workers finish

func (c *Crawler) Run(ctx context.Context, seeds []string) ([]Result, error) {
	jobs := make(chan string, c.maxURLs)
	results := make(chan Result)
	pending := 0

	var wg sync.WaitGroup

	for range c.workers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case url, ok := <-jobs:
					if !ok {
						return
					}

					result, err := c.process(ctx, url)
					if err != nil {
						return
					}

					select {
					case <-ctx.Done():
						return
					case results <- result:
					}
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	enqueue := func(url string) bool {
		if c.VisitedCount() >= c.maxURLs {
			return false
		}

		if !c.markVisited(url) {
			return false
		}
		select {
		case <-ctx.Done():
			return false
		case jobs <- url:
			pending++
			return true
		}
	}

	for _, seed := range seeds {
		enqueue(seed)
	}

	var out []Result

	for pending > 0 {
		select {
		case <-ctx.Done():
			close(jobs)
			return nil, ctx.Err()
		case result := <-results:
			pending--
			out = append(out, result)
			for _, link := range result.Links {
				enqueue(link)
			}
		}
	}
	close(jobs)
	for result := range results {
		out = append(out, result)
	}
	return out, ctx.Err()
}

func (c *Crawler) SetMaxURLs(n int) {
	c.maxURLs = n
}

func (c *Crawler) markVisited(url string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.visited[url]; ok {
		return false
	}

	c.visited[url] = struct{}{}
	return true
}

func (c *Crawler) Visited(url string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, ok := c.visited[url]
	return ok
}

func (c *Crawler) VisitedCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.visited)
}

func (c *Crawler) process(ctx context.Context, url string) (Result, error) {
	select {
	case <-ctx.Done():
		return Result{}, ctx.Err()
	default:
		return Result{URL: url, Links: []string{
			url + "/a",
			url + "/b",
		}}, nil
	}
}
