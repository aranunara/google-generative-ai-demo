package repositories

import (
	"context"

	"tryon-demo/internal/domain/entities"
)

type AIService interface {
	GenerateTryOn(ctx context.Context, request *entities.TryOnRequest) (*entities.TryOnResult, error)
}