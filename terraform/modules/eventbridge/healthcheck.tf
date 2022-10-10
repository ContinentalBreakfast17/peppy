resource "aws_cloudwatch_event_rule" "healthcheck" {
  name                = "${var.name}-healthcheck-cron"
  description         = "Triggers a healthcheck on a schedule"
  is_enabled          = var.cron.healthcheck.enabled
  schedule_expression = var.cron.healthcheck.schedule
  tags                = local.tags
}

resource "aws_cloudwatch_event_target" "healthcheck_lambda" {
  rule = aws_cloudwatch_event_rule.healthcheck.name
  arn  = var.cron.healthcheck.lambda

  retry_policy {
    maximum_event_age_in_seconds = 60
    maximum_retry_attempts       = 1
  }
}

resource "aws_lambda_permission" "healthcheck_lambda" {
  statement_id  = "AllowExecutionFromEventBridge"
  action        = "lambda:InvokeFunction"
  function_name = var.cron.healthcheck.lambda
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.healthcheck.arn
}