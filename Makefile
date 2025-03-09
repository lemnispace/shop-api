.PHONY: build-% test test-coverage clean dynamo-local dynamo-stop dynamo-init run dev test-unit test-api test-pattern test-race deploy

build-%:
	@echo "Building $*..."
	GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o ./build/$*/bootstrap ./cmd/$*

# Build all components
build: build-shop
	@echo "Build complete"

# Run API with local DynamoDB
run:
	@echo "Starting API server with local DynamoDB..."
	AWS_PROFILE=local AWS_ENDPOINT_URL=http://localhost:8000 go run ./cmd/shop

# Run API with production DynamoDB (requires AWS credentials)
run-prod:
	@echo "Starting API server with AWS DynamoDB..."
	go run ./cmd/shop

# Run with the automated dev script (recommended for local development)
dev:
	@echo "Starting full development environment with local DynamoDB..."
	./scripts/dev.sh

# Run all tests (excluding vendor directory)
test:
	@echo "Running all tests with local DynamoDB..."
	./scripts/run-tests.sh

# Run unit tests only
test-unit:
	@echo "Running unit tests with local DynamoDB..."
	./scripts/run-tests.sh ./tests/unit/...

# Run API tests only
test-api:
	@echo "Running API tests with local DynamoDB..."
	./scripts/run-tests.sh ./tests/api/...

# Run tests for a specific test pattern (usage: make test-pattern PATTERN=TestCollectionProducts)
test-pattern:
	@if [ -z "$(PATTERN)" ]; then \
		echo "Error: PATTERN parameter is required. Usage: make test-pattern PATTERN=TestCollectionProducts"; \
		exit 1; \
	fi
	@echo "Running tests matching pattern '$(PATTERN)' with local DynamoDB..."
	./scripts/run-tests.sh ./tests/... -run $(PATTERN)

# Run tests with race detection
test-race:
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

# Start local DynamoDB for development
dynamo-local:
	@echo "Starting local DynamoDB on port 8000..."
	docker run -d --name dynamodb-local -p 8000:8000 amazon/dynamodb-local -jar DynamoDBLocal.jar -sharedDb

# Stop local DynamoDB
dynamo-stop:
	@echo "Stopping local DynamoDB..."
	docker stop dynamodb-local
	docker rm dynamodb-local

# Create required DynamoDB tables for local development
dynamo-init:
	@echo "Creating required DynamoDB tables..."
	aws dynamodb create-table \
		--endpoint-url http://localhost:8000 \
		--table-name ShopAPI \
		--attribute-definitions \
			AttributeName=PK,AttributeType=S \
			AttributeName=SK,AttributeType=S \
		--key-schema \
			AttributeName=PK,KeyType=HASH \
			AttributeName=SK,KeyType=RANGE \
		--provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 \
		--table-class STANDARD

# Deploy API to AWS (requires proper AWS credentials)
deploy:
	@echo "Deploying to AWS..."
	terraform -chdir=./terraform apply