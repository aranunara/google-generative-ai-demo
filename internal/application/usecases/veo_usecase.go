package usecases

import (
	"context"
	"log/slog"
	"tryon-demo/internal/domain/entities"
	"tryon-demo/internal/domain/services"
	"tryon-demo/internal/domain/valueobjects"
)

type VeoUseCase struct {
	veoDomainService    *services.VeoDomainService
	imagenDomainService *services.ImagenDomainService
}

func NewVeoUseCase(
	veoDomainService *services.VeoDomainService,
	imagenDomainService *services.ImagenDomainService,
) *VeoUseCase {
	return &VeoUseCase{
		veoDomainService:    veoDomainService,
		imagenDomainService: imagenDomainService,
	}
}

type VeoInput struct {
	// 画像生成用
	ImagenPrompt  string
	ImagenModel   string
	ImageData     []byte
	ImageMimeType string

	// 動画生成用
	VideoPrompt string
	VideoModel  string
}

type VeoOutput struct {
	Video []byte
}

func (uc *VeoUseCase) Execute(ctx context.Context, input VeoInput) (*VeoOutput, error) {
	// ImageDataがnilかつImagenPromptが入力されている場合は画像生成を行う
	if input.ImageData == nil && input.ImagenPrompt != "" {
		slog.Info("Execute Image Generation", "ImagenPrompt", input.ImagenPrompt, "ImagenModel", input.ImagenModel)
		imagenRequest := entities.NewImagenRequest(input.ImagenPrompt, input.ImagenModel)
		imagenOutput, err := uc.imagenDomainService.ProcessImagen(ctx, imagenRequest)
		if err != nil {
			return nil, err
		}

		slog.Info("Successfully generated image")

		input.ImageData = imagenOutput.Images()[0].Data()
		input.ImageMimeType = string(imagenOutput.Images()[0].MimeType())
	}

	imageData, err := valueobjects.NewImageData(input.ImageData, input.ImageMimeType)
	if err != nil {
		return nil, err
	}

	// 動画生成を行う
	slog.Info("Execute Video Generation", "VideoPrompt", input.VideoPrompt, "VideoModel", input.VideoModel)
	veoRequest := entities.NewVeoRequest(imageData, input.VideoModel, input.VideoPrompt)
	veoOutput, err := uc.veoDomainService.ProcessVeo(ctx, veoRequest)
	if err != nil {
		return nil, err
	}

	videoData := valueobjects.NewVideoData(veoOutput.Video().Data())

	slog.Info("Successfully generated video")

	return &VeoOutput{
		Video: videoData.Data(),
	}, nil
}
