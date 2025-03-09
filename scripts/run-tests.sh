#!/bin/bash
set -e

DYNAMO_PORT=8000
TEST_TIMEOUT=60s  # Set test timeout to 60 seconds
TEST_TABLE="ShopAPITest"
FORCE_RECREATE_TABLE=${FORCE_RECREATE_TABLE:-true}  # Default to true for backward compatibility

# Print with color
print_info() {
    echo -e "\033[1;36m[INFO]\033[0m $1"
}

print_success() {
    echo -e "\033[1;32m[SUCCESS]\033[0m $1"
}

print_error() {
    echo -e "\033[1;31m[ERROR]\033[0m $1" >&2
}

print_warning() {
    echo -e "\033[1;33m[WARNING]\033[0m $1" >&2
}

# Ensure DynamoDB local is running
ensure_dynamodb_running() {
    print_info "Checking if DynamoDB Local is running..."
    
    local max_attempts=3
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if docker ps | grep -q "dynamodb-local"; then
            print_success "DynamoDB Local is already running"
            return 0
        else
            print_info "Starting DynamoDB Local (attempt $attempt/$max_attempts)..."
            if docker ps -a | grep -q "dynamodb-local"; then
                # Container exists but is not running
                docker start dynamodb-local || true
            else
                # Container doesn't exist, create it
                docker run -d --name dynamodb-local -p $DYNAMO_PORT:$DYNAMO_PORT amazon/dynamodb-local -jar DynamoDBLocal.jar -sharedDb || true
            fi
            
            # Wait for DynamoDB to be ready
            print_info "Waiting for DynamoDB to be ready..."
            for i in {1..10}; do
                if curl -s http://localhost:$DYNAMO_PORT >/dev/null; then
                    print_success "DynamoDB is ready"
                    return 0
                fi
                sleep 1
            done
            
            attempt=$((attempt + 1))
            if [ $attempt -le $max_attempts ]; then
                print_warning "DynamoDB not responding, retrying..."
                # Try to stop the container to restart fresh
                docker stop dynamodb-local > /dev/null 2>&1 || true
            fi
        fi
    done
    
    print_error "DynamoDB failed to start properly after $max_attempts attempts"
    return 1
}

# Set up AWS local profile
setup_aws_local() {
    print_info "Setting up AWS local profile for tests"
    aws configure set aws_access_key_id test --profile local
    aws configure set aws_secret_access_key test --profile local
    aws configure set region us-east-1 --profile local
    aws configure set output json --profile local
    aws configure set cli_pager "" --profile local
}

# Ensure test table exists
ensure_test_table_exists() {
    print_info "Checking if DynamoDB table '$TEST_TABLE' exists..."
    
    # Check if the table exists
    if aws dynamodb describe-table --table-name $TEST_TABLE --endpoint-url http://localhost:$DYNAMO_PORT --profile local --no-cli-pager > /dev/null 2>&1; then
        if [ "$FORCE_RECREATE_TABLE" = "true" ]; then
            print_info "Table '$TEST_TABLE' exists, but recreating for a clean test environment"
            aws dynamodb delete-table --table-name $TEST_TABLE --endpoint-url http://localhost:$DYNAMO_PORT --profile local --no-cli-pager > /dev/null
            
            # Wait for table to be deleted
            print_info "Waiting for table to be deleted..."
            while aws dynamodb describe-table --table-name $TEST_TABLE --endpoint-url http://localhost:$DYNAMO_PORT --profile local --no-cli-pager > /dev/null 2>&1; do
                sleep 1
            done
            
            # Now recreate it
            create_test_table
        else
            print_success "Table '$TEST_TABLE' already exists, using existing table"
        fi
    else
        print_info "Table '$TEST_TABLE' does not exist, creating..."
        create_test_table
    fi
    
    return 0
}

# Create the test table with proper indexes
create_test_table() {
    # Create the table
    aws dynamodb create-table \
        --table-name $TEST_TABLE \
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
        --billing-mode PROVISIONED \
        --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 \
        --endpoint-url http://localhost:$DYNAMO_PORT \
        --profile local \
        --no-cli-pager > /dev/null
        
    # Check if table creation was successful
    if [ $? -eq 0 ]; then
        print_success "Created table '$TEST_TABLE'"
        
        # Wait for the table to become active
        print_info "Waiting for table '$TEST_TABLE' to become active..."
        aws dynamodb wait table-exists \
            --table-name $TEST_TABLE \
            --endpoint-url http://localhost:$DYNAMO_PORT \
            --profile local \
            --no-cli-pager > /dev/null 2>&1
            
        print_success "Table '$TEST_TABLE' is now active"
    else
        print_error "Failed to create table '$TEST_TABLE'"
        return 1
    fi
}

# Run the tests
run_tests() {
    print_info "Running tests..."
    
    # Set trap to catch timeout or interrupt
    trap 'print_warning "Tests interrupted or timed out"; exit 1' TERM INT
    
    # Pass environment variable to control table recreation to the tests
    export FORCE_RECREATE_TABLE
    
    # Export test related environment variables
    export DYNAMODB_TABLE=$TEST_TABLE
    export DYNAMODB_ENDPOINT=http://localhost:$DYNAMO_PORT
    export AWS_PROFILE=local
    
    # If arguments are provided, run tests for those paths
    # Otherwise, run all tests
    if [ $# -eq 0 ]; then
        go test -timeout=$TEST_TIMEOUT -v ./tests/... || {
            code=$?
            if [ $code -eq 124 ]; then
                print_error "Tests timed out after ${TEST_TIMEOUT}"
            else
                print_error "Tests failed with exit code $code"
            fi
            return $code
        }
    else
        go test -timeout=$TEST_TIMEOUT -v "$@" || {
            code=$?
            if [ $code -eq 124 ]; then
                print_error "Tests timed out after ${TEST_TIMEOUT}"
            else
                print_error "Tests failed with exit code $code"
            fi
            return $code
        }
    fi
    
    return 0
}

# Main execution
ensure_dynamodb_running || exit 1
setup_aws_local
ensure_test_table_exists || exit 1

# Run tests and capture result
run_tests "$@"
result=$?

# Report results
if [ $result -eq 0 ]; then
    print_success "Tests completed successfully!"
else
    print_error "Tests failed with exit code $result"
    exit $result
fi 