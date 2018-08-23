terraform {
  backend "s3" {
    bucket = "dilbert-feed-terraform"
    key    = "terraform.tfstate"
    region = "eu-central-1"
  }
}
