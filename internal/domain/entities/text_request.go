package entities

type TextRequest struct {
	prompt string

	// 対象とするモデル
	model string
}

func NewTextRequest(prompt string, model string) *TextRequest {
	return &TextRequest{
		prompt: prompt,
		model:  model,
	}
}

func (r *TextRequest) Prompt() string {
	return r.prompt
}

func (r *TextRequest) Model() string {
	return r.model
}
