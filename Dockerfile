# Stage 1: Build static binary
FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.version=${VERSION}" -o /bin/takesort ./cmd/takesort/

# Stage 2: Minimal runtime image
FROM alpine:3.21

COPY --from=builder /bin/takesort /usr/local/bin/takesort

ENTRYPOINT ["/usr/local/bin/takesort"]
