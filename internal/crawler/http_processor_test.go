package crawler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestHTTPProcessorFetchesURL(t *testing.T) {
	body := "hello crawlr"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(body))
	}))

	defer server.Close()

	p := NewHTTPProcessor(server.Client())
	result, err := p.Process(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Process returned error: %v", err)
	}

	if result.URL != server.URL {
		t.Fatalf("expected URL %q, got %q", server.URL, result.URL)
	}

	if result.StatusCode != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, result.StatusCode)
	}
	if result.ContentType != "text/html" {
		t.Fatalf("expected content type %q, got %q", "text/html", result.ContentType)
	}

	if result.BodyBytes != int64(len(body)) {
		t.Fatalf("expected body size %d, got %d", len(body), result.BodyBytes)
	}

	if string(result.Body) != body {
		t.Fatalf("expected body %q, got %q", body, string(result.Body))
	}
}

func TestHttPProcessorRespectsTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))

	defer server.Close()

	client := &http.Client{
		Timeout: 50 * time.Millisecond,
	}

	p := NewHTTPProcessor(client)

	_, err := p.Process(context.Background(), server.URL)

	if err == nil {
		t.Fatalf("expected timeout error")
	}
}

func TestHTTPProcessorTruncatesLargeBodies(t *testing.T) {
	body := strings.Repeat("a", 20)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(body))
	}))

	defer server.Close()

	p := NewHTTPProcessor(server.Client())
	p.maxBodyBytes = 10

	result, err := p.Process(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("process returned error: %v", err)
	}
	if result.BodyBytes != 10 {
		t.Fatalf("expected body size 10, got %d", result.BodyBytes)
	}

	if len(result.Body) != 10 {
		t.Fatalf("expected body length 10, got %d", len(result.Body))
	}

	if string(result.Body) != strings.Repeat("a", 10) {
		t.Fatalf("expected truncated body, got %q", string(result.Body))
	}
}

func TestHTTPProcessorExtractsHTMLLinks(t *testing.T) {
	body := `
		<html>
			<body>
				<a href="https://example.com/about">About</a>
				<a href="https://example.com/contact">Contact</a>
			</body>
		</html>
	`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	p := NewHTTPProcessor(server.Client())

	result, err := p.Process(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Process returned error: %v", err)
	}

	want := []string{
		"https://example.com/about",
		"https://example.com/contact",
	}

	if !slices.Equal(result.Links, want) {
		t.Fatalf("expected links %v, got %v", want, result.Links)
	}
}
