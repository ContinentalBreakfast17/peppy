variable "name" {
  description = "Name prefix for all resources"
  type        = string
}

variable "healthcheck" {
  description = "Healthcheck configuration"
  type = object({
    function_name = string
  })
}