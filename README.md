# Shop API

A comprehensive REST API for e-commerce operations. This API allows clients to manage products, collections, orders, and other e-commerce related resources.

## Table of Contents

- [Overview](#overview)
- [Key Features](#key-features)
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

## Key Features

- **Product Management**: Create, read, update, and delete products, variants, and images
- **Collection Management**: Group products into collections for easier organization
- **Order Processing**: Handle orders, fulfillment, and shipping
- **Pagination, Filtering, and Sorting**: Efficiently browse and search through resources
- **Rate Limiting**: Protect the API from abuse
- **API Versioning**: Ensure backward compatibility

## Technology Stack

- **Programming Language**: Go (chosen for performance, strong typing, and concurrency support)
- **Database**: Amazon DynamoDB (providing high scalability and low-latency data access)
- **Compute**: AWS Lambda (serverless architecture for cost efficiency and automatic scaling)
- **Infrastructure**: Terraform (infrastructure as code for consistent deployments)
- **API Gateway**: Amazon API Gateway (for request routing, throttling, and authentication)
- **Monitoring**: AWS CloudWatch (for logging, metrics, and alerting)
- **CI/CD**: GitHub Actions (for automated testing and deployment)

### System Architecture

The API follows a serverless microservices architecture:

1. API Gateway receives and routes HTTP requests
2. Lambda functions process specific resource operations
3. DynamoDB stores and manages shop data
4. CloudWatch monitors system health and performance

## Getting Started

### System Prerequisites

- Go 1.22 or higher
- Docker and Docker Compose (for local DynamoDB)
- AWS CLI v2
- Make

### Installation

1. Clone the repository:

```bash
git clone https://github.com/lemnispace/shop-api.git
cd shop-api
```

2.Install dependencies:

```bash
go mod download
```

3.Set up local AWS credentials for development:

```bash
aws configure set aws_access_key_id test --profile local
aws configure set aws_secret_access_key test --profile local
aws configure set region us-east-1 --profile local
```

4.Start local DynamoDB:

```bash
make dynamo-local
```

5.Initialize DynamoDB table:

```bash
make dynamo-init
```

### Configuration

For local development, you can use the default configuration provided by the `dev.sh` script.

For production deployment, create a `.env` file in the project root with:

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

### Running the API

We use DynamoDB for all environments (both local development and production) to maintain consistency across all stages of development.

#### Local Development (Recommended)

For a complete development environment setup with local DynamoDB:

```bash
make dev
```

This script automates the following tasks:

- Kills any existing processes on port 8080
- Ensures DynamoDB Local is running (starts if needed)
- Sets up AWS local credentials
- Creates the required DynamoDB table if it doesn't exist
- Builds and runs the API with proper environment variables

#### Manual Setup

If you prefer a more manual approach:

1. Start local DynamoDB:

```bash
make dynamo-local
```

2.Initialize DynamoDB tables:

```bash
make dynamo-init
```

3.Run the API with local DynamoDB:

```bash
make run
```

#### Using AWS DynamoDB

To run the API using your AWS DynamoDB (requires proper AWS credentials):

```bash
make run-prod
```

### Testing

Run the test suite:

```bash
make test
```

For unit tests only:

```bash
make test-unit
```

For API tests only:

```bash
make test-api
```

For test coverage:

```bash
make test-coverage
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

API documentation is available in [API_DESIGN.md](API_DESIGN.md).

### Base URL

- Local: `http://localhost:8080/v1`
- Production: `https://api.lemnispace.com/v1`

## Security Considerations

- HTTPS encryption for all API communications
- Input validation and sanitization to prevent injection attacks
- DDoS protection through AWS Shield
- PCI DSS compliance for payment information
