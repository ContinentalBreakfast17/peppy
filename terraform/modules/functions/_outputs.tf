output "region" {
  value = data.aws_region.current.name
}

output "ip_lookup" {
  value = {
    arn = aws_lambda_function.ip_lookup.arn,
  }
}