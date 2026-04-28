.PHONY: all generate test build

# Default target
all: test build

# --- CODE GENERATION ---
# The main generate target now has dependencies, ensuring correct order.
generate: generate-common generate-authorizer generate-server

# NEW: Generate the common types first into their own package.
generate-common:
	@echo "Generating common API types..."
	@oapi-codegen -generate "types" \
		-package commonapi \
		-o internal/transport/http/commonapi/generated.go \
		api/common.yaml

# UPDATED: Generate authorizer, mapping common.yaml to the Go package we just created.
# Replace 'streambear' with your actual go module name from go.mod
generate-authorizer:
	@echo "Generating authorizer API..."
	@oapi-codegen -generate "types,chi-server" \
		-package authorizerapi \
		-o internal/transport/http/authorizerapi/generated.go \
		--import-mapping=common.yaml:streambear/internal/transport/http/commonapi \
		api/authorizer.yaml

# UPDATED: Do the same for the server.
generate-server:
	@echo "Generating server API..."
	@oapi-codegen -generate "types,chi-server" \
		-package serverapi \
		-o internal/transport/http/serverapi/generated.go \
		--import-mapping=common.yaml:streambear/internal/transport/http/commonapi \
		api/server.yaml


# --- OTHER TARGETS ---
build:
	@echo "Building streambear binary..."
	@go build -o ./bin/streambear ./cmd/streambear

test:
	@echo "Running tests..."
	@go test -v ./...