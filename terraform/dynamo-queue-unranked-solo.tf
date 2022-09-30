resource "aws_dynamodb_table" "queue_unranked_solo" {
  name             = "${var.name}-queue-unraked-solo"
  billing_mode     = "PAY_PER_REQUEST"
  table_class      = "STANDARD"
  hash_key         = "user"
  stream_enabled   = true
  stream_view_type = "NEW_AND_OLD_IMAGES"

  global_secondary_index {
    name            = local.dynamo_indexes.queue_sort
    hash_key        = "queue"
    range_key       = "join_time"
    projection_type = "ALL"
  }

  attribute {
    name = "user"
    type = "S"
  }

  attribute {
    name = "queue"
    type = "S"
  }

  attribute {
    name = "join_time"
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

data "aws_dynamodb_table" "queue_unranked_solo_us_east_2" {
  provider = aws.us_east_2
  name     = aws_dynamodb_table.queue_unranked_solo.id
}

resource "aws_lambda_event_source_mapping" "queue_processer_unranked_solo_us_east_1" {
  event_source_arn  = aws_dynamodb_table.queue_unranked_solo.stream_arn
  function_name     = module.functions_us_east_1.queue_processer_unranked_solo.arn
  starting_position = "LATEST"
}

resource "aws_lambda_event_source_mapping" "queue_processer_unranked_solo_us_east_2" {
  provider          = aws.us_east_2
  event_source_arn  = data.aws_dynamodb_table.queue_unranked_solo_us_east_2.stream_arn
  function_name     = module.functions_us_east_2.queue_processer_unranked_solo.arn
  starting_position = "LATEST"
}