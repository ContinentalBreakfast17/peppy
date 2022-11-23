module "functions_us_east_1" {
  source = "./modules/functions"
  name   = var.name
  code   = var.function_code

  functions = {
    healthcheck = {
      role    = aws_iam_role.healthcheck.arn
      api_url = local.regional_url
      table   = aws_dynamodb_table.healthcheck.id
    }

    healthcheck_responder = {
      role    = aws_iam_role.healthcheck_responder.arn
      api_url = local.regional_url
    }

    ip_lookup = {
      role       = aws_iam_role.ip_lookup.arn
      secret_arn = aws_secretsmanager_secret.ip_lookup_token.arn
    }

    match_publisher = {
      role    = aws_iam_role.match_publisher.arn
      api_url = local.regional_url
    }

    queue_processer_unranked_solo = {
      role         = aws_iam_role.queue_processer_unranked_solo.arn
      queue_index  = local.dynamo_indexes.queue_sort
      queue_table  = aws_dynamodb_table.queue_unranked_solo.id
      match_table  = aws_dynamodb_table.match_publisher.id
      lock_table   = aws_dynamodb_table.process_lock.id
      lock_regions = local.lock_regions
    }
  }
}

module "functions_us_east_2" {
  source    = "./modules/functions"
  providers = { aws = aws.us_east_2 }
  name      = var.name
  code      = var.function_code

  functions = {
    healthcheck = {
      role    = aws_iam_role.healthcheck.arn
      api_url = local.regional_url
      table   = aws_dynamodb_table.healthcheck.id
    }

    healthcheck_responder = {
      role    = aws_iam_role.healthcheck_responder.arn
      api_url = local.regional_url
    }

    ip_lookup = {
      role       = aws_iam_role.ip_lookup.arn
      secret_arn = aws_secretsmanager_secret.ip_lookup_token.arn
    }

    match_publisher = {
      role    = aws_iam_role.match_publisher.arn
      api_url = local.regional_url
    }

    queue_processer_unranked_solo = {
      role         = aws_iam_role.queue_processer_unranked_solo.arn
      queue_index  = local.dynamo_indexes.queue_sort
      queue_table  = aws_dynamodb_table.queue_unranked_solo.id
      match_table  = aws_dynamodb_table.match_publisher.id
      lock_table   = aws_dynamodb_table.process_lock.id
      lock_regions = local.lock_regions
    }
  }
}