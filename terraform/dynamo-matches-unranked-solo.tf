resource "aws_dynamodb_table" "match_publisher" {
  name             = "${var.name}-matches-unranked-solo"
  billing_mode     = "PAY_PER_REQUEST"
  table_class      = "STANDARD"
  hash_key         = "match"
  stream_enabled   = true
  stream_view_type = "NEW_AND_OLD_IMAGES"

  attribute {
    name = "match"
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

data "aws_dynamodb_table" "match_publisher_us_east_2" {
  provider = aws.us_east_2
  name     = aws_dynamodb_table.match_publisher.id
}

resource "aws_lambda_event_source_mapping" "match_publisher_us_east_1" {
  enabled                            = var.process_toggles.match_publisher
  event_source_arn                   = aws_dynamodb_table.match_publisher.stream_arn
  function_name                      = module.functions_us_east_1.match_publisher.arn
  starting_position                  = "LATEST"
  maximum_batching_window_in_seconds = 2
  maximum_retry_attempts             = 6

  # todo: swap filter for something that enforces at least one regional item?
  filter_criteria {
    filter {
      pattern = jsonencode(local.dynamo_filters.queue_process)
    }
  }
}

resource "aws_lambda_event_source_mapping" "match_publisher_us_east_2" {
  provider                           = aws.us_east_2
  enabled                            = var.process_toggles.match_publisher
  event_source_arn                   = data.aws_dynamodb_table.match_publisher_us_east_2.stream_arn
  function_name                      = module.functions_us_east_2.match_publisher.arn
  starting_position                  = "LATEST"
  maximum_batching_window_in_seconds = 2
  maximum_retry_attempts             = 6

  filter_criteria {
    filter {
      pattern = jsonencode(local.dynamo_filters.queue_process)
    }
  }
}