variable "name" {
  description = "Name prefix for all resources"
  type        = string
}

variable "cron" {
  description = "Cron configuration"
  type = object({
    healthcheck = object({
      schedule = string
      enabled  = bool
      lambda   = string
    })
  })
}