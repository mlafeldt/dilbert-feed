terraform {
  backend "s3" {
    bucket = "dilbert-feed-terraform"
    key    = "prod/terraform.tfstate"
    region = "eu-central-1"
  }
}

data "aws_caller_identity" "current" {}

module "dilbert_feed" {
  source          = "../../modules/dilbert-feed"
  stage           = "prod"
  bucket_name     = "dilbert-feed"
  function_prefix = "arn:aws:lambda:eu-central-1:${data.aws_caller_identity.current.account_id}:function:dilbert-feed-prod-"
}