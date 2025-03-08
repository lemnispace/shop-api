.PHONY: build-% test test-all test-coverage clean

build-%:
	@echo "Building $*..."
	GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o ./build/$*/bootstrap ./cmd/$*

# Run all tests (excluding vendor directory)
test:
	@echo "Running all tests..."
	go test -v ./tests/...

# Run tests with more verbose output and race detection
test-verbose:
	@echo "Running tests with race detection..."
	go test -v -race ./tests/...

# Run tests and generate coverage report
test-coverage:
	@echo "Running tests with coverage report..."
	go test -v -coverprofile=coverage.out ./tests/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf ./build/
	rm -f coverage.out coverage.html