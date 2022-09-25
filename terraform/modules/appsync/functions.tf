resource "aws_appsync_function" "cache_ip" {
  api_id                    = aws_appsync_graphql_api.this.id
  data_source               = aws_appsync_datasource.ip_cache.name
  name                      = "cache_ip"
  request_mapping_template  = file("${var.paths.vtl}/cache-ip.req.vm")
  response_mapping_template = file("${var.paths.vtl}/cache-ip.resp.vm")
}

resource "aws_appsync_function" "check_ip_cache" {
  api_id                    = aws_appsync_graphql_api.this.id
  data_source               = aws_appsync_datasource.ip_cache.name
  name                      = "check_ip_cache"
  request_mapping_template  = file("${var.paths.vtl}/check-ip-cache.req.vm")
  response_mapping_template = file("${var.paths.vtl}/check-ip-cache.resp.vm")
}

resource "aws_appsync_function" "enqueue_unranked_solo" {
  api_id      = aws_appsync_graphql_api.this.id
  data_source = aws_appsync_datasource.queue_unranked_solo.name
  name        = "enqueue_unranked_solo"
  request_mapping_template = join("\n", [
    "$util.quiet($ctx.stash.put(\"dequeue_tables\", []))",
    "$util.quiet($ctx.stash.put(\"queue_table\", \"${var.tables.queue_unranked_solo}\"))",
    file("${var.paths.vtl}/enqueue.req.vm"),
  ])
  response_mapping_template = file("${var.paths.vtl}/enqueue.resp.vm")
}

resource "aws_appsync_function" "get_mmr_unranked_solo" {
  api_id                    = aws_appsync_graphql_api.this.id
  data_source               = aws_appsync_datasource.mmr_unranked_solo.name
  name                      = "get_mmr_unranked_solo"
  request_mapping_template  = file("${var.paths.vtl}/get-mmr.req.vm")
  response_mapping_template = file("${var.paths.vtl}/get-mmr.resp.vm")
}

resource "aws_appsync_function" "lookup_ip" {
  api_id                    = aws_appsync_graphql_api.this.id
  data_source               = aws_appsync_datasource.ip_lookup.name
  name                      = "lookup_ip"
  request_mapping_template  = file("${var.paths.vtl}/lookup-ip.req.vm")
  response_mapping_template = file("${var.paths.vtl}/lookup-ip.resp.vm")
}

resource "aws_appsync_function" "match_unranked_solo" {
  api_id                    = aws_appsync_graphql_api.this.id
  data_source               = aws_appsync_datasource.noop.name
  name                      = "match_unranked_solo"
  request_mapping_template  = file("${var.paths.vtl}/match-single.req.vm")
  response_mapping_template = file("${var.paths.vtl}/match-single.resp.vm")
}

resource "aws_appsync_function" "publish_enqueue" {
  api_id      = aws_appsync_graphql_api.this.id
  data_source = aws_appsync_datasource.events.name
  name        = "publish_enqueue"
  request_mapping_template = join("\n", [
    "$util.quiet($ctx.stash.put(\"event_bus\", \"${var.event_bus}\"))",
    file("${var.paths.vtl}/publish-enqueue.req.vm"),
  ])
  response_mapping_template = file("${var.paths.vtl}/publish-enqueue.resp.vm")
}