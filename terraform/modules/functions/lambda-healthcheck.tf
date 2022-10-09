locals {
  healthcheck_name = "${var.name}-healthcheck"
}

resource "aws_lambda_function" "healthcheck" {
  function_name    = local.healthcheck_name
  role             = var.functions.healthcheck.role
  source_code_hash = var.functions.healthcheck.source_hash
  filename         = var.functions.healthcheck.source_file
  description      = "Performs a healthcheck on the API"
  handler          = "bootstrap"
  runtime          = "provided.al2"
  architectures    = ["arm64"]
  timeout          = 20
  memory_size      = 128
  tags             = local.tags

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