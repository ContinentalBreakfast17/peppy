module "eventbridge_us_east_1" {
  source = "./modules/eventbridge"
  name   = var.name

  cron = {
    healthcheck = {
      schedule = "rate(1 minute)"
      enabled  = true
      lambda   = module.functions_us_east_1.healthcheck.arn
    }
  }
}

module "eventbridge_us_east_2" {
  source    = "./modules/eventbridge"
  providers = { aws = aws.us_east_2 }
  name      = var.name

  cron = {
    healthcheck = {
      schedule = "rate(1 minute)"
      enabled  = true
      lambda   = module.functions_us_east_2.healthcheck.arn
    }
  }
}