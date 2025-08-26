package services

import (
	"context"
	"fmt"
	"strings"

	"tryon-demo/internal/domain/entities"
	"tryon-demo/internal/domain/repositories"
)

type TryOnDomainService struct {
	aiService repositories.AIService
}

func NewTryOnDomainService(aiService repositories.AIService) *TryOnDomainService {
	return &TryOnDomainService{
		aiService: aiService,
	}
}

func (s *TryOnDomainService) ProcessTryOn(ctx context.Context, request *entities.TryOnRequest) (*entities.TryOnResult, error) {
	if err := s.validateRequest(request); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	if err := request.PrepareImages(); err != nil {
		return nil, fmt.Errorf("image preparation failed: %w", err)
	}

	result, err := s.aiService.GenerateTryOn(ctx, request)
	if err != nil {
		if s.isQuotaError(err) {
			return nil, fmt.Errorf("service temporarily unavailable due to high demand: %w", err)
		}
		return nil, fmt.Errorf("try-on generation failed: %w", err)
	}

	if !result.HasImages() {
		return nil, fmt.Errorf("no images generated")
	}

	return result, nil
}

func (s *TryOnDomainService) validateRequest(request *entities.TryOnRequest) error {
	if request.PersonImage() == nil {
		return fmt.Errorf("person image is required")
	}

	if request.GarmentImage() == nil {
		return fmt.Errorf("garment image is required")
	}

	if request.Parameters() == nil {
		return fmt.Errorf("parameters are required")
	}

	return nil
}

func (s *TryOnDomainService) isQuotaError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "quota exceeded") ||
		strings.Contains(errStr, "resourceexhausted")
}