package usecases

import (
	"context"
	"fmt"

	"tryon-demo/internal/domain/entities"
	"tryon-demo/internal/domain/repositories"
	"tryon-demo/internal/domain/services"
	"tryon-demo/internal/domain/valueobjects"
)

type TryOnUseCase struct {
	tryOnRepo     repositories.TryOnRepository
	domainService *services.TryOnDomainService
}

func NewTryOnUseCase(
	tryOnRepo repositories.TryOnRepository,
	domainService *services.TryOnDomainService,
) *TryOnUseCase {
	return &TryOnUseCase{
		tryOnRepo:     tryOnRepo,
		domainService: domainService,
	}
}

type TryOnInput struct {
	PersonImageData  []byte
	PersonMimeType   string
	GarmentImageData []byte
	GarmentMimeType  string
	Parameters       *TryOnParametersInput
}

type TryOnParametersInput struct {
	AddWatermark       bool
	BaseSteps          int
	PersonGeneration   string
	SafetySetting      string
	SampleCount        int
	Seed               int
	StorageURI         string
	OutputMimeType     string
	CompressionQuality int
}

type TryOnOutput struct {
	RequestID entities.TryOnRequestID
	Images    []ImageOutput
}

type ImageOutput struct {
	Data []byte
	Type string
}

func (uc *TryOnUseCase) Execute(ctx context.Context, input TryOnInput) (*TryOnOutput, error) {
	personImage, err := valueobjects.NewImageData(input.PersonImageData, input.PersonMimeType)
	if err != nil {
		return nil, fmt.Errorf("invalid person image: %w", err)
	}

	garmentImage, err := valueobjects.NewImageData(input.GarmentImageData, input.GarmentMimeType)
	if err != nil {
		return nil, fmt.Errorf("invalid garment image: %w", err)
	}

	parameters, err := uc.convertParameters(input.Parameters)
	if err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	request, err := entities.NewTryOnRequest(personImage, garmentImage, parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if err := uc.tryOnRepo.Save(ctx, request); err != nil {
		return nil, fmt.Errorf("failed to save request: %w", err)
	}

	result, err := uc.domainService.ProcessTryOn(ctx, request)
	if err != nil {
		return nil, err
	}

	if err := uc.tryOnRepo.SaveResult(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to save result: %w", err)
	}

	output := &TryOnOutput{
		RequestID: request.ID(),
		Images:    make([]ImageOutput, len(result.Images())),
	}

	for i, img := range result.Images() {
		output.Images[i] = ImageOutput{
			Data: img.Data(),
			Type: string(parameters.OutputMimeType()),
		}
	}

	return output, nil
}

func (uc *TryOnUseCase) convertParameters(input *TryOnParametersInput) (*valueobjects.TryOnParameters, error) {
	if input == nil {
		return valueobjects.DefaultTryOnParameters(), nil
	}

	personGen := valueobjects.PersonGeneration(input.PersonGeneration)
	safetySetting := valueobjects.SafetySetting(input.SafetySetting)
	mimeType := valueobjects.MimeType(input.OutputMimeType)

	return valueobjects.NewTryOnParameters(
		input.AddWatermark,
		input.BaseSteps,
		personGen,
		safetySetting,
		input.SampleCount,
		input.Seed,
		input.StorageURI,
		mimeType,
		input.CompressionQuality,
	)
}
