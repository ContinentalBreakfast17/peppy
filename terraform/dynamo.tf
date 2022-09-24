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
  hash_key         = "version"
  range_key        = "user"
  stream_enabled   = true
  stream_view_type = "NEW_AND_OLD_IMAGES"

  attribute {
    name = "version"
    type = "S"
  }

  attribute {
    name = "user"
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
  name             = "${var.name}-mmr-unraked-solo"
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