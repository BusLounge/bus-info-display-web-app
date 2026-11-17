# --- build ---
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# --- run ---
FROM alpine:3.20
# tzdata: the trip generator loads Asia/Colombo. ca-certificates: TLS to the database.
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 app \
    && mkdir -p /data/media \
    && chown app /data/media
WORKDIR /app
COPY --from=build /out/server ./server
# The server reads this file relative to its working directory.
COPY internal/config/config.json ./internal/config/config.json
USER app
ENV PORT=8083 MEDIA_DIR=/data/media
EXPOSE 8083
# Uploaded media lives in /data/media: Docker Compose mounts a named volume there,
# Railway attaches its own volume (Railway does not allow the VOLUME instruction).
HEALTHCHECK --interval=30s --timeout=3s --start-period=20s \
  CMD wget -qO- http://127.0.0.1:${PORT}/health || exit 1
CMD ["./server"]
