package main

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	readability "codeberg.org/readeck/go-readability/v2"
	"golang.org/x/net/html"
)

var buildVersion = "dev"
var gitRev = "HEAD"
var urlPattern = regexp.MustCompile(`\bhttps?://\S+`)

var (
	homeTmpl    = template.Must(template.ParseFiles("templates/layout.html", "templates/home.html"))
	articleTmpl = template.Must(template.ParseFiles("templates/layout.html", "templates/article.html"))
	errorTmpl   = template.Must(template.ParseFiles("templates/layout.html", "templates/error.html"))
)

type pageData struct {
	Build      string
	GitRev     string
	URL        string
	URLEncoded string
	Host       string
	Title      string
	Excerpt    string
	Article    template.HTML
	Error      error
}

func getURL(r *http.Request) string {
	q := r.URL.Query()
	u := strings.TrimSpace(q.Get("url"))
	if u == "" {
		if m := urlPattern.FindString(q.Get("text")); m != "" {
			u = m
		} else if m := urlPattern.FindString(q.Get("title")); m != "" {
			u = m
		}
	}
	if u == "" {
		return ""
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		u = "https://" + u
	}
	return u
}

func rewriteLinks(n *html.Node, base *url.URL) {
	if n.Type == html.ElementNode && n.Data == "a" {
		hasRel := false
		for i, a := range n.Attr {
			if a.Key == "href" {
				if resolved, err := base.Parse(a.Val); err == nil {
					n.Attr[i].Val = "/?url=" + url.QueryEscape(resolved.String())
				}
			}
			if a.Key == "rel" {
				hasRel = true
			}
		}
		if !hasRel {
			n.Attr = append(n.Attr, html.Attribute{Key: "rel", Val: "nofollow"})
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		rewriteLinks(c, base)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	rawURL := getURL(r)
	base := pageData{
		Build:  buildVersion,
		GitRev: gitRev,
		URL:    rawURL,
		Host:   r.Host,
	}

	if rawURL == "" {
		homeTmpl.ExecuteTemplate(w, "layout", base)
		return
	}
	base.URLEncoded = url.QueryEscape(rawURL)

	renderError := func(err error) {
		base.Error = err
		w.WriteHeader(http.StatusBadGateway)
		errorTmpl.ExecuteTemplate(w, "layout", base)
	}

	resp, err := http.Get(rawURL) //nolint:gosec
	if err != nil {
		renderError(err)
		return
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		renderError(fmt.Errorf("Invalid content type %q", ct))
		return
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		renderError(err)
		return
	}

	article, err := readability.FromReader(resp.Body, parsedURL)
	if err != nil {
		renderError(err)
		return
	}
	if article.Node == nil {
		renderError(fmt.Errorf("Error processing document html"))
		return
	}

	rewriteLinks(article.Node, parsedURL)

	var sb strings.Builder
	if err := article.RenderHTML(&sb); err != nil {
		renderError(err)
		return
	}

	base.Title = article.Title()
	base.Excerpt = article.Excerpt()
	base.Article = template.HTML(sb.String()) //nolint:gosec

	articleTmpl.ExecuteTemplate(w, "layout", base)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("request", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr, "duration", time.Since(start))
	})
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/site.webmanifest", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/site.webmanifest")
	})
	slog.Info("listening on :8080")
	if err := http.ListenAndServe(":8080", loggingMiddleware(mux)); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
