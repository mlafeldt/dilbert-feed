from aws_cdk import (
    aws_codebuild as codebuild,
    aws_events as events,
    aws_events_targets as targets,
    aws_iam as iam,
    aws_lambda as lambda_,
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
        heartbeat_endpoint: str,
        bucket_name: str = None,
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
            function_name=f"{name}-get-strip",
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
            function_name=f"{name}-gen-feed",
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
            function_name=f"{name}-heartbeat",
            code=lambda_.Code.asset("bin/heartbeat"),
            environment={"HEARTBEAT_ENDPOINT": heartbeat_endpoint},
            **LAMBDA_DEFAULTS,
        )

        retry = {
            "errors": ["States.TaskFailed"],
            "interval": core.Duration.seconds(10),
            "max_attempts": 2,
            "backoff_rate": 2.0,
        }

        steps = (
            sfn.Task(
                self,
                "GetStrip",
                task=sfn_tasks.InvokeFunction(get_strip),
                result_path="$.strip",
            )
            .add_retry(**retry)
            .next(
                sfn.Task(
                    self,
                    "GenFeed",
                    task=sfn_tasks.InvokeFunction(gen_feed),
                    result_path="$.feed",
                ).add_retry(**retry)
            )
            .next(
                sfn.Task(
                    self,
                    "SendHeartbeat",
                    task=sfn_tasks.InvokeFunction(heartbeat),
                    result_path="$.heartbeat",
                ).add_retry(**retry)
            )
        )

        sm = sfn.StateMachine(
            self, "StateMachine", state_machine_name=name, definition=steps
        )

        cron = events.Rule(
            self,
            "Cron",
            description="Update Dilbert feed",
            rule_name=f"{name}-cron",
            schedule=events.Schedule.expression("cron(0 6 * * ? *)"),
        )
        cron.add_target(targets.SfnStateMachine(sm))


class DilbertFeedCICDStack(core.Stack):
    def __init__(self, app: core.App, name: str, **kwargs,) -> None:
        super().__init__(app, name, **kwargs)

        project = codebuild.Project(
            self,
            "CodeBuild",
            project_name=name,
            source=codebuild.Source.git_hub(
                owner="mlafeldt",
                repo="dilbert-feed",
                webhook_filters=[
                    codebuild.FilterGroup.in_event_of(
                        codebuild.EventAction.PUSH
                    ).and_branch_is("master")
                ],
            ),
            build_spec=codebuild.BuildSpec.from_object(
                {
                    "version": "0.2",
                    "env": {"variables": {"GO111MODULE": "on"}},
                    "phases": {
                        "install": {
                            "runtime-versions": {"golang": "1.13", "nodejs": "10"},
                            "commands": ["npm install -g aws-cdk"],
                        },
                        "build": {"commands": ["env", "make dev"]},
                    },
                }
            ),
        )
        project.add_to_role_policy(iam.PolicyStatement(actions=["*"], resources=["*"]))


app = core.App()

DilbertFeedStack(
    app,
    "dilbert-feed-dev",
    heartbeat_endpoint="https://hc-ping.com/33868fe9-9efc-414a-b882-a598a2b09dea",
    tags={"STAGE": "dev"},
)
DilbertFeedStack(
    app,
    "dilbert-feed-prod",
    bucket_name="dilbert-feed",
    heartbeat_endpoint="https://hc-ping.com/4fb7e55d-fe13-498b-bfaf-73cbf20e279e",
    tags={"STAGE": "prod"},
)
DilbertFeedCICDStack(
    app, "dilbert-feed-cicd", tags={"STAGE": "prod"},
)

app.synth()
