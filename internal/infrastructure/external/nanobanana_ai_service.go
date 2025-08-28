package external

import (
	"context"
	"fmt"
	"log/slog"
	"tryon-demo/internal/domain/entities"
	"tryon-demo/internal/domain/repositories"
	"tryon-demo/internal/domain/valueobjects"

	"google.golang.org/genai"
)

type NanobananaAIService struct {
	genAIClient *genai.Client
}

func NewNanobananaAIService(genAIClient *genai.Client) repositories.NanobananaAIService {
	return &NanobananaAIService{
		genAIClient: genAIClient,
	}
}

func (s *NanobananaAIService) ModifyImage(ctx context.Context, request *entities.NanobananaModifyRequest) (*entities.NanobananaResult, error) {
	slog.Info("ModifyImage", "model", request.Model(), "prompt", request.Prompt())
	imageParts := &genai.Part{
		InlineData: &genai.Blob{
			MIMEType: request.ImageData().MimeType(),
			Data:     request.ImageData().Data(),
		},
	}

	parts := []*genai.Part{
		genai.NewPartFromText(request.Prompt()),
		imageParts,
	}

	contents := []*genai.Content{
		genai.NewContentFromParts(parts, genai.RoleUser),
	}

	// 2025/08/28時点で、「gemini-2.5-flash-image-preview」は、複数候補を返せないようになっている。
	// 2025/08/28 04:04:36 Error executing Nanobanana use case: failed to modify image: failed to generate content: Error 400, Message: Multiple candidates is not enabled for models/gemini-2.5-flash-image-preview, Status: INVALID_ARGUMENT, Details: []
	// MediaResolutionの指定も不可
	// Media resolution is not enabled for models/gemini-2.5-flash-image-preview,
	resultGenerateContent, errGenerateContent := s.genAIClient.Models.GenerateContent(
		ctx,
		request.Model(),
		contents,
		&genai.GenerateContentConfig{},
	)

	if errGenerateContent != nil {
		return nil, fmt.Errorf("failed to generate content: %w", errGenerateContent)
	}

	result := entities.NewNanobananaResult("", nil)

	for _, part := range resultGenerateContent.Candidates[0].Content.Parts {
		if part.Text != "" {
			result.SetResponse(part.Text)
		} else if part.InlineData != nil {
			imageBytes := part.InlineData.Data
			imageData, err := valueobjects.NewImageData(imageBytes, part.InlineData.MIMEType)
			if err != nil {
				return nil, fmt.Errorf("failed to create image data: %w", err)
			}
			result.SetImageData(imageData)
		}
	}

	return result, nil
}
