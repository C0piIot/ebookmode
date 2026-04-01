FROM golang:1.25-alpine AS base
WORKDIR /app
ARG BUILD_VERSION=dev
ENV BUILD_VERSION=$BUILD_VERSION
ARG GIT_REV=HEAD
ENV GIT_REV=$GIT_REV

EXPOSE 8080

FROM base AS development
CMD ["go", "run", "."]

FROM base AS builder
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o ebookmode .

FROM alpine:3.21 AS production
RUN adduser -D -u 1000 app
WORKDIR /app
COPY --from=builder /app/ebookmode .
COPY templates/ templates/
COPY static/ static/
USER app
EXPOSE 8080
CMD ["./ebookmode"]
