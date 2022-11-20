resource "aws_kms_key" "main" {
  description             = "${var.name}-main primary key"
  key_usage               = "ENCRYPT_DECRYPT"
  enable_key_rotation     = true
  multi_region            = true
  deletion_window_in_days = 7
  policy                  = data.aws_iam_policy_document.main_key.json
}

resource "aws_kms_alias" "main" {
  name          = "alias/${var.name}-main"
  target_key_id = aws_kms_key.main.arn
}

data "aws_iam_policy_document" "main_key" {
  statement {
    sid       = "AllowRoot"
    effect    = "Allow"
    actions   = ["kms:*"]
    resources = ["*"]
    principals {
      type = "AWS"
      identifiers = concat(
        [data.aws_caller_identity.current.account_id],
        [for user in data.aws_iam_group.admins.users : user.arn],
      )
    }
  }

  statement {
    sid       = "AllowCloudwatchAlarms"
    effect    = "Allow"
    actions   = ["kms:Decrypt", "kms:GenerateDataKey*"]
    resources = ["*"]
    principals {
      type        = "Service"
      identifiers = ["cloudwatch.amazonaws.com", "sns.amazonaws.com"]
    }
  }
}

module "main_key_replica_us_east_2" {
  source      = "./modules/kms-replica"
  providers   = { aws = aws.us_east_2 }
  name        = "${var.name}-main"
  primary_arn = aws_kms_key.main.arn
  policy      = data.aws_iam_policy_document.main_key.json
}