# https://github.com/kristofferahl/terraform-provider-healthchecksio
# read API key from HEALTHCHECKSIO_API_KEY env var
provider "healthchecksio" {}

resource "healthchecksio_check" "heartbeat" {
  name     = "${var.service}"
  schedule = "${var.heartbeat_schedule}"
  timezone = "UTC"
  grace    = 3600                        # grace period of 1h
  tags     = ["${var.stage}"]
}
