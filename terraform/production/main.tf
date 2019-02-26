terraform {
  backend "s3" {
    bucket = "dilbert-feed-terraform"
    key    = "production/terraform.tfstate"
    region = "eu-central-1"
  }
}

module "dilbert_feed" {
  source = "../dilbert-feed"
  stage  = "production"
}
