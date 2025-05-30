FROM golang:1.24-alpine AS builder
WORKDIR /app
RUN apk add git
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=vendor -ldflags="-s -w 'main.version=$(git describe --tags --always --dirty)' -X 'main.commit=$(git rev-parse --short HEAD)' -X 'main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)' -X 'main.trimpath=$(go env GOROOT)' -trimpath" -o server main.go

# Minimal runtime image using Chainguard Wolfi base
FROM cgr.dev/chainguard/wolfi-base:latest
WORKDIR /app
COPY --from=builder /app/server ./server
EXPOSE 8080
ENTRYPOINT ["/app/server"]
