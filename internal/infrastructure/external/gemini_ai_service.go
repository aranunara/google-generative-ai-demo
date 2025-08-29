package external

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
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
	translatePrompt := ""

	// veoが含まれている場合はveoのプロンプトを生成
	if strings.Contains(request.Model(), "veo") {
		slog.Info("TranslateToEnglish", "use prompt template", "veo")
		translatePrompt = buildVeoPrompt(request.Prompt(), request.Model())
	} else if strings.Contains(request.Model(), "imagen") {
		slog.Info("TranslateToEnglish", "use prompt template", "imagen")
		translatePrompt = buildImageGenerationPrompt(request.Prompt(), request.Model())
	} else {
		slog.Info("TranslateToEnglish", "use prompt template", "default")
		translatePrompt = "Translate the following text into English. The translation should be accurate and natural in tone.\n"
		translatePrompt += "Target Text: '" + request.Prompt() + "'\n"
		translatePrompt += "English Translation:"
	}

	slog.Info("TranslateToEnglish", "translatePrompt", translatePrompt)

	resp, err := s.genAIClient.Models.GenerateContent(ctx,
		"gemini-2.5-flash",
		genai_std.Text(translatePrompt),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	respText := resp.Text()

	slog.Info("TranslateToEnglish", "after generate content", respText)

	return entities.NewTextResult(respText), nil
}

func buildVeoPrompt(inputPrompt string, model string) string {
	var sb strings.Builder

	sb.WriteString("Translate the following input text into a single, highly optimized English prompt for " + model + ", a video generation model. The output MUST BE the optimized English prompt only.\n")

	sb.WriteString("The prompt should be concise, visually evocative, and detailed, aiming to generate a compelling video sequence. If the input text is already in English, refine and enhance it to include specific visual elements, actions, atmosphere, and potential camera perspectives suitable for video generation. If the input text is in Japanese, translate it directly into such a detailed and optimized English prompt, incorporating these visual enhancements.\n")

	sb.WriteString("Strictly adhere to the following output constraints:\n")
	sb.WriteString("1. Output Only the Prompt: The entire response must consist of the optimized English prompt.\n")
	sb.WriteString("2. No Prefixes or Explanations: Do not include any prefacing phrases (e.g., \"Here is...\", \"The output is...\", \"As requested...\"), no multiple options, no explanations, no commentary, and no surrounding text whatsoever.\n")
	sb.WriteString("3. No Suggestions or Advice: Do not offer advice on which prompt is best or suggest alternatives.\n")

	sb.WriteString("Input:\n")
	sb.WriteString(inputPrompt)
	sb.WriteString("\n\n")
	sb.WriteString("Expected Output Format:\n")
	sb.WriteString("[Only the optimized English prompt]")

	return sb.String()
}

func buildImageGenerationPrompt(inputPrompt string, model string) string {
	var sb strings.Builder

	sb.WriteString("Translate the following text into an optimized English prompt for " + model + ", focusing on descriptive and evocative visual elements. ")
	sb.WriteString("The output must be a direct, concise, and visually evocative prompt suitable for directing video actions. ")
	sb.WriteString("If the input text is already in English, refine and enhance it with relevant visual details (e.g., style, composition, lighting, atmosphere) to maximize its effectiveness as a video prompt. ")
	sb.WriteString("If the input text is in Japanese, translate it directly into such a detailed and optimized English prompt. ")
	sb.WriteString("Absolutely do not provide multiple options, explanations, commentary, suggestions, or any prefacing/accompanying remarks. ")
	sb.WriteString("The final output must be only the optimized English prompt.")
	sb.WriteString("Input:\n")
	sb.WriteString(inputPrompt)
	sb.WriteString("\n\n")
	sb.WriteString("Expected Output Format:\n")
	sb.WriteString("[Highly descriptive and visually detailed English prompt for image generation]")

	return sb.String()
}
