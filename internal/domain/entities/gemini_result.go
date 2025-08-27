package entities

import "tryon-demo/internal/domain/valueobjects"

type GeminiResult struct {
	response string

	images []*valueobjects.ImageData
}

func NewGeminiResult(response string, images []*valueobjects.ImageData) *GeminiResult {
	return &GeminiResult{
		response: response,
		images:   images,
	}
}

func (r *GeminiResult) Response() string {
	return r.response
}

func (r *GeminiResult) Images() []*valueobjects.ImageData {
	return r.images
}
