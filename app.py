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
            code=lambda_.Code.asset("./build/get-strip.zip"),
            handler="get-strip",
            runtime=lambda_.Runtime.GO_1_X,
            memory_size=128,
            timeout=core.Duration.seconds(30),
        )

        gen_feed = lambda_.Function(
            self,
            "GenFeed",
            code=lambda_.Code.asset("./build/gen-feed.zip"),
            handler="gen-feed",
            runtime=lambda_.Runtime.GO_1_X,
            memory_size=128,
            timeout=core.Duration.seconds(30),
        )

        heartbeat = lambda_.Function(
            self,
            "Heartbeat",
            code=lambda_.Code.asset("./build/heartbeat.zip"),
            handler="heartbeat",
            runtime=lambda_.Runtime.GO_1_X,
            memory_size=128,
            timeout=core.Duration.seconds(30),
        )


app = core.App()
DilbertFeedStack(app, "dilbert-feed-cdk-dev", tags={"Environment": "dev"})
DilbertFeedStack(app, "dilbert-feed-cdk-prod", tags={"Environment": "prod"})
app.synth()
