package entities

import "tryon-demo/internal/domain/valueobjects"

// 画像加工リクエスト
type NanobananaModifyRequest struct {
	model     string
	prompt    string
	imageData *valueobjects.ImageData
}

func NewNanobananaModifyRequest(model string, prompt string, imageData *valueobjects.ImageData) *NanobananaModifyRequest {
	if model == "" {
		// デフォルトモデル
		model = "gemini-2.5-flash-image-preview"
	}

	return &NanobananaModifyRequest{
		model:     model,
		prompt:    prompt,
		imageData: imageData,
	}
}

func (r *NanobananaModifyRequest) Model() string {
	return r.model
}

func (r *NanobananaModifyRequest) Prompt() string {
	return r.prompt
}

func (r *NanobananaModifyRequest) SetPrompt(prompt string) {
	r.prompt = prompt
}

func (r *NanobananaModifyRequest) ImageData() *valueobjects.ImageData {
	return r.imageData
}
