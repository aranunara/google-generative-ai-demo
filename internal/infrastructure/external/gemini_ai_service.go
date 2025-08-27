package external

import (
	"context"
	"fmt"
	"tryon-demo/internal/domain/entities"
	"tryon-demo/internal/domain/repositories"

	genai_std "google.golang.org/genai"
)

type GeminiAIService struct {
	genAIClient *genai_std.Client
}

func NewGeminiAIService(genAIClient *genai_std.Client) repositories.TextAIService {
	return &GeminiAIService{
		genAIClient: genAIClient,
	}
}

func (s *GeminiAIService) GenerateText(ctx context.Context, request *entities.TextRequest) (*entities.TextResult, error) {

	resp, err := s.genAIClient.Models.GenerateContent(ctx,
		"gemini-2.5-flash",
		genai_std.Text(request.Prompt()),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	respText := resp.Text()

	return entities.NewTextResult(respText), nil
}

func (s *GeminiAIService) TranslateToEnglish(ctx context.Context, request *entities.TextRequest) (*entities.TextResult, error) {
	translatePrompt := "Translate the following text into English. The translation should be accurate and natural in tone.\n"
	translatePrompt += "Target Text: '" + request.Prompt() + "'\n"
	translatePrompt += "English Translation:"

	resp, err := s.genAIClient.Models.GenerateContent(ctx,
		"gemini-2.5-flash",
		genai_std.Text(translatePrompt),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	respText := resp.Text()

	return entities.NewTextResult(respText), nil
}

// func (s *GeminiAIService) GenerateWithTextImage(ctx context.Context, request *entities.GeminiRequest) (*entities.GeminiResult, error) {
// 	genai_std.Part{

// 	}
// }
