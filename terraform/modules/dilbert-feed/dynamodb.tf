resource "aws_dynamodb_table" "metadata" {
  name         = "${var.service}-${var.stage}"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "date"

  attribute {
    name = "date"
    type = "S"
  }

  point_in_time_recovery {
    enabled = true
  }
}
