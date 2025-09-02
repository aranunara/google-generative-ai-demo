package entities

import "tryon-demo/internal/domain/valueobjects"

// 画像加工リクエスト
type NanobananaModifyRequest struct {
	model       string
	prompt      string
	imageDatas  []*valueobjects.ImageData // 複数画像対応
	isTranslate bool
}

func NewNanobananaModifyRequest(model string, prompt string, imageDatas []*valueobjects.ImageData) *NanobananaModifyRequest {
	if model == "" {
		// デフォルトモデル
		model = "gemini-2.5-flash-image-preview"
	}

	return &NanobananaModifyRequest{
		model:       model,
		prompt:      prompt,
		imageDatas:  imageDatas,
		isTranslate: false,
	}
}

// 複数画像対応の新しいコンストラクタ
func NewNanobananaModifyRequestWithMultipleImages(model string, prompt string, imageDatas []*valueobjects.ImageData) *NanobananaModifyRequest {
	if model == "" {
		// デフォルトモデル
		model = "gemini-2.5-flash-image-preview"
	}

	return &NanobananaModifyRequest{
		model:       model,
		prompt:      prompt,
		imageDatas:  imageDatas,
		isTranslate: false,
	}
}

func (r *NanobananaModifyRequest) IsTranslate() bool {
	return r.isTranslate
}

func (r *NanobananaModifyRequest) SetIsTranslate(isTranslate bool) {
	r.isTranslate = isTranslate
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

// 複数画像を取得するメソッド
func (r *NanobananaModifyRequest) ImageDatas() []*valueobjects.ImageData {
	return r.imageDatas
}

// 画像の数を取得するメソッド
func (r *NanobananaModifyRequest) ImageCount() int {
	if len(r.imageDatas) > 0 {
		return len(r.imageDatas)
	}
	return 0
}
