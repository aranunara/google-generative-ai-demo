package entities

type ImagenRequest struct {
	prompt           string
	imagenModel      string
	numberOfImages   int
	aspectRatio      string
	negativePrompt   string
	seed             int64
	includeRaiReason bool
}

func NewImagenRequest(prompt, imagenModel string) *ImagenRequest {
	return &ImagenRequest{
		prompt:           prompt,
		imagenModel:      imagenModel,
		numberOfImages:   1,
		aspectRatio:      "1:1",
		negativePrompt:   "",
		seed:             0,
		includeRaiReason: false,
	}
}

func NewImagenRequestWithConfig(prompt, imagenModel string, numberOfImages int, aspectRatio, negativePrompt string, seed int64, includeRaiReason bool) *ImagenRequest {
	return &ImagenRequest{
		prompt:           prompt,
		imagenModel:      imagenModel,
		numberOfImages:   numberOfImages,
		aspectRatio:      aspectRatio,
		negativePrompt:   negativePrompt,
		seed:             seed,
		includeRaiReason: includeRaiReason,
	}
}

func (r *ImagenRequest) Prompt() string {
	return r.prompt
}

func (r *ImagenRequest) SetPrompt(prompt string) {
	r.prompt = prompt
}

func (r *ImagenRequest) ImagenModel() string {
	return r.imagenModel
}

func (r *ImagenRequest) NumberOfImages() int {
	return r.numberOfImages
}

func (r *ImagenRequest) AspectRatio() string {
	return r.aspectRatio
}

func (r *ImagenRequest) NegativePrompt() string {
	return r.negativePrompt
}

func (r *ImagenRequest) Seed() int64 {
	return r.seed
}

func (r *ImagenRequest) IncludeRaiReason() bool {
	return r.includeRaiReason
}
