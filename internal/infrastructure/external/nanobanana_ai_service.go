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
	slog.Info("ModifyImage", "model", request.Model(), "prompt", request.Prompt(), "imageCount", request.ImageCount())

	parts := []*genai.Part{
		genai.NewPartFromText(request.Prompt()),
	}

	// 複数画像がある場合は複数画像を追加
	if len(request.ImageDatas()) > 0 {
		for _, imageData := range request.ImageDatas() {
			imagePart := &genai.Part{
				InlineData: &genai.Blob{
					MIMEType: imageData.MimeType(),
					Data:     imageData.Data(),
				},
			}
			parts = append(parts, imagePart)
		}
	} else {
		return nil, fmt.Errorf("image data is required")
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

	// レスポンスの詳細をログ出力
	slog.Info("Gemini API response",
		"candidatesCount", len(resultGenerateContent.Candidates),
		"partsCount", len(resultGenerateContent.Candidates[0].Content.Parts))

	for i, part := range resultGenerateContent.Candidates[0].Content.Parts {
		slog.Info("Processing part", "index", i, "hasText", part.Text != "", "hasInlineData", part.InlineData != nil)

		if part.Text != "" {
			result.SetResponse(part.Text)
			slog.Info("Set text response", "text", part.Text)
		} else if part.InlineData != nil {
			imageBytes := part.InlineData.Data
			slog.Info("Processing image data", "mimeType", part.InlineData.MIMEType, "dataSize", len(imageBytes))

			imageData, err := valueobjects.NewImageData(imageBytes, part.InlineData.MIMEType)
			if err != nil {
				return nil, fmt.Errorf("failed to create image data: %w", err)
			}
			result.SetImageData(imageData)
			slog.Info("Successfully set image data")
		}
	}

	// 最終結果の確認
	if result.ImageData() == nil {
		slog.Warn("No image data in response", "responseText", result.Response())
		return nil, fmt.Errorf("no image data received from Gemini API")
	}

	return result, nil
}
