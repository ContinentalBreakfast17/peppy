data "archive_file" "fn_ip_lookup" {
  type        = "zip"
  source_dir  = "${path.root}/../functions/ip-lookup/target/lambda/ip-lookup"
  output_path = "${path.root}/../functions/ip-lookup/target/lambda/ip-lookup/code.zip"
  excludes    = ["*.zip"]
}

data "archive_file" "fn_queue_processer_unranked_solo" {
  type        = "zip"
  source_dir  = "${path.root}/../functions/test"
  output_path = "${path.root}/../functions/test/code.zip"
  excludes    = ["*.zip"]
}

module "functions_us_east_1" {
  source = "./modules/functions"
  name   = var.name

  functions = {
    ip_lookup = {
      role        = aws_iam_role.ip_lookup.arn
      source_file = data.archive_file.fn_ip_lookup.output_path
      source_hash = data.archive_file.fn_ip_lookup.output_base64sha256
      secret_arn  = aws_secretsmanager_secret.ip_lookup_token.arn
    }

    queue_processer_unranked_solo = {
      role        = aws_iam_role.queue_processer_unranked_solo.arn
      source_file = data.archive_file.fn_queue_processer_unranked_solo.output_path
      source_hash = data.archive_file.fn_queue_processer_unranked_solo.output_base64sha256
      table       = aws_dynamodb_table.queue_unranked_solo.id
      index       = local.dynamo_indexes.queue_sort
    }
  }
}

module "functions_us_east_2" {
  source    = "./modules/functions"
  providers = { aws = aws.us_east_2 }
  name      = var.name

  functions = {
    ip_lookup = {
      role        = aws_iam_role.ip_lookup.arn
      source_file = data.archive_file.fn_ip_lookup.output_path
      source_hash = data.archive_file.fn_ip_lookup.output_base64sha256
      secret_arn  = aws_secretsmanager_secret.ip_lookup_token.arn
    }

    queue_processer_unranked_solo = {
      role        = aws_iam_role.queue_processer_unranked_solo.arn
      source_file = data.archive_file.fn_queue_processer_unranked_solo.output_path
      source_hash = data.archive_file.fn_queue_processer_unranked_solo.output_base64sha256
      table       = aws_dynamodb_table.queue_unranked_solo.id
      index       = local.dynamo_indexes.queue_sort
    }
  }
}