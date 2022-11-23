locals {
  healthcheck_name = "${var.name}-healthcheck"
}

data "aws_s3_object" "healthcheck" {
  bucket = local.code_bucket
  key    = "${var.code.object_prefix}rust/target/lambda/healthcheck/bootstrap.zip"
}

resource "aws_lambda_function" "healthcheck" {
  function_name     = local.healthcheck_name
  role              = var.functions.healthcheck.role
  s3_bucket         = data.aws_s3_object.healthcheck.bucket
  s3_key            = data.aws_s3_object.healthcheck.key
  s3_object_version = data.aws_s3_object.healthcheck.version_id
  description       = "Performs a healthcheck on the API"
  handler           = "bootstrap"
  runtime           = "provided.al2"
  architectures     = ["arm64"]
  timeout           = 20
  memory_size       = 128
  tags              = local.tags

  environment {
    variables = {
      TABLE   = var.functions.healthcheck.table,
      API_URL = replace(var.functions.healthcheck.api_url, "<region>", data.aws_region.current.name),
    }
  }

  depends_on = [aws_cloudwatch_log_group.healthcheck]
}

resource "aws_cloudwatch_log_group" "healthcheck" {
  name              = "/aws/lambda/${local.healthcheck_name}"
  retention_in_days = 7
  tags              = local.tags
}