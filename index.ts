import 'source-map-support/register'
import * as cdk from '@aws-cdk/core'
import * as events from '@aws-cdk/aws-events'
import * as lambda from '@aws-cdk/aws-lambda'
import * as logs from '@aws-cdk/aws-logs'
import * as s3 from '@aws-cdk/aws-s3'
import * as sfn from '@aws-cdk/aws-stepfunctions'
import * as ssm from '@aws-cdk/aws-ssm'
import * as targets from '@aws-cdk/aws-events-targets'
import * as tasks from '@aws-cdk/aws-stepfunctions-tasks'

export class DilbertFeedStack extends cdk.Stack {
  constructor(scope: cdk.App, id: string, props: cdk.StackProps) {
    super(scope, id, props)

    const stripsDir = 'strips'
    const feedPath = 'v1/rss.xml'

    const bucket = new s3.Bucket(this, 'Bucket', {
      publicReadAccess: true,
      encryption: s3.BucketEncryption.S3_MANAGED,
    })
    bucket.addLifecycleRule({
      id: 'DeleteStripsAfter30Days',
      prefix: `${stripsDir}/`,
      expiration: cdk.Duration.days(30),
    })

    const getStrip = new lambda.Function(this, 'GetStripFunc', {
      functionName: `${id}-get-strip`,
      code: lambda.Code.fromAsset('bin/get-strip'),
      handler: 'handler',
      runtime: lambda.Runtime.GO_1_X,
      memorySize: 128,
      timeout: cdk.Duration.seconds(10),
      logRetention: logs.RetentionDays.ONE_MONTH,
      environment: {
        BUCKET_NAME: bucket.bucketName,
        STRIPS_DIR: stripsDir,
      },
    })
    bucket.grantPut(getStrip)

    const genFeed = new lambda.Function(this, 'GenFeedFunc', {
      functionName: `${id}-gen-feed`,
      code: lambda.Code.fromAsset('bin/gen-feed'),
      handler: 'handler',
      runtime: lambda.Runtime.GO_1_X,
      memorySize: 128,
      timeout: cdk.Duration.seconds(10),
      logRetention: logs.RetentionDays.ONE_MONTH,
      environment: {
        BUCKET_NAME: bucket.bucketName,
        STRIPS_DIR: stripsDir,
        FEED_PATH: feedPath,
      },
    })
    bucket.grantReadWrite(genFeed)

    const heartbeatEndpoint = ssm.StringParameter.valueForStringParameter(this, `/${id}/heartbeat-endpoint`)
    const heartbeat = new lambda.Function(this, 'HeartbeatFunc', {
      functionName: `${id}-heartbeat`,
      code: lambda.Code.fromAsset('bin/heartbeat'),
      handler: 'bootstrap',
      runtime: lambda.Runtime.PROVIDED_AL2,
      memorySize: 128,
      timeout: cdk.Duration.seconds(10),
      logRetention: logs.RetentionDays.ONE_MONTH,
      environment: {
        HEARTBEAT_ENDPOINT: heartbeatEndpoint,
        RUST_LOG: 'debug',
      },
    })

    const retryProps = {
      errors: ['States.TaskFailed'],
      interval: cdk.Duration.seconds(10),
      maxAttempts: 2,
      backoffRate: 2.0,
    }

    const steps = new tasks.LambdaInvoke(this, 'GetStrip', {
      lambdaFunction: getStrip,
      resultPath: '$.strip',
    })
      .addRetry(retryProps)
      .next(
        new tasks.LambdaInvoke(this, 'GenFeed', {
          lambdaFunction: genFeed,
          resultPath: '$.feed',
        }).addRetry(retryProps)
      )
      .next(
        new tasks.LambdaInvoke(this, 'SendHeartbeat', {
          lambdaFunction: heartbeat,
          resultPath: '$.heartbeat',
        }).addRetry(retryProps)
      )

    const sm = new sfn.StateMachine(this, 'StateMachine', {
      stateMachineName: id,
      definition: steps,
    })

    const cron = new events.Rule(this, 'Cron', {
      description: 'Update Dilbert feed',
      schedule: events.Schedule.expression('cron(0 8 * * ? *)'),
    })
    cron.addTarget(new targets.SfnStateMachine(sm))

    new cdk.CfnOutput(this, 'BucketName', { value: bucket.bucketName })
    new cdk.CfnOutput(this, 'FeedUrl', { value: `https://${bucket.bucketRegionalDomainName}/${feedPath}` })
    new cdk.CfnOutput(this, 'HeartbeatEndpoint', { value: heartbeatEndpoint })
  }
}

const app = new cdk.App()
new DilbertFeedStack(app, 'dilbert-feed-dev', { tags: { STAGE: 'dev' } })
new DilbertFeedStack(app, 'dilbert-feed-prod', { tags: { STAGE: 'prod' } })
app.synth()
