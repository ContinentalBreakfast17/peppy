resource "aws_dynamodb_table" "healthcheck" {
  name             = "${var.name}-healthcheck"
  billing_mode     = "PAY_PER_REQUEST"
  table_class      = "STANDARD"
  hash_key         = "region"
  range_key        = "id"
  stream_enabled   = true
  stream_view_type = "NEW_AND_OLD_IMAGES"

  attribute {
    name = "region"
    type = "S"
  }

  attribute {
    name = "id"
    type = "S"
  }

  ttl {
    attribute_name = "ttl"
    enabled        = true
  }

  server_side_encryption {
    enabled     = true
    kms_key_arn = aws_kms_key.main.arn
  }

  replica {
    region_name    = module.main_key_replica_us_east_2.region
    kms_key_arn    = module.main_key_replica_us_east_2.key.arn
    propagate_tags = true
  }
}

data "aws_dynamodb_table" "healthcheck_us_east_2" {
  provider = aws.us_east_2
  name     = aws_dynamodb_table.healthcheck.id
}

resource "aws_lambda_event_source_mapping" "healthcheck_us_east_1" {
  event_source_arn       = aws_dynamodb_table.healthcheck.stream_arn
  function_name          = module.functions_us_east_1.healthcheck_responder.arn
  starting_position      = "LATEST"
  maximum_retry_attempts = 2

  filter_criteria {
    filter {
      pattern = jsonencode(local.dynamo_filters.healthcheck)
    }
  }
}

resource "aws_lambda_event_source_mapping" "healthcheck_us_east_2" {
  provider               = aws.us_east_2
  event_source_arn       = data.aws_dynamodb_table.healthcheck_us_east_2.stream_arn
  function_name          = module.functions_us_east_2.healthcheck_responder.arn
  starting_position      = "LATEST"
  maximum_retry_attempts = 2

  filter_criteria {
    filter {
      pattern = jsonencode(local.dynamo_filters.healthcheck)
    }
  }
}