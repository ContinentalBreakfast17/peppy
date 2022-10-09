data "archive_file" "fn_healthcheck" {
  type        = "zip"
  source_dir  = "${path.root}/../functions/healthcheck/target/lambda/healthcheck"
  output_path = "${path.root}/../functions/healthcheck/target/lambda/healthcheck/code.zip"
  excludes    = ["*.zip"]
}

data "archive_file" "fn_healthcheck_responder" {
  type        = "zip"
  source_dir  = "${path.root}/../functions/process-healthcheck/target/lambda/process-healthcheck"
  output_path = "${path.root}/../functions/process-healthcheck/target/lambda/process-healthcheck/code.zip"
  excludes    = ["*.zip"]
}

data "archive_file" "fn_ip_lookup" {
  type        = "zip"
  source_dir  = "${path.root}/../functions/ip-lookup/target/lambda/ip-lookup"
  output_path = "${path.root}/../functions/ip-lookup/target/lambda/ip-lookup/code.zip"
  excludes    = ["*.zip"]
}


data "archive_file" "fn_match_publisher" {
  type        = "zip"
  source_dir  = "${path.root}/../functions/process-match/target/lambda/process-match"
  output_path = "${path.root}/../functions/process-match/target/lambda/process-match/code.zip"
  excludes    = ["*.zip"]
}


data "archive_file" "fn_queue_processer_unranked_solo" {
  type        = "zip"
  source_dir  = "${path.root}/../functions/process-queue/target/lambda/unranked-solo"
  output_path = "${path.root}/../functions/process-queue/target/lambda/unranked-solo/code.zip"
  excludes    = ["*.zip"]
}

module "functions_us_east_1" {
  source = "./modules/functions"
  name   = var.name

  functions = {
    healthcheck = {
      role        = aws_iam_role.healthcheck.arn
      source_file = data.archive_file.fn_healthcheck.output_path
      source_hash = data.archive_file.fn_healthcheck.output_base64sha256
      api_url     = local.regional_url
      table       = aws_dynamodb_table.healthcheck.id
    }

    healthcheck_responder = {
      role        = aws_iam_role.healthcheck_responder.arn
      source_file = data.archive_file.fn_healthcheck_responder.output_path
      source_hash = data.archive_file.fn_healthcheck_responder.output_base64sha256
      api_url     = local.regional_url
    }

    ip_lookup = {
      role        = aws_iam_role.ip_lookup.arn
      source_file = data.archive_file.fn_ip_lookup.output_path
      source_hash = data.archive_file.fn_ip_lookup.output_base64sha256
      secret_arn  = aws_secretsmanager_secret.ip_lookup_token.arn
    }

    match_publisher = {
      role        = aws_iam_role.match_publisher.arn
      source_file = data.archive_file.fn_match_publisher.output_path
      source_hash = data.archive_file.fn_match_publisher.output_base64sha256
      api_url     = local.regional_url
    }

    queue_processer_unranked_solo = {
      role         = aws_iam_role.queue_processer_unranked_solo.arn
      source_file  = data.archive_file.fn_queue_processer_unranked_solo.output_path
      source_hash  = data.archive_file.fn_queue_processer_unranked_solo.output_base64sha256
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

  functions = {
    healthcheck = {
      role        = aws_iam_role.healthcheck.arn
      source_file = data.archive_file.fn_healthcheck.output_path
      source_hash = data.archive_file.fn_healthcheck.output_base64sha256
      api_url     = local.regional_url
      table       = aws_dynamodb_table.healthcheck.id
    }

    healthcheck_responder = {
      role        = aws_iam_role.healthcheck_responder.arn
      source_file = data.archive_file.fn_healthcheck_responder.output_path
      source_hash = data.archive_file.fn_healthcheck_responder.output_base64sha256
      api_url     = local.regional_url
    }

    ip_lookup = {
      role        = aws_iam_role.ip_lookup.arn
      source_file = data.archive_file.fn_ip_lookup.output_path
      source_hash = data.archive_file.fn_ip_lookup.output_base64sha256
      secret_arn  = aws_secretsmanager_secret.ip_lookup_token.arn
    }

    match_publisher = {
      role        = aws_iam_role.match_publisher.arn
      source_file = data.archive_file.fn_match_publisher.output_path
      source_hash = data.archive_file.fn_match_publisher.output_base64sha256
      api_url     = local.regional_url
    }

    queue_processer_unranked_solo = {
      role         = aws_iam_role.queue_processer_unranked_solo.arn
      source_file  = data.archive_file.fn_queue_processer_unranked_solo.output_path
      source_hash  = data.archive_file.fn_queue_processer_unranked_solo.output_base64sha256
      queue_index  = local.dynamo_indexes.queue_sort
      queue_table  = aws_dynamodb_table.queue_unranked_solo.id
      match_table  = aws_dynamodb_table.match_publisher.id
      lock_table   = aws_dynamodb_table.process_lock.id
      lock_regions = local.lock_regions
    }
  }
}