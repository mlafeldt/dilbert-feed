provider "aws" {
  region = "eu-central-1"
}

resource "aws_s3_bucket" "terraform" {
  bucket = "dilbert-feed-terraform"
  acl    = "private"

  versioning {
    enabled = true
  }

  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        sse_algorithm = "AES256"
      }
    }
  }

  lifecycle {
    prevent_destroy = true
  }
}
