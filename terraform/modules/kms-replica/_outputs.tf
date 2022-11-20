output "key" {
  value = {
    arn = aws_kms_replica_key.this.arn,
    id  = aws_kms_replica_key.this.key_id,
  }
}

output "region" {
  value = data.aws_region.current.name
}