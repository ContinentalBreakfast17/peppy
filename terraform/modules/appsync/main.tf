locals {
  schema = join("\n\n", [
    for filename in fileset("${var.paths.schema}", "*.graphql") : file(filename)
  ])
}

resource "aws_appsync_graphql_api" "this" {
  name                = var.name
  schema              = local.schema
  tags                = local.tags
  authentication_type = "API_KEY" # this will likely change

  log_config {
    cloudwatch_logs_role_arn = var.role
    field_log_level          = var.log_level
  }
}

resource "aws_appsync_datasource" "noop" {
  api_id = aws_appsync_graphql_api.this.id
  name   = "no_op"
  type   = "NONE"
}

# todo: event bus source

resource "aws_appsync_datasource" "ip_cache" {
  api_id           = aws_appsync_graphql_api.this.id
  name             = "ip_cache_table"
  service_role_arn = var.role
  type             = "AMAZON_DYNAMODB"

  dynamodb_config {
    table_name = var.tables.ip_cache
  }
}

resource "aws_appsync_datasource" "queue_unranked_solo" {
  api_id           = aws_appsync_graphql_api.this.id
  name             = "queue_unranked_solo"
  service_role_arn = var.role
  type             = "AMAZON_DYNAMODB"

  dynamodb_config {
    table_name = var.tables.queue_unranked_solo
  }
}

resource "aws_appsync_datasource" "mmr_unranked_solo" {
  api_id           = aws_appsync_graphql_api.this.id
  name             = "mmr_unranked_solo"
  service_role_arn = var.role
  type             = "AMAZON_DYNAMODB"

  dynamodb_config {
    table_name = var.tables.mmr_unranked_solo
  }
}

resource "aws_appsync_datasource" "ip_lookup" {
  api_id           = aws_appsync_graphql_api.this.id
  name             = "ip_lookup_function"
  service_role_arn = var.role
  type             = "AWS_LAMBDA"

  lambda_config {
    function_arn = var.functions.ip_lookup
  }
}