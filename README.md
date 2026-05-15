# Google Generative AI Demo

Google Cloud Vertex AI / Gemini API の生成AI機能を試せる統合デモアプリケーションです。バーチャル試着・画像生成・動画生成・画像編集の4機能を、それぞれ独立した画面とAPIで提供します。

## プロジェクト概要

Google の各種生成AIモデルを1つのGoアプリケーションから呼び出せるデモです。Clean Architecture を採用し、機能ごとにユースケース・ドメインサービス・外部API接続を分離しています。

### 提供機能

| 機能 | 画面 / エンドポイント | 内容 |
|------|----------------------|------|
| バーチャル試着 (Virtual Try-On) | `GET /` / `POST /tryon` | 人物画像と衣服画像（複数可）を組み合わせて試着結果を生成 |
| 画像生成 (Imagen) | `GET /imagen` / `POST /imagen` | テキストプロンプトから画像を生成 |
| 動画生成 (Veo) | `GET /veo` / `POST /veo` | 初期画像とプロンプトから動画を生成 |
| 画像編集 (Nano Banana) | `GET /nanobanana/image-editing` / `POST /nanobanana/image-editing` | 画像（複数可）とプロンプトから編集後画像を生成 |

## 技術スタック

- **言語**: Go 1.24.2
- **アーキテクチャ**: Clean Architecture
- **開発手法**: TDD (Test-Driven Development)
- **フレームワーク / SDK**:
  - Gorilla Mux (HTTPルーター)
  - Google Cloud Vertex AI / Gen AI SDK
- **インフラ**: Docker, Google Cloud Run, Cloud Build, Artifact Registry
- **開発ツール**: Air (ホットリロード), Docker Compose

## 利用モデル

| 機能 | 対応モデル | デフォルト |
|------|-----------|-----------|
| Virtual Try-On | `virtual-try-on-preview-08-04` | `virtual-try-on-preview-08-04` |
| Imagen | `imagen-4.0-ultra-generate-001` / `imagen-4.0-fast-generate-001` / `imagen-4.0-generate-001` / `imagen-3.0-generate-002` | `imagen-3.0-generate-002` |
| Veo | `veo-3.0-generate-preview` / `veo-3.0-fast-generate-preview` / `veo-2.0-generate-001` | `veo-3.0-generate-preview` |
| Nano Banana | `gemini-2.5-flash-image-preview` | `gemini-2.5-flash-image-preview` |

## アーキテクチャ

Clean Architecture パターンを採用し、レイヤー分離による保守性と拡張性を実現しています。機能（TryOn / Imagen / Veo / Nanobanana）ごとにエンティティ・ユースケース・ドメインサービス・外部API実装を分割しています。

```plaintext
internal/
├── domain/              # ドメイン層
│   ├── entities/        # エンティティ（tryon / imagen / veo / nanobanana / gemini ...）
│   ├── valueobjects/    # 値オブジェクト（image / video / parameters）
│   ├── repositories/    # リポジトリ・サービスインターフェース
│   └── services/        # ドメインサービス（機能別）
├── application/         # アプリケーション層
│   ├── usecases/        # ユースケース（機能別ビジネスロジック）
│   └── services/        # アプリケーションサービス（パラメータ解析等）
└── infrastructure/      # インフラストラクチャ層
    ├── api/             # HTTPハンドラー（機能別）
    ├── external/        # 外部API接続（Vertex AI / Gemini）
    ├── repositories/    # データ永続化実装
    └── services/        # クライアントプール等
```

## セットアップ

### 前提条件

- Go 1.24.2+
- Docker & Docker Compose
- Google Cloud SDK（ローカル認証に使用）
- **重要**: Vertex AI リージョンは `us-central1` を使用してください
  - `asia-northeast1` では Virtual Try-On API が利用できません
- 以下の API を有効化した API キーを作成
  - Vertex AI API
  - Generative Language API
- Artifact Registry に `us-central1` で `genai-playground` リポジトリを作成（Cloud Build デプロイ時）

### 設定ファイル

`config.yaml` で接続情報を指定します。存在しない場合は `config.example.yaml` から自動生成されます。

