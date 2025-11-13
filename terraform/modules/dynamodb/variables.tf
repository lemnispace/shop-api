variable "table_name" {
  description = "Name of the DynamoDB table"
  type        = string
}

variable "billing_mode" {
  description = "Billing mode for the table (PROVISIONED or PAY_PER_REQUEST)"
  type        = string
  default     = "PAY_PER_REQUEST"

  validation {
    condition     = contains(["PROVISIONED", "PAY_PER_REQUEST"], var.billing_mode)
    error_message = "billing_mode must be either PROVISIONED or PAY_PER_REQUEST"
  }
}

variable "table_read_capacity" {
  description = "Read capacity units for the table (only used if billing_mode is PROVISIONED)"
  type        = number
  default     = 5
}

variable "table_write_capacity" {
  description = "Write capacity units for the table (only used if billing_mode is PROVISIONED)"
  type        = number
  default     = 5
}

variable "gsi_read_capacity" {
  description = "Read capacity units for GSIs (only used if billing_mode is PROVISIONED)"
  type        = number
  default     = 5
}

variable "gsi_write_capacity" {
  description = "Write capacity units for GSIs (only used if billing_mode is PROVISIONED)"
  type        = number
  default     = 5
}

variable "enable_pitr" {
  description = "Enable point-in-time recovery"
  type        = bool
  default     = false
}

variable "enable_encryption" {
  description = "Enable server-side encryption"
  type        = bool
  default     = true
}

variable "ttl_attribute" {
  description = "Attribute name for TTL (empty string to disable)"
  type        = string
  default     = ""
}

variable "environment" {
  description = "Environment name (local, dev, staging, prod)"
  type        = string
}

variable "tags" {
  description = "Additional tags for the table"
  type        = map(string)
  default     = {}
}
