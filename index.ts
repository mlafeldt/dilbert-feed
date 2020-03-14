import cdk = require('@aws-cdk/core');

interface DilbertFeedStackProps extends cdk.StackProps {
  bucketName?: string;
  heartbeatEndpoint: string;
}

export class DilbertFeedStack extends cdk.Stack {
  constructor(scope: cdk.App, id: string, props: DilbertFeedStackProps) {
    super(scope, id, props);
  }
}

const app = new cdk.App();

// TODO: remove -ts suffix
new DilbertFeedStack(app, 'dilbert-feed-dev-ts', {
  heartbeatEndpoint: 'https://hc-ping.com/33868fe9-9efc-414a-b882-a598a2b09dea',
  tags: { STAGE: 'dev' }
});
new DilbertFeedStack(app, 'dilbert-feed-prod-ts', {
  bucketName: 'dilbert-feed-ts',
  heartbeatEndpoint: 'https://hc-ping.com/4fb7e55d-fe13-498b-bfaf-73cbf20e279e',
  tags: { STAGE: 'prod' }
});

app.synth();
