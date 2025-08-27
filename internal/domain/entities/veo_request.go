package entities

import "tryon-demo/internal/domain/valueobjects"

type VeoRequest struct {
	// 複数画像は非対応。動画を生成する初期画像を指定するのみ。
	images *valueobjects.ImageData

	veoModel string

	// 動画生成のプロンプト
	videoPrompt string
}

func NewVeoRequest(
	imageData *valueobjects.ImageData,
	veoModel string,
	videoPrompt string,
) *VeoRequest {
	return &VeoRequest{
		images:      imageData,
		veoModel:    veoModel,
		videoPrompt: videoPrompt,
	}
}

func (r *VeoRequest) Images() *valueobjects.ImageData {
	return r.images
}

func (r *VeoRequest) VideoPrompt() string {
	return r.videoPrompt
}

func (r *VeoRequest) SetVideoPrompt(videoPrompt string) {
	r.videoPrompt = videoPrompt
}

func (r *VeoRequest) VeoModel() string {
	return r.veoModel
}
