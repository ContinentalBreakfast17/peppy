variable "name" {
  description = "Name prefix for all resources"
  type        = string
}

variable "functions" {
  description = "Function configuration"
  type = object({
    ip_lookup = object({
      role        = string
      source_hash = string
      source_file = string
      secret_arn  = string
    })
  })
}