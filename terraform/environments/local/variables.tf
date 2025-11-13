variable "aws_region" {
  description = "AWS region (used even for LocalStack)"
  type        = string
  default     = "us-east-1"
}

variable "localstack_endpoint" {
  description = "LocalStack endpoint URL"
  type        = string
  default     = "http://localhost:4566"
}

variable "table_name" {
  description = "DynamoDB table name"
  type        = string
  default     = "ShopAPI"
}

variable "s3_bucket_name" {
  description = "S3 bucket name for local storage"
  type        = string
  default     = "lemnispace-local-storage"
}
