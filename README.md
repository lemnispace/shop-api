# Shop API

A comprehensive REST API for e-commerce operations. This API allows clients to manage products, collections, orders, and other e-commerce related resources.

## Table of Contents

- [Overview](#overview)
- [Project Key Features](#key-features)
- [Technology Stack](#technology-stack)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Installation](#installation)
  - [Configuration](#configuration)
- [Development](#development)
  - [Running the API](#running-the-api)
  - [Testing](#testing)
  - [Code Structure](#code-structure)
  - [AWS Configuration](#aws-configuration)
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

### System Architecture

The API follows a serverless microservices architecture:

1. API Gateway receives and routes HTTP requests
2. Lambda functions process specific resource operations
3. DynamoDB stores and manages shop data

#### Database Design

This API uses DynamoDB following a single table design pattern. The single table design consolidates multiple entity types into one DynamoDB table, leveraging composite primary keys (partition and sort keys) to efficiently store and query heterogeneous data.

Key aspects of our DynamoDB implementation:

- **Partition Key**: Uses entity type prefixes (e.g., "PRODUCT#", "COLLECTION#") to distinguish between different data types
- **Sort Key**: Enables hierarchical relationships and efficient range queries
- **Secondary Indexes**: GSIs and LSIs for flexible access patterns beyond the primary key
- **Item Collections**: Related items are grouped together using the same partition key
- **Sparse Indexes**: Optimized for specific query patterns while minimizing storage costs

This approach offers several advantages

- Reduced latency by minimizing the number of round trips to the database
- Lower costs by consolidating data into a single table
- Simplified transaction management across related entities
- Improved query flexibility through strategic use of indexes

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
make test        # Run integration tests
make test-clean  # Stop test services
make run         # Run API directly (requires services running)
make build       # Build Lambda functions
make deploy      # Deploy to AWS
```

### Running the API

We use DynamoDB and MinIO (S3-compatible) for all environments to maintain consistency across development and production.

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
