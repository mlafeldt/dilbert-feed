import cdk = require('@aws-cdk/core');
import lambda = require('@aws-cdk/aws-lambda');
import s3 = require('@aws-cdk/aws-s3');

interface DilbertFeedStackProps extends cdk.StackProps {
  bucketName?: string;
  heartbeatEndpoint: string;
}

export class DilbertFeedStack extends cdk.Stack {
  constructor(scope: cdk.App, id: string, props: DilbertFeedStackProps) {
    super(scope, id, props);

    const stripsDir = 'strips/';

    const bucket = new s3.Bucket(this, 'Bucket', {
      bucketName: props.bucketName,
      publicReadAccess: true,
      encryption: s3.BucketEncryption.S3_MANAGED
    });
    bucket.addLifecycleRule({
      id: 'DeleteStripsAfter30Days',
      prefix: stripsDir,
      expiration: cdk.Duration.days(30)
    });

    const getStrip = new lambda.Function(this, 'GetStripFunc', {
      functionName: `${id}-get-strip`,
      code: lambda.Code.fromAsset('bin/get-strip'),
      handler: 'handler',
      runtime: lambda.Runtime.GO_1_X,
      memorySize: 128,
      timeout: cdk.Duration.seconds(10),
      environment: {
        BUCKET_NAME: bucket.bucketName,
        BUCKET_PREFIX: stripsDir
      }
    });
    bucket.grantPut(getStrip);

    const genFeed = new lambda.Function(this, 'GenFeedFunc', {
      functionName: `${id}-gen-feed`,
      code: lambda.Code.fromAsset('bin/gen-feed'),
      handler: 'handler',
      runtime: lambda.Runtime.GO_1_X,
      memorySize: 128,
      timeout: cdk.Duration.seconds(10),
      environment: {
        BUCKET_NAME: bucket.bucketName,
        BUCKET_PREFIX: stripsDir
      }
    });
    bucket.grantPut(genFeed);

    // @ts-ignore
    const heartbeat = new lambda.Function(this, 'HeartbeatFunc', {
      functionName: `${id}-heartbeat`,
      code: lambda.Code.fromAsset('heartbeat'),
      handler: 'index.handler',
      runtime: lambda.Runtime.NODEJS_12_X,
      memorySize: 128,
      timeout: cdk.Duration.seconds(10),
      environment: {
        HEARTBEAT_ENDPOINT: props.heartbeatEndpoint
      }
    });

    new cdk.CfnOutput(this, 'BucketName', { value: bucket.bucketName });
    new cdk.CfnOutput(this, 'HeartbeatEndpoint', { value: props.heartbeatEndpoint });
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
