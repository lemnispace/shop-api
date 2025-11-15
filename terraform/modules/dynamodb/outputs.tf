output "table_name" {
  description = "Name of the DynamoDB table"
  value       = aws_dynamodb_table.shop_table.name
}

output "table_arn" {
  description = "ARN of the DynamoDB table"
  value       = aws_dynamodb_table.shop_table.arn
}

output "table_id" {
  description = "ID of the DynamoDB table"
  value       = aws_dynamodb_table.shop_table.id
}

output "table_stream_arn" {
  description = "ARN of the table stream"
  value       = aws_dynamodb_table.shop_table.stream_arn
}

output "gsi_names" {
  description = "Names of all GSIs"
  value       = ["GSI1", "GSI2", "ProductsByStatus"]
}
