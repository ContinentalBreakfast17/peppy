locals {
  healthcheck_responder_name = "${var.name}-healthcheck-responder"
}

resource "aws_lambda_function" "healthcheck_responder" {
  function_name    = local.healthcheck_responder_name
  role             = var.functions.healthcheck_responder.role
  source_code_hash = var.functions.healthcheck_responder.source_hash
  filename         = var.functions.healthcheck_responder.source_file
  description      = "Responds to async healthchecks"
  handler          = "bootstrap"
  runtime          = "provided.al2"
  architectures    = ["arm64"]
  timeout          = 5
  memory_size      = 128
  tags             = local.tags

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