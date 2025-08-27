package repositories

import (
	"context"
	"fmt"
	"sync"

	"tryon-demo/internal/domain/entities"
	domainrepos "tryon-demo/internal/domain/repositories"
)

type MemoryTryOnRepository struct {
	requests map[entities.TryOnRequestID]*entities.TryOnRequest
	results  map[entities.TryOnRequestID]*entities.TryOnResult
	mu       sync.RWMutex
}

func NewMemoryTryOnRepository() domainrepos.TryOnRepository {
	return &MemoryTryOnRepository{
		requests: make(map[entities.TryOnRequestID]*entities.TryOnRequest),
		results:  make(map[entities.TryOnRequestID]*entities.TryOnResult),
	}
}

func (r *MemoryTryOnRepository) Save(ctx context.Context, request *entities.TryOnRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.requests[request.ID()] = request
	return nil
}

func (r *MemoryTryOnRepository) FindByID(ctx context.Context, id entities.TryOnRequestID) (*entities.TryOnRequest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	request, exists := r.requests[id]
	if !exists {
		return nil, fmt.Errorf("request not found: %s", id)
	}

	return request, nil
}

func (r *MemoryTryOnRepository) SaveResult(ctx context.Context, result *entities.TryOnResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.results[result.RequestID()] = result
	return nil
}

func (r *MemoryTryOnRepository) FindResultByRequestID(ctx context.Context, requestID entities.TryOnRequestID) (*entities.TryOnResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result, exists := r.results[requestID]
	if !exists {
		return nil, fmt.Errorf("result not found for request: %s", requestID)
	}

	return result, nil
}
