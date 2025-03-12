#!/bin/bash
set -e

DYNAMO_PORT=8000
TEST_TIMEOUT=60s  # Set test timeout to 60 seconds
TEST_TABLE="ShopAPITest"
FORCE_RECREATE_TABLE=${FORCE_RECREATE_TABLE:-true}  # Default to true for backward compatibility

# Utility functions for printing colored output
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

print_header() {
    echo -e "\033[1;34m$1\033[0m"
}

# Check if a Docker container is running
is_container_running() {
    local container_name=$1
    docker ps --format '{{.Names}}' | grep -q "^$container_name$"
}

# Check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Ensure Docker network exists
ensure_docker_network() {
    if ! docker network ls | grep -q "lemnispace-network"; then
        print_info "Creating Docker network 'lemnispace-network'..."
        docker network create lemnispace-network &>/dev/null
        print_success "Docker network created"
    else
        print_info "Docker network 'lemnispace-network' already exists"
    fi
}

# Check if we're using Docker Compose
if [ -f "docker-compose.yml" ]; then
    USE_COMPOSE=true
    print_info "Found docker-compose.yml, will use Docker Compose for local services"
else
    USE_COMPOSE=false
    print_info "No docker-compose.yml found, will use individual containers"
fi

# Ensure DynamoDB is running
ensure_dynamodb_running() {
    if $USE_COMPOSE; then
        if ! docker-compose ps -q dynamodb-local 2>/dev/null | xargs docker ps -q | grep -q .; then
            print_info "Starting all services with Docker Compose..."
            docker-compose up -d
        else
            print_info "DynamoDB is already running via Docker Compose"
        fi
    else
        if ! is_container_running "dynamodb-local"; then
            print_info "Starting DynamoDB Local..."
            docker run -d --name dynamodb-local -p 8000:8000 amazon/dynamodb-local -jar DynamoDBLocal.jar -sharedDb
            # Wait for DynamoDB to be ready
            print_info "Waiting for DynamoDB to be ready..."
            for i in {1..10}; do
                if curl -s http://localhost:8000 >/dev/null; then
                    print_success "DynamoDB is ready"
                    break
                fi
                if [ $i -eq 10 ]; then
                    print_error "DynamoDB failed to start properly"
                    exit 1
                fi
                sleep 1
            done
        else
            print_info "DynamoDB Local is already running"
        fi
    fi
}

# Ensure S3 is running
ensure_s3_running() {
    if $USE_COMPOSE; then
        # S3 should already be started by Docker Compose in ensure_dynamodb_running
        print_info "S3 should be running via Docker Compose"
        
        # Verify MinIO is actually accessible
        if ! curl -s http://localhost:9000/minio/health/live >/dev/null; then
            print_warning "MinIO does not seem to be accessible, waiting for it to be ready..."
            for i in {1..10}; do
                if curl -s http://localhost:9000/minio/health/live >/dev/null; then
                    print_success "MinIO is ready"
                    break
                fi
                if [ $i -eq 10 ]; then
                    print_warning "MinIO might not be ready but continuing anyway"
                fi
                sleep 1
            done
        else
            print_info "MinIO is accessible"
        fi
    else
        if ! is_container_running "minio-local"; then
            print_info "Starting MinIO (S3) Local..."
            docker run -d --name minio-local \
                -p 9000:9000 -p 9001:9001 \
                -e MINIO_ROOT_USER=minioadmin \
                -e MINIO_ROOT_PASSWORD=minioadmin \
                minio/minio server /data --console-address ":9001"
            # Wait for MinIO to be ready
            print_info "Waiting for MinIO to be ready..."
            for i in {1..10}; do
                if curl -s http://localhost:9000/minio/health/live >/dev/null; then
                    print_success "MinIO is ready"
                    break
                fi
                if [ $i -eq 10 ]; then
                    print_error "MinIO failed to start properly"
                    exit 1
                fi
                sleep 1
            done
            
            # Create required buckets
            print_info "Setting up S3 buckets..."
            docker run --rm --link minio-local:minio \
                minio/mc /bin/sh -c " \
                mc config host add myminio http://minio:9000 minioadmin minioadmin && \
                mc mb --ignore-existing myminio/lemnispace-services && \
                mc mb --ignore-existing myminio/user-product-files && \
                mc policy set download myminio/lemnispace-services && \
                mc policy set download myminio/user-product-files"
        else
            print_info "MinIO (S3) Local is already running"
            # Verify it's actually accessible
            if ! curl -s http://localhost:9000/minio/health/live >/dev/null; then
                print_warning "MinIO is running but not accessible, restarting it..."
                docker restart minio-local
                print_info "Waiting for MinIO to be ready after restart..."
                for i in {1..10}; do
                    if curl -s http://localhost:9000/minio/health/live >/dev/null; then
                        print_success "MinIO is ready"
                        break
                    fi
                    if [ $i -eq 10 ]; then
                        print_error "MinIO failed to start properly after restart"
                        exit 1
                    fi
                    sleep 1
                done
            else
                print_info "MinIO is accessible"
            fi
        fi
    fi
}

