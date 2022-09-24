data "archive_file" "fn_ip_lookup" {
  type        = "zip"
  source_dir  = "${path.root}/../functions/ip-lookup/target/lambda/ip-lookup"
  output_path = "${path.root}/../functions/ip-lookup/target/lambda/ip-lookup/code.zip"
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
  }
}