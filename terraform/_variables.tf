variable "name" {
  description = "Name prefix for all resources"
  type        = string
}

variable "enable_queue_processing" {
  description = "Whether or not to enable matchmaking queue processing"
  type        = bool
  default     = true
}

variable "process_toggles" {
  description = "Can be used to shutoff certain stream processers"
  type        = object({
    match_publisher = bool
    queue_unranked_solo = bool
  })
  default = {
    match_publisher = true
    queue_unranked_solo = true
  }
}