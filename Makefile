.PHONY: help lint test run docker-build docker-run docker-clean

help:
	@echo "Available targets:"
	@echo "  lint           - Run golangci-lint"
	@echo "  test           - Run tests with coverage"
	@echo "  run            - Run locally"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run Docker container"
	@echo "  docker-clean   - Remove Docker image"

lint:
	@golangci-lint run ./...

test:
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out

run:
	@go run ./cmd/main

docker-build:
	@docker build -t mock-oidc:latest .

docker-run: docker-build
	@docker run -it -p 8080:8080 \
		-e LOG_LEVEL=info \
		-e GIN_MODE=release \
		mock-oidc:latest

docker-clean:
	@docker rmi mock-oidc:latest || true

.PHONY: clean
clean:
	@rm -f coverage.out
	@go clean
