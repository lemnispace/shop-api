output "lambda_function_name" {
  value = aws_lambda_function.ShopFunction.function_name
}

output "dynamodb_table_name" {
  value = aws_dynamodb_table.shop_table.name
}
