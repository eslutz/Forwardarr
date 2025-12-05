# Build Stage
FROM golang:1.25-alpine AS builder

ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X github.com/eslutz/gluetarr/pkg/version.Version=${VERSION} -X github.com/eslutz/gluetarr/pkg/version.Commit=${COMMIT} -X github.com/eslutz/gluetarr/pkg/version.Date=${DATE}" \
    -o gluetarr ./cmd/gluetarr

# Runtime Stage
FROM alpine:3.23
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
# Create non-root user
RUN addgroup -g 1000 gluetarr && adduser -u 1000 -G gluetarr -D gluetarr

COPY --from=builder /app/gluetarr /app/gluetarr

USER gluetarr
EXPOSE 9090

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9090/health || exit 1

ENTRYPOINT ["/app/gluetarr"]
