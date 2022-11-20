resource "aws_sns_topic" "alarms" {
  name              = "${var.name}-alarms"
  kms_master_key_id = var.kms_key_id
  tags              = local.tags
}

resource "aws_sns_topic_subscription" "alarms_emails" {
  for_each  = toset(var.send_alarms_to)
  topic_arn = aws_sns_topic.alarms.arn
  protocol  = "email"
  endpoint  = each.key
}

resource "aws_sns_topic_policy" "alarms" {
  arn    = aws_sns_topic.alarms.arn
  policy = data.aws_iam_policy_document.alarms_topic_policy.json
}

data "aws_iam_policy_document" "alarms_topic_policy" {
  statement {
    effect    = "Allow"
    actions   = ["SNS:Publish"]
    resources = [aws_sns_topic.alarms.arn]

    principals {
      type        = "Service"
      identifiers = ["cloudwatch.amazonaws.com"]
    }

    condition {
      test     = "StringEquals"
      variable = "aws:SourceAccount"
      values   = [data.aws_caller_identity.current.id]
    }

    condition {
      test     = "ArnLike"
      variable = "aws:SourceArn"
      values   = ["arn:aws:cloudwatch:*:${data.aws_caller_identity.current.id}:alarm:${var.name}-*"]
    }
  }
}