module "alarms_us_east_1" {
  source = "./modules/alarms"
  name   = var.name

  healthcheck = {
    function_name = module.functions_us_east_1.healthcheck.name
  }
}

module "alarms_us_east_2" {
  source    = "./modules/alarms"
  providers = { aws = aws.us_east_2 }
  name      = var.name

  healthcheck = {
    function_name = module.functions_us_east_1.healthcheck.name
  }
}