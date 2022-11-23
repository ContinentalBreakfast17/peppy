locals {
  ip_lookup_name = "${var.name}-ip-lookup"
}

data "aws_s3_object" "ip_lookup" {
  bucket = local.code_bucket
  key    = "${var.code.object_prefix}rust/target/lambda/ip-lookup/bootstrap.zip"
}

resource "aws_lambda_function" "ip_lookup" {
  function_name     = local.ip_lookup_name
  role              = var.functions.ip_lookup.role
  s3_bucket         = data.aws_s3_object.ip_lookup.bucket
  s3_key            = data.aws_s3_object.ip_lookup.key
  s3_object_version = data.aws_s3_object.ip_lookup.version_id
  description       = "Gets geolocation data from an ip address"
  handler           = "bootstrap"
  runtime           = "provided.al2"
  architectures     = ["arm64"]
  timeout           = 3
  memory_size       = 128
  tags              = local.tags

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