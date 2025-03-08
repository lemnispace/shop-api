#!/bin/bash
set -e

# Build the API
echo "Building Shop API..."
make build-shop

# Run the tests
echo "Running tests..."
make test

# Run with race detection
echo "Running tests with race detection..."
make test-verbose

# Generate coverage report
echo "Generating coverage report..."
make test-coverage

# If coverage.html exists, print its location
if [ -f "coverage.html" ]; then
  echo "Coverage report generated at: $(pwd)/coverage.html"
fi

echo "Tests completed successfully!" 