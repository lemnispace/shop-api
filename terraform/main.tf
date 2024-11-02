terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
  backend "s3" {
    bucket         = "lemnispace-terraform-state"
    key            = "shop-api/terraform.tfstate"
    region         = "us-east-1"
    dynamodb_table = "terraform-state-lock"
  }
}

provider "aws" {
  region = var.aws_region
}

data "terraform_remote_state" "lemnispace_services" {
  backend = "s3"
  config = {
    bucket         = "lemnispace-terraform-state"
    key            = "infra/terraform.tfstate"
    region         = "us-east-1"
    dynamodb_table = "terraform-state-lock"
  }
}

module "shop_route" {
  source            = "./modules/routes"
  lambda_endpoint   = "/shop"
  lambda_invoke_arn = aws_lambda_function.ShopFunction.invoke_arn
  api_id            = data.terraform_remote_state.lemnispace_services.outputs.api_id
}

data "archive_file" "ShopFunction" {
  type        = "zip"
  source_file = "${path.module}/../build/shop/bootstrap"
  output_path = "${path.module}/../build/shop/ShopFunction.zip"
}

resource "aws_s3_object" "shop_service" {
  bucket = data.terraform_remote_state.lemnispace_services.outputs.services_s3_bucket_id
  key    = "ShopFunction.zip"
  source = data.archive_file.ShopFunction.output_path
  etag   = filemd5(data.archive_file.ShopFunction.output_path)
}

resource "aws_lambda_function" "ShopFunction" {
  filename         = data.archive_file.ShopFunction.output_path
  function_name    = "ShopFunction"
  role             = data.terraform_remote_state.lemnispace_services.outputs.execute_lambda_role_arn
  handler          = "main"
  runtime          = "go1.x"
  source_code_hash = data.archive_file.ShopFunction.output_base64sha256
  timeout          = 30
  memory_size      = 512

  environment {
    variables = {
      ALLOWED_ORIGINS = var.allow_origins
      ROOT_PATH       = "/shop"
      DYNAMODB_TABLE  = aws_dynamodb_table.shop_table.name
      S3_BUCKET       = aws_s3_bucket.file_storage.id
    }
  }
}

resource "aws_lambda_permission" "shop_service" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.ShopFunction.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${data.terraform_remote_state.lemnispace_services.outputs.api_execution_arn}/*/*"
}

resource "aws_dynamodb_table" "shop_table" {
  name         = "shop-table"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "PK"
  range_key    = "SK"

  attribute {
    name = "PK"
    type = "S"
  }

  attribute {
    name = "SK"
    type = "S"
  }

  tags = {
    Name = "shop-table"
  }
}

resource "aws_s3_bucket" "file_storage" {
  bucket = "shop-file-storage-${data.aws_caller_identity.current.account_id}"

  tags = {
    Name = "shop-file-storage"
  }
}

resource "aws_s3_bucket_public_access_block" "file_storage" {
  bucket = aws_s3_bucket.file_storage.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

data "aws_caller_identity" "current" {}
