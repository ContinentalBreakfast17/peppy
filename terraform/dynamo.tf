locals {
  dynamo_indexes = {
    queue_sort = "queue_sort",
  }
}

resource "aws_dynamodb_table" "ip_cache" {
  name             = "${var.name}-ip-cache"
  billing_mode     = "PAY_PER_REQUEST"
  table_class      = "STANDARD"
  hash_key         = "ip"
  stream_enabled   = true
  stream_view_type = "NEW_AND_OLD_IMAGES"

  attribute {
    name = "ip"
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

resource "aws_dynamodb_table" "mmr_unranked_solo" {
  name             = "${var.name}-mmr-unranked-solo"
  billing_mode     = "PAY_PER_REQUEST"
  table_class      = "STANDARD"
  hash_key         = "user"
  stream_enabled   = true
  stream_view_type = "NEW_AND_OLD_IMAGES"

  attribute {
    name = "user"
    type = "S"
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