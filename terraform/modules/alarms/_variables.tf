variable "name" {
  description = "Name prefix for all resources"
  type        = string
}

variable "send_alarms_to" {
  description = "List of email addresses to send alarms to"
  type        = list(string)
}

variable "kms_key_id" {
  description = "KMS key id for sns topic encryption"
  type        = string
}

variable "healthcheck" {
  description = "Healthcheck configuration"
  type = object({
    function_name = string
  })
}