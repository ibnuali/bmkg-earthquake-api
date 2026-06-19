# Stage 1: Build
FROM golang:1.26.4-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w" \
    -o /app/earthquake-api ./cmd/api

# Stage 2: Run
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -g '' appuser

COPY --from=builder /app/earthquake-api /usr/local/bin/

USER appuser

EXPOSE 8080

ENTRYPOINT ["earthquake-api"]
