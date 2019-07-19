data "template_file" "state_machine" {
  template = "${file("${path.module}/state_machine.json")}"

  vars = {
    function_prefix    = "${var.function_prefix}"
    metadata_table     = "${aws_dynamodb_table.metadata.name}"
    heartbeat_endpoint = "https://hc-ping.com/${healthchecksio_check.heartbeat.id}"
  }
}

resource "aws_sfn_state_machine" "state_machine" {
  name       = "${var.service}-${var.stage}"
  role_arn   = "${aws_iam_role.state_machine.arn}"
  definition = "${data.template_file.state_machine.rendered}"
}

resource "aws_iam_role" "state_machine" {
  name        = "${var.service}-${var.stage}-state-machine-role"
  description = "Allow state machine to do its thing"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "states.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "state_machine" {
  name = "state-machine"
  role = "${aws_iam_role.state_machine.id}"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "lambda:InvokeFunction"
      ],
      "Resource": [
        "${var.function_prefix}*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "dynamodb:PutItem"
      ],
      "Resource": [
        "${aws_dynamodb_table.metadata.arn}"
      ]
    }
  ]
}
EOF
}
