variable "name" {
  description = "Name prefix for all resources"
  type        = string
}

variable "role" {
  description = "Role used by appsync to access datasources/logs/whatever else"
  type        = string
}

variable "log_level" {
  description = "Appsync log level (ERROR, ALL, NONE)"
  type        = string
  default     = "ALL"
}

variable "paths" {
  description = "Filesystem locations"
  type = object({
    schema = string
    vtl    = string
  })
}

variable "event_bus" {
  description = "Event bus used to process events"
  type        = string
}

variable "tables" {
  description = "Table names"
  type = object({
    ip_cache            = string
    queue_unranked_solo = string
    mmr_unranked_solo   = string
  })
}

variable "functions" {
  description = "Function arns"
  type = object({
    ip_lookup = string
  })
}