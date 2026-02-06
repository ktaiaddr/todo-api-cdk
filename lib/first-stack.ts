import * as cdk from 'aws-cdk-lib/core';
import * as dynamodb from 'aws-cdk-lib/aws-dynamodb';
import * as apigateway from 'aws-cdk-lib/aws-apigateway';
import { GoFunction } from '@aws-cdk/aws-lambda-go-alpha';
import { Construct } from 'constructs';

export class FirstStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    // DynamoDB テーブル
    const todosTable = new dynamodb.Table(this, 'TodosTable', {
      tableName: 'Todos',
      partitionKey: { name: 'id', type: dynamodb.AttributeType.STRING },
      billingMode: dynamodb.BillingMode.PAY_PER_REQUEST,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    // Go Lambda 関数
    const todoFunction = new GoFunction(this, 'TodoHandler', {
      entry: 'lambda',
      functionName: 'todo-api-handler',
      timeout: cdk.Duration.seconds(30),
      memorySize: 128,
      environment: {
        TABLE_NAME: todosTable.tableName,
      },
    });

    // Lambda に DynamoDB の読み書き権限を付与
    todosTable.grantReadWriteData(todoFunction);

    // API Gateway REST API
    const api = new apigateway.RestApi(this, 'TodoApi', {
      restApiName: 'Todo API',
      defaultCorsPreflightOptions: {
        allowOrigins: apigateway.Cors.ALL_ORIGINS,
        allowMethods: apigateway.Cors.ALL_METHODS,
      },
    });

    const lambdaIntegration = new apigateway.LambdaIntegration(todoFunction);
    const methodOptions = { apiKeyRequired: true };

    // /todos リソース
    const todos = api.root.addResource('todos');
    todos.addMethod('GET', lambdaIntegration, methodOptions);
    todos.addMethod('POST', lambdaIntegration, methodOptions);

    // /todos/{id} リソース
    const todoById = todos.addResource('{id}');
    todoById.addMethod('GET', lambdaIntegration, methodOptions);
    todoById.addMethod('PUT', lambdaIntegration, methodOptions);
    todoById.addMethod('DELETE', lambdaIntegration, methodOptions);

    // API キー
    const apiKey = api.addApiKey('TodoApiKey');

    // 使用量プラン
    const usagePlan = api.addUsagePlan('TodoUsagePlan', {
      name: 'TodoUsagePlan',
      throttle: {
        rateLimit: 10,    // 1秒あたり10リクエスト
        burstLimit: 20,   // バースト上限20
      },
      quota: {
        limit: 1000,      // 1日あたり1000リクエスト
        period: apigateway.Period.DAY,
      },
    });
    usagePlan.addApiKey(apiKey);
    usagePlan.addApiStage({ stage: api.deploymentStage });

    // API URL を出力
    new cdk.CfnOutput(this, 'ApiUrl', {
      value: api.url,
      description: 'TODO API Gateway URL',
    });

    // API キーの確認コマンドを出力
    new cdk.CfnOutput(this, 'ApiKeyId', {
      value: apiKey.keyId,
      description: 'API Key ID（値の確認: aws apigateway get-api-key --api-key <ID> --include-value）',
    });
  }
}
