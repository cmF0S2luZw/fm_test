BINARY=pm
MAIN_DIR=./cmd
CONFIG_DIR=./example/configs

.DEFAULT_GOAL := help


help:
	@echo -e "\033[1;33mAvailable commands:\033[0m"
	@echo ""
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / { printf "  \033[0;32m%-15s\033[0m \033[1;33m%s\033[0m\n", $$1, $$2 }' $(MAKEFILE_LIST)

build:
	@echo -e "\033[0;32mBuilding $(BINARY)...\033[0m"
	go build -o $(BINARY) $(MAIN_DIR)/main.go
	@echo -e "\033[0;32mBuilt: ./$(BINARY)\033[0m"

test:
	@echo -e "\033[0;32mRunning tests...\033[0m"
	go test -v ./...

test-coverage:
	@echo -e "\033[0;32mRunning tests with coverage...\033[0m"
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo -e "\033[0;32mCoverage report: coverage.html\033[0m"

deps:
	@echo -e "\033[0;32mUpdating dependencies...\033[0m"
	go mod tidy
	go mod verify
	@echo -e "\033[0;32mDependencies ready\033[0m"

clean:
	@echo -e "\033[0;32mCleaning...\033[0m"
	rm -f $(BINARY)
	rm -f coverage.out coverage.html
	@echo -e "\033[0;32mDone\033[0m"

create: build
	@if [ -z "$(CONFIG)" ]; then \
		echo -e "\033[1;33mUse: make create CONFIG=packet.json\033[0m"; \
		exit 1; \
	fi
	@echo -e "\033[0;32mExecuting: pm create $(CONFIG)\033[0m"
	./$(BINARY) create $(CONFIG)

update: build
	@if [ -z "$(CONFIG)" ]; then \
		echo -e "\033[1;33mUse: make update CONFIG=packages.json\033[0m"; \
		exit 1; \
	fi
	@echo -e "\033[0;32mExecuting: pm update $(CONFIG)\033[0m"
	./$(BINARY) update $(CONFIG)

example-configs:
	mkdir -p $(CONFIG_DIR)
	@echo -e "\033[0;32mCreating example configs...\033[0m"

	@echo '{
  "name": "app",
  "ver": "1.0",
  "targets": [
	"./test_data/*.txt",
	{ "path": "./test_data/*.log", "exclude": "*.tmp" }
	],
	"packets": [
	{ "name": "utils", "ver": "1.5" }
	]
	}' > $(CONFIG_DIR)/packet.json

	@echo '{
	"packages": [
	{ "name": "app", "ver": "1.0" },
	{ "name": "utils" }
	]
	}' > $(CONFIG_DIR)/packages.json

	@echo -e "\033[0;32mExamples saved to: $(CONFIG_DIR)/\033[0m"

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo -e "\033[1;33mgolangci-lint not installed. Install: https://golangci-lint.run/usage/install/\033[0m"; \
	fi