data "aws_dynamodb_table" "queue_unranked_solo_us_east_2" {
  provider = aws.us_east_2
  name     = aws_dynamodb_table.queue_unranked_solo.id
}

resource "aws_lambda_event_source_mapping" "queue_processer_unranked_solo_us_east_1" {
  event_source_arn  = aws_dynamodb_table.queue_unranked_solo.stream_arn
  function_name     = module.functions_us_east_1.queue_processer_unranked_solo.arn
  starting_position = "LATEST"
}

resource "aws_lambda_event_source_mapping" "queue_processer_unranked_solo_us_east_2" {
  provider          = aws.us_east_2
  event_source_arn  = data.aws_dynamodb_table.queue_unranked_solo_us_east_2.stream_arn
  function_name     = module.functions_us_east_2.queue_processer_unranked_solo.arn
  starting_position = "LATEST"
}