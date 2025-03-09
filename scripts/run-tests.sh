#!/bin/bash
set -e

DYNAMO_PORT=8000
TEST_TIMEOUT=60s  # Set test timeout to 60 seconds

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
}

# Run the tests
run_tests() {
    print_info "Running tests..."
    
    # Set trap to catch timeout or interrupt
    trap 'print_warning "Tests interrupted or timed out"; exit 1' TERM INT
    
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