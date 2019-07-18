terraform {
  backend "s3" {
    bucket = "dilbert-feed-terraform"
    key    = "prod/terraform.tfstate"
    region = "eu-central-1"
  }
}

module "dilbert_feed" {
  source = "../../modules/dilbert-feed"
  stage  = "prod"
}
