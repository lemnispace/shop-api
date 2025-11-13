.PHONY: build-% run deploy test dev-up dev-down dev-logs localstack-up localstack-down localstack-terraform-init localstack-terraform-apply localstack-terraform-destroy localstack-test localstack-setup

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
	@echo "  - LocalStack: http://localhost:4566"

# Stop all local development services
dev-down:
	@echo "Stopping development services..."
	docker compose down

# View logs from development services
dev-logs:
	docker compose logs -f

# LocalStack: Start LocalStack service
localstack-up:
	@echo "Starting LocalStack..."
	docker compose up -d localstack
	@echo "LocalStack is running at http://localhost:4566"
	@echo "Waiting for LocalStack to be ready..."
	@sleep 5
	@curl -s http://localhost:4566/_localstack/health | grep -q '"dynamodb": "available"' && echo "LocalStack is ready!" || echo "LocalStack may not be fully ready yet"

# LocalStack: Stop LocalStack service
localstack-down:
	@echo "Stopping LocalStack..."
	docker compose stop localstack

# LocalStack: Apply Terraform configuration
localstack-terraform-init:
	@echo "Initializing Terraform for LocalStack..."
	cd terraform/environments/local && terraform init

localstack-terraform-apply:
	@echo "Applying Terraform configuration to LocalStack..."
	@echo "Make sure LocalStack is running (make localstack-up)"
	cd terraform/environments/local && \
	AWS_ACCESS_KEY_ID=test \
	AWS_SECRET_ACCESS_KEY=test \
	AWS_REGION=us-east-1 \
	terraform apply -auto-approve

localstack-terraform-destroy:
	@echo "Destroying Terraform resources in LocalStack..."
	cd terraform/environments/local && \
	AWS_ACCESS_KEY_ID=test \
	AWS_SECRET_ACCESS_KEY=test \
	AWS_REGION=us-east-1 \
	terraform destroy -auto-approve

# LocalStack: Test infrastructure
localstack-test:
	@echo "Testing LocalStack infrastructure..."
	@echo "1. Checking DynamoDB table..."
	aws dynamodb describe-table \
		--table-name ShopAPI \
		--endpoint-url http://localhost:4566 \
		--region us-east-1 \
		--no-cli-pager || echo "Table not found"
	@echo ""
	@echo "2. Listing S3 buckets..."
	aws s3 ls \
		--endpoint-url http://localhost:4566 \
		--region us-east-1 || echo "No buckets found"

# LocalStack: Full setup (start LocalStack + apply Terraform)
localstack-setup: localstack-up localstack-terraform-init localstack-terraform-apply
	@echo "LocalStack setup complete!"
	@echo "You can now run the API with: make run"

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
