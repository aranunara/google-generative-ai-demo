package repositories

import (
	"context"
	
	"tryon-demo/internal/domain/entities"
)

type TryOnRepository interface {
	Save(ctx context.Context, request *entities.TryOnRequest) error
	FindByID(ctx context.Context, id entities.TryOnRequestID) (*entities.TryOnRequest, error)
	SaveResult(ctx context.Context, result *entities.TryOnResult) error
	FindResultByRequestID(ctx context.Context, requestID entities.TryOnRequestID) (*entities.TryOnResult, error)
}