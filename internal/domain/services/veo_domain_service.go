package services

import (
	"context"
	"fmt"
	"strings"

	"tryon-demo/internal/domain/entities"
	"tryon-demo/internal/domain/repositories"
)

type VeoDomainService struct {
	veoAIService repositories.VeoAIService
}

func NewVeoDomainService(veoAIService repositories.VeoAIService) *VeoDomainService {
	return &VeoDomainService{
		veoAIService: veoAIService,
	}
}

func (s *VeoDomainService) ProcessVeo(
	ctx context.Context,
	request *entities.VeoRequest,
) (*entities.VeoResult, error) {
	if err := s.validateRequest(request); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	result, err := s.veoAIService.GenerateVideo(ctx, request)
	if err != nil {
		if s.isQuotaError(err) {
			return nil, fmt.Errorf("service temporarily unavailable due to high demand: %w", err)
		}
		return nil, fmt.Errorf("veo generation failed: %w", err)
	}

	// 動画が生成されなかった場合はエラー
	if result.Video() == nil {
		return nil, fmt.Errorf("no video generated")
	}

	return result, nil
}

func (s *VeoDomainService) validateRequest(request *entities.VeoRequest) error {
	if request.Images() == nil {
		return fmt.Errorf("images are required")
	}

	return nil
}

func (s *VeoDomainService) isQuotaError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "quota exceeded") ||
		strings.Contains(errStr, "resourceexhausted")
}
