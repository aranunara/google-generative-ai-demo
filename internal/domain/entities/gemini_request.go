package entities

import "tryon-demo/internal/domain/valueobjects"

type GeminiRequest struct {
	prompt string

	images []*valueobjects.ImageData
}

func NewGeminiRequest(prompt string, images []*valueobjects.ImageData) *GeminiRequest {
	return &GeminiRequest{
		prompt: prompt,
		images: images,
	}
}

func (r *GeminiRequest) Prompt() string {
	return r.prompt
}

func (r *GeminiRequest) SetPrompt(prompt string) {
	r.prompt = prompt
}

func (r *GeminiRequest) Images() []*valueobjects.ImageData {
	return r.images
}
