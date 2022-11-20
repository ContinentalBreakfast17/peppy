module "appsync_us_east_1" {
  source    = "./modules/appsync"
  name      = var.name
  role      = aws_iam_role.appsync.arn
  log_level = "ALL"

  paths = {
    schema = "${path.root}/../schema"
    vtl    = "${path.root}/../vtl"
  }

  dns = {
    domain_name = local.domain_name
    zone_id     = data.aws_route53_zone.domain_name.zone_id
    cert        = aws_acm_certificate.api_wildcard.arn
  }

  alarms = {
    cron = module.alarms_us_east_1.healthcheck_failures.errors_fail.alarm_name
  }

  tables = {
    healthcheck         = aws_dynamodb_table.healthcheck.id
    ip_cache            = aws_dynamodb_table.ip_cache.id
    queue_unranked_solo = aws_dynamodb_table.queue_unranked_solo.id
    mmr_unranked_solo   = aws_dynamodb_table.mmr_unranked_solo.id
  }

  functions = {
    ip_lookup = module.functions_us_east_1.ip_lookup.arn
  }
}

module "appsync_us_east_2" {
  source    = "./modules/appsync"
  providers = { aws = aws.us_east_2 }
  name      = var.name
  role      = aws_iam_role.appsync.arn
  log_level = "ALL"

  paths = {
    schema = "${path.root}/../schema"
    vtl    = "${path.root}/../vtl"
  }

  dns = {
    domain_name = local.domain_name
    zone_id     = data.aws_route53_zone.domain_name.zone_id
    cert        = aws_acm_certificate.api_wildcard.arn
  }

  alarms = {
    cron = module.alarms_us_east_2.healthcheck_failures.errors_fail.alarm_name
  }

  tables = {
    healthcheck         = aws_dynamodb_table.healthcheck.id
    ip_cache            = aws_dynamodb_table.ip_cache.id
    queue_unranked_solo = aws_dynamodb_table.queue_unranked_solo.id
    mmr_unranked_solo   = aws_dynamodb_table.mmr_unranked_solo.id
  }

  functions = {
    ip_lookup = module.functions_us_east_2.ip_lookup.arn
  }
}