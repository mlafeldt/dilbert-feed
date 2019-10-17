from aws_cdk import (
    aws_events as events,
    aws_lambda as lambda_,
    aws_events_targets as targets,
    core,
)


class DilbertFeedStack(core.Stack):
    def __init__(self, app: core.App, name: str, **kwargs) -> None:
        super().__init__(app, name, **kwargs)

        get_strip = lambda_.Function(
            self,
            "GetStrip",
            code=lambda_.Code.asset("bin/get-strip"),
            handler="handler",
            runtime=lambda_.Runtime.GO_1_X,
            memory_size=128,
            timeout=core.Duration.seconds(30),
        )

        gen_feed = lambda_.Function(
            self,
            "GenFeed",
            code=lambda_.Code.asset("bin/gen-feed"),
            handler="handler",
            runtime=lambda_.Runtime.GO_1_X,
            memory_size=128,
            timeout=core.Duration.seconds(30),
        )

        heartbeat = lambda_.Function(
            self,
            "Heartbeat",
            code=lambda_.Code.asset("bin/heartbeat"),
            handler="handler",
            runtime=lambda_.Runtime.GO_1_X,
            memory_size=128,
            timeout=core.Duration.seconds(30),
        )


app = core.App()
DilbertFeedStack(app, "dilbert-feed-cdk-dev", tags={"Environment": "dev"})
DilbertFeedStack(app, "dilbert-feed-cdk-prod", tags={"Environment": "prod"})
app.synth()
