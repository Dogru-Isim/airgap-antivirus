# Configuration
BINARY_NAME ?= antivirus
VERSION ?= 0.1.0
BUILD_DIR ?= $(shell pwd)/build
GO_BUILD_FLAGS ?= -v
GO_TEST_FLAGS ?= -race -cover -v
GO_LDFLAGS ?= -s -w -X main.version=$(VERSION)

# Tools
GOLANGCI_LINT = golangci-lint
DELVE = dlv

.PHONY: all
all: build

##@ Development

.PHONY: run
run: ## Run the application
	@go build $(GO_BUILD_FLAGS) -ldflags '$(GO_LDFLAGS)' -o $(BUILD_DIR)/$(BINARY_NAME) ./$(BINARY_NAME) && $(BUILD_DIR)/$(BINARY_NAME) && rm $(BUILD_DIR)/$(BINARY_NAME)

.PHONY: debug
debug: ## Debug the application with Delve
	@$(DELVE) debug $(shell pwd)/cmd/build/

.PHONY: test
test: ## Run unit tests
	@go test $(GO_TEST_FLAGS) ../... -o $(shell pwd)/cmd/build/

.PHONY: test-cover
test-cover: test ## Run tests and generate coverage report
	@go tool cover -html=coverage.out

.PHONY: lint
lint: ## Run linter
	@$(GOLANGCI_LINT) run

.PHONY: tidy
tidy: ## Tidy dependencies
	@go mod tidy -v

##@ Build

.PHONY: build
build: ## Build binary
	@go build $(GO_BUILD_FLAGS) -ldflags '$(GO_LDFLAGS)' -o $(BUILD_DIR)/$(BINARY_NAME) ./$(BINARY_NAME)

.PHONY: build-all
build-all: ## Build for all platforms
	@GOOS=linux GOARCH=amd64 go build $(GO_BUILD_FLAGS) -ldflags '$(GO_LDFLAGS)' -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./$(BINARY_NAME)
	@GOOS=linux GOARCH=arm64 go build $(GO_BUILD_FLAGS) -ldflags '$(GO_LDFLAGS)' -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./$(BINARY_NAME)
	#@GOOS=darwin GOARCH=amd64 go build $(GO_BUILD_FLAGS) -ldflags '$(GO_LDFLAGS)' -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./$(BINARY_NAME)
	#@GOOS=windows GOARCH=amd64 go build $(GO_BUILD_FLAGS) -ldflags '$(GO_LDFLAGS)' -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./$(BINARY_NAME)

##@ Docker

.PHONY: docker-build
docker-build: ## Build Docker image
	@docker build -t $(BINARY_NAME):$(VERSION) .

.PHONY: docker-run
docker-run: ## Run Docker container
	@docker run -p 8080:8080 $(BINARY_NAME):$(VERSION)

##@ Utility

.PHONY: clean
clean: ## Clean build artifacts
	@rm -rf $(BUILD_DIR) coverage.out

.PHONY: gen
gen: ## Generate code (protobuf, mocks, etc.)
	@go generate ./...

.PHONY: check-vuln
check-vuln: ## Check for vulnerabilities
	@govulncheck ./...

.PHONY: vendor
vendor: ## Vendor dependencies
	@go mod vendor

##@ Help

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

