# ebookmode

Mozilla's reader mode as a web service, designed for ebook browsers and slow connections. Enter any URL and get back a clean, readable version of the article stripped of ads, navigation, and clutter.

Built with Go and [go-readability](https://codeberg.org/readeck/go-readability).

## Features

- Extracts article content using Mozilla's Readability algorithm
- Rewrites all links so you can keep browsing in reader mode
- Bookmarklet support — drag to your bookmarks bar to convert any page
- Browser-local bookmarks saved in `localStorage`
- PWA manifest for installing as a home screen app
- Web Share API integration

## Requirements

- Docker and Docker Compose

## Run in development

```bash
docker compose up
```

The server starts at <http://localhost:8080>. Source files are mounted into the container; restart the container to pick up changes.

## Build for production

```bash
docker build --target production -t ebookmode .
docker run -p 8080:8080 ebookmode
```

To pass a build version label:

```bash
docker build --target production --build-arg BUILD_VERSION=1.0.0 -t ebookmode .
```

## Run tests

```bash
docker run --rm -v "$(pwd)":/app -w /app -e GOTOOLCHAIN=auto golang:1.24-alpine go test ./...
```

## Usage

Navigate to the running instance and enter a URL, or use the bookmarklet:

```
javascript: (() => { window.location.href='https://<your-host>/?url=' + encodeURIComponent(window.location.href) })()
```

The service also accepts URLs via the `text` and `title` query parameters, which matches the format used by the browser Web Share Target API.

## License

GNU General Public License v3.0 — see [LICENSE](LICENSE).
