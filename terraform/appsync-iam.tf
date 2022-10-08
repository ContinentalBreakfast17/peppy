data "aws_iam_policy_document" "appsync_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["appsync.amazonaws.com"]
    }
  }
}

data "aws_iam_policy" "appsync_logs" {
  name = "AWSAppSyncPushToCloudWatchLogs"
}

resource "aws_iam_role" "appsync" {
  name               = "appsync"
  path               = "/${var.name}/"
  assume_role_policy = data.aws_iam_policy_document.appsync_assume_role.json
}

resource "aws_iam_role_policy_attachment" "appsync_kms" {
  role       = aws_iam_role.appsync.name
  policy_arn = aws_iam_policy.kms_crypto.arn
}

resource "aws_iam_role_policy_attachment" "appsync_logs" {
  role       = aws_iam_role.appsync.name
  policy_arn = data.aws_iam_policy.appsync_logs.arn
}

resource "aws_iam_role_policy" "appsync_custom" {
  role   = aws_iam_role.appsync.name
  policy = data.aws_iam_policy_document.appsync_custom.json
}

data "aws_iam_policy_document" "appsync_custom" {
  statement {
    sid     = "AllowDynamoActions"
    effect  = "Allow"
    actions = ["dynamodb:*Item"]
    resources = [
      replace(aws_dynamodb_table.healthcheck.arn, "us-east-1", "*"),
      replace(aws_dynamodb_table.ip_cache.arn, "us-east-1", "*"),
      replace(aws_dynamodb_table.queue_unranked_solo.arn, "us-east-1", "*"),
      replace(aws_dynamodb_table.mmr_unranked_solo.arn, "us-east-1", "*"),
    ]
  }

  statement {
    sid       = "AllowLambdaInvocation"
    effect    = "Allow"
    actions   = ["lambda:InvokeFunction"]
    resources = [replace(module.functions_us_east_1.ip_lookup.arn, "us-east-1", "*")]
  }
}