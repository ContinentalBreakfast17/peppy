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