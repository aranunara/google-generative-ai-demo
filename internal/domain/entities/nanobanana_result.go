package entities

import "tryon-demo/internal/domain/valueobjects"

type NanobananaResult struct {
	response  string
	imageData *valueobjects.ImageData
}

func NewNanobananaResult(response string, imageData *valueobjects.ImageData) *NanobananaResult {
	return &NanobananaResult{
		response:  response,
		imageData: imageData,
	}
}

func (r *NanobananaResult) Response() string {
	return r.response
}

func (r *NanobananaResult) SetResponse(response string) {
	r.response = response
}

func (r *NanobananaResult) ImageData() *valueobjects.ImageData {
	return r.imageData
}

func (r *NanobananaResult) SetImageData(imageData *valueobjects.ImageData) {
	r.imageData = imageData
}
