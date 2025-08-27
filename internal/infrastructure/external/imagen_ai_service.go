package external

import (
	"context"
	"fmt"
	"log/slog"

	genai_std "google.golang.org/genai"

	"tryon-demo/internal/domain/entities"
	"tryon-demo/internal/domain/repositories"
	"tryon-demo/internal/domain/valueobjects"
)

type ImagenAIService struct {
	genAIClient *genai_std.Client
}

func NewImagenAIService(genAIClient *genai_std.Client) repositories.ImagenAIService {
	return &ImagenAIService{
		genAIClient: genAIClient,
	}
}

func (s *ImagenAIService) GenerateImage(
	ctx context.Context,
	request *entities.ImagenRequest,
) (*entities.ImagenResult, error) {
	slog.Info("GenerateImage", "request", request)

	// GenerateImagesConfigを構築
	config := &genai_std.GenerateImagesConfig{
		NumberOfImages:   int32(request.NumberOfImages()),
		AspectRatio:      request.AspectRatio(),
		IncludeRAIReason: request.IncludeRaiReason(),
	}

	// NegativePromptが指定されている場合のみ設定
	if request.NegativePrompt() != "" {
		config.NegativePrompt = request.NegativePrompt()
	}

	// Seedが指定されている場合のみ設定
	if request.Seed() != 0 {
		seedValue := int32(request.Seed())
		config.Seed = &seedValue
	}

	imagenResponse, err := s.genAIClient.Models.GenerateImages(
		ctx,
		request.ImagenModel(),
		request.Prompt(),
		config,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate images: %w", err)
	}

	images := make([]*valueobjects.ImageData, len(imagenResponse.GeneratedImages))

	for i, GeneratedImages := range imagenResponse.GeneratedImages {
		image, err := valueobjects.NewImageData(
			GeneratedImages.Image.ImageBytes,
			GeneratedImages.Image.MIMEType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create image data: %w", err)
		}
		images[i] = image
	}

	return entities.NewImagenResult(images), nil
}

func (s *ImagenAIService) Close() error {
	if s.genAIClient != nil {
		// GenAI Clientはリソースクリーンアップ不要
		s.genAIClient = nil
	}
	return nil
}
