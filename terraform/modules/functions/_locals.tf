locals {
  tags        = { region = data.aws_region.current.name }
  code_bucket = "${var.code.bucket_prefix}${data.aws_region.current.name}"
}