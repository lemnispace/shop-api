variable "aws_region" {
  description = "AWS region"
  default     = "us-east-1"
}

variable "project_name" {
  description = "The name of the deployment repository"
  type        = string
  default     = "shop-api"
}

variable "allow_origins" {
  description = "Allowed origins for CORS"
  type        = string
}

variable "root_path" {
  description = "Root path for the API"
  type        = string
}
