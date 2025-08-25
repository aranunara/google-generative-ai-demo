# Google Virtual Try-On Demo

Google Try-on APIを使用したバーチャル試着デモアプリケーション（Go + Clean Architecture + TDD）

## 特徴

- **クリーンアーキテクチャ**: レイヤー分離による保守性の向上
- **TDD**: 包括的なテストスイート
- **Google Try-on API**: 最新のバーチャル試着技術
- **Docker対応**: コンテナ化されたデプロイメント
- **Cloud Run対応**: スケーラブルなサーバーレス実行

## アーキテクチャ

```
internal/
├── domain/          # エンティティ、値オブジェクト、リポジトリインターフェース
├── usecase/         # アプリケーションロジック
├── infrastructure/ # 外部依存実装（GenAI、画像処理）
└── interface/      # ハンドラー、プレゼンター
```

## セットアップ

### 前提条件

- Go 1.24+
- Docker & Docker Compose

### クイックスタート

```bash
# セットアップスクリプト実行（推奨）
./scripts/setup-local.sh

# モックモード（認証不要、すぐに試せる）
docker compose -f compose.dev.yml --profile mock up --build
```

### 認証方法

#### 方法1: モックモード（最も簡単）
```bash
# 認証不要、テスト用のモック画像を返します
docker compose -f compose.dev.yml --profile mock up --build
# http://localhost:8081 でアクセス
```

#### 方法2: Google Cloud認証（本格利用）

##### 🎯 簡単な自動セットアップ（推奨）

```bash
# 認証設定から起動まで自動化
./scripts/run-with-auth.sh
```

##### 🔧 手動セットアップ

```bash
# ステップ1: 認証設定
gcloud auth application-default login

# ステップ2: 必要なAPIを有効化
gcloud services enable aiplatform.googleapis.com
gcloud services enable generativelanguage.googleapis.com

# ステップ3: プロジェクトID設定
export PROJECT_ID=your-project-id

# ステップ4: 起動
docker compose -f compose.auth.yml up --build
# アクセス: http://localhost:8082
```

##### 🔥 Google Cloud SDK内蔵版（最も確実）

```bash
# ホスト側で認証（一度のみ）
gcloud auth application-default login
export PROJECT_ID=your-project-id

# Google Cloud SDK内蔵版でテスト
./scripts/test-gcloud.sh
# アクセス: http://localhost:8083
```

##### 🗝️ サービスアカウントキー使用

```bash
# 1. Google Cloud Consoleでサービスアカウントキー作成
# 2. キーファイルをダウンロード
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account-key.json
export PROJECT_ID=your-project-id
docker compose -f compose.auth.yml up --build
```

### 開発環境

```bash
# ホットリロード付き開発
docker compose -f compose.dev.yml up --build

# ローカル直接実行
go run main.go

# テスト実行
go test ./test/...
```

### 本番環境

```bash
# nginx付き本番環境
docker compose --profile production up --build
```

## API仕様

### POST /tryon

バーチャル試着を実行します。

**Request:**
- `person_image`: 人物画像ファイル (multipart/form-data)
- `garment_image`: 衣服画像ファイル (multipart/form-data)

**Response:**
- 成功: バーチャル試着結果画像 (image/jpeg)
- エラー: JSON形式のエラーメッセージ

**制限:**
- 各画像ファイルは最大25MB
- API呼び出し頻度制限: 10回/分（nginx使用時）

### GET /healthz

ヘルスチェックエンドポイント

## デプロイ

### Cloud Run

```bash
# プロジェクトIDを設定
gcloud config set project YOUR_PROJECT_ID

# コンテナをCloud Runにデプロイ
gcloud run deploy tryon-demo \
  --source . \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --set-env-vars PROJECT_ID=YOUR_PROJECT_ID,LOCATION=us-central1 \
  --memory 2Gi \
  --timeout 300s
```

## 環境変数

| 変数名 | 説明 | デフォルト値 |
|--------|------|-------------|
| PROJECT_ID | Google CloudプロジェクトID | 必須 |
| LOCATION | リージョン | us-central1 |
| VTO_MODEL | 使用するモデル | virtual-try-on-preview-08-04 |
| GOOGLE_APPLICATION_CREDENTIALS | サービスアカウントキーのパス | 必須 |
| PORT | サーバーポート | 8080 |

## 開発

### テスト実行

```bash
# 全テスト実行
go test ./test/...

# カバレッジ付き
go test -cover ./test/...

# 特定のテスト
go test ./test/domain/
```

### コードフォーマット

```bash
go fmt ./...
go vet ./...
```

## トラブルシューティング

### 認証エラー

```bash
# サービスアカウントキーの確認
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json
gcloud auth application-default print-access-token
```

### メモリエラー

Cloud Runでメモリ不足が発生する場合は、メモリ制限を増やしてください：

```bash
gcloud run services update tryon-demo --memory 4Gi
```