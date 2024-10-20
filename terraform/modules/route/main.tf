resource "aws_apigatewayv2_route" "shop_route" {
  api_id    = var.api_id
  route_key = "ANY ${var.lambda_endpoint}"
  target    = "integrations/${aws_apigatewayv2_integration.shop_integration.id}"
}

resource "aws_apigatewayv2_integration" "shop_integration" {
  api_id             = var.api_id
  integration_type   = "AWS_PROXY"
  integration_uri    = var.lambda_invoke_arn
  integration_method = "POST"
}

variable "api_id" {
  type = string
}

variable "lambda_endpoint" {
  type = string
}

variable "lambda_invoke_arn" {
  type = string
}
