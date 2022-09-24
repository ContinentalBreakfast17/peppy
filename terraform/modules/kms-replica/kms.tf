resource "aws_kms_replica_key" "this" {
  primary_key_arn         = var.primary_arn
  description             = "${var.name} replica key - ${data.aws_region.current.name}"
  deletion_window_in_days = 7
  policy                  = var.policy
  tags                    = local.tags
}

resource "aws_kms_alias" "this" {
  name          = "alias/${var.name}"
  target_key_id = aws_kms_replica_key.this.arn
}