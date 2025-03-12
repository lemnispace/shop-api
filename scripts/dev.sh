#!/bin/bash
set -e

# Print with color functions
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

# Source environment variables if .env.local exists
ENV_FILE=".env.local"
if [ -f "$ENV_FILE" ]; then
    print_info "Loading environment variables from $ENV_FILE"
    export $(grep -v '^#' "$ENV_FILE" | xargs)
else
    print_error "$ENV_FILE not found. Please create it from .env.example"
    exit 1
fi

# Default values if not set in .env.local
PORT=${PORT:-8080}
DYNAMO_PORT=${DYNAMO_PORT:-8000}
DYNAMO_TABLE=${DYNAMODB_TABLE:-"ShopAPI"}
S3_ENDPOINT=${S3_ENDPOINT:-"http://localhost:9000"}
S3_REGION=${S3_REGION:-"us-east-1"}
S3_ACCESS_KEY=${S3_ACCESS_KEY:-"minioadmin"}
S3_SECRET_KEY=${S3_SECRET_KEY:-"minioadmin"}
S3_SERVICES_BUCKET=${S3_SERVICES_BUCKET:-"lemnispace-services"}
S3_USER_FILES_BUCKET=${S3_USER_FILES_BUCKET:-"user-product-files"}
API_PATH="cmd/shop"
DEBUG=${DEBUG:-true}
RUN_LOCAL=${RUN_LOCAL:-true}

# Check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Kill processes using specified port
kill_process_on_port() {
    local port=$1
    local pid=$(lsof -ti :$port 2>/dev/null)
    
    if [ -n "$pid" ]; then
        print_info "Found process(es) using port $port: $pid - killing..."
        kill -9 $pid 2>/dev/null || true
        print_success "Stopped process(es) on port $port"
    else
        print_info "No process found using port $port"
    fi
}

# Check if a Docker container is running
is_container_running() {
    local container_name=$1
    docker ps --format '{{.Names}}' | grep -q "^$container_name$"
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

# Check if we're using Docker Compose or individual containers
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
            print_info "Starting DynamoDB local..."
            make dynamo-local
        else
            print_info "DynamoDB local is already running"
        fi
    fi
}

# Ensure S3 is running
ensure_s3_running() {
    if $USE_COMPOSE; then
        # S3 should already be started by Docker Compose in ensure_dynamodb_running
        print_info "S3 should be running via Docker Compose"
    else
        if ! is_container_running "minio-local"; then
            print_info "Starting MinIO (S3) local..."
            make s3-local
            # Wait for MinIO to start
            sleep 5
            print_info "Creating S3 buckets..."
            make s3-init
        else
            print_info "MinIO (S3) local is already running"
        fi
    fi
}

# Check if DynamoDB table exists
check_dynamodb_table() {
    if aws dynamodb describe-table --table-name "$DYNAMO_TABLE" --endpoint-url http://localhost:$DYNAMO_PORT --region us-east-1 &>/dev/null; then
        print_info "DynamoDB table '$DYNAMO_TABLE' exists"
        return 0
    else
        print_info "DynamoDB table '$DYNAMO_TABLE' does not exist, creating..."
        return 1
    fi
}

# Check if S3 buckets exist
check_s3_buckets() {
    # Use aws CLI to check if buckets exist
    if aws --endpoint-url $S3_ENDPOINT s3 ls s3://$S3_SERVICES_BUCKET &>/dev/null && \
       aws --endpoint-url $S3_ENDPOINT s3 ls s3://$S3_USER_FILES_BUCKET &>/dev/null; then
        print_info "S3 buckets exist"
        return 0
    else
        print_info "Some S3 buckets do not exist, will create them"
        return 1
    fi
}

# Create required S3 buckets
create_s3_buckets() {
    print_info "Creating required S3 buckets using Docker..."
    # Use Docker directly (more reliable)
    if docker run --rm --network lemnispace-network \
        minio/mc /bin/sh -c " \
        mc config host add myminio http://minio-local:9000 minioadmin minioadmin && \
        mc mb --ignore-existing myminio/lemnispace-services && \
        mc mb --ignore-existing myminio/user-product-files && \
        mc policy set download myminio/lemnispace-services && \
        mc policy set download myminio/user-product-files" &>/dev/null; then
        print_success "S3 buckets created successfully"
    else
        print_warning "Docker-based bucket creation failed, trying AWS CLI..."
        # Fallback to AWS CLI
        if aws --endpoint-url $S3_ENDPOINT s3 mb s3://$S3_SERVICES_BUCKET &>/dev/null; then
            aws --endpoint-url $S3_ENDPOINT s3api put-bucket-policy \
                --bucket $S3_SERVICES_BUCKET \
                --policy '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:GetObject","Resource":"arn:aws:s3:::'"$S3_SERVICES_BUCKET"'/*"}]}' &>/dev/null
            print_success "Created $S3_SERVICES_BUCKET bucket"
        else
            print_warning "Failed to create $S3_SERVICES_BUCKET bucket"
        fi
        
        if aws --endpoint-url $S3_ENDPOINT s3 mb s3://$S3_USER_FILES_BUCKET &>/dev/null; then
            aws --endpoint-url $S3_ENDPOINT s3api put-bucket-policy \
                --bucket $S3_USER_FILES_BUCKET \
                --policy '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:GetObject","Resource":"arn:aws:s3:::'"$S3_USER_FILES_BUCKET"'/*"}]}' &>/dev/null
            print_success "Created $S3_USER_FILES_BUCKET bucket"
        else
            print_warning "Failed to create $S3_USER_FILES_BUCKET bucket"
        fi
    fi
}

# Main script execution
print_info "Starting development environment setup..."

# Kill any existing processes on the API port
print_info "Checking for processes using port $PORT..."
kill_process_on_port $PORT

# Ensure Docker network exists
ensure_docker_network

# Ensure AWS CLI is configured for local development
print_info "Setting up AWS CLI for local development..."
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1

# Start required local services
ensure_dynamodb_running
ensure_s3_running

# Check and create DynamoDB table if needed
if ! check_dynamodb_table; then
    make dynamo-init
fi

# Check and create S3 buckets if needed
if ! check_s3_buckets; then
    create_s3_buckets
fi

# Set environment variables for the application
export DYNAMODB_TABLE=$DYNAMO_TABLE
export DYNAMODB_ENDPOINT=http://localhost:$DYNAMO_PORT
export S3_ENDPOINT=$S3_ENDPOINT
export S3_REGION=$S3_REGION
export S3_ACCESS_KEY=$S3_ACCESS_KEY
export S3_SECRET_KEY=$S3_SECRET_KEY
export S3_SERVICES_BUCKET=$S3_SERVICES_BUCKET
export S3_USER_FILES_BUCKET=$S3_USER_FILES_BUCKET
export PORT=$PORT
export DEBUG=$DEBUG

# Build and run the application
if $RUN_LOCAL; then
    print_success "Environment configured. Starting API on port $PORT..."
    if [ "$DEBUG" = "true" ]; then
        go run $API_PATH -port=$PORT
    else
        go run $API_PATH -port=$PORT 2>&1 | grep -v DEBUG
    fi
else
    # Configure for AWS
    print_info "Environment configured for AWS."
    print_info "You can now run 'go run $API_PATH' with your AWS credentials"
fi 