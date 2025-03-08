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
- [Contributing](#contributing)
- [License](#license)

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

- Go 1.19 or higher
- AWS CLI (for deployment)
- Docker and Docker Compose (for local development)
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

3.Build the project:

```bash
make build
```

### Configuration

Create a `.env` file in the project root with the following variables:

```bash
AWS_PROFILE=profile-name
AWS_SSO_START_URL=start-url
AWS_REGION=region
AWS_SSO_ACCOUNT_ID=account-id
AWS_SSO_ROLE_NAME=role-name
AWS_OUTPUT=json
AWS_SDK_LOAD_CONFIG=1
```

## Development

### Running the API

To run the API locally:

```bash
make run
```

This will start the API server at <http://localhost:8080>.

For local development with hot reloading:

```bash
make dev
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

The `setup-aws.sh` script automates the setup of AWS CLI configuration for AWS Single Sign-On (SSO) authentication.
It creates the necessary AWS configuration files and initiates the SSO login process.

### Prerequisites

- AWS CLI v2 installed
- The following environment variables must be set:
  - `AWS_PROFILE`: The name of the AWS profile to create
  - `AWS_SSO_START_URL`: Your organization's SSO start URL
  - `AWS_SSO_ACCOUNT_ID`: Your AWS account ID
  - `AWS_SSO_ROLE_NAME`: The IAM role name to assume
  - `AWS_REGION`: (Optional) AWS region (defaults to us-east-1)
  - `AWS_OUTPUT`: (Optional) Output format (defaults to json)

### Usage

1. Set the required environment variables
2. Ensure the script is executable:
   ```chmod +x setup-aws.sh```
3. Run the script:
   ```./setup-aws.sh```

### What it does

1. Creates `~/.aws` directory if it doesn't exist
2. Generates AWS CLI configuration file with SSO settings
3. Attempts automatic SSO login
4. Provides feedback on the login status

#### When to use

- Initial development environment setup
- When setting up new AWS SSO access
- After refreshing AWS SSO credentials
- When configuring CI/CD pipelines that need AWS authentication

#### Notes

- If automatic login fails, you'll need to run the SSO login command manually
- The script is non-destructive and can be run multiple times
- Existing AWS configurations will be overwritten for the specified profile

## Deployment

### AWS Setup

Configure AWS credentials:

```bash
./setup-aws.sh
```

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

### Authentication

The API uses Bearer tokens for authentication:

```bash
Authorization: Bearer {your_access_token}
```

### Base URL

```bash
https://api.lemnispace.com/v1
```

## Security Considerations

- HTTPS encryption for all API communications
- Input validation and sanitization to prevent injection attacks
- DDoS protection through AWS Shield
- PCI DSS compliance for payment information
