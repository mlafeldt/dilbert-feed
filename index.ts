import cdk = require('@aws-cdk/core');
import events = require('@aws-cdk/aws-events');
import lambda = require('@aws-cdk/aws-lambda');
import s3 = require('@aws-cdk/aws-s3');
import sfn = require('@aws-cdk/aws-stepfunctions');
import ssm = require('@aws-cdk/aws-ssm');
import targets = require('@aws-cdk/aws-events-targets');
import tasks = require('@aws-cdk/aws-stepfunctions-tasks');

export class DilbertFeedStack extends cdk.Stack {
  constructor(scope: cdk.App, id: string, props: cdk.StackProps) {
    super(scope, id, props);

    const stripsDir = 'strips/';

    const bucket = new s3.Bucket(this, 'Bucket', {
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

    const heartbeatEndpoint = ssm.StringParameter.valueForStringParameter(this, `/${id}/heartbeat-endpoint`);
    const heartbeat = new lambda.Function(this, 'HeartbeatFunc', {
      functionName: `${id}-heartbeat`,
      code: lambda.Code.fromAsset('heartbeat'),
      handler: 'index.handler',
      runtime: lambda.Runtime.NODEJS_12_X,
      memorySize: 128,
      timeout: cdk.Duration.seconds(10),
      environment: {
        HEARTBEAT_ENDPOINT: heartbeatEndpoint
      }
    });

    const taskRetry = {
      errors: ['States.TaskFailed'],
      interval: cdk.Duration.seconds(10),
      maxAttempts: 2,
      backoffRate: 2.0
    };

    const steps = new sfn.Task(this, 'GetStrip', {
      task: new tasks.InvokeFunction(getStrip),
      resultPath: '$.strip'
    })
      .addRetry(taskRetry)
      .next(
        new sfn.Task(this, 'GenFeed', {
          task: new tasks.InvokeFunction(genFeed),
          resultPath: '$.feed'
        }).addRetry(taskRetry)
      )
      .next(
        new sfn.Task(this, 'SendHeartbeat', {
          task: new tasks.InvokeFunction(heartbeat),
          resultPath: '$.heartbeat'
        }).addRetry(taskRetry)
      );

    const sm = new sfn.StateMachine(this, 'StateMachine', {
      stateMachineName: id,
      definition: steps
    });

    const cron = new events.Rule(this, 'Cron', {
      description: 'Update Dilbert feed',
      ruleName: `${id}-cron`,
      schedule: events.Schedule.expression('cron(0 6 * * ? *)')
    });
    cron.addTarget(new targets.SfnStateMachine(sm));

    new cdk.CfnOutput(this, 'BucketName', { value: bucket.bucketName });
    new cdk.CfnOutput(this, 'HeartbeatEndpoint', { value: heartbeatEndpoint });
  }
}

const app = new cdk.App();
new DilbertFeedStack(app, 'dilbert-feed-dev', { tags: { STAGE: 'dev' } });
new DilbertFeedStack(app, 'dilbert-feed-prod', { tags: { STAGE: 'prod' } });
app.synth();
