resource "aws_cloudwatch_metric_alarm" "healthcheck_failures" {
  for_each = {
    errors_fail = { name = "system-down", alarm_at = 3 }
  }

  alarm_name          = "${var.name}-${each.value.name}"
  alarm_description   = "[${var.name}/healthcheck-failures/${each.value.name}] - healthcheck function is experiencing errors"
  namespace           = "AWS/Lambda"
  metric_name         = "Errors"
  statistic           = "Sum"
  comparison_operator = "GreaterThanOrEqualToThreshold"
  threshold           = 1
  evaluation_periods  = 5
  period              = 60
  datapoints_to_alarm = each.value.alarm_at
  treat_missing_data  = "breaching"
  tags                = local.tags

  dimensions = {
    FunctionName = var.healthcheck.function_name
  }
}