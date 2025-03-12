#!/bin/bash
set -e

# Constants
DYNAMO_PORT=8000
DYNAMO_TABLE="ShopAPI_Test"
S3_PORT=9000
S3_CONSOLE_PORT=9001
NETWORK_NAME="lemnispace-network"

# Buckets that must exist for tests
S3_SERVICES_BUCKET="lemnispace-services"
S3_USER_FILES_BUCKET="user-product-files"

# Utility functions
print_info() {
    echo -e "\033[0;34m[INFO] $1\033[0m"
}

print_success() {
    echo -e "\033[0;32m[SUCCESS] $1\033[0m"
}

print_warning() {
    echo -e "\033[0;33m[WARNING] $1\033[0m"
}

print_error() {
    echo -e "\033[0;31m[ERROR] $1\033[0m"
}

# Check if we should use Docker Compose
use_docker_compose() {
    if [ -f "docker-compose.yml" ]; then
        return 0  # true
    else
        return 1  # false
    fi
}

# Ensure Docker network exists
ensure_network() {
    if ! docker network ls | grep -q "$NETWORK_NAME"; then
        print_info "Creating Docker network '$NETWORK_NAME'..."
        docker network create $NETWORK_NAME &>/dev/null || true
        print_success "Docker network created"
    fi
}

# Start services using Docker Compose
start_compose_services() {
    print_info "Starting services with Docker Compose..."
    docker-compose up -d
    
    # Verify services are running
    wait_for_service "DynamoDB" $DYNAMO_PORT
    wait_for_service "MinIO" $S3_PORT
}

# Start individual services
start_individual_services() {
    # Start DynamoDB if not running
    if ! docker ps --format '{{.Names}}' | grep -q "dynamodb-local"; then
        print_info "Starting DynamoDB Local..."
        docker run -d --name dynamodb-local \
            --network $NETWORK_NAME \
            -p $DYNAMO_PORT:$DYNAMO_PORT \
            amazon/dynamodb-local -jar DynamoDBLocal.jar -sharedDb
    fi
    
    # Start MinIO if not running
    if ! docker ps --format '{{.Names}}' | grep -q "minio-local"; then
        print_info "Starting MinIO (S3) Local..."
        docker run -d --name minio-local \
            --network $NETWORK_NAME \
            -p $S3_PORT:$S3_PORT -p $S3_CONSOLE_PORT:$S3_CONSOLE_PORT \
            -e MINIO_ROOT_USER=minioadmin \
            -e MINIO_ROOT_PASSWORD=minioadmin \
            minio/minio server /data --console-address ":$S3_CONSOLE_PORT"
    fi
    
    # Wait for services to be ready
    wait_for_service "DynamoDB" $DYNAMO_PORT
    wait_for_service "MinIO" $S3_PORT
}

# Wait for a service to be ready
wait_for_service() {
    local service_name=$1
    local port=$2
    local max_attempts=10
    local attempt=1
    
    print_info "Waiting for $service_name to be ready..."
    while ! curl -s http://localhost:$port >/dev/null; do
        if [ $attempt -ge $max_attempts ]; then
            print_warning "$service_name not responding after $max_attempts attempts, continuing anyway..."
            return
        fi
        sleep 1
        attempt=$((attempt + 1))
    done
    print_success "$service_name is ready"
}

