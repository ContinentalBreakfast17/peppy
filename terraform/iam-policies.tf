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