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
    "timeout": core.Duration.seconds(30),
}


class DilbertFeedStack(core.Stack):
    def __init__(self, app: core.App, name: str, **kwargs) -> None:
        super().__init__(app, name, **kwargs)

        bucket = s3.Bucket(
            self,
            "dilbert-feed",
            encryption=s3.BucketEncryption.S3_MANAGED,
            public_read_access=True,
        )
        bucket.add_lifecycle_rule(
            id="DeleteStripsAfter30Days",
            prefix="strips/",
            expiration=core.Duration.days(30),
        )

        get_strip = lambda_.Function(
            self,
            "GetStripFunc",
            code=lambda_.Code.asset("bin/get-strip"),
            environment={"BUCKET_NAME": bucket.bucket_name, "BUCKET_PREFIX": "strips/"},
            **LAMBDA_DEFAULTS,
        )

        gen_feed = lambda_.Function(
            self,
            "GenFeedFunc",
            code=lambda_.Code.asset("bin/gen-feed"),
            environment={"BUCKET_NAME": bucket.bucket_name, "BUCKET_PREFIX": "strips/"},
            **LAMBDA_DEFAULTS,
        )

        bucket.grant_put(get_strip)
        bucket.grant_put(gen_feed)

        # heartbeat = lambda_.Function(
        #     self,
        #     "Heartbeat",
        #     code=lambda_.Code.asset("bin/heartbeat"),
        #     **LAMBDA_DEFAULTS,
        # )

        definition = sfn.Task(
            self,
            "GetStrip",
            task=sfn_tasks.InvokeFunction(get_strip),
            result_path="$.strip",
        ).next(
            sfn.Task(
                self,
                "GenFeed",
                task=sfn_tasks.InvokeFunction(gen_feed),
                result_path="$.feed",
            )
        )

        sm = sfn.StateMachine(
            self,
            "StateMachine",
            definition=definition,
            timeout=core.Duration.seconds(30),
        )

        cron = events.Rule(
            self, "Cron", schedule=events.Schedule.expression("cron(0 6 * * ? *)")
        )
        cron.add_target(targets.SfnStateMachine(sm))


app = core.App()
DilbertFeedStack(app, "dilbert-feed-cdk-dev", tags={"STAGE": "dev"})
DilbertFeedStack(app, "dilbert-feed-cdk-prod", tags={"STAGE": "prod"})
app.synth()
