#!/bin/bash
set -e

# Print with color functions
print_info() {
    echo -e "\033[1;36m[INFO]\033[0m $1"
}

print_success() {
    echo -e "\033[1;32m[SUCCESS]\033[0m $1"
}

print_error() {
    echo -e "\033[1;31m[ERROR]\033[0m $1" >&2
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

# Ensure DynamoDB local is running
ensure_dynamodb_running() {
    if docker ps | grep -q "dynamodb-local"; then
        print_info "DynamoDB Local is already running"
    else
        print_info "Starting DynamoDB Local..."
        if docker ps -a | grep -q "dynamodb-local"; then
            # Container exists but is not running
            docker start dynamodb-local
        else
            # Container doesn't exist, create it
            docker run -d --name dynamodb-local -p $DYNAMO_PORT:$DYNAMO_PORT amazon/dynamodb-local -jar DynamoDBLocal.jar -sharedDb
        fi
        print_success "DynamoDB Local started on port $DYNAMO_PORT"
    fi
    
    # Wait for DynamoDB to be ready
    print_info "Waiting for DynamoDB to be ready..."
    for i in {1..10}; do
        if curl -s http://localhost:$DYNAMO_PORT >/dev/null; then
            print_success "DynamoDB is ready"
            break
        fi
        if [ $i -eq 10 ]; then
            print_error "DynamoDB failed to start properly"
            exit 1
        fi
        sleep 1
    done
}

# Ensure DynamoDB table exists
ensure_table_exists() {
    print_info "Checking if DynamoDB table '$DYNAMO_TABLE' exists..."
    
    # Check if table exists
    if ! aws dynamodb describe-table --table-name $DYNAMO_TABLE --endpoint-url http://localhost:$DYNAMO_PORT >/dev/null 2>&1; then
        print_info "Table '$DYNAMO_TABLE' does not exist, creating..."
        
        # Create table
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
                AttributeName=EntityType,AttributeType=S \
            --key-schema \
                AttributeName=PK,KeyType=HASH \
                AttributeName=SK,KeyType=RANGE \
            --global-secondary-indexes \
                "[
                    {
                        \"IndexName\": \"GSI1\",
                        \"KeySchema\": [
                            {\"AttributeName\": \"GSI1PK\", \"KeyType\": \"HASH\"},
                            {\"AttributeName\": \"GSI1SK\", \"KeyType\": \"RANGE\"}
                        ],
                        \"Projection\": {\"ProjectionType\": \"ALL\"},
                        \"ProvisionedThroughput\": {\"ReadCapacityUnits\": 5, \"WriteCapacityUnits\": 5}
                    },
                    {
                        \"IndexName\": \"GSI2\",
                        \"KeySchema\": [
                            {\"AttributeName\": \"GSI2PK\", \"KeyType\": \"HASH\"},
                            {\"AttributeName\": \"GSI2SK\", \"KeyType\": \"RANGE\"}
                        ],
                        \"Projection\": {\"ProjectionType\": \"ALL\"},
                        \"ProvisionedThroughput\": {\"ReadCapacityUnits\": 5, \"WriteCapacityUnits\": 5}
                    },
                    {
                        \"IndexName\": \"EntityTypeIndex\",
                        \"KeySchema\": [
                            {\"AttributeName\": \"EntityType\", \"KeyType\": \"HASH\"}
                        ],
                        \"Projection\": {\"ProjectionType\": \"ALL\"},
                        \"ProvisionedThroughput\": {\"ReadCapacityUnits\": 5, \"WriteCapacityUnits\": 5}
                    }
                ]" \
            --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 \
            --table-class STANDARD >/dev/null
        
        print_success "Created DynamoDB table: $DYNAMO_TABLE"
    else
        print_info "Table '$DYNAMO_TABLE' already exists"
    fi
}

# Set up AWS local credentials
setup_aws_local() {
    if ! aws configure list --profile local >/dev/null 2>&1; then
        print_info "Setting up AWS local profile"
        aws configure set aws_access_key_id test --profile local
        aws configure set aws_secret_access_key test --profile local
        aws configure set region us-east-1 --profile local
        print_success "AWS local profile configured"
    fi
}

# Build and run the application
build_and_run() {
    print_info "Building application..."
    cd /workspaces/shop-api
    
    # Build using go build to catch any compilation errors
    if ! go build -o /tmp/shop-api-test ./cmd/shop >/dev/null 2>&1; then
        print_error "Build failed, check your code"
        go build -o /tmp/shop-api-test ./cmd/shop
        exit 1
    fi
    
    # Remove the test binary
    rm -f /tmp/shop-api-test
    
    print_success "Build succeeded"
    print_info "Starting API server on port $PORT with local DynamoDB..."
    
    # Run the application with environment variables
    export AWS_PROFILE=local
    export AWS_ENDPOINT_URL=http://localhost:$DYNAMO_PORT
    export DYNAMODB_TABLE=$DYNAMO_TABLE
    export DEBUG=$DEBUG
    export PORT=$PORT
    export RUN_LOCAL=$RUN_LOCAL
    export LOG_LEVEL=${LOG_LEVEL:-debug}
    
    go run cmd/shop/main.go
}

# Main execution
print_info "Starting development environment setup with DynamoDB..."

# Check for required tools
for cmd in aws docker go lsof curl; do
    if ! command_exists $cmd; then
        print_error "$cmd is required but not installed"
        exit 1
    fi
done

# Kill existing processes
kill_process_on_port $PORT

# Ensure DynamoDB is running
ensure_dynamodb_running

# Set up AWS local profile
setup_aws_local

# Ensure table exists
ensure_table_exists

# Build and run the application
build_and_run 