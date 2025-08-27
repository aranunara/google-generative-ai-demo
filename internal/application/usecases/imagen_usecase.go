package usecases

import (
	"context"
	"tryon-demo/internal/domain/entities"
	"tryon-demo/internal/domain/services"
)

type ImagenUseCase struct {
	domainService *services.ImagenDomainService
}

func NewImagenUseCase(
	domainService *services.ImagenDomainService,
) *ImagenUseCase {
	return &ImagenUseCase{
		domainService: domainService,
	}
}

type ImagenInput struct {
	Prompt           string
	ImagenModel      string
	NumberOfImages   int
	AspectRatio      string
	NegativePrompt   string
	Seed             int64
	IncludeRaiReason bool
}

type ImagenOutput struct {
	Images []ImageOutput
}

func (uc *ImagenUseCase) Execute(ctx context.Context, input ImagenInput) (*ImagenOutput, error) {

	request := entities.NewImagenRequestWithConfig(
		input.Prompt,
		input.ImagenModel,
		input.NumberOfImages,
		input.AspectRatio,
		input.NegativePrompt,
		input.Seed,
		input.IncludeRaiReason,
	)

	result, err := uc.domainService.ProcessImagen(ctx, request)
	if err != nil {
		return nil, err
	}

	output := &ImagenOutput{
		Images: make([]ImageOutput, len(result.Images())),
	}

	for i, img := range result.Images() {
		output.Images[i] = ImageOutput{
			Data: img.Data(),
			Type: string(img.MimeType()),
		}
	}

	return output, nil
}
