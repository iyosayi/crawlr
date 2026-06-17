package crawler

import (
	"context"
	"net/url"
	"sync"
	"time"
)

type Crawler struct {
	workers      int
	maxURLs      int
	maxDepth     int
	processor    Processor
	sameHostOnly bool
}

type runState struct {
	mu      sync.Mutex
	visited map[string]struct{}
}

type Result struct {
	URL         string
	Links       []string
	Err         error
	Depth       int
	StatusCode  int
	ContentType string
	BodyBytes   int64
	Body        []byte
}

type Options struct {
	Workers      int
	MaxURLs      int
	MaxDepth     int
	SameHostOnly bool
	Processor    Processor
}

type RunResult struct {
	Results      []Result
	VisitedCount int
	FailedCount  int
	StartedAt    time.Time
	FinishedAt   time.Time
	Duration     time.Duration
}

type Processor func(ctx context.Context, url string) (Result, error)

type Job struct {
	URL   string
	Depth int
}

func newRunState() *runState {
	return &runState{
		visited: make(map[string]struct{}),
	}
}

func buildRunResult(startedAt time.Time, results []Result, visitedCount int) RunResult {
	finishedAt := time.Now()

	return RunResult{
		Results:      results,
		VisitedCount: visitedCount,
		FailedCount:  countFailures(results),
		StartedAt:    startedAt,
		FinishedAt:   finishedAt,
		Duration:     finishedAt.Sub(startedAt),
	}
}

func countFailures(results []Result) int {
	count := 0

	for _, result := range results {
		if result.Err != nil {
			count++
		}
	}
	return count
}

func hostOf(rawURL string) (string, bool) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", false
	}
	if u.Host == "" {
		return "", false
	}
	return u.Host, true
}

func New(opts Options) *Crawler {
	if opts.Workers <= 0 {
		opts.Workers = 1
	}

	if opts.MaxURLs <= 0 {
		opts.MaxURLs = 10
	}

	if opts.MaxDepth < 0 {
		opts.MaxDepth = 0
	}

	if opts.Processor == nil {
		opts.Processor = func(ctx context.Context, url string) (Result, error) {
			return Result{URL: url}, nil
		}
	}
	return &Crawler{
		workers:      opts.Workers,
		maxURLs:      opts.MaxURLs,
		processor:    opts.Processor,
		maxDepth:     opts.MaxDepth,
		sameHostOnly: opts.SameHostOnly,
	}
}

// Key ownership rule:
// workers send results
// the coordinator closes results after all workers finish

func (c *Crawler) startWorkers(ctx context.Context, jobs <-chan Job, results chan<- Result) *sync.WaitGroup {
	var wg sync.WaitGroup

	for range c.workers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-jobs:
					if !ok {
						return
					}

					result, err := c.process(ctx, job.URL)
					if err != nil {
						result = Result{
							URL:   job.URL,
							Err:   err,
							Depth: job.Depth,
						}
					}
					result.Depth = job.Depth

					select {
					case <-ctx.Done():
						return
					case results <- result:
					}
				}
			}
		}()
	}
	return &wg
}

func (c *Crawler) scheduleSeeds(seeds []string, allowedHosts map[string]struct{}, enqueue func(Job) bool) int {
	scheduled := 0

	for _, seed := range seeds {
		normalized, ok := normalizeURL(seed)
		if !ok {
			continue
		}

		if c.sameHostOnly {
			host, ok := hostOf(normalized)
			if !ok {
				continue
			}
			allowedHosts[host] = struct{}{}
		}

		if enqueue(Job{URL: normalized, Depth: 0}) {
			scheduled++
		}
	}
	return scheduled
}

func (c *Crawler) runScheduler(ctx context.Context, results <-chan Result, closeJobs func(), enqueue func(Job) bool, initialPending int, state *runState) ([]Result, error) {
	pending := initialPending
	out := make([]Result, 0, state.visitedCount())

	for pending > 0 {
		select {
		case <-ctx.Done():
			closeJobs()

			for result := range results {
				out = append(out, result)
			}
			return out, ctx.Err()
		case result, ok := <-results:
			if !ok {
				closeJobs()
				return out, ctx.Err()
			}
			pending--
			out = append(out, result)
			pending += c.scheduleDiscoveredLinks(result, enqueue)
		}
	}
	closeJobs()
	for result := range results {
		out = append(out, result)
	}

	return out, ctx.Err()
}

func (c *Crawler) scheduleDiscoveredLinks(result Result, enqueue func(Job) bool) int {
	scheduled := 0

	for _, link := range result.Links {
		if enqueue(Job{URL: link, Depth: result.Depth + 1}) {
			scheduled++
		}
	}
	return scheduled
}

func (c *Crawler) enqueue(ctx context.Context, jobs chan<- Job, state *runState, allowedHosts map[string]struct{}, job Job) bool {

	if job.Depth > c.maxDepth {
		return false
	}

	if c.sameHostOnly {
		host, ok := hostOf(job.URL)
		if !ok {
			return false
		}

		if _, ok := allowedHosts[host]; !ok {
			return false
		}
	}

	if !state.trySchedule(c.maxURLs, job.URL) {
		return false
	}
	select {
	case <-ctx.Done():
		return false
	case jobs <- job:
		return true
	}
}

func (c *Crawler) Run(ctx context.Context, seeds []string) (RunResult, error) {
	if err := ctx.Err(); err != nil {
		return RunResult{}, err
	}
	startedAt := time.Now()

	jobs := make(chan Job, c.maxURLs)
	results := make(chan Result)
	state := newRunState()
	allowedHosts := make(map[string]struct{})

	var closeJobsOnce sync.Once

	closeJobs := func() {
		closeJobsOnce.Do(func() {
			close(jobs)
		})
	}

	wg := c.startWorkers(ctx, jobs, results)

	go func() {
		wg.Wait()
		close(results)
	}()

	enqueue := func(job Job) bool {
		return c.enqueue(ctx, jobs, state, allowedHosts, job)
	}
	pending := c.scheduleSeeds(seeds, allowedHosts, enqueue)

	out, err := c.runScheduler(ctx, results, closeJobs, enqueue, pending, state)

	return buildRunResult(startedAt, out, state.visitedCount()), err
}

func (c *runState) trySchedule(maxURLs int, url string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.visited) >= maxURLs {
		return false
	}

	if _, ok := c.visited[url]; ok {
		return false
	}

	c.visited[url] = struct{}{}
	return true
}

func (s *runState) visitedCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.visited)
}

func (c *Crawler) process(ctx context.Context, url string) (Result, error) {
	return c.processor(ctx, url)
}
