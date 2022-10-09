locals {
  global_request_template  = <<EOT
$util.quiet($ctx.stash.put("entry_time", $util.time.nowEpochSeconds()))
$util.quiet($ctx.stash.put("graphql", $ctx.info))
$util.quiet($ctx.stash.put("region", "${data.aws_region.current.name}"))
{}
EOT
  global_response_template = "$util.toJson($ctx.result)"
}

# not worth unit tests
resource "aws_appsync_resolver" "Query_region" {
  api_id      = aws_appsync_graphql_api.this.id
  data_source = aws_appsync_datasource.noop.name
  type        = "Query"
  field       = "region"

  request_template  = jsonencode({ version = "2017-02-28", payload = {} })
  response_template = jsonencode({ region = data.aws_region.current.name })
}

resource "aws_appsync_resolver" "Subscription_healthcheck" {
  api_id = aws_appsync_graphql_api.this.id
  kind   = "PIPELINE"
  type   = "Subscription"
  field  = "healthcheck"

  request_template = join("\n", [
    # using an empty ip will get info about the requesting lambda's ip
    "$util.quiet($ctx.stash.put(\"ip\", \"\"))",
    "$util.quiet($ctx.stash.put(\"healthcheck_table\", \"${var.tables.healthcheck}\"))",
    local.global_request_template,
  ])
  response_template = "#return"

  pipeline_config {
    functions = [
      # make sure the region can look up ip addresses first
      aws_appsync_function.lookup_ip.function_id,
      aws_appsync_function.post_healthcheck.function_id,
    ]
  }
}

resource "aws_appsync_resolver" "Mutation_publishHealth" {
  api_id = aws_appsync_graphql_api.this.id
  kind   = "PIPELINE"
  type   = "Mutation"
  field  = "publishHealth"

  request_template  = local.global_request_template
  response_template = jsonencode({ id = "$ctx.args.id" })

  pipeline_config {
    functions = [
      aws_appsync_function.health_response.function_id,
    ]
  }
}

resource "aws_appsync_resolver" "Mutation_publishMatch" {
  api_id = aws_appsync_graphql_api.this.id
  kind   = "PIPELINE"
  type   = "Mutation"
  field  = "publishMatch"

  request_template  = local.global_request_template
  response_template = local.global_response_template

  pipeline_config {
    functions = [
      aws_appsync_function.publish_match.function_id,
    ]
  }
}

resource "aws_appsync_resolver" "Subscription_joinUnrankedSoloQueue" {
  api_id = aws_appsync_graphql_api.this.id
  kind   = "PIPELINE"
  type   = "Subscription"
  field  = "joinUnrankedSoloQueue"

  request_template = join("\n", [
    "$util.quiet($ctx.stash.put(\"user\", $ctx.args.userId))",
    "$util.quiet($ctx.stash.put(\"dequeue_tables\", []))",
    "$util.quiet($ctx.stash.put(\"queue_table\", \"${var.tables.queue_unranked_solo}\"))",
    local.global_request_template,
  ])
  response_template = "#return"

  pipeline_config {
    functions = [
      aws_appsync_function.check_ip_cache.function_id,
      aws_appsync_function.lookup_ip.function_id,
      aws_appsync_function.cache_ip.function_id,
      aws_appsync_function.get_mmr_unranked_solo.function_id,
      aws_appsync_function.enqueue.function_id,
    ]
  }
}