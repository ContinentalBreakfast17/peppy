output "key" {
  value = {
    arn = aws_kms_replica_key.this.arn,
  }
}

output "region" {
  value = data.aws_region.current.name
}