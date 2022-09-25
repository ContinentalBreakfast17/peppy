# common iam policies applied across multiple resources
resource "aws_iam_policy" "kms_decrypt" {
  name        = "kms-decrypt"
  path        = "/${var.name}/"
  description = "Enables KMS decryption for ${var.name}"
  policy      = data.aws_iam_policy_document.kms_decrypt.json
}

data "aws_iam_policy_document" "kms_decrypt" {
  statement {
    sid       = "AllowKmsDecryption"
    effect    = "Allow"
    actions   = ["kms:Decrypt"]
    resources = [replace(aws_kms_key.main.arn, "us-east-1", "*")]
  }
}

resource "aws_iam_policy" "kms_crypto" {
  name        = "kms-crypto"
  path        = "/${var.name}/"
  description = "Enables full KMS crypto operations for ${var.name}"
  policy      = data.aws_iam_policy_document.kms_crypto.json
}

data "aws_iam_policy_document" "kms_crypto" {
  statement {
    sid       = "AllowKmsCryptoOps"
    effect    = "Allow"
    actions   = ["kms:CreateGrant", "kms:Decrypt", "kms:Encrypt", "kms:GenerateDataPair*", "kms:ReEncrypt*"]
    resources = [replace(aws_kms_key.main.arn, "us-east-1", "*")]
  }
}