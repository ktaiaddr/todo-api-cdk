# サーバーレス TODO API

AWS CDK (TypeScript) + Go Lambda + DynamoDB で構築した TODO 管理 API です。

## アーキテクチャ

```
クライアント → API Gateway (REST) → Lambda (Go) → DynamoDB
```

| リソース | 詳細 |
|----------|------|
| API Gateway | REST API、CORS 有効、API キー認証 + 使用量プラン |
| Lambda | Go (provided.al2)、128MB、30秒タイムアウト |
| DynamoDB | `Todos` テーブル、オンデマンド課金 |

## API エンドポイント

| メソッド | パス | 説明 |
|----------|------|------|
| GET | `/todos` | TODO 一覧取得 |
| GET | `/todos/{id}` | TODO 単体取得 |
| POST | `/todos` | TODO 作成 |
| PUT | `/todos/{id}` | TODO 更新 |
| DELETE | `/todos/{id}` | TODO 削除 |

## プロジェクト構成

```
├── bin/first.ts           # CDK アプリエントリポイント
├── lib/first-stack.ts     # CDK スタック定義（DynamoDB + Lambda + API Gateway）
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

デプロイ完了後、`ApiUrl` として API Gateway の URL が出力されます。

デプロイ後に API URL と API キーを確認するには：

```bash
# API URL を確認
aws cloudformation describe-stacks --stack-name FirstStack \
  --query "Stacks[0].Outputs[?OutputKey=='ApiUrl'].OutputValue" --output text

# API キーの値を確認（ApiKeyId はデプロイ時の出力に表示される）
aws apigateway get-api-key --api-key <ApiKeyId> --include-value
```

## 使い方

全リクエストに `x-api-key` ヘッダーが必要です。

```bash
# 変数にセット
API_URL=$(aws cloudformation describe-stacks --stack-name FirstStack \
  --query "Stacks[0].Outputs[?OutputKey=='ApiUrl'].OutputValue" --output text)
API_KEY=$(aws apigateway get-api-key --include-value \
  --api-key $(aws cloudformation describe-stacks --stack-name FirstStack \
    --query "Stacks[0].Outputs[?OutputKey=='ApiKeyId'].OutputValue" --output text) \
  | jq -r '.value')

# 作成
curl -X POST "${API_URL}todos" \
  -H "Content-Type: application/json" \
  -H "x-api-key: ${API_KEY}" \
  -d '{"title": "タスク名", "content": "タスクの詳細"}'

# 一覧取得
curl -H "x-api-key: ${API_KEY}" "${API_URL}todos"

# 単体取得
curl -H "x-api-key: ${API_KEY}" "${API_URL}todos/{id}"

# 更新
curl -X PUT "${API_URL}todos/{id}" \
  -H "Content-Type: application/json" \
  -H "x-api-key: ${API_KEY}" \
  -d '{"completed": true}'

# 削除
curl -X DELETE -H "x-api-key: ${API_KEY}" "${API_URL}todos/{id}"
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
