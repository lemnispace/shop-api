terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  # Remote state backend for dev environment
  backend "s3" {
    bucket         = "lemnispace-terraform-state"
    key            = "shop-api/dev/terraform.tfstate"
    region         = "us-east-1"
    dynamodb_table = "terraform-state-lock"
    encrypt        = true
  }
}

provider "aws" {
  region = var.aws_region
}

# Get shared infrastructure outputs
data "terraform_remote_state" "lemnispace_services" {
  backend = "s3"
  config = {
    bucket         = "lemnispace-terraform-state"
    key            = "infra/terraform.tfstate"
    region         = "us-east-1"
    dynamodb_table = "terraform-state-lock"
  }
}

# Create DynamoDB table using the module
module "dynamodb" {
  source = "../../modules/dynamodb"

  table_name   = var.table_name
  billing_mode = var.billing_mode

  table_read_capacity  = var.table_read_capacity
  table_write_capacity = var.table_write_capacity
  gsi_read_capacity    = var.gsi_read_capacity
  gsi_write_capacity   = var.gsi_write_capacity

  enable_pitr       = var.enable_pitr
  enable_encryption = true

  environment = "dev"

  tags = {
    Environment = "dev"
    Service     = "shop-api"
  }
}

# Lambda function (keeping existing pattern from main.tf)
data "archive_file" "ShopFunction" {
  type        = "zip"
  source_file = "${path.module}/../../../build/shop/bootstrap"
  output_path = "${path.module}/../../../build/shop/ShopFunction.zip"
}

resource "aws_s3_object" "shop_service" {
  bucket = data.terraform_remote_state.lemnispace_services.outputs.services_s3_bucket_id
  key    = "dev/ShopFunction.zip"
  source = data.archive_file.ShopFunction.output_path
  etag   = filemd5(data.archive_file.ShopFunction.output_path)
}

resource "aws_lambda_function" "ShopFunction" {
  filename         = data.archive_file.ShopFunction.output_path
  function_name    = "${var.function_name_prefix}ShopFunction"
  role             = data.terraform_remote_state.lemnispace_services.outputs.execute_lambda_role_arn
  handler          = "main"
  runtime          = "provided.al2023"
  source_code_hash = data.archive_file.ShopFunction.output_base64sha256
  timeout          = 30
  memory_size      = 512

  environment {
    variables = {
      ALLOWED_ORIGINS = var.allow_origins
      ROOT_PATH       = "/shop"
      DYNAMODB_TABLE  = module.dynamodb.table_name
      S3_BUCKET       = data.terraform_remote_state.lemnispace_services.outputs.user_product_files_s3_bucket_id
      ENVIRONMENT     = "dev"
    }
  }

  tags = {
    Environment = "dev"
    Service     = "shop-api"
  }
}

resource "aws_lambda_permission" "shop_service" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.ShopFunction.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${data.terraform_remote_state.lemnispace_services.outputs.api_execution_arn}/*/*"
}

# API Gateway route
module "shop_route" {
  source            = "../../modules/routes"
  lambda_endpoint   = "/shop"
  lambda_invoke_arn = aws_lambda_function.ShopFunction.invoke_arn
  api_id            = data.terraform_remote_state.lemnispace_services.outputs.api_id
}
