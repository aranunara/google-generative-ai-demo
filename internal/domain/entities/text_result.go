package entities

type TextResult struct {
	text string
}

func NewTextResult(text string) *TextResult {
	return &TextResult{
		text: text,
	}
}

func (r *TextResult) Text() string {
	return r.text
}
