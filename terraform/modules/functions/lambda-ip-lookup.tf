locals {
  ip_lookup_name = "${var.name}-ip-lookup"
}

resource "aws_lambda_function" "ip_lookup" {
  function_name    = local.ip_lookup_name
  role             = var.functions.ip_lookup.role
  source_code_hash = var.functions.ip_lookup.source_hash
  filename         = var.functions.ip_lookup.source_file
  description      = "Gets geolocation data from an ip address"
  handler          = "bootstrap"
  runtime          = "provided.al2"
  architectures    = ["arm64"]
  timeout          = 3
  memory_size      = 128
  tags             = local.tags

  environment {
    variables = {
      SECRET_ARN = replace(var.functions.ip_lookup.secret_arn, "us-east-1", data.aws_region.current.name),
    }
  }

  depends_on = [aws_cloudwatch_log_group.ip_lookup]
}

resource "aws_cloudwatch_log_group" "ip_lookup" {
  name              = "/aws/lambda/${local.ip_lookup_name}"
  retention_in_days = 7
  tags              = local.tags
}