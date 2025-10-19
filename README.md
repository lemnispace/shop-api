# Shop API

A comprehensive REST API for e-commerce operations. This API allows clients to manage products, collections, orders, and other e-commerce related resources.

## Table of Contents

- [Overview](#overview)
- [Project Key Features](#key-features)
- [Technology Stack](#technology-stack)
- [Quick Start](#quick-start)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Installation](#installation)
  - [Configuration](#configuration)
- [Development](#development)
  - [Running the API](#running-the-api)
  - [Manual API Testing](#manual-api-testing)
  - [Testing](#testing)
  - [Code Structure](#code-structure)
  - [AWS Configuration](#aws-configuration)
- [Security & Authentication](#security--authentication)
- [Troubleshooting](#troubleshooting)
- [Deployment](#deployment)
- [API Documentation](#api-documentation)

## Overview

Shop API serves as a robust backend for e-commerce applications. It provides essential functionality for managing products, collections, orders, and more. The API is designed to be scalable, extensible, and easy to integrate with.

### Purpose and Goals

- Create a RESTful API with comprehensive e-commerce functionality
- Facilitate data migration from existing systems
- Enable synchronization between different data sources
- Provide enhanced functionality tailored to shop-specific requirements
- Ensure scalability to handle varying levels of traffic and data volume
- Simplify integration for developers

## Project Key Features

- **Product Management**: Create, read, update, and delete products, variants, and images
- **Collection Management**: Group products into collections for easier organization
- **Order Processing**: Handle orders, fulfillment, and shipping
- **Customization Service**: Upload, process, and manage product customization images
- **Pagination, Filtering, and Sorting**: Efficiently browse and search through resources
- **Rate Limiting**: Protect the API from abuse
- **API Versioning**: Ensure backward compatibility
- **Infrastructure as Code**: All infrastructure is managed through Terraform.

## Technology Stack

- **Programming Language**: Go (chosen for performance, strong typing, and concurrency support)
- **Database**: Amazon DynamoDB (providing high scalability and low-latency data access)
- **Compute**: AWS Lambda (serverless architecture for cost efficiency and automatic scaling)
- **Infrastructure**: Terraform (infrastructure as code for consistent deployments)
- **API Gateway**: Amazon API Gateway (for request routing, throttling, and authentication)
- **CI/CD**: GitHub Actions (for automated testing and deployment)

## Quick Start

Get started in 2 minutes:

```bash
# Start all services (DynamoDB, MinIO, API)
docker compose up -d

# Verify services are running
sleep 5
curl http://localhost:8080/health

# Run tests
docker compose exec -T shop-api go test ./... -v

# Stop all services when done
docker compose down
```

**Service Endpoints**:
- API: `http://localhost:8080/v1`
- DynamoDB: `http://localhost:8000`
- MinIO Console: `http://localhost:9001` (credentials: minioadmin/minioadmin)

### System Architecture

The API follows a serverless microservices architecture:

1. API Gateway receives and routes HTTP requests
2. Lambda functions process specific resource operations
3. DynamoDB stores and manages shop data

#### Database Design

This API uses DynamoDB following a single table design pattern. The single table design consolidates multiple entity types into one DynamoDB table, leveraging composite primary keys (partition and sort keys) to efficiently store and query heterogeneous data.

Key aspects of our DynamoDB implementation:

- **Partition Key (PK)**: Uses entity type prefixes to distinguish between different data types:
  - `PRODUCT#{productId}` - Product entities
  - `COLLECTION#{collectionId}` - Collection entities
  - `PRODUCT#{productId}` paired with `VARIANT#{variantId}` - Product variants
  - `COLLECTION#{collectionId}` paired with `PRODUCT#{productId}` - Collection membership
  - `CUSTOMER#{customerId}` - Customer entities
  - `ORDER#{orderId}` - Order entities
  - And more...

- **Sort Key (SK)**: Enables hierarchical relationships and efficient range queries
  - `METADATA` - Main entity metadata
  - Entity relationships (e.g., `PRODUCT#{id}` for collection products)

- **Secondary Indexes**:
  - **GSI1**: For customer/user lookups and email searches
  - **GSI2**: For status-based queries (e.g., orders by status)

- **Item Collections**: Related items are grouped together using the same partition key for efficient batch operations

- **Sparse Indexes**: Optimized for specific query patterns while minimizing storage costs

**Example Access Patterns**:

```
Get product by ID:
  PK = PRODUCT#prod_123
  SK = METADATA

Get all variants for a product:
  PK = PRODUCT#prod_123
  SK = begins_with(VARIANT#)

Get products in a collection:
  PK = COLLECTION#col_456
  SK = begins_with(PRODUCT#)

Get customer's orders:
  GSI1PK = CUSTOMER#cus_789
  GSI1SK = begins_with(ORDER#)
```

This approach offers several advantages:

- Reduced latency by minimizing the number of round trips to the database
- Lower costs by consolidating data into a single table
- Simplified transaction management across related entities
- Improved query flexibility through strategic use of indexes
- Efficient filtering by membership (e.g., collection filtering uses PK=COLLECTION#{id}, SK=PRODUCT#{id} relationships)

## Getting Started

### Recommended: DevContainer Setup

**This project uses DevContainers for the best development experience.**

DevContainers provide a consistent, containerized development environment with all dependencies pre-installed (Go 1.23, AWS CLI, Docker, Terraform).

#### Quick Start with DevContainer

1. **Prerequisites**:
   - Visual Studio Code
   - Docker Desktop
   - [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)

2. **Open in DevContainer**:
   ```bash
   git clone https://github.com/lemnispace/shop-api.git
   cd shop-api
   code .
   ```
   - When prompted, click "Reopen in Container"
   - VS Code will build and start the devcontainer (first time takes a few minutes)

3. **Run Tests** (inside devcontainer):
   ```bash
   make test        # Runs integration tests with local DynamoDB + MinIO
   ```

4. **Start Development Services**:
   ```bash
   make dev-up      # Start all services (DynamoDB, MinIO, API)
   make run         # Run API server directly (requires services running)
   ```

5. **Access Services**:
   - API: http://localhost:8080
   - DynamoDB: http://localhost:8000
   - MinIO: http://localhost:9000
   - MinIO Console: http://localhost:9001 (user: minioadmin, password: minioadmin)

6. **Stop Services**:
   ```bash
   make dev-down    # Stop all services
   make test-clean  # Clean up test services
   ```

### Alternative: Local Setup (without DevContainer)

If you prefer not to use DevContainers:

#### System Prerequisites

- Go 1.23 or higher
- Docker and Docker Compose
- AWS CLI v2
- Make

#### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/lemnispace/shop-api.git
   cd shop-api
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Start local services:
   ```bash
   make dev-up      # Starts DynamoDB, MinIO, and API
   ```

4. Run tests:
   ```bash
   make test        # Run integration tests
   ```

### Configuration

#### Local Development (DevContainer or Local)

The default configuration works out of the box with local services. Environment variables are set in `docker-compose.yml`:

- `DYNAMODB_ENDPOINT=http://localhost:8000`
- `S3_ENDPOINT=http://localhost:9000`
- `AWS_ACCESS_KEY_ID=minioadmin`
- `AWS_SECRET_ACCESS_KEY=minioadmin`

#### Production Deployment

Create a `.env` file in the project root with:

```bash
AWS_PROFILE=profile-name
AWS_SSO_START_URL=start-url
AWS_REGION=region
AWS_SSO_ACCOUNT_ID=account-id
AWS_SSO_ROLE_NAME=role-name
AWS_OUTPUT=json
AWS_SDK_LOAD_CONFIG=1
DYNAMODB_TABLE=your-dynamodb-table
```

## Development

### Available Make Commands

```bash
make dev-up      # Start all development services
make dev-down    # Stop all services
make dev-logs    # Follow service logs
make test        # Run integration tests with Go installed
make test-clean  # Stop test services
make run         # Run API directly (requires services running)
make build       # Build Lambda functions
make deploy      # Deploy to AWS
```

### Running the API

We use DynamoDB and MinIO (S3-compatible) for all environments to maintain consistency across development and production.

#### Option 1: Using Make (Recommended)

```bash
# Start all services
make dev-up

# In another terminal, run the API (Go must be installed)
make run

# View logs
make dev-logs

# Stop all services
make dev-down
```

#### Option 2: Using Docker Compose Directly

```bash
# Start infrastructure services
docker compose up -d dynamodb-local minio create-dynamodb-table createbuckets

# Start the API service (builds and runs)
docker compose up --build shop-api

# Stop all services
docker compose down
```

#### Option 3: Docker Compose (Without Building)

```bash
# If you've already built the image
docker compose up shop-api

# Or build specifically
docker compose build shop-api
```

### Code Structure

The codebase is organized as follows:

- `cmd/`: Entry points for the application
- `internal/`: Internal packages
  - `handlers/`: HTTP request handlers
  - `models/`: Data models
  - `services/`: Business logic
  - `utils/`: Utility functions
  - `routers/`: Router configuration
- `tests/`: Test files
  - `api/`: API integration tests
  - `unit/`: Unit tests
- `scripts/`: Helper scripts
- `terraform/`: Infrastructure as code

### Manual API Testing

Test the API using curl without authentication:

```bash
# Health check (should return immediately)
curl http://localhost:8080/health

# List products (public endpoint)
curl http://localhost:8080/v1/products

# Count all products
curl http://localhost:8080/v1/products/count

# Filter products by status
curl 'http://localhost:8080/v1/products?status=active'

# Filter products by collection (only returns products in that collection)
curl 'http://localhost:8080/v1/products?collection=<collection-id>'

# Count products in a collection
curl 'http://localhost:8080/v1/products/count?collection=<collection-id>'

# List collections
curl http://localhost:8080/v1/collections
```

**Testing Protected Endpoints** (require authentication):

Protected endpoints (POST, PUT, DELETE) require a JWT token in the `Authorization` header:

```bash
# This will fail (no auth)
curl -X POST http://localhost:8080/v1/products \
  -H "Content-Type: application/json" \
  -d '{"title": "Test"}'
# Returns: 401 Unauthorized

# In local development (RUN_LOCAL=true), auth is optional for some endpoints
```

### Testing

#### Option 1: Using Make

```bash
# Starts services, runs tests, cleans up
make test
```

#### Option 2: Using Docker Compose Directly

```bash
# Start infrastructure for testing
docker compose up -d dynamodb-local minio createbuckets create-dynamodb-table

# Wait for services to be ready
sleep 8

# Run tests inside the shop-api container
docker compose exec -T shop-api go test ./... -v

# Clean up
docker compose down
```

#### Option 3: Run Tests Manually with Go Installed

```bash
# Start infrastructure first
docker compose up -d dynamodb-local minio createbuckets create-dynamodb-table

# Set required environment variables and run tests
DYNAMODB_ENDPOINT=http://localhost:8000 \
S3_ENDPOINT=http://localhost:9000 \
S3_USE_PATH_STYLE=true \
AWS_ACCESS_KEY_ID=minioadmin \
AWS_SECRET_ACCESS_KEY=minioadmin \
AWS_REGION=us-east-1 \
go test -v ./... -count=1
```

### AWS Configuration

For production deployment, you'll need to set up AWS credentials. The `setup-aws.sh` script automates the setup of AWS CLI configuration for AWS Single Sign-On (SSO) authentication.

#### Prerequisites

- AWS CLI v2 installed
- The following environment variables must be set:
  - `AWS_PROFILE`: The name of the AWS profile to create
  - `AWS_SSO_START_URL`: Your organization's SSO start URL
  - `AWS_SSO_ACCOUNT_ID`: Your AWS account ID
  - `AWS_SSO_ROLE_NAME`: The IAM role name to assume
  - `AWS_REGION`: AWS region (defaults to us-east-1)

#### Usage

1. Set the required environment variables
2. Run the script:

   ```bash
   ./scripts/setup-aws.sh
   ```

## Security & Authentication

### Authentication Requirements

The Shop API uses JWT (JSON Web Tokens) for authentication. Different endpoints have different requirements:

#### Public Endpoints (No Authentication Required)

- `GET /v1/products` - List products with optional filtering
- `GET /v1/products/count` - Get product count
- `GET /v1/products/:productId` - Get single product details
- `GET /v1/collections` - List collections
- `GET /v1/collections/:collectionId` - Get collection details

#### Protected Endpoints (Authentication Required)

All write operations (POST, PUT, DELETE) require authentication via JWT token:

- `POST /v1/products` - Create product
- `PUT /v1/products/:productId` - Update product
- `DELETE /v1/products/:productId` - Delete product
- `POST /v1/orders` - Create order
- `GET /v1/orders/:orderId` - Get customer's order (requires order ownership)
- `POST /v1/orders/:orderId/payment-intent` - Create payment intent (requires order ownership)
- And all other write operations...

### JWT Authentication

In production, provide JWT token in the `Authorization` header:

```bash
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -X POST http://localhost:8080/v1/products \
  -H "Content-Type: application/json" \
  -d '{"title": "New Product"}'
```

### Customer Ownership Verification

The API enforces strict customer ownership verification on sensitive operations:

- **Orders**: Users can only view/modify their own orders
- **Payments**: Users can only pay for their own orders
- **Customizations**: Users can only access their own customization images

This prevents cross-customer data access vulnerabilities.

### Local Development (RUN_LOCAL=true)

In local development mode (`RUN_LOCAL=true` environment variable), authentication may be relaxed for testing, but ownership verification is still enforced.

## Troubleshooting

### Issue: "Go: command not found"

**Problem**: You need to run tests but Go is not installed locally.

**Solution**: Use Docker Compose to run tests:

```bash
docker compose exec -T shop-api go test ./... -v
```

The shop-api container has Go 1.23 pre-installed. You don't need Go on your host machine.

### Issue: "Port already in use" (Address already in use)

**Problem**: Docker Compose can't bind to port 8080 or another port.

**Solution**: Check what's using the port and stop it:

```bash
# Find what's using port 8080
lsof -i :8080

# Stop the container using the port
docker compose down

# Or, modify docker-compose.yml to use different ports
# Change "8080:8080" to "8081:8080" for example
```

### Issue: "Connection refused" when API starts

**Problem**: API container starts but can't connect to DynamoDB or MinIO.

**Solution**: Ensure infrastructure services started first:

```bash
# Check if services are running
docker compose ps

# Start infrastructure services
docker compose up -d dynamodb-local minio createbuckets create-dynamodb-table

# Wait for services to be ready
sleep 8

# Then start API
docker compose up shop-api
```

### Issue: "Collection filter returns all products"

**Problem**: `/v1/products?collection=xyz` returns all products instead of just those in the collection.

**Solution**: Ensure you're using the latest code. Collection filtering was fully implemented to actually query collection-product relationships from DynamoDB (Pattern: `PK=COLLECTION#{id}, SK=PRODUCT#{id}`). Update your code if needed.

### Issue: "Authentication failures" on write endpoints

**Problem**: `401 Unauthorized` when trying to create/update products.

**Solution**:

1. Verify you're providing a valid JWT token in the `Authorization` header
2. In local development, check that `RUN_LOCAL=true` is set if you want to disable auth
3. Public read endpoints don't require auth; only write operations do

### Issue: "DynamoDB table not found"

**Problem**: Errors about table "ShopAPI" not existing.

**Solution**: The table should be created automatically by the `create-dynamodb-table` service. If not:

```bash
# Restart the table creation service
docker compose up create-dynamodb-table

# Or verify the table exists
aws dynamodb describe-table --table-name ShopAPI \
  --endpoint-url http://localhost:8000 \
  --region us-east-1
```

### Issue: "MinIO buckets not created"

**Problem**: Errors about S3 buckets not existing.

**Solution**: The buckets are created by the `createbuckets` service:

```bash
# Verify buckets exist in MinIO console
# Open http://localhost:9001 (user: minioadmin, password: minioadmin)

# Or restart the bucket creation service
docker compose up createbuckets
```

## Deployment

For production deployment, we use Terraform to manage AWS resources:

```bash
make deploy
```

This will:

1. Apply the Terraform configuration in the `terraform/` directory
2. Deploy the API to AWS Lambda and API Gateway
3. Configure DynamoDB tables in AWS

### Deployment Pipeline

The deployment process follows these steps:

1. Code changes pushed to GitHub repository
2. Automated tests run in CI/CD pipeline
3. Terraform plans generated and reviewed
4. Infrastructure changes applied via Terraform
5. API deployed to staging environment for integration testing
6. Production deployment with traffic shifting for zero downtime

## API Documentation

Detailed API documentation is available in [API_DESIGN.md](API_DESIGN.md).

### Customization Service

The Shop API includes a customization service that allows users to upload, process, and manage images for product customization. This service is built on top of S3 for file storage and DynamoDB for metadata management.

#### Key Features

- **Image Upload**: Upload images for product customization with user-specific access control
- **Image Processing**: Apply operations like resizing, cropping, and background removal
- **Presigned URLs**: Generate secure, time-limited URLs for image access
- **Metadata Management**: Store and retrieve image metadata
- **Lifecycle Management**: Automatically expire unused images
- **User-Specific Customizations**: Each customization is linked to a specific user
- **Cart Integration**: Link customizations to specific cart items for checkout

#### API Endpoints

- `POST /v1/customizations/images`: Upload a new customization image
- `GET /v1/customizations/images?userId={userId}`: List all customization images for a user
- `GET /v1/customizations/images?userId={userId}&productId={productId}&variantId={variantId}`: List user's customizations for a specific product variant
- `GET /v1/customizations/images/{imageId}?userId={userId}`: Get image metadata (user verification required)
- `POST /v1/customizations/images/{imageId}/process?userId={userId}`: Process an image with operations
- `POST /v1/customizations/images/{imageId}/link?userId={userId}`: Link an image to a cart item
- `DELETE /v1/customizations/images/{imageId}?userId={userId}`: Delete an image

#### User-Specific Access Control

The customization service implements strict user-specific access control:

1. Each image is associated with the user who uploaded it
2. Users can only access, process, or delete their own images
3. User verification is required for all operations to prevent unauthorized access
4. Images in S3 are organized by user ID to maintain separation

#### Image Operations

The customization service supports the following image operations:

- **Resize**: Change the dimensions of an image
- **Crop**: Crop an image to a specific region
- **Remove Background**: Automatically remove the background from an image

#### Cart Integration

The customization service integrates with the cart API:

1. When a user creates a customization, they can upload images
2. The customization is linked to a specific product variant
3. When the user adds the customized product to their cart, the customization is linked to the cart item
4. During checkout, the customization data is included with the order

This ensures that each user's customizations are private and only visible to them, while still allowing the customization to be included in their order.

### Base URL

- Local: `http://localhost:8080/v1`
- Production: `https://api.lemnispace.com/v1`

## Security Considerations

- HTTPS encryption for all API communications
- Input validation and sanitization to prevent injection attacks
- DDoS protection through AWS Shield
- PCI DSS compliance for payment information
