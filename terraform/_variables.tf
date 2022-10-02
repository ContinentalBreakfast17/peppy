variable "name" {
  description = "Name prefix for all resources"
  type        = string
}

variable "enable_queue_processing" {
  description = "Whether or not to enable matchmaking queue processing"
  type        = bool
  default     = true
}