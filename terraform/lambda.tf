data "aws_lambda_function" "get_strip" {
  function_name = "${var.service}-${var.stage}-get-strip"
}

data "aws_lambda_function" "gen_feed" {
  function_name = "${var.service}-${var.stage}-gen-feed"
}
