locals {
  match_publisher_name = "${var.name}-match-publisher"
}

resource "aws_lambda_function" "match_publisher" {
  function_name    = local.match_publisher_name
  role             = var.functions.match_publisher.role
  source_code_hash = var.functions.match_publisher.source_hash
  filename         = var.functions.match_publisher.source_file
  description      = "Notifies players of matches"
  handler          = "bootstrap"
  runtime          = "provided.al2"
  architectures    = ["arm64"]
  timeout          = 15
  memory_size      = 128
  tags             = local.tags

  environment {
    variables = {
      API_URL = replace(var.functions.match_publisher.api_url, "<region>", data.aws_region.current.name),
    }
  }

  depends_on = [aws_cloudwatch_log_group.match_publisher]
}

resource "aws_cloudwatch_log_group" "match_publisher" {
  name              = "/aws/lambda/${local.match_publisher_name}"
  retention_in_days = 7
  tags              = local.tags
}