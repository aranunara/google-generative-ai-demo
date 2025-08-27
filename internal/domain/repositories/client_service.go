package repositories

import (
	"context"

	"cloud.google.com/go/vertexai/genai" // VertexAI用
	genai_std "google.golang.org/genai"  // 標準GenAI用
)

// AIクライアント共通設定
type AIClientConfig struct {
	ProjectID string
	Location  string
}

// VertexAI Client Pool Service
// TryOn機能で使用するVertex AI専用クライアントプール
type VertexAIClientPool interface {
	// VertexAI用クライアントを取得
	GetVertexAIClient(ctx context.Context) (*genai.Client, error)

	// リソースのクリーンアップ
	Close() error
}

// GenAI Client Pool Service
// Imagen/Veo機能で使用する標準GenAI専用クライアントプール
type GenAIClientPool interface {
	// 標準GenAI用クライアントを取得
	GetGenAIClient(ctx context.Context, isCloudBuild bool, geminiApiKey string) (*genai_std.Client, error)

	// リソースのクリーンアップ
	Close() error
}

// Client Pool Service
// 全AIクライアントプールを統合管理するサービス
type ClientPoolService interface {
	// VertexAIクライアントプールを取得
	VertexAIPool() VertexAIClientPool

	// GenAIクライアントプールを取得
	GenAIPool() GenAIClientPool

	// 設定情報を取得
	Config() *AIClientConfig

	// 全リソースのクリーンアップ
	Close() error
}