# Function to check if S3 buckets exist and create them if necessary
ensure_s3_buckets_exist() {
    echo "[INFO] Verifying required S3 buckets..."
    
    # Check if buckets exist with AWS CLI
    if aws --endpoint-url http://localhost:9000 s3 ls s3://lemnispace-services &>/dev/null && \
       aws --endpoint-url http://localhost:9000 s3 ls s3://user-product-files &>/dev/null; then
        echo "[INFO] S3 buckets already exist"
        return 0
    fi
    
    echo "[INFO] Some buckets don't exist, creating using Docker..."
    
    # Wait for MinIO to be fully ready
    sleep 3
    
    # Use Docker to create buckets (more reliable when using Docker Compose)
    # Configure MinIO client, create the buckets, and set download policies
    if docker run --rm --network lemnispace-network \
        minio/mc /bin/sh -c " \
        mc config host add myminio http://minio-local:9000 minioadmin minioadmin && \
        mc mb -p myminio/lemnispace-services && \
        mc mb -p myminio/user-product-files && \
        mc policy set download myminio/lemnispace-services && \
        mc policy set download myminio/user-product-files" &>/dev/null; then
        
        echo "[SUCCESS] Created S3 buckets successfully"
        return 0
    else
        echo "[WARNING] Docker-based bucket creation failed, trying AWS CLI as fallback..."
        
        # Create lemnispace-services bucket with AWS CLI
        if aws --endpoint-url http://localhost:9000 s3 mb s3://lemnispace-services &>/dev/null; then
            aws --endpoint-url http://localhost:9000 s3api put-bucket-policy \
                --bucket lemnispace-services \
                --policy '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:GetObject","Resource":"arn:aws:s3:::lemnispace-services/*"}]}' &>/dev/null
            echo "[SUCCESS] Created lemnispace-services bucket"
        else
            echo "[ERROR] Failed to create lemnispace-services bucket"
        fi
        
        # Create user-product-files bucket with AWS CLI
        if aws --endpoint-url http://localhost:9000 s3 mb s3://user-product-files &>/dev/null; then
            aws --endpoint-url http://localhost:9000 s3api put-bucket-policy \
                --bucket user-product-files \
                --policy '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:GetObject","Resource":"arn:aws:s3:::user-product-files/*"}]}' &>/dev/null
            echo "[SUCCESS] Created user-product-files bucket"
        else
            echo "[ERROR] Failed to create user-product-files bucket"
        fi
        
        # Despite errors, continue execution as the test might create buckets on demand
        echo "[WARNING] Continuing despite S3 bucket creation issues"
        return 0
    fi
}

# Check if DynamoDB table exists
check_dynamodb_table() {
    local table_name=$1
    if aws dynamodb describe-table --table-name $table_name --endpoint-url http://localhost:8000 --region us-east-1 &>/dev/null; then
        print_info "DynamoDB table '$table_name' exists"
        return 0
    else
        print_info "DynamoDB table '$table_name' does not exist, creating..."
        return 1
    fi
}

# Ensure DynamoDB table exists
ensure_table_exists() {
    DYNAMO_TABLE="ShopAPI_Test"
    
    # Check if table exists
    if ! check_dynamodb_table $DYNAMO_TABLE; then
        # Create table
        aws dynamodb create-table \
            --endpoint-url http://localhost:8000 \
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
            --global-secondary-indexes '[{"IndexName":"GSI1","KeySchema":[{"AttributeName":"GSI1PK","KeyType":"HASH"},{"AttributeName":"GSI1SK","KeyType":"RANGE"}],"Projection":{"ProjectionType":"ALL"},"ProvisionedThroughput":{"ReadCapacityUnits":5,"WriteCapacityUnits":5}},{"IndexName":"GSI2","KeySchema":[{"AttributeName":"GSI2PK","KeyType":"HASH"},{"AttributeName":"GSI2SK","KeyType":"RANGE"}],"Projection":{"ProjectionType":"ALL"},"ProvisionedThroughput":{"ReadCapacityUnits":5,"WriteCapacityUnits":5}},{"IndexName":"GSI3","KeySchema":[{"AttributeName":"GSI3PK","KeyType":"HASH"},{"AttributeName":"GSI3SK","KeyType":"RANGE"}],"Projection":{"ProjectionType":"ALL"},"ProvisionedThroughput":{"ReadCapacityUnits":5,"WriteCapacityUnits":5}}]' \
            --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 \
            --table-class STANDARD >/dev/null
        
        print_success "Created DynamoDB table: $DYNAMO_TABLE"
    fi
}

