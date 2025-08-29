package services

import (
	"context"
	"fmt"
	"strings"

	"tryon-demo/internal/domain/entities"
	"tryon-demo/internal/domain/repositories"
)

type ImagenDomainService struct {
	imageAIService repositories.ImagenAIService
	textAIService  repositories.TextAIService
}

func NewImagenDomainService(
	aiService repositories.ImagenAIService,
	textAIService repositories.TextAIService,
) *ImagenDomainService {
	return &ImagenDomainService{
		imageAIService: aiService,
		textAIService:  textAIService,
	}
}

func (s *ImagenDomainService) ProcessImagen(
	ctx context.Context,
	request *entities.ImagenRequest,
) (*entities.ImagenResult, error) {
	if err := s.validateRequest(request); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	// プロンプトを英語に翻訳
	textRequest := entities.NewTextRequest(request.Prompt(), request.ImagenModel())
	textResult, err := s.textAIService.TranslateToEnglish(ctx, textRequest)
	if err != nil {
		return nil, fmt.Errorf("text generation failed: %w", err)
	}

	request.SetPrompt(textResult.Text())

	result, err := s.imageAIService.GenerateImage(ctx, request)
	if err != nil {
		if s.isQuotaError(err) {
			return nil, fmt.Errorf("service temporarily unavailable due to high demand: %w", err)
		}
		return nil, fmt.Errorf("imagen generation failed: %w", err)
	}

	// 画像が生成されなかった場合はエラー
	if len(result.Images()) == 0 {
		return nil, fmt.Errorf("no images generated")
	}

	return result, nil
}

func (s *ImagenDomainService) validateRequest(request *entities.ImagenRequest) error {
	if request.Prompt() == "" {
		return fmt.Errorf("prompt is required")
	}

	return nil
}

func (s *ImagenDomainService) isQuotaError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "quota exceeded") ||
		strings.Contains(errStr, "resourceexhausted")
}
