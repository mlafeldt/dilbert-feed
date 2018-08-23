terraform {
  backend "s3" {
    bucket = "dilbert-feed-terraform"
    key    = "terraform.tfstate"
    region = "eu-central-1"
  }
}

module "dilbert_feed_production" {
  source = "dilbert-feed"
  stage  = "production"
}
