.PHONY: build-% run deploy test dev-up dev-down dev-logs

build-%:
	@echo "Building $*..."
	GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o ./build/$*/bootstrap ./cmd/$*

# Build all components
build: build-shop
	@echo "Build complete"

# Start all local development services (DynamoDB, MinIO, API)
dev-up:
	@echo "Starting all development services..."
	docker compose up -d
	@echo "Development services are running!"
	@echo "  - API: http://localhost:8080"
	@echo "  - DynamoDB: http://localhost:8000"
	@echo "  - MinIO: http://localhost:9000 (Console: http://localhost:9001)"

# Stop all local development services
dev-down:
	@echo "Stopping development services..."
	docker compose down

# View logs from development services
dev-logs:
	docker compose logs -f

# Run API with local DynamoDB (use this inside devcontainer)
run:
	@echo "Starting API server with local DynamoDB..."
	@echo "Make sure DynamoDB and MinIO are running (make dev-up)"
	go run ./cmd/shop

# Deploy API to AWS (requires proper AWS credentials)
deploy:
	@echo "Deploying to AWS..."
	terraform -chdir=./terraform apply

# Run integration tests (use this inside devcontainer or locally with Go installed)
test:
	@echo "Starting local DynamoDB and MinIO for tests..."
	docker compose up -d dynamodb-local minio createbuckets create-dynamodb-table
	@echo "Waiting for services to be ready..."
	@sleep 8
	@echo "Running integration tests..."
	DYNAMODB_ENDPOINT=http://localhost:8000 \
	S3_ENDPOINT=http://localhost:9000 \
	S3_USE_PATH_STYLE=true \
	AWS_ACCESS_KEY_ID=minioadmin \
	AWS_SECRET_ACCESS_KEY=minioadmin \
	AWS_REGION=us-east-1 \
	go test -v ./... -count=1
	@echo "Tests complete."

# Clean up test services
test-clean:
	@echo "Stopping test services..."
	docker compose down
	@echo "Test services stopped."