# Create DynamoDB test table
create_dynamo_table() {
    print_info "Setting up DynamoDB test table..."
    
    # Check if table exists
    if aws dynamodb describe-table --table-name $DYNAMO_TABLE --endpoint-url http://localhost:$DYNAMO_PORT --region us-east-1 &>/dev/null; then
        print_info "DynamoDB table '$DYNAMO_TABLE' already exists"
    else
        print_info "Creating DynamoDB table '$DYNAMO_TABLE'..."
        aws dynamodb create-table \
            --endpoint-url http://localhost:$DYNAMO_PORT \
            --table-name $DYNAMO_TABLE \
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
            --global-secondary-indexes '[
                {
                    "IndexName": "GSI1",
                    "KeySchema": [
                        {"AttributeName": "GSI1PK", "KeyType": "HASH"},
                        {"AttributeName": "GSI1SK", "KeyType": "RANGE"}
                    ],
                    "Projection": {"ProjectionType": "ALL"},
                    "ProvisionedThroughput": {"ReadCapacityUnits": 5, "WriteCapacityUnits": 5}
                },
                {
                    "IndexName": "GSI2",
                    "KeySchema": [
                        {"AttributeName": "GSI2PK", "KeyType": "HASH"},
                        {"AttributeName": "GSI2SK", "KeyType": "RANGE"}
                    ],
                    "Projection": {"ProjectionType": "ALL"},
                    "ProvisionedThroughput": {"ReadCapacityUnits": 5, "WriteCapacityUnits": 5}
                },
                {
                    "IndexName": "GSI3",
                    "KeySchema": [
                        {"AttributeName": "GSI3PK", "KeyType": "HASH"},
                        {"AttributeName": "GSI3SK", "KeyType": "RANGE"}
                    ],
                    "Projection": {"ProjectionType": "ALL"},
                    "ProvisionedThroughput": {"ReadCapacityUnits": 5, "WriteCapacityUnits": 5}
                }
            ]' \
            --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 >/dev/null && \
        print_success "DynamoDB table created" || print_error "Failed to create DynamoDB table"
    fi
}

# Create S3 buckets
create_s3_buckets() {
    print_info "Setting up S3 test buckets..."
    
    # Check if buckets exist using AWS CLI
    if ! aws --endpoint-url http://localhost:$S3_PORT s3 ls s3://$S3_SERVICES_BUCKET &>/dev/null; then
        print_info "Creating bucket: $S3_SERVICES_BUCKET"
        aws --endpoint-url http://localhost:$S3_PORT s3 mb s3://$S3_SERVICES_BUCKET &>/dev/null || true
        aws --endpoint-url http://localhost:$S3_PORT s3api put-bucket-policy \
            --bucket $S3_SERVICES_BUCKET \
            --policy '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:GetObject","Resource":"arn:aws:s3:::'$S3_SERVICES_BUCKET'/*"}]}' &>/dev/null || true
    fi
    
    if ! aws --endpoint-url http://localhost:$S3_PORT s3 ls s3://$S3_USER_FILES_BUCKET &>/dev/null; then
        print_info "Creating bucket: $S3_USER_FILES_BUCKET"
        aws --endpoint-url http://localhost:$S3_PORT s3 mb s3://$S3_USER_FILES_BUCKET &>/dev/null || true
        aws --endpoint-url http://localhost:$S3_PORT s3api put-bucket-policy \
            --bucket $S3_USER_FILES_BUCKET \
            --policy '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:GetObject","Resource":"arn:aws:s3:::'$S3_USER_FILES_BUCKET'/*"}]}' &>/dev/null || true
    fi
}

# Set environment variables for testing
set_test_env_vars() {
    export DYNAMODB_ENDPOINT="http://localhost:$DYNAMO_PORT"
    export DYNAMODB_TABLE=$DYNAMO_TABLE
    export S3_ENDPOINT="http://localhost:$S3_PORT"
    export S3_REGION="us-east-1"
    export S3_SERVICES_BUCKET=$S3_SERVICES_BUCKET
    export S3_USER_FILES_BUCKET=$S3_USER_FILES_BUCKET
    export AWS_ACCESS_KEY_ID="minioadmin"
    export AWS_SECRET_ACCESS_KEY="minioadmin"
    export TEST_MODE="true"
    
    print_success "Environment variables configured for testing"
}

# Run tests
run_tests() {
    print_info "Running tests: $*"
    
    if [ -z "$*" ]; then
        # Run all tests if no specific paths provided
        go test -v ./tests/...
    else
        # Run specified tests
        go test -v $*
    fi
}

# Main execution
main() {
    print_info "Setting up test environment..."
    
    # Make sure AWS CLI is available
    if ! command -v aws &>/dev/null; then
        print_error "AWS CLI not found. Please install it to run tests."
        exit 1
    fi
    
    # Create Docker network
    ensure_network
    
    # Start services
    if use_docker_compose; then
        start_compose_services
    else
        start_individual_services
    fi
    
    # Set up infrastructure
    create_dynamo_table
    create_s3_buckets
    
    # Set environment variables
    set_test_env_vars
    
    # Run the tests
    run_tests "$@"
    
    print_success "Tests completed successfully"
}

# Execute main function with all arguments
main "$@" 