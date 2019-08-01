variable "service" {
  default = "dilbert-feed"
}

variable "update_schedule" {
  default = "cron(0 6 * * ? *)"
}

variable "heartbeat_schedule" {
  default = "0 6 * * *"
}

variable "stage" {}
variable "bucket_name" {}
variable "function_prefix" {}
