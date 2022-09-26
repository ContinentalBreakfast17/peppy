data "aws_iam_policy_document" "lambda_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

data "aws_iam_policy" "lambda_logs" {
  name = "AWSLambdaBasicExecutionRole"
}

########
#
# Ip Lookup
#
resource "aws_iam_role" "ip_lookup" {
  name               = "ip-lookup"
  path               = "/${var.name}/functions/"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume_role.json
}

resource "aws_iam_role_policy_attachment" "ip_lookup_kms" {
  role       = aws_iam_role.ip_lookup.name
  policy_arn = aws_iam_policy.kms_decrypt.arn
}

resource "aws_iam_role_policy_attachment" "ip_lookup_logs" {
  role       = aws_iam_role.ip_lookup.name
  policy_arn = data.aws_iam_policy.lambda_logs.arn
}

resource "aws_iam_role_policy" "ip_lookup_custom" {
  role   = aws_iam_role.ip_lookup.name
  policy = data.aws_iam_policy_document.ip_lookup_custom.json
}

data "aws_iam_policy_document" "ip_lookup_custom" {
  statement {
    sid       = "AllowGetIpCredential"
    effect    = "Allow"
    actions   = ["secretsmanager:GetSecretValue"]
    resources = [replace(aws_secretsmanager_secret.ip_lookup_token.arn, "us-east-1", "*")]
  }
}

########
#
# Queue Processer - unranked solo
#
resource "aws_iam_role" "queue_processer_unranked_solo" {
  name               = "queue-processer-unranked-solo"
  path               = "/${var.name}/functions/"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume_role.json
}

resource "aws_iam_role_policy_attachment" "queue_processer_unranked_solo_kms" {
  role       = aws_iam_role.queue_processer_unranked_solo.name
  policy_arn = aws_iam_policy.kms_crypto.arn
}

resource "aws_iam_role_policy_attachment" "queue_processer_unranked_solo_logs" {
  role       = aws_iam_role.queue_processer_unranked_solo.name
  policy_arn = data.aws_iam_policy.lambda_logs.arn
}

resource "aws_iam_role_policy" "queue_processer_unranked_solo_custom" {
  role   = aws_iam_role.queue_processer_unranked_solo.name
  policy = data.aws_iam_policy_document.queue_processer_unranked_solo_custom.json
}

data "aws_iam_policy_document" "queue_processer_unranked_solo_custom" {
  statement {
    sid       = "AllowReadDynamoStream"
    effect    = "Allow"
    actions   = ["dynamodb:GetRecords", "dynamodb:GetShardIterator", "dynamodb:DescribeStream", "dynamodb:ListStreams"]
    resources = ["${replace(aws_dynamodb_table.queue_unranked_solo.arn, "us-east-1", "*")}/stream/*"]
  }

  statement {
    sid       = "AllowReadDynamo"
    effect    = "Allow"
    actions   = ["dynamodb:Query"]
    resources = ["${replace(aws_dynamodb_table.queue_unranked_solo.arn, "us-east-1", "*")}/index/${local.dynamo_indexes.queue_sort}"]
  }
}