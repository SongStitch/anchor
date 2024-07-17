.DEFAULT_GOAL := help
BINARY_NAME=anchor
LDFLAGS=-s -w

# Package path where the version and commit variables are located
PKG := github.com/songstitch/anchor/cmd

# Dynamically set version and commit using git
VERSION := $(shell git describe --tags --abbrev=0)
COMMIT := $(shell git rev-parse --short HEAD)

# Append -X flags to LDFLAGS
LDFLAGS += -X '$(PKG).version=$(VERSION)' -X '$(PKG).commit=$(COMMIT)'

help: ## Prints the help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
    sort | \
    awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

shellcheck: ## Runs shellcheck on the codebase
	# https://github.com/koalaman/shellcheck
	@printf "%s\n" "==== Running shellcheck ====="
	find . -iname "*.sh" -not -path "./vendor/*" -exec shellcheck {} \;

hadolint: ## Runs hadolint on the Dockerfile
	# https://github.com/hadolint/hadolint/
	@printf "%s\n" "==== Running hadolint ====="
	hadolint *Dockerfile.template

prettier-format: ## Runs prettier format on the codebase
	# https://github.com/prettier/prettier
	@printf "%s\n" "==== Running prettier format ====="
	prettier -w . --log-level=error

prettier-lint: ## Runs prettier lint checking on the codebase
	@printf "%s\n" "==== Running prettier lint check ====="
	prettier -c . --log-level=error

typos: ## Runs typos on the codebase
	# https://github.com/crate-ci/typos
	@printf "%s\n" "==== Running typos ====="
	typos

markdown-toc: ## Generates the MarkDown TOC and writes it to README.md
	# https://github.com/jonschlinkert/markdown-toc
	@printf "%s\n" "==== Generating MarkDown TOC ====="
	markdown-toc -i README.md
	prettier -w README.md --log-level=error

lint: hadolint prettier-lint shellcheck typos go-vet go-staticcheck gosec ## Runs all the linters

format: prettier-format format-go golines-format ## Runs all the formatters

documentation: markdown-toc ## Generates the documentation

format-lint: format lint ## Runs all the formatters and linters

mod: ## Runs go mod tidy, vendor, and verify
	go mod tidy
	go mod vendor
	go mod verify

go-update: ## Updates all the go dependencies
	go list -mod=readonly -m -f '{{if not .Indirect}}{{if not .Main}}{{.Path}}{{end}}{{end}}' all | xargs go get -u
	$(MAKE) mod

gosec: ## Runs gosec on the codebase
	# https://github.com/securego/gosec
	gosec -severity medium -exclude-dir ./vendor/ ./...

format-go: ## Runs go-fmt on the codebase
	@printf "%s\n" "==== Running go-fmt ====="
	gofmt -s -w  *.go cmd/ pkg/

golines-format: ## Runs golines on the codebase
	# https://github.com/segmentio/golines
	@printf "%s\n" "==== Run golines ====="
	golines --write-output --ignored-dirs=vendor .

go-vet: ## Runs go vet on the codebase
	@printf "%s\n" "==== Running go vet ====="
	go vet ./...

go-staticcheck: ## Runs staticcheck on the codebase
	# https://github.com/dominikh/go-tools
	staticcheck ./...

run: ## Runs the binary
	go run main.go $(ARGS)

test: ## Runs the tests
	go test -v ./...

build: format-lint ## Builds the binary for your current platform
	go build -o bin/${BINARY_NAME} main.go

build-prod: format-lint ## Builds the binary for your current platform with production flags
	go build -ldflags="${LDFLAGS}" -o bin/${BINARY_NAME} main.go

build-darwin: format-lint ## Builds the binary for darwin
	@printf "%s\n" "==== Building for Darwin ====="
	env GOOS=darwin GOARCH=arm64 go build -o bin/${BINARY_NAME}_darwin_arm64 main.go
	env GOOS=darwin GOARCH=amd64 go build -o bin/${BINARY_NAME}_darwin_amd64 main.go
	lipo -create -output bin/${BINARY_NAME}_darwin bin/${BINARY_NAME}_darwin_arm64 bin/${BINARY_NAME}_darwin_amd64

build-darwin-prod: ## Builds the binary for darwin with production flags
	@printf "%s\n" "==== Building for Darwin (Production) ====="
	env GOOS=darwin GOARCH=arm64 go build -ldflags="${LDFLAGS}" -o bin/${BINARY_NAME}_darwin_arm64 main.go
	env GOOS=darwin GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o bin/${BINARY_NAME}_darwin_amd64 main.go

build-linux-arm64: format-lint ## Builds the binary for linux arm64
	@printf "%s\n" "==== Building for linux arm64 ====="
	env GOOS=linux GOARCH=arm64 go build -o bin/${BINARY_NAME}_linux_arm64 main.go

build-linux-arm64-prod: ## Builds the binary for linux arm64 with production flags
	@printf "%s\n" "==== Building for linux arm64 (Production) ====="
	env GOOS=linux GOARCH=arm64 go build -ldflags="${LDFLAGS}" -o bin/${BINARY_NAME}_linux_arm64 main.go

build-linux-amd64: format-lint ## Builds the binary for linux amd64
	@printf "%s\n" "==== Building for linux amd64 ====="
	env GOOS=linux GOARCH=amd64 go build -o bin/${BINARY_NAME}_linux_amd64 main.go

build-linux-amd64-prod: ## Builds the binary for linux amd64 with production flags
	@printf "%s\n" "==== Building for linux amd64 (Production) ====="
	env GOOS=linux GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o bin/${BINARY_NAME}_linux_amd64 main.go

build-windows-amd64: format-lint ## Builds the binary for windows amd64
	@printf "%s\n" "==== Building for windows amd64 ====="
	env GOOS=windows GOARCH=amd64 go build -o bin/${BINARY_NAME}_windows_amd64 main.go

build-windows-amd64-prod: ## Builds the binary for windows amd64 with production flags
	@printf "%s\n" "==== Building for windows amd64 (Production) ====="
	env GOOS=windows GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o bin/${BINARY_NAME}_windows_amd64 main.go

build-all: build-darwin build-linux-arm64 build-linux-amd64 build-windows-amd64 ## Builds the binary for all platforms

build-all-prod: build-darwin-prod build-linux-arm64-prod build-linux-amd64-prod build-windows-amd64-prod ## Builds the binary for all platforms with production flags

docker-auth: ## Authenticate to Docker Registry (GHCR)
	@printf "%s\n" "==== Authenticating to Docker Registry (GHCR) ====="
	gh auth token | docker login ghcr.io -u username --password-stdin

clean: ## Cleans the build artifacts
	rm -rf bin/*
