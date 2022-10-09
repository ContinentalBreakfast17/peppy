output "region" {
  value = data.aws_region.current.name
}

output "healthcheck" {
  value = {
    arn = aws_lambda_function.healthcheck_responder.arn,
  }
}

output "healthcheck_responder" {
  value = {
    arn = aws_lambda_function.healthcheck_responder.arn,
  }
}

output "ip_lookup" {
  value = {
    arn = aws_lambda_function.ip_lookup.arn,
  }
}

output "match_publisher" {
  value = {
    arn = aws_lambda_function.match_publisher.arn,
  }
}

output "queue_processer_unranked_solo" {
  value = {
    arn = aws_lambda_function.queue_processer_unranked_solo.arn,
  }
}