locals {
  healthcheck_responder_name = "${var.name}-healthcheck-responder"
}

data "aws_s3_object" "healthcheck_responder" {
  bucket = local.code_bucket
  key    = "${var.code.object_prefix}rust/target/lambda/process-healthcheck/bootstrap.zip"
}

resource "aws_lambda_function" "healthcheck_responder" {
  function_name     = local.healthcheck_responder_name
  role              = var.functions.healthcheck_responder.role
  s3_bucket         = data.aws_s3_object.healthcheck_responder.bucket
  s3_key            = data.aws_s3_object.healthcheck_responder.key
  s3_object_version = data.aws_s3_object.healthcheck_responder.version_id
  description       = "Responds to async healthchecks"
  handler           = "bootstrap"
  runtime           = "provided.al2"
  architectures     = ["arm64"]
  timeout           = 5
  memory_size       = 128
  tags              = local.tags

  environment {
    variables = {
      API_URL = replace(var.functions.healthcheck_responder.api_url, "<region>", data.aws_region.current.name),
    }
  }

  depends_on = [aws_cloudwatch_log_group.healthcheck_responder]
}

resource "aws_cloudwatch_log_group" "healthcheck_responder" {
  name              = "/aws/lambda/${local.healthcheck_responder_name}"
  retention_in_days = 7
  tags              = local.tags
}