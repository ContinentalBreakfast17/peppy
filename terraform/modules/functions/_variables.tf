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

    queue_processer_unranked_solo = object({
      role         = string
      source_hash  = string
      source_file  = string
      match_table  = string
      queue_table  = string
      queue_index  = string
      lock_table   = string
      lock_regions = list(string)
    })

    match_publisher = object({
      role        = string
      source_hash = string
      source_file = string
      api_url     = string
    })
  })
}