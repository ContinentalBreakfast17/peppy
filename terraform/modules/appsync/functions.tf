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

resource "aws_appsync_function" "enqueue" {
  api_id                    = aws_appsync_graphql_api.this.id
  data_source               = aws_appsync_datasource.queue_unranked_solo.name
  name                      = "enqueue"
  request_mapping_template  = file("${var.paths.vtl}/enqueue.req.vm")
  response_mapping_template = file("${var.paths.vtl}/enqueue.resp.vm")
}

resource "aws_appsync_function" "get_mmr_unranked_solo" {
  api_id                    = aws_appsync_graphql_api.this.id
  data_source               = aws_appsync_datasource.mmr_unranked_solo.name
  name                      = "get_mmr_unranked_solo"
  request_mapping_template  = file("${var.paths.vtl}/get-mmr.req.vm")
  response_mapping_template = file("${var.paths.vtl}/get-mmr.resp.vm")
}

resource "aws_appsync_function" "health_response" {
  api_id                    = aws_appsync_graphql_api.this.id
  data_source               = aws_appsync_datasource.healthcheck.name
  name                      = "health_response"
  request_mapping_template  = file("${var.paths.vtl}/healthcheck-response.req.vm")
  response_mapping_template = file("${var.paths.vtl}/healthcheck-response.resp.vm")
}

resource "aws_appsync_function" "lookup_ip" {
  api_id                    = aws_appsync_graphql_api.this.id
  data_source               = aws_appsync_datasource.ip_lookup.name
  name                      = "lookup_ip"
  request_mapping_template  = file("${var.paths.vtl}/lookup-ip.req.vm")
  response_mapping_template = file("${var.paths.vtl}/lookup-ip.resp.vm")
}

resource "aws_appsync_function" "post_healthcheck" {
  api_id                    = aws_appsync_graphql_api.this.id
  data_source               = aws_appsync_datasource.healthcheck.name
  name                      = "post_healthcheck"
  request_mapping_template  = file("${var.paths.vtl}/post-healthcheck.req.vm")
  response_mapping_template = file("${var.paths.vtl}/post-healthcheck.resp.vm")
}

resource "aws_appsync_function" "publish_match" {
  api_id                    = aws_appsync_graphql_api.this.id
  data_source               = aws_appsync_datasource.noop.name
  name                      = "publish_match"
  request_mapping_template  = file("${var.paths.vtl}/match.req.vm")
  response_mapping_template = file("${var.paths.vtl}/match.resp.vm")
}