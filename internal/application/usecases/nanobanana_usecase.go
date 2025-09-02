package usecases

import (
	"context"
	"fmt"
	"tryon-demo/internal/domain/entities"
	"tryon-demo/internal/domain/repositories"
	"tryon-demo/internal/domain/valueobjects"
)

type NanobananaUseCase struct {
	nanobananaService repositories.NanobananaAIService
}

func NewNanobananaUseCase(nanobananaService repositories.NanobananaAIService) *NanobananaUseCase {
	return &NanobananaUseCase{
		nanobananaService: nanobananaService,
	}
}

type NanobananaInput struct {
	Model      string
	Prompt     string
	ImageDatas []*valueobjects.ImageData // 複数画像対応
}

type NanobananaOutput struct {
	Image    *valueobjects.ImageData
	Response string
}

func (uc *NanobananaUseCase) ModifyImage(ctx context.Context, input NanobananaInput) (*NanobananaOutput, error) {
	request := entities.NewNanobananaModifyRequestWithMultipleImages(input.Model, input.Prompt, input.ImageDatas)

	result, err := uc.nanobananaService.ModifyImage(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to modify image: %w", err)
	}

	return &NanobananaOutput{
		Image:    result.ImageData(),
		Response: result.Response(),
	}, nil
}
