from aws_cdk import (
    aws_events as events,
    aws_lambda as lambda_,
    aws_events_targets as targets,
    aws_s3 as s3,
    aws_stepfunctions as sfn,
    aws_stepfunctions_tasks as sfn_tasks,
    core,
)

LAMBDA_DEFAULTS = {
    "handler": "handler",
    "runtime": lambda_.Runtime.GO_1_X,
    "memory_size": 128,
    "timeout": core.Duration.seconds(10),
}
STRIPS_DIR = "strips/"


class DilbertFeedStack(core.Stack):
    def __init__(
        self,
        app: core.App,
        name: str,
        bucket_name: str = None,
        heartbeat_endpoint: str = None,
        **kwargs,
    ) -> None:
        super().__init__(app, name, **kwargs)

        bucket = s3.Bucket(
            self,
            "Bucket",
            bucket_name=bucket_name,
            public_read_access=True,
            encryption=s3.BucketEncryption.S3_MANAGED,
        )
        bucket.add_lifecycle_rule(
            id="DeleteStripsAfter30Days",
            prefix=STRIPS_DIR,
            expiration=core.Duration.days(30),
        )

        get_strip = lambda_.Function(
            self,
            "GetStripFunc",
            code=lambda_.Code.asset("bin/get-strip"),
            environment={
                "BUCKET_NAME": bucket.bucket_name,
                "BUCKET_PREFIX": STRIPS_DIR,
            },
            **LAMBDA_DEFAULTS,
        )
        bucket.grant_put(get_strip)

        gen_feed = lambda_.Function(
            self,
            "GenFeedFunc",
            code=lambda_.Code.asset("bin/gen-feed"),
            environment={
                "BUCKET_NAME": bucket.bucket_name,
                "BUCKET_PREFIX": STRIPS_DIR,
            },
            **LAMBDA_DEFAULTS,
        )
        bucket.grant_put(gen_feed)

        heartbeat = lambda_.Function(
            self,
            "HeartbeatFunc",
            code=lambda_.Code.asset("bin/heartbeat"),
            environment={"HEARTBEAT_ENDPOINT": heartbeat_endpoint},
            **LAMBDA_DEFAULTS,
        )

        steps = (
            sfn.Task(
                self,
                "GetStrip",
                task=sfn_tasks.InvokeFunction(get_strip),
                result_path="$.strip",
            )
            .next(
                sfn.Task(
                    self,
                    "GenFeed",
                    task=sfn_tasks.InvokeFunction(gen_feed),
                    result_path="$.feed",
                )
            )
            .next(
                sfn.Task(
                    self,
                    "SendHeartbeat",
                    task=sfn_tasks.InvokeFunction(heartbeat),
                    result_path="$.heartbeat",
                )
            )
        )

        sm = sfn.StateMachine(
            self, "StateMachine", state_machine_name=name, definition=steps
        )

        cron = events.Rule(
            self,
            "Cron",
            description="Update Dilbert feed",
            rule_name=name + "-cron",
            schedule=events.Schedule.expression("cron(0 6 * * ? *)"),
        )
        cron.add_target(targets.SfnStateMachine(sm))


app = core.App()

DilbertFeedStack(
    app,
    "dilbert-feed-cdk-dev",
    heartbeat_endpoint="https://hc-ping.com/07321d8b-251b-4cf8-aaec-73e152eee601",
    tags={"STAGE": "dev"},
)
# DilbertFeedStack(
#     app, "dilbert-feed-cdk-prod", bucket_name="dilbert-feed-cdk", tags={"STAGE": "prod"}
# )

app.synth()
