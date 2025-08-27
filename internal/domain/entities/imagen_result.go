package entities

import "tryon-demo/internal/domain/valueobjects"

type ImagenResult struct {
	images []*valueobjects.ImageData
}

func NewImagenResult(images []*valueobjects.ImageData) *ImagenResult {
	return &ImagenResult{images: images}
}

func (r *ImagenResult) Images() []*valueobjects.ImageData {
	return r.images
}
