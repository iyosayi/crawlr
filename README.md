# crawlr

A concurrent web crawler written in Go. `crawlr` uses a worker-pool architecture to crawl URLs from seed pages, follow discovered links, deduplicate visits, and respect crawl limits and cancellation.

## Status

This project is in active development. The concurrency and scheduling layer is implemented and tested. Current work is focused on building the crawler architecture incrementally, starting with worker pools, scheduling, cancellation, and URL deduplication before introducing real HTTP fetching and HTML parsing.

## Features

- **Worker pool** — configurable number of concurrent workers
- **Deduplication** — each URL is visited at most once
- **Crawl limit** — cap the total number of URLs to fetch (default: 10)
- **Context cancellation** — workers and the coordinator respect `context.Context` cancellation

## Project structure

```
crawlr/
├── cmd/crawlr/          # CLI entrypoint (not yet implemented)
├── internal/crawler/    # Core crawler package
│   ├── crawler.go
│   └── crawler_test.go
└── go.mod
```

## Requirements

- Go 1.25.4 or later

## Installation

```bash
git clone https://github.com/iyosayi/crawlr.git
cd crawlr
```

## Usage

The crawler is currently usable as a Go library. The CLI in `cmd/crawlr` is a placeholder.

```go
package main

import (
    "context"
    "fmt"

    "github.com/iyosayi/crawlr/internal/crawler"
)

func main() {
    c := crawler.New(crawler.Options{
    Workers: 3,
    MaxURLs: 10,
}) 

    seeds := []string{
        "https://example.com",
    }

    results, err := c.Run(context.Background(), seeds)
    if err != nil {
        panic(err)
    }

    for _, r := range results {
        fmt.Printf("%s -> %v\n", r.URL, r.Links)
    }
}
```

### API

| Function / method | Description |
|-------------------|-------------|
| `New(workers int)` | Create a crawler with the given number of workers |
| `Run(ctx, seeds []string)` | Start crawling from seed URLs; returns results and any error |

### Result

Each crawled URL produces a `Result`:

```go
type Result struct {
    URL   string
    Links []string
}
```

## How it works

```
Seeds → Coordinator → jobs channel → Workers → process() → results channel → Coordinator
                                              ↑                                    |
                                              └──────── enqueue new links ─────────┘
```

1. Seed URLs are enqueued on a job channel.
2. Worker goroutines pull jobs, call `process()` on each URL, and send results back.
3. The coordinator collects results and enqueues any newly discovered links.
4. Crawling stops when there is no pending work, the URL limit is reached, or the context is cancelled.

## Running tests

```bash
go test ./...
```

With verbose output:

```bash
go test ./... -v
```

### Race Detection

```bash
go test -race ./...
```

## Roadmap

### Stage 1 — Concurrent Scheduler ✅

- [x] Worker pool
- [x] Dynamic job scheduling
- [x] Dynamic URL discovery
- [x] URL deduplication

### Stage 2 — Real Fetching

- [ ] HTTP client integration
- [ ] Request timeouts
- [ ] Context-aware requests
- [ ] Error handling and retries

### Stage 3 — HTML Parsing

- [ ] Extract links from HTML
- [ ] Normalize discovered URLs
- [ ] Relative URL resolution

### Stage 4 — Crawl Control

- [ ] Maximum depth
- [ ] Domain scoping
- [ ] URL normalization
- [ ] Maximum page limits

### Stage 5 — Polite Crawling

- [ ] robots.txt support
- [ ] Crawl delays
- [ ] User-Agent configuration

### Stage 6 — Production Hardening

- [ ] CLI interface
- [ ] Structured logging
- [ ] Metrics
- [ ] Benchmarking
- [ ] go test -race verification

## Concepts Practiced

This project is focused on learning and applying:

- Goroutines
- Worker pools
- Channels
- Select statements
- Context cancellation
- WaitGroups
- Mutexes
- Race detection
- Concurrent scheduling
- Ownership patterns
- Backpressure
- Channel ownership
- Goroutine lifecycle management

## License

MIT
