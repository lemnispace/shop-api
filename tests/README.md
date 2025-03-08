# Shop API Tests

This directory contains tests for the Shop API. The tests are organized to be separate from the production code so they are not bundled and shipped to AWS.

## Test Structure

- `api/` - API integration tests for testing the full HTTP endpoints
- `unit/` - Unit tests for testing individual components in isolation
- `integration/` - Integration tests for testing interactions between components

## Running Tests

You can run the tests using the following Make commands:

```bash
# Run all tests
make test

# Run tests with race detection
make test-verbose

# Run tests and generate coverage report
make test-coverage
```

Or you can use the provided script:

```bash
./scripts/run-tests.sh
```

## Test Implementation Strategy

1. **API Tests**: These test the full HTTP request/response cycle by creating a test server with the actual handlers and routes. They verify that the API endpoints work as expected.

2. **Unit Tests**: These test individual components (like services) in isolation. They verify the business logic works correctly.

3. **Integration Tests**: These test the interactions between components, such as how the handlers use the services.

## GitHub Actions Integration

The tests are automatically run in GitHub Actions CI workflow whenever changes are pushed to the main branches. The workflow is defined in `.github/workflows/ci.yml`.
