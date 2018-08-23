resource "aws_sfn_state_machine" "state_machine" {
  name     = "${var.service}-${var.stage}"
  role_arn = "${aws_iam_role.state_machine.arn}"

  definition = <<EOF
{
  "StartAt": "GetStrip",
  "States": {
    "GetStrip": {
      "Type": "Task",
      "Resource": "${data.aws_lambda_function.get_strip.arn}",
      "Next": "GenFeed",
      "Retry": [
        {
          "ErrorEquals": ["States.TaskFailed"],
          "IntervalSeconds": 60,
          "MaxAttempts": 2,
          "BackoffRate": 2.0
        }
      ]
    },
    "GenFeed": {
      "Type": "Task",
      "Resource": "${data.aws_lambda_function.gen_feed.arn}",
      "Retry": [
        {
          "ErrorEquals": ["States.TaskFailed"],
          "IntervalSeconds": 60,
          "MaxAttempts": 2,
          "BackoffRate": 2.0
        }
      ],
      "End": true
    }
  }
}
EOF
}

resource "aws_iam_role" "state_machine" {
  name        = "${var.service}-${var.stage}-state-machine-role"
  description = "Allow state machines to invoke Lambda functions"

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
  name = "invoke-lambda-functions"
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
        "${data.aws_lambda_function.get_strip.arn}",
        "${data.aws_lambda_function.gen_feed.arn}"
      ]
    }
  ]
}
EOF
}
