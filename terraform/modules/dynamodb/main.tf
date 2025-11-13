resource "aws_dynamodb_table" "shop_table" {
  name         = var.table_name
  billing_mode = var.billing_mode
  hash_key     = "PK"
  range_key    = "SK"

  # Primary key attributes
  attribute {
    name = "PK"
    type = "S"
  }

  attribute {
    name = "SK"
    type = "S"
  }

  # GSI1 attributes - for status-based queries and customer lookups
  attribute {
    name = "GSI1PK"
    type = "S"
  }

  attribute {
    name = "GSI1SK"
    type = "S"
  }

  # GSI2 attributes - for SKU lookups and additional indexes
  attribute {
    name = "GSI2PK"
    type = "S"
  }

  attribute {
    name = "GSI2SK"
    type = "S"
  }

  # ProductsByStatus GSI attributes - for efficient product listing
  attribute {
    name = "EntityType"
    type = "S"
  }

  attribute {
    name = "CreatedAt"
    type = "S"
  }

  # GSI1: Status-based queries and customer lookups
  global_secondary_index {
    name            = "GSI1"
    hash_key        = "GSI1PK"
    range_key       = "GSI1SK"
    projection_type = "ALL"

    # Only set throughput for PROVISIONED billing mode
    read_capacity  = var.billing_mode == "PROVISIONED" ? var.gsi_read_capacity : null
    write_capacity = var.billing_mode == "PROVISIONED" ? var.gsi_write_capacity : null
  }

  # GSI2: SKU lookups and additional indexes
  global_secondary_index {
    name            = "GSI2"
    hash_key        = "GSI2PK"
    range_key       = "GSI2SK"
    projection_type = "ALL"

    read_capacity  = var.billing_mode == "PROVISIONED" ? var.gsi_read_capacity : null
    write_capacity = var.billing_mode == "PROVISIONED" ? var.gsi_write_capacity : null
  }

  # ProductsByStatus: Efficient product listing by entity type and creation date
  # This GSI allows us to Query products instead of Scan, improving performance
  # and enabling proper cursor-based pagination
  global_secondary_index {
    name            = "ProductsByStatus"
    hash_key        = "EntityType"
    range_key       = "CreatedAt"
    projection_type = "ALL"

    read_capacity  = var.billing_mode == "PROVISIONED" ? var.gsi_read_capacity : null
    write_capacity = var.billing_mode == "PROVISIONED" ? var.gsi_write_capacity : null
  }

  # Only set throughput for PROVISIONED billing mode
  read_capacity  = var.billing_mode == "PROVISIONED" ? var.table_read_capacity : null
  write_capacity = var.billing_mode == "PROVISIONED" ? var.table_write_capacity : null

  # Point-in-time recovery for production
  point_in_time_recovery {
    enabled = var.enable_pitr
  }

  # Server-side encryption
  server_side_encryption {
    enabled = var.enable_encryption
  }

  # TTL configuration (optional)
  dynamic "ttl" {
    for_each = var.ttl_attribute != "" ? [1] : []
    content {
      attribute_name = var.ttl_attribute
      enabled        = true
    }
  }

  tags = merge(
    var.tags,
    {
      Name        = var.table_name
      Environment = var.environment
      ManagedBy   = "terraform"
    }
  )
}
