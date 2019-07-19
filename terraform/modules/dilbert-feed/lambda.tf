data "aws_region" "current" {}
data "aws_caller_identity" "current" {}

locals {
  func_prefix    = "arn:aws:lambda:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:function:${var.service}-${var.stage}-"
  get_strip_func = "${local.func_prefix}get-strip"
  gen_feed_func  = "${local.func_prefix}gen-feed"
  heartbeat_func = "${local.func_prefix}heartbeat"
}