```yaml
location: us-central1
vto_model: virtual-try-on-preview-08-04
api_key: <Gemini / Generative Language API キー>
# GCS保存先URI
gcs_uri: gs://your-gcs-bucket
```

`PROJECT_ID` は Google Cloud 認証情報（`gcloud config`）から自動取得されます。

### 実行方法

```bash
# 認証確認から起動まで自動化（Docker Compose で起動）
./scripts/run-local.sh
```

このスクリプトが自動的に以下を行います：

- Google Cloud 認証（ADC）状態の確認・設定
- プロジェクト / アカウントの選択・切り替え
- `config.yaml` の読み込みと環境変数の設定
- Docker Compose によるアプリケーションの起動（ホットリロード対応）

起動後、各機能には以下の URL からアクセスできます。

- バーチャル試着: `http://localhost:<port>/`
- 画像生成: `http://localhost:<port>/imagen`
- 動画生成: `http://localhost:<port>/veo`
- 画像編集: `http://localhost:<port>/nanobanana/image-editing`

### テスト

```bash
# 全テスト実行
go test ./...

# カバレッジ付きテスト
go test -cover ./...

# 特定パッケージのテスト実行
go test ./internal/domain/...
```

### コード品質

```bash
# フォーマット
go fmt ./...

# 静的解析
go vet ./...
```

### デプロイ

`cloudbuild.yaml` により、Cloud Build でイメージをビルドして Artifact Registry に push し、Cloud Run（サービス名: `tryon-demo`）へデプロイします。リージョン・サービスアカウント等は Cloud Build の代入変数（`_PROJECT_REGION`, `_SERVICE_ACCOUNT` 等）で指定します。

## API仕様

### POST /tryon

バーチャル試着を実行します。

**Request (multipart/form-data):**

- `person_image`: 人物画像ファイル（1枚）
- `garment_image`: 衣服画像ファイル（複数指定可）

**Response:**

- 成功: JSON `{ "success": true, "images": [{ "id": "image_0", "data": "<base64>", "type": "<mime>" }] }`
- エラー: JSON `{ "error": "<メッセージ>" }`

**制限:**

- リクエストボディは最大 10MB
- API呼び出し頻度制限: 10回/分（nginx 使用時）

**コスト:**

- 生成1回につき約20円（Vertex AI Virtual Try-On API 利用料金）

### POST /imagen

テキストプロンプトから画像を生成します。

**Request (multipart/form-data):**

- `prompt`: 生成プロンプト（必須）
- `imagenModel`: 使用モデル（省略時 `imagen-3.0-generate-002`）
- `numberOfImages`: 生成枚数
- `aspectRatio`: アスペクト比（省略時 `1:1`）
- `negativePrompt`: ネガティブプロンプト
- `seed`: シード値
- `includeRaiReason`: RAI 理由を含めるか（`true` / 省略）

**Response:** JSON（base64 画像配列）／エラー時は `{ "error": ... }`

### POST /veo

初期画像とプロンプトから動画を生成します。

**Request (multipart/form-data):**

- `image`: 初期画像ファイル（1枚）
- `videoPrompt`: 動画生成プロンプト
- `imagenPrompt`: 初期画像を Imagen で生成する場合のプロンプト
- `veoModel`: 使用モデル（省略時 `veo-3.0-generate-preview`）

**Response:** JSON（base64 動画データ配列）／エラー時は `{ "error": ... }`

### POST /nanobanana/image-editing

画像とプロンプトから編集後の画像を生成します。

**Request (multipart/form-data):**

- `images`: 入力画像ファイル（複数指定可）
- `prompt`: 編集プロンプト
- `model`: 使用モデル（省略時 `gemini-2.5-flash-image-preview`）

**Response:** JSON（base64 画像配列）／エラー時は `{ "error": ... }`

**制限:**

- リクエストボディは最大 32MB

### その他のエンドポイント

| メソッド / パス | 内容 |
|----------------|------|
| `GET /healthz` | ヘルスチェック（`ok` を返す） |
| `GET /api/sample-images` | サンプル画像一覧の取得 |
| `GET /api/sample-image` | サンプル画像の取得 |
| `GET /static/...` | 静的ファイル配信 |
