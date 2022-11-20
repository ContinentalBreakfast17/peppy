module "alarms_us_east_1" {
  source         = "./modules/alarms"
  name           = var.name
  kms_key_id     = aws_kms_key.main.key_id
  send_alarms_to = var.send_alarms_to

  healthcheck = {
    function_name = module.functions_us_east_1.healthcheck.name
  }
}

module "alarms_us_east_2" {
  source         = "./modules/alarms"
  providers      = { aws = aws.us_east_2 }
  name           = var.name
  kms_key_id     = module.main_key_replica_us_east_2.key.id
  send_alarms_to = var.send_alarms_to

  healthcheck = {
    function_name = module.functions_us_east_1.healthcheck.name
  }
}