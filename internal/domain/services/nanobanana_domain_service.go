package services

import (
	"context"
	"fmt"
	"strings"
	"tryon-demo/internal/domain/entities"
	"tryon-demo/internal/domain/repositories"
)

type NanobananaDomainService struct {
	nanobananaService repositories.NanobananaAIService
	textAIService     repositories.TextAIService
}

func NewNanobananaDomainService(
	nanobananaService repositories.NanobananaAIService,
	textAIService repositories.TextAIService,
) repositories.NanobananaAIService {
	return &NanobananaDomainService{
		nanobananaService: nanobananaService,
		textAIService:     textAIService,
	}
}

func (s *NanobananaDomainService) ModifyImage(
	ctx context.Context,
	request *entities.NanobananaModifyRequest,
) (*entities.NanobananaResult, error) {
	if err := s.validateRequest(request); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	if request.Prompt() != "" {
		textRequest := entities.NewTextRequest(request.Prompt(), request.Model())
		textResult, err := s.textAIService.TranslateToEnglish(ctx, textRequest)
		if err != nil {
			return nil, fmt.Errorf("text generation failed: %w", err)
		}

		prompt := strings.Join([]string{
			`You are an expert image editor and virtual photographer, specializing in creating high-quality, professional e-commerce product photos. Your goal is to transform the provided raw image into a polished, market-ready fashion advertisement.`,
			`**Instructions for Image Optimization:**`,
			`Analyze the provided image and perform the following edits based on the specific instructions below.`,
			`**1. User Instructions:**`,
			textResult.Text(),
			`**2. Lighting & Atmosphere:**`,
			`- Adjust the lighting to be softer and more diffused, removing harsh shadows.`,
			`- Ensure the model is naturally grounded, with the lighting creating a clear separation from the background without a "floating" effect.`,
			`**3. Background:**`,
			`- Keep the background clean and minimalist, but adjust its tone to a slightly warmer gray.`,
			`**4. Final Output:**`,
			`- The final image should be a high-resolution, professional e-commerce photo with a polished look.`,
		}, "")

		request.SetPrompt(prompt)
	}

	return s.nanobananaService.ModifyImage(ctx, request)
}

func (s *NanobananaDomainService) validateRequest(request *entities.NanobananaModifyRequest) error {
	if request.Prompt() == "" {
		return fmt.Errorf("prompt is required")
	}

	if request.ImageData() == nil {
		return fmt.Errorf("image data is required")
	}

	return nil
}
