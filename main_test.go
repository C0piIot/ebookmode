package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// --- getURL ---

func TestGetURL(t *testing.T) {
	tests := []struct {
		query string
		want  string
	}{
		{"url=https://example.com", "https://example.com"},
		{"url=http://example.com", "http://example.com"},
		{"url=example.com", "https://example.com"},
		{"url=+https://example.com+", "https://example.com"},
		{"text=check+out+https://example.com+today", "https://example.com"},
		{"title=see+https://example.com", "https://example.com"},
		{"url=https://primary.com&text=https://secondary.com", "https://primary.com"},
		{"text=https://text.com&title=https://title.com", "https://text.com"},
		{"", ""},
		{"url=&text=hello&title=world", ""},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/?"+tt.query, nil)
			if got := getURL(r); got != tt.want {
				t.Errorf("getURL(%q) = %q, want %q", tt.query, got, tt.want)
			}
		})
	}
}

// --- rewriteLinks ---

func parseFragment(t *testing.T, s string) []*html.Node {
	t.Helper()
	ctx := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
	nodes, err := html.ParseFragment(strings.NewReader(s), ctx)
	if err != nil {
		t.Fatalf("parse fragment: %v", err)
	}
	return nodes
}

func renderNodes(nodes []*html.Node) string {
	var sb strings.Builder
	for _, n := range nodes {
		html.Render(&sb, n) //nolint:errcheck
	}
	return sb.String()
}

func TestRewriteLinks_RelativeURL(t *testing.T) {
	base, _ := url.Parse("https://example.com/article")
	nodes := parseFragment(t, `<a href="/about">About</a>`)
	for _, n := range nodes {
		rewriteLinks(n, base)
	}
	got := renderNodes(nodes)
	want := "/?url=https%3A%2F%2Fexample.com%2Fabout"
	if !strings.Contains(got, want) {
		t.Errorf("relative href not rewritten; got %q", got)
	}
}

func TestRewriteLinks_AbsoluteURL(t *testing.T) {
	base, _ := url.Parse("https://example.com/")
	nodes := parseFragment(t, `<a href="https://other.com/page">Link</a>`)
	for _, n := range nodes {
		rewriteLinks(n, base)
	}
	got := renderNodes(nodes)
	want := "/?url=https%3A%2F%2Fother.com%2Fpage"
	if !strings.Contains(got, want) {
		t.Errorf("absolute href not rewritten; got %q", got)
	}
}

func TestRewriteLinks_AddsRelNofollow(t *testing.T) {
	base, _ := url.Parse("https://example.com/")
	nodes := parseFragment(t, `<a href="/page">Link</a>`)
	for _, n := range nodes {
		rewriteLinks(n, base)
	}
	got := renderNodes(nodes)
	if !strings.Contains(got, `rel="nofollow"`) {
		t.Errorf("rel=nofollow not added; got %q", got)
	}
}

func TestRewriteLinks_PreservesExistingRel(t *testing.T) {
	base, _ := url.Parse("https://example.com/")
	nodes := parseFragment(t, `<a href="/page" rel="noopener">Link</a>`)
	for _, n := range nodes {
		rewriteLinks(n, base)
	}
	got := renderNodes(nodes)
	if strings.Count(got, "rel=") != 1 {
		t.Errorf("expected exactly one rel attribute; got %q", got)
	}
}

func TestRewriteLinks_Nested(t *testing.T) {
	base, _ := url.Parse("https://example.com/")
	nodes := parseFragment(t, `<div><p><a href="/deep">Deep</a></p></div>`)
	for _, n := range nodes {
		rewriteLinks(n, base)
	}
	got := renderNodes(nodes)
	if !strings.Contains(got, "/?url=") {
		t.Errorf("nested link not rewritten; got %q", got)
	}
}

// --- handler ---

const articleHTML = `<!DOCTYPE html>
<html>
<head><title>Test Article</title></head>
<body>
<article>
<h1>Test Article Title</h1>
<p>Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod
tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam,
quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo.</p>
<p>Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore
eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident.</p>
<a href="/related">Related article</a>
</article>
</body>
</html>`

func TestHandlerHomePage(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "ebookmode") {
		t.Errorf("home page missing expected content")
	}
}

func TestHandlerArticle(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, articleHTML)
	}))
	defer ts.Close()

	r := httptest.NewRequest("GET", "/?url="+url.QueryEscape(ts.URL), nil)
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "Test Article Title") {
		t.Errorf("article title missing from response")
	}
	// links should be rewritten
	if !strings.Contains(body, "/?url=") {
		t.Errorf("links not rewritten in article")
	}
}

func TestHandlerInvalidContentType(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		fmt.Fprint(w, "%PDF-1.4")
	}))
	defer ts.Close()

	r := httptest.NewRequest("GET", "/?url="+url.QueryEscape(ts.URL), nil)
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Invalid content type") {
		t.Errorf("expected content-type error message; got: %s", w.Body.String())
	}
}

func TestHandlerUnreachableURL(t *testing.T) {
	r := httptest.NewRequest("GET", "/?url=http://localhost:1", nil)
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", w.Code)
	}
}

func TestHandlerURLFromTextParam(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, articleHTML)
	}))
	defer ts.Close()

	sharedText := "Read this article " + ts.URL
	r := httptest.NewRequest("GET", "/?text="+url.QueryEscape(sharedText), nil)
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
}
