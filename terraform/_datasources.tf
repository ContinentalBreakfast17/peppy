data "aws_caller_identity" "current" {}

data "aws_iam_group" "admins" {
  # todo: should be a var
  group_name = "infra-admins"
}