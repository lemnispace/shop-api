output "dynamodb_table_name" {
  description = "Name of the DynamoDB table"
  value       = module.dynamodb.table_name
}

output "dynamodb_table_arn" {
  description = "ARN of the DynamoDB table"
  value       = module.dynamodb.table_arn
}

output "s3_bucket_name" {
  description = "Name of the S3 bucket"
  value       = aws_s3_bucket.local_storage.bucket
}

output "gsi_names" {
  description = "Names of all GSIs"
  value       = module.dynamodb.gsi_names
}

output "localstack_endpoint" {
  description = "LocalStack endpoint being used"
  value       = var.localstack_endpoint
}
