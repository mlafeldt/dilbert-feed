terraform {
  backend "s3" {
    bucket = "dilbert-feed-terraform"
    key    = "dev/terraform.tfstate"
    region = "eu-central-1"
  }
}

data "aws_caller_identity" "current" {}

module "dilbert_feed" {
  source          = "../../modules/dilbert-feed"
  stage           = "dev"
  function_prefix = "arn:aws:lambda:eu-central-1:${data.aws_caller_identity.current.account_id}:function:dilbert-feed-dev-"
}
