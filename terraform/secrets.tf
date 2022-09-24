resource "aws_secretsmanager_secret" "ip_lookup_token" {
  name_prefix = "${var.name}-ip-lookup-token"
  description = "API token for ip lookup service"
  kms_key_id  = aws_kms_key.main.arn

  replica {
    region     = module.main_key_replica_us_east_2.region
    kms_key_id = module.main_key_replica_us_east_2.key.arn
  }
}