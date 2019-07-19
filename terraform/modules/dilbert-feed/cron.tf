resource "aws_cloudwatch_event_rule" "cron" {
  name                = "${var.service}-${var.stage}-cron"
  description         = "Update Dilbert feed"
  schedule_expression = "${var.update_schedule}"
}

resource "aws_cloudwatch_event_target" "cron" {
  rule     = "${aws_cloudwatch_event_rule.cron.name}"
  arn      = "${aws_sfn_state_machine.state_machine.id}"
  role_arn = "${aws_iam_role.cron.arn}"
}

resource "aws_iam_role" "cron" {
  name        = "${var.service}-${var.stage}-cron-role"
  description = "Allow CloudWatch Events to start state machine"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "events.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "cron" {
  name = "start-state-machine"
  role = "${aws_iam_role.cron.id}"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "states:StartExecution",
      "Resource": "${aws_sfn_state_machine.state_machine.id}"
    }
  ]
}
EOF
}
