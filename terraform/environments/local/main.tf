terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  # No remote backend for local development
  # State is stored locally in terraform.tfstate
}

# Configure AWS provider to use LocalStack
provider "aws" {
  region = var.aws_region

  # LocalStack configuration
  access_key = "test"
  secret_key = "test"

  # Skip credential validation for LocalStack
  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true

  # LocalStack endpoints
  endpoints {
    dynamodb   = var.localstack_endpoint
    s3         = var.localstack_endpoint
    apigateway = var.localstack_endpoint
  }

  # Use path-style S3 URLs for LocalStack
  s3_use_path_style = true
}

# Create DynamoDB table using the module
module "dynamodb" {
  source = "../../modules/dynamodb"

  table_name   = var.table_name
  billing_mode = "PROVISIONED" # LocalStack requires PROVISIONED mode

  table_read_capacity  = 5
  table_write_capacity = 5
  gsi_read_capacity    = 5
  gsi_write_capacity   = 5

  enable_pitr       = false # Disable PITR for local development
  enable_encryption = false # Disable encryption for local development

  environment = "local"

  tags = {
    Environment = "local"
    Purpose     = "development"
  }
}

# Create S3 bucket for local development (replacing MinIO)
resource "aws_s3_bucket" "local_storage" {
  bucket = var.s3_bucket_name

  tags = {
    Name        = var.s3_bucket_name
    Environment = "local"
  }
}

# Make the bucket publicly readable for local development
resource "aws_s3_bucket_public_access_block" "local_storage" {
  bucket = aws_s3_bucket.local_storage.id

  block_public_acls       = false
  block_public_policy     = false
  ignore_public_acls      = false
  restrict_public_buckets = false
}
