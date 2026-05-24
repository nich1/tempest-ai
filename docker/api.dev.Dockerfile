# syntax=docker/dockerfile:1
# Dev image for the API: hot-reloads with air on file changes.
FROM golang:1.25-alpine
WORKDIR /src
RUN apk add --no-cache git ca-certificates && \
    go install github.com/air-verse/air@v1.65.3
EXPOSE 8080
CMD ["air", "-c", "/src/docker/air.api.toml"]
