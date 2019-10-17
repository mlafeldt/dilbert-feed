from aws_cdk import (
    aws_events as events,
    aws_lambda as lambda_,
    aws_events_targets as targets,
    core,
)


class DilbertFeedStack(core.Stack):
    def __init__(self, app: core.App, name: str, **kwargs) -> None:
        super().__init__(app, name, **kwargs)

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
DilbertFeedStack(app, "DilbertFeedDev", tags={"Environment": "dev"})
DilbertFeedStack(app, "DilbertFeedProd", tags={"Environment": "prod"})
app.synth()
