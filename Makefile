.PHONY: build test lint run clean docker-build docker-up docker-down docker-logs

BINARY := takesort

build:
	go build -o bin/$(BINARY) ./cmd/takesort/

test:
	go test ./... -v

lint:
	golangci-lint run ./...

run: build
	./bin/$(BINARY)

clean:
	rm -rf bin/

docker-build:
	docker build -t $(BINARY) .

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f
