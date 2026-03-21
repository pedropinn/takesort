# Stage 1: Build static binary
FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.version=${VERSION}" -o /bin/takesort ./cmd/takesort/

# Stage 2: Minimal runtime image
FROM alpine:latest

RUN addgroup -S takesort && adduser -S takesort -G takesort

COPY --from=builder /bin/takesort /usr/local/bin/takesort

RUN mkdir -p /temp/conflicts /temp/errors /media && \
    chown -R takesort:takesort /temp /media

USER takesort

ENTRYPOINT ["/usr/local/bin/takesort"]
