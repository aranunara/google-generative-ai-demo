package usecases

import (
	"context"
	"fmt"
	"sync"

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

type GarmentImageData struct {
	Data     []byte
	MimeType string
}

type TryOnInput struct {
	PersonImageData  []byte
	PersonMimeType   string
	GarmentImageData []GarmentImageData
	Parameters       *TryOnParametersInput
}

type TryOnParametersInput struct {
	AddWatermark       bool
	BaseSteps          int
	PersonGeneration   string
	SafetySetting      string
	SampleCount        int
	Seed               int
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

	var garmentImageDatas []*valueobjects.ImageData
	for _, garmentImageData := range input.GarmentImageData {
		garmentImage, err := valueobjects.NewImageData(garmentImageData.Data, garmentImageData.MimeType)
		if err != nil {
			return nil, fmt.Errorf("invalid garment image: %w", err)
		}
		garmentImageDatas = append(garmentImageDatas, garmentImage)
	}

	parameters, err := uc.convertParameters(input.Parameters)
	if err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	var wg sync.WaitGroup

	// 結果を保存するチャネル
	resultCh := make(chan *entities.TryOnResult)
	errCh := make(chan error)

	for _, garmentImage := range garmentImageDatas {
		wg.Add(1)
		go func(garmentImage *valueobjects.ImageData) {
			defer wg.Done()
			request, err := entities.NewTryOnRequest(personImage, garmentImage, parameters)
			if err != nil {
				errCh <- fmt.Errorf("failed to create request: %w", err)
				return
			}

			if err := uc.tryOnRepo.Save(ctx, request); err != nil {
				errCh <- fmt.Errorf("failed to save request: %w", err)
				return
			}

			result, err := uc.domainService.ProcessTryOn(ctx, request)
			if err != nil {
				errCh <- err
				return
			}
			resultCh <- result
		}(garmentImage)
	}

	go func() {
		wg.Wait()
		close(resultCh)
		close(errCh)
	}()

	var output *TryOnOutput

	for result := range resultCh {
		if err := uc.tryOnRepo.SaveResult(ctx, result); err != nil {
			return nil, fmt.Errorf("failed to save result: %w", err)
		}

		if output == nil {
			output = &TryOnOutput{
				RequestID: result.RequestID(),
			}
		}

		for _, img := range result.Images() {
			output.Images = append(output.Images, ImageOutput{
				Data: img.Data(),
				Type: string(parameters.OutputMimeType()),
			})
		}
	}

	for err := range errCh {
		return nil, err
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
		"", // Storage URI is removed
		mimeType,
		input.CompressionQuality,
	)
}
