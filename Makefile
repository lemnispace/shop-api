.PHONY: build-% run deploy test

build-%:
	@echo "Building $*..."
	GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o ./build/$*/bootstrap ./cmd/$*

# Build all components
build: build-shop
	@echo "Build complete"

# Run API with local DynamoDB
run:
	@echo "Starting API server with local DynamoDB..."
	./scripts/dev.sh

# Deploy API to AWS (requires proper AWS credentials)
deploy:
	@echo "Deploying to AWS..."
	terraform -chdir=./terraform apply

# Run integration tests using local infrastructure
test:
	@echo "Starting local DynamoDB and MinIO for tests..."
	docker-compose up -d dynamodb-local minio
	@echo "Waiting for services to be ready..."
	@sleep 5 # Simple wait, can be replaced with health checks
	@echo "Running integration tests..."
	DYNAMODB_ENDPOINT=http://localhost:8000 \
	S3_ENDPOINT=http://localhost:9000 \
	S3_USE_PATH_STYLE=true \
	AWS_ACCESS_KEY_ID=minioadmin \
	AWS_SECRET_ACCESS_KEY=minioadmin \
	AWS_REGION=us-east-1 \
	go test -v ./... -count=1
	@echo "Stopping local services..."
	docker-compose down
	@echo "Tests complete."
