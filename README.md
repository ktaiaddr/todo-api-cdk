# サーバーレス TODO API

AWS CDK (TypeScript) + Go Lambda + DynamoDB で構築した TODO 管理 API です。
Cognito 認証により、ユーザーごとに TODO を管理できます。

## アーキテクチャ

```
クライアント → Cognito（サインアップ・ログイン → JWT 発行）
           → API Gateway (REST) → Lambda (Go) → DynamoDB
```

| リソース | 詳細 |
|----------|------|
| Cognito | ユーザープール、メール + パスワード認証、確認コード送信 |
| API Gateway | REST API、CORS 有効、Cognito オーソライザー + API キー認証 |
| Lambda | Go (provided.al2)、128MB、30秒タイムアウト |
| DynamoDB | `Todos` テーブル、PK: `userId` / SK: `id`、オンデマンド課金 |

## API エンドポイント

全リクエストに `Authorization: Bearer <IdToken>` と `x-api-key` ヘッダーが必要です。

| メソッド | パス | 説明 |
|----------|------|------|
| GET | `/todos` | 自分の TODO 一覧取得 |
| GET | `/todos/{id}` | 自分の TODO 単体取得 |
| POST | `/todos` | TODO 作成 |
| PUT | `/todos/{id}` | 自分の TODO 更新 |
| DELETE | `/todos/{id}` | 自分の TODO 削除 |

## プロジェクト構成

```
├── bin/first.ts           # CDK アプリエントリポイント
├── lib/first-stack.ts     # CDK スタック定義（Cognito + DynamoDB + Lambda + API Gateway）
├── lambda/
│   ├── main.go            # Go Lambda ハンドラー（ルーティング + CRUD）
│   ├── go.mod
│   └── go.sum
├── cdk.json
├── package.json
└── tsconfig.json
```

## 前提条件

- Node.js
- Go 1.21+
- AWS CLI（認証情報設定済み）

## セットアップ

```bash
# 依存パッケージのインストール
npm install

# CDK ブートストラップ（アカウント/リージョンにつき初回のみ）
npx cdk bootstrap

# デプロイ
npx cdk deploy
```

デプロイ完了後、`ApiUrl`、`UserPoolId`、`UserPoolClientId` が出力されます。

## 認証フロー（CLI での検証）

```bash
# 変数にセット
USER_POOL_ID=$(aws cloudformation describe-stacks --stack-name FirstStack \
  --query "Stacks[0].Outputs[?OutputKey=='UserPoolId'].OutputValue" --output text)
CLIENT_ID=$(aws cloudformation describe-stacks --stack-name FirstStack \
  --query "Stacks[0].Outputs[?OutputKey=='UserPoolClientId'].OutputValue" --output text)
API_URL=$(aws cloudformation describe-stacks --stack-name FirstStack \
  --query "Stacks[0].Outputs[?OutputKey=='ApiUrl'].OutputValue" --output text)
API_KEY=$(aws apigateway get-api-key --include-value \
  --api-key $(aws cloudformation describe-stacks --stack-name FirstStack \
    --query "Stacks[0].Outputs[?OutputKey=='ApiKeyId'].OutputValue" --output text) \
  | jq -r '.value')

# 1. サインアップ
aws cognito-idp sign-up \
  --client-id ${CLIENT_ID} \
  --username user@example.com \
  --password 'Password123!'

# 2. メール確認（管理者として強制確認）
aws cognito-idp admin-confirm-sign-up \
  --user-pool-id ${USER_POOL_ID} \
  --username user@example.com

# 3. ログイン → JWT 取得
aws cognito-idp initiate-auth \
  --client-id ${CLIENT_ID} \
  --auth-flow USER_PASSWORD_AUTH \
  --auth-parameters USERNAME=user@example.com,PASSWORD='Password123!'

# 4. IdToken を変数にセット（--query で直接取得するとコピペミスを防げる）
TOKEN=$(aws cognito-idp initiate-auth \
  --client-id ${CLIENT_ID} \
  --auth-flow USER_PASSWORD_AUTH \
  --auth-parameters USERNAME=user@example.com,PASSWORD='Password123!' \
  --query 'AuthenticationResult.IdToken' --output text)
```

## 使い方

```bash
# 作成
curl -X POST "${API_URL}todos" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "x-api-key: ${API_KEY}" \
  -d '{"title": "タスク名", "content": "タスクの詳細"}'

# 一覧取得（自分の TODO のみ）
curl -H "Authorization: Bearer ${TOKEN}" \
     -H "x-api-key: ${API_KEY}" \
     "${API_URL}todos"

# 単体取得
curl -H "Authorization: Bearer ${TOKEN}" \
     -H "x-api-key: ${API_KEY}" \
     "${API_URL}todos/{id}"

# 更新
curl -X PUT "${API_URL}todos/{id}" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "x-api-key: ${API_KEY}" \
  -d '{"completed": true}'

# 削除
curl -X DELETE \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "x-api-key: ${API_KEY}" \
  "${API_URL}todos/{id}"

# 認証なしで 401 が返ることを確認
curl -H "x-api-key: ${API_KEY}" "${API_URL}todos"
```

## レート制限

| 項目 | 値 |
|------|-----|
| リクエスト上限 | 1,000 回/日 |
| レートリミット | 10 リクエスト/秒 |
| バースト上限 | 20 リクエスト |

## その他のコマンド

```bash
# CloudFormation テンプレートの生成（ローカル検証）
npx cdk synth

# デプロイ済みスタックとの差分確認
npx cdk diff

# 全リソース削除
npx cdk destroy
aws logs delete-log-group --log-group-name /aws/lambda/todo-api-handler
```

claude --resume c1447cc7-4cad-4fc8-97f5-82d7744bd53a