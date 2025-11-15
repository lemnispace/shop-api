variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "table_name" {
  description = "DynamoDB table name"
  type        = string
  default     = "shop-api-dev"
}

variable "billing_mode" {
  description = "DynamoDB billing mode (PROVISIONED or PAY_PER_REQUEST)"
  type        = string
  default     = "PAY_PER_REQUEST"
}

variable "table_read_capacity" {
  description = "Read capacity units (only used if billing_mode is PROVISIONED)"
  type        = number
  default     = 5
}

variable "table_write_capacity" {
  description = "Write capacity units (only used if billing_mode is PROVISIONED)"
  type        = number
  default     = 5
}

variable "gsi_read_capacity" {
  description = "GSI read capacity units (only used if billing_mode is PROVISIONED)"
  type        = number
  default     = 5
}

variable "gsi_write_capacity" {
  description = "GSI write capacity units (only used if billing_mode is PROVISIONED)"
  type        = number
  default     = 5
}

variable "enable_pitr" {
  description = "Enable point-in-time recovery"
  type        = bool
  default     = true
}

variable "function_name_prefix" {
  description = "Prefix for Lambda function names"
  type        = string
  default     = "dev-"
}

variable "allow_origins" {
  description = "Allowed CORS origins"
  type        = string
  default     = "*"
}
