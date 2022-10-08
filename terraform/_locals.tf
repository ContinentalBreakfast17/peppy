locals {
  dynamo_indexes = {
    queue_sort = "queue_sort",
  }

  dynamo_filters = {
    # todo: check region match
    queue_process = {
      eventName = ["MODIFY", "INSERT"]
    }

    # todo: check region match
    publish_match = {
      eventName = ["MODIFY", "INSERT"]
    }

    # todo: check region match
    healthcheck = {
      eventName = ["INSERT"]
    }
  }
}

locals {
  domain_name  = "${var.subdomain}.${var.domain_name}"
  regional_url = "https://<region>.${local.domain_name}/graphql"
}