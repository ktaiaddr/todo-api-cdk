# サーバーレス TODO API

AWS CDK (TypeScript) + Go Lambda + DynamoDB で構築した TODO 管理 API です。

## アーキテクチャ

```
クライアント → API Gateway (REST) → Lambda (Go) → DynamoDB
```

| リソース | 詳細 |
|----------|------|
| API Gateway | REST API、CORS 有効 |
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

デプロイ後に API URL を再確認するには：

```bash
# CloudFormation の出力から確認
aws cloudformation describe-stacks --stack-name FirstStack \
  --query "Stacks[0].Outputs[?OutputKey=='ApiUrl'].OutputValue" --output text
```

## 使い方

```bash
# API URL を変数にセット（上記コマンドの出力値を使用）
API_URL="https://xxxxx.execute-api.ap-northeast-1.amazonaws.com/prod"

# 作成
curl -X POST "${API_URL}/todos" \
  -H "Content-Type: application/json" \
  -d '{"title": "タスク名", "content": "タスクの詳細"}'

# 一覧取得
curl "${API_URL}/todos"

# 単体取得
curl "${API_URL}/todos/{id}"

# 更新
curl -X PUT "${API_URL}/todos/{id}" \
  -H "Content-Type: application/json" \
  -d '{"completed": true}'

# 削除
curl -X DELETE "${API_URL}/todos/{id}"
```

## その他のコマンド

```bash
# CloudFormation テンプレートの生成（ローカル検証）
npx cdk synth

# デプロイ済みスタックとの差分確認
npx cdk diff

# 全リソース削除
npx cdk destroy
```
