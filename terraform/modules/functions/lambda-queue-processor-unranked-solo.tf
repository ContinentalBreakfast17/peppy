locals {
  queue_processer_unranked_solo_name = "${var.name}-queue-processor-unranked-solo"
}

resource "aws_lambda_function" "queue_processer_unranked_solo" {
  function_name    = local.queue_processer_unranked_solo_name
  role             = var.functions.queue_processer_unranked_solo.role
  source_code_hash = var.functions.queue_processer_unranked_solo.source_hash
  filename         = var.functions.queue_processer_unranked_solo.source_file
  description      = "Makes matches based on the unranked solo queue"
  handler          = "test.handler"
  runtime          = "python3.9"
  architectures    = ["arm64"]
  timeout          = 15
  memory_size      = 128
  tags             = local.tags

  environment {
    variables = {
      QUEUE_TABLE  = var.functions.queue_processer_unranked_solo.queue_table,
      QUEUE_INDEX  = var.functions.queue_processer_unranked_solo.queue_index,
      MATCH_TABLE  = var.functions.queue_processer_unranked_solo.match_table,
      LOCK_TABLE   = var.functions.queue_processer_unranked_solo.lock_table,
      LOCK_REGIONS = join(",", var.functions.queue_processer_unranked_solo.lock_regions),
    }
  }

  depends_on = [aws_cloudwatch_log_group.queue_processer_unranked_solo]
}

resource "aws_cloudwatch_log_group" "queue_processer_unranked_solo" {
  name              = "/aws/lambda/${local.queue_processer_unranked_solo_name}"
  retention_in_days = 7
  tags              = local.tags
}