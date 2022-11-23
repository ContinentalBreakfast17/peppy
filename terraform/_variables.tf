variable "name" {
  description = "Name prefix for all resources"
  type        = string
}

variable "domain_name" {
  description = "Name of public route 53 zone used for the API"
  type        = string
}

variable "subdomain" {
  description = "Subdomain for the API"
  type        = string
  default     = "slippi"
}

variable "send_alarms_to" {
  description = "List of email addresses to send alarms to"
  type        = list(string)
}

variable "function_code" {
  description = "Bucket details for where lambda code is stored"
  type = object({
    bucket_prefix = string
    object_prefix = string
  })
}

variable "enable_queue_processing" {
  description = "Whether or not to enable matchmaking queue processing"
  type        = bool
  default     = true
}

variable "process_toggles" {
  description = "Can be used to shutoff certain stream processers"
  type = object({
    match_publisher     = bool
    queue_unranked_solo = bool
  })
  default = {
    match_publisher     = true
    queue_unranked_solo = true
  }
}