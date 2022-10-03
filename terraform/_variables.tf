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