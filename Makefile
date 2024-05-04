.DEFAULT_GOAL := build
BINARY_NAME=docker-lock

mod:
	go mod tidy
	go mod vendor
	go mod verify

format-go:
	@printf "%s\n" "==== Running go-fmt ====="
	gofmt -s -w .

go-staticcheck:
	# https://github.com/dominikh/go-tools
	staticcheck ./...

golines-format:
	@printf "%s\n" "==== Running golines ====="
	golines --write-output --ignored-dirs=vendor .

lint: go-staticcheck

format: format-go golines-format

format-lint: format lint

build: format-lint
	go build -o bin/${BINARY_NAME} *.go
