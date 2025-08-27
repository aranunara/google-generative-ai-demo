package repositories

import (
	"context"

	"tryon-demo/internal/domain/entities"
)

// Vertex AIサービス
type VertexAIService interface {
	GenerateTryOn(ctx context.Context, request *entities.TryOnRequest) (*entities.TryOnResult, error)

	Close() error
}

// Veo（動画生成）サービス
type VeoAIService interface {
	GenerateVideo(ctx context.Context, request *entities.VeoRequest) (*entities.VeoResult, error)

	Close() error
}

// Imagen（画像生成）サービス
type ImagenAIService interface {
	GenerateImage(ctx context.Context, request *entities.ImagenRequest) (*entities.ImagenResult, error)

	Close() error
}
