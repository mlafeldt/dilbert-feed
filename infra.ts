import 'source-map-support/register'
import * as cdk from 'aws-cdk-lib'
import { Construct } from 'constructs'
import * as events from 'aws-cdk-lib/aws-events'
import * as lambda from 'aws-cdk-lib/aws-lambda'
import * as logs from 'aws-cdk-lib/aws-logs'
import * as s3 from 'aws-cdk-lib/aws-s3'
import * as sfn from 'aws-cdk-lib/aws-stepfunctions'
import * as ssm from 'aws-cdk-lib/aws-ssm'
import * as targets from 'aws-cdk-lib/aws-events-targets'
import * as tasks from 'aws-cdk-lib/aws-stepfunctions-tasks'

const LAMBDA_DEFAULTS = {
  handler: 'bootstrap',
  runtime: lambda.Runtime.PROVIDED_AL2,
  architecture: lambda.Architecture.ARM_64,
  memorySize: 128,
  timeout: cdk.Duration.seconds(10),
  logRetention: logs.RetentionDays.ONE_MONTH,
  tracing: lambda.Tracing.ACTIVE,
}

const RETRY_PROPS = {
  errors: ['States.TaskFailed'],
  interval: cdk.Duration.seconds(10),
  maxAttempts: 2,
  backoffRate: 2.0,
}

class DilbertFeedStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props: cdk.StackProps) {
    super(scope, id, props)

    const stripsDir = 'strips'
    const feedPath = 'v2/rss.xml'

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
      ...LAMBDA_DEFAULTS,
      functionName: `${id}-get-strip`,
      code: lambda.Code.fromAsset('bin/get-strip'),
      environment: {
        BUCKET_NAME: bucket.bucketName,
        STRIPS_DIR: stripsDir,
      },
    })
    bucket.grantPut(getStrip)

    const genFeed = new lambda.Function(this, 'GenFeedFunc', {
      ...LAMBDA_DEFAULTS,
      functionName: `${id}-gen-feed`,
      code: lambda.Code.fromAsset('bin/gen-feed'),
      environment: {
        BUCKET_NAME: bucket.bucketName,
        STRIPS_DIR: stripsDir,
        FEED_PATH: feedPath,
      },
    })
    bucket.grantReadWrite(genFeed)

    const heartbeatEndpoint = ssm.StringParameter.valueForStringParameter(this, `/${id}/heartbeat-endpoint`)
    const heartbeat = new lambda.Function(this, 'HeartbeatFunc', {
      ...LAMBDA_DEFAULTS,
      functionName: `${id}-heartbeat`,
      code: lambda.Code.fromAsset('bin/heartbeat'),
      environment: {
        HEARTBEAT_ENDPOINT: heartbeatEndpoint,
      },
    })

    const steps = new tasks.LambdaInvoke(this, 'GetStrip', {
      lambdaFunction: getStrip,
      resultPath: '$.strip',
    })
      .addRetry(RETRY_PROPS)
      .next(
        new tasks.LambdaInvoke(this, 'GenFeed', {
          lambdaFunction: genFeed,
          resultPath: '$.feed',
        }).addRetry(RETRY_PROPS)
      )
      .next(
        new tasks.LambdaInvoke(this, 'SendHeartbeat', {
          lambdaFunction: heartbeat,
          resultPath: '$.heartbeat',
        }).addRetry(RETRY_PROPS)
      )

    const sm = new sfn.StateMachine(this, 'StateMachine', {
      stateMachineName: id,
      definition: steps,
      tracingEnabled: true,
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
