variable "name" {
  description = "Name prefix for all resources"
  type        = string
}

variable "code" {
  description = "Bucket details for where lambda code is stored"
  type = object({
    bucket_prefix = string
    object_prefix = string
  })
}

variable "functions" {
  description = "Function configuration"
  type = object({
    healthcheck = object({
      role    = string
      api_url = string
      table   = string
    })

    healthcheck_responder = object({
      role    = string
      api_url = string
    })

    ip_lookup = object({
      role       = string
      secret_arn = string
    })

    queue_processer_unranked_solo = object({
      role         = string
      match_table  = string
      queue_table  = string
      queue_index  = string
      lock_table   = string
      lock_regions = list(string)
    })

    match_publisher = object({
      role    = string
      api_url = string
    })
  })
}