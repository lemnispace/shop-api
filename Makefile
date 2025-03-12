.PHONY: build-% test test-coverage clean dynamo-local dynamo-stop dynamo-init s3-local s3-stop s3-init docker-local docker-stop run test-unit test-api test-pattern test-race deploy

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

# Run API with production DynamoDB (requires AWS credentials)
run-prod:
	@echo "Starting API server with AWS DynamoDB..."
	go run ./cmd/shop

# Run all tests
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

# Start local services using Docker Compose
docker-local:
	@echo "Starting local DynamoDB and S3 services..."
	docker-compose up -d
	@echo "Services are now running. DynamoDB on port 8000, MinIO on ports 9000 (API) and 9001 (Console)"

# Stop local services
docker-stop:
	@echo "Stopping local services..."
	docker-compose down

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
	@aws dynamodb create-table \
		--endpoint-url http://localhost:8000 \
		--table-name ShopAPI \
		--attribute-definitions \
			AttributeName=PK,AttributeType=S \
			AttributeName=SK,AttributeType=S \
			AttributeName=GSI1PK,AttributeType=S \
			AttributeName=GSI1SK,AttributeType=S \
			AttributeName=GSI2PK,AttributeType=S \
			AttributeName=GSI2SK,AttributeType=S \
			AttributeName=GSI3PK,AttributeType=S \
			AttributeName=GSI3SK,AttributeType=S \
		--key-schema \
			AttributeName=PK,KeyType=HASH \
			AttributeName=SK,KeyType=RANGE \
		--global-secondary-indexes '[{"IndexName":"GSI1","KeySchema":[{"AttributeName":"GSI1PK","KeyType":"HASH"},{"AttributeName":"GSI1SK","KeyType":"RANGE"}],"Projection":{"ProjectionType":"ALL"},"ProvisionedThroughput":{"ReadCapacityUnits":5,"WriteCapacityUnits":5}},{"IndexName":"GSI2","KeySchema":[{"AttributeName":"GSI2PK","KeyType":"HASH"},{"AttributeName":"GSI2SK","KeyType":"RANGE"}],"Projection":{"ProjectionType":"ALL"},"ProvisionedThroughput":{"ReadCapacityUnits":5,"WriteCapacityUnits":5}},{"IndexName":"GSI3","KeySchema":[{"AttributeName":"GSI3PK","KeyType":"HASH"},{"AttributeName":"GSI3SK","KeyType":"RANGE"}],"Projection":{"ProjectionType":"ALL"},"ProvisionedThroughput":{"ReadCapacityUnits":5,"WriteCapacityUnits":5}}]' \
		--provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 \
		--table-class STANDARD > /dev/null
	@echo "DynamoDB table 'ShopAPI' created successfully"

# Start local S3 (MinIO) for development
s3-local:
	@echo "Starting local S3 (MinIO) on ports 9000 (API) and 9001 (Console)..."
	docker run -d --name minio-local \
		-p 9000:9000 -p 9001:9001 \
		-e MINIO_ROOT_USER=minioadmin \
		-e MINIO_ROOT_PASSWORD=minioadmin \
		minio/minio server /data --console-address ":9001"

# Stop local S3 (MinIO)
s3-stop:
	@echo "Stopping local S3 (MinIO)..."
	docker stop minio-local
	docker rm minio-local

# Create required S3 buckets for local development
s3-init:
	@echo "Creating required S3 buckets..."
	docker run --rm --network lemnispace-network \
		minio/mc /bin/sh -c " \
		mc config host add myminio http://minio-local:9000 minioadmin minioadmin && \
		mc mb -p myminio/lemnispace-services && \
		mc mb -p myminio/user-product-files && \
		mc policy set download myminio/lemnispace-services && \
		mc policy set download myminio/user-product-files" > /dev/null 2>&1 || { \
		echo "Warning: Docker-based bucket creation failed, trying AWS CLI..."; \
		aws --endpoint-url http://localhost:9000 s3 mb s3://lemnispace-services > /dev/null 2>&1 && \
		aws --endpoint-url http://localhost:9000 s3api put-bucket-policy \
			--bucket lemnispace-services \
			--policy '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:GetObject","Resource":"arn:aws:s3:::lemnispace-services/*"}]}' > /dev/null 2>&1; \
		aws --endpoint-url http://localhost:9000 s3 mb s3://user-product-files > /dev/null 2>&1 && \
		aws --endpoint-url http://localhost:9000 s3api put-bucket-policy \
			--bucket user-product-files \
			--policy '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:GetObject","Resource":"arn:aws:s3:::user-product-files/*"}]}' > /dev/null 2>&1; \
	}
	@echo "S3 buckets created and configured"

# Deploy API to AWS (requires proper AWS credentials)
deploy:
	@echo "Deploying to AWS..."
	terraform -chdir=./terraform apply