data "aws_lambda_function" "get_strip" {
  function_name = "${var.service}-${var.stage}-get-strip"
  qualifier     = ""
}

data "aws_lambda_function" "gen_feed" {
  function_name = "${var.service}-${var.stage}-gen-feed"
  qualifier     = ""
}

data "aws_lambda_function" "heartbeat" {
  function_name = "${var.service}-${var.stage}-heartbeat"
  qualifier     = ""
}
