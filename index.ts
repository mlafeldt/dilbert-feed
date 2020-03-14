import cdk = require('@aws-cdk/core');

export class DilbertFeedStack extends cdk.Stack {
  constructor(scope: cdk.App, id: string, props?: cdk.StackProps) {
    super(scope, id, props);
  }
}

const app = new cdk.App();
new DilbertFeedStack(app, 'dilbert-feed-dev-ts', { tags: { STAGE: 'dev' } });
new DilbertFeedStack(app, 'dilbert-feed-prod-ts', { tags: { STAGE: 'prod' } });
app.synth();