# Set environment variables for testing
set_test_env_vars() {
    # Set environment variables for testing
    export DYNAMODB_ENDPOINT="http://localhost:8000"
    export DYNAMODB_TABLE="ShopAPI_Test"
    export S3_ENDPOINT="http://localhost:9000"
    export S3_REGION="us-east-1"
    export S3_SERVICES_BUCKET="lemnispace-services"
    export S3_USER_FILES_BUCKET="user-product-files"
    export AWS_ACCESS_KEY_ID="minioadmin"
    export AWS_SECRET_ACCESS_KEY="minioadmin"
    export TEST_MODE="true"
    
    print_success "Environment variables configured for testing"
}

# Run the tests with the provided arguments
run_tests() {
    print_info "Running tests: $*"
    if [ -z "$*" ]; then
        # Run all tests if no specific tests provided
        go test -v ./tests/...
    else
        # Run specified tests
        go test -v $*
    fi
}

# Start services using Docker Compose
start_services_with_docker_compose() {
    print_info "Starting all services with Docker Compose..."
    docker-compose up -d
    
    # Wait for services to be ready
    wait_for_service "dynamodb-local" 8000
    wait_for_service "minio-local" 9000
}

# Start services individually
start_local_services() {
    print_info "Starting individual services..."
    
    # Start DynamoDB
    if ! docker ps | grep -q "dynamodb-local"; then
        print_info "Starting DynamoDB on port 8000..."
        docker run -d --name dynamodb-local -p 8000:8000 amazon/dynamodb-local -jar DynamoDBLocal.jar -sharedDb
        wait_for_service "dynamodb-local" 8000
    else
        print_info "DynamoDB already running"
    fi
    
    # Start MinIO
    if ! docker ps | grep -q "minio-local"; then
        print_info "Starting MinIO on ports 9000 (API) and 9001 (Console)..."
        docker run -d --name minio-local \
            -p 9000:9000 -p 9001:9001 \
            -e MINIO_ROOT_USER=minioadmin \
            -e MINIO_ROOT_PASSWORD=minioadmin \
            minio/minio server /data --console-address ":9001"
        wait_for_service "minio-local" 9000
    else
        print_info "MinIO already running"
    fi
}

# Wait for a service to be ready
wait_for_service() {
    local service_name=$1
    local port=$2
    local max_attempts=30
    local attempt=1
    
    print_info "Waiting for $service_name to be ready..."
    while ! curl -s http://localhost:$port >/dev/null; do
        if [ $attempt -ge $max_attempts ]; then
            print_error "Timed out waiting for $service_name to start"
            return 1
        fi
        sleep 1
        attempt=$((attempt + 1))
    done
    print_success "$service_name is ready"
}

# Check if S3 service is running
check_s3_service() {
    print_info "S3 should be running via Docker Compose"
    
    # Check if MinIO is accessible
    if curl -s http://localhost:9000 >/dev/null; then
        print_info "MinIO is accessible"
    else
        print_warning "MinIO is not accessible, some tests may fail"
    fi
}

# Cleanup resources after tests
cleanup() {
    print_info "Cleaning up resources..."
    
    if [ "$USE_DOCKER_COMPOSE" = true ]; then
        docker-compose down
    else
        # Stop DynamoDB
        if docker ps | grep -q "dynamodb-local"; then
            docker stop dynamodb-local
            docker rm dynamodb-local
        fi
        
        # Stop MinIO
        if docker ps | grep -q "minio-local"; then
            docker stop minio-local
            docker rm minio-local
        fi
    fi
    
    print_success "Cleanup completed"
}

# Main script execution
main() {
    print_header "Setting up test environment..."

    # Check for Docker Compose and set appropriate variables
    if [ -f "docker-compose.yml" ]; then
        print_info "Found docker-compose.yml, will use Docker Compose for local services"
        USE_DOCKER_COMPOSE=true
    else
        USE_DOCKER_COMPOSE=false
    fi

    # Ensure the Docker network exists
    ensure_docker_network
    
    # Start services based on availability
    if [ "$USE_DOCKER_COMPOSE" = true ]; then
        start_services_with_docker_compose
    else
        start_local_services
    fi
    
    # Check for S3 service
    check_s3_service
    
    # Ensure S3 buckets exist
    ensure_s3_buckets_exist || true  # Continue even if this fails
    
    # Ensure DynamoDB table exists
    ensure_table_exists || true  # Continue even if this fails
    
    # Set environment variables for testing
    set_test_env_vars
    
    # Run the tests with the provided arguments
    run_tests "$@"
    
    # Cleanup if needed
    if [ "$CLEANUP_AFTER_TESTS" = true ]; then
        cleanup
    fi
}

# Main execution
main "$@"

print_success "Tests completed successfully" 