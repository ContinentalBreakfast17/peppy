locals {
  global_request_template  = <<EOT
$util.quiet($ctx.stash.put("entry_time", $util.time.nowEpochSeconds()))
$util.quiet($ctx.stash.put("graphql", $ctx.info))
{}
EOT
  global_response_template = "$util.toJson($ctx.result)"
}

resource "aws_appsync_resolver" "Mutation_unrankedSoloMatch" {
  api_id = aws_appsync_graphql_api.this.id
  kind   = "PIPELINE"
  type   = "Mutation"
  field  = "unrankedSoloMatch"

  request_template  = local.global_request_template
  response_template = local.global_response_template

  pipeline_config {
    functions = [
      aws_appsync_function.match_unranked_solo.function_id,
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
    "$util.quiet($ctx.stash.put(\"queue_version\", \"v1\"))",
    local.global_request_template,
  ])
  response_template = "#return"

  pipeline_config {
    functions = [
      aws_appsync_function.check_ip_cache.function_id,
      aws_appsync_function.lookup_ip.function_id,
      aws_appsync_function.cache_ip.function_id,
      aws_appsync_function.get_mmr_unranked_solo.function_id,
      aws_appsync_function.enqueue_unranked_solo.function_id,
      aws_appsync_function.publish_enqueue.function_id,
    ]
  }
}