package entities

import "tryon-demo/internal/domain/valueobjects"

type VeoResult struct {
	video *valueobjects.VideoData
}

func NewVeoResult(video *valueobjects.VideoData) *VeoResult {
	return &VeoResult{video: video}
}

func (r *VeoResult) Video() *valueobjects.VideoData {
	return r.video
}
