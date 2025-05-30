FROM golang:1.24-alpine AS builder
WORKDIR /app
RUN apk add git
COPY . .
RUN CGO_ENABLED=0 go build -mod=vendor -ldflags="-s -w"  -trimpath -o server main.go

# Minimal runtime image using Chainguard Wolfi base
FROM cgr.dev/chainguard/wolfi-base:latest
WORKDIR /app
COPY --from=builder /app/server ./server
EXPOSE 8080
ENTRYPOINT ["/app/server"]
