locals {
  dynamo_indexes = {
    queue_sort = "queue_sort",
  }

  dynamo_filters = {
    queue_process = {
      eventName = ["MODIFY", "INSERT"]
    }
  }
}

locals {
  domain_name  = "${var.subdomain}.${var.domain_name}"
  regional_url = "https://<region>.${local.domain_name}/graphql"
}