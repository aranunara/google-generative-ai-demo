package entities

type TextRequest struct {
	prompt string
}

func NewTextRequest(prompt string) *TextRequest {
	return &TextRequest{
		prompt: prompt,
	}
}

func (r *TextRequest) Prompt() string {
	return r.prompt
}
