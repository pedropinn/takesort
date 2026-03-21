.PHONY: build test lint run clean docker-build docker-up docker-down docker-logs docker-push release

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

docker-push:
ifndef DOCKERHUB_USERNAME
	$(error DOCKERHUB_USERNAME is not set. Usage: make docker-push DOCKERHUB_USERNAME=yourusername)
endif
	docker build -t $(DOCKERHUB_USERNAME)/$(BINARY):latest .
	docker push $(DOCKERHUB_USERNAME)/$(BINARY):latest

release:
ifndef VERSION
	$(error VERSION is not set. Usage: make release VERSION=v1.0.0)
endif
ifeq ($(filter v%,$(VERSION)),)
	$(error VERSION must start with 'v' (e.g., v1.0.0). Got: $(VERSION))
endif
	git tag $(VERSION)
	git push origin $(VERSION)
