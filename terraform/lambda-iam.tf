data "aws_iam_policy_document" "lambda_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
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
# Process queue
#