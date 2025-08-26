# Virtual Try-On Demo

Google Cloud Vertex AI Virtual Try-On APIを利用したバーチャル試着デモアプリケーション

## プロジェクト概要

人物画像と衣服画像を組み合わせて、AIが生成するバーチャル試着結果を提供するWebアプリケーションです。Google CloudのVertex AI Virtual Try-On APIを活用し、リアルタイムでの試着体験を実現します。

## 技術スタック

- **言語**: Go 1.24.2
- **アーキテクチャ**: Clean Architecture
- **開発手法**: TDD (Test-Driven Development)
- **フレームワーク**: 
  - Gorilla Mux (HTTPルーター)
  - Google Cloud Vertex AI SDK
- **インフラ**: Docker, Google Cloud Run, nginx
- **開発ツール**: Air (ホットリロード)

## 主な機能

- **バーチャル試着**: 人物画像と衣服画像から試着結果を生成
- **RESTful API**: シンプルなAPI設計
- **ヘルスチェック**: アプリケーション稼働状況の監視
- **レート制限**: 過負荷防止のための呼び出し制限
- **エラーハンドリング**: 包括的なエラー処理と適切なレスポンス

## アーキテクチャ

Clean Architectureパターンを採用し、レイヤー分離による保守性と拡張性を実現しています。

```
internal/
├── domain/              # ドメイン層
│   ├── entities/       # エンティティ（ビジネスオブジェクト）
│   ├── valueobjects/   # 値オブジェクト
│   ├── repositories/   # リポジトリインターフェース
│   └── services/       # ドメインサービス
├── application/        # アプリケーション層
│   ├── usecases/      # ユースケース（ビジネスロジック）
│   └── services/      # アプリケーションサービス
└── infrastructure/    # インフラストラクチャ層
    ├── api/           # HTTPハンドラー
    ├── external/      # 外部API接続（Vertex AI）
    └── repositories/  # データ永続化実装
```

## セットアップ

### 前提条件

- Go 1.24.2+
- Docker & Docker Compose
- Google Cloud SDK (認証使用時)

### 実行方法

```bash
# 認証設定から起動まで自動化
./scripts/run-local.sh
```

このスクリプトが自動的に以下を行います：

- Google Cloud認証の確認・設定
- プロジェクトIDの設定
- 必要な環境変数の設定
- アプリケーションの起動

### テスト

```bash
# 全テスト実行
go test ./test/...

# カバレッジ付きテスト
go test -cover ./test/...

# 特定のテスト実行
go test ./test/domain/
```

### コード品質

```bash
# フォーマット
go fmt ./...

# 静的解析
go vet ./...
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