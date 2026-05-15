package external

import (
	"context"
	"fmt"

	genai_std "google.golang.org/genai"

	"tryon-demo/internal/domain/entities"
	"tryon-demo/internal/domain/repositories"
	"tryon-demo/internal/domain/valueobjects"
)

type VertexAIService struct {
	vtoModel       string
	vertexAIClient *genai_std.Client
}

func NewVertexAIService(
	vtoModel string,
	vertexAIClient *genai_std.Client,
) repositories.VertexAIService {
	return &VertexAIService{
		vtoModel:       vtoModel,
		vertexAIClient: vertexAIClient,
	}
}

// GenerateTryOn は Virtual Try-On を実行する。
// google.golang.org/genai の RecontextImage（Virtual Try-On 専用 API）を利用する。
// 認証はクライアント側（client_pool_service）の Backend 設定に従う
// （Vertex express mode = API キー、または ADC）。
func (s *VertexAIService) GenerateTryOn(ctx context.Context, request *entities.TryOnRequest) (*entities.TryOnResult, error) {
	person := request.PersonImage()
	garment := request.GarmentImage()
	params := request.Parameters()

	source := &genai_std.RecontextImageSource{
		PersonImage: &genai_std.Image{
			ImageBytes: person.Data(),
			MIMEType:   imageMIME(person),
		},
		ProductImages: []*genai_std.ProductImage{
			{
				ProductImage: &genai_std.Image{
					ImageBytes: garment.Data(),
					MIMEType:   imageMIME(garment),
				},
			},
		},
	}

	config := &genai_std.RecontextImageConfig{
		NumberOfImages:    genai_std.Ptr(int32(params.SampleCount())),
		BaseSteps:         genai_std.Ptr(int32(params.BaseSteps())),
		PersonGeneration:  toGenaiPersonGeneration(params.PersonGeneration()),
		SafetyFilterLevel: toGenaiSafetyFilterLevel(params.SafetySetting()),
		OutputMIMEType:    string(params.OutputMimeType()),
	}
	if params.CompressionQuality() > 0 {
		config.OutputCompressionQuality = genai_std.Ptr(int32(params.CompressionQuality()))
	}
	// 旧REST実装と同じく、ウォーターマーク無効かつ Seed 指定時のみ Seed を送る。
	// 注: RecontextImageConfig に addWatermark に相当する項目は無いため、
	// ウォーターマークの有無はサービス側既定に従う（既知の挙動差）。
	if !params.AddWatermark() && params.Seed() > 0 {
		config.Seed = genai_std.Ptr(int32(params.Seed()))
	}

	resp, err := s.vertexAIClient.Models.RecontextImage(ctx, s.vtoModel, source, config)
	if err != nil {
		return nil, fmt.Errorf("failed to recontext image: %w", err)
	}

	var images []*valueobjects.ImageData
	for _, generated := range resp.GeneratedImages {
		if generated.Image == nil || len(generated.Image.ImageBytes) == 0 {
			continue
		}

		imageData, err := valueobjects.NewImageData(generated.Image.ImageBytes, generated.Image.MIMEType)
		if err != nil {
			continue
		}

		images = append(images, imageData)
	}

	if len(images) == 0 {
		return nil, fmt.Errorf("no valid image data found in response")
	}

	return entities.NewTryOnResult(request.ID(), images), nil
}

func (s *VertexAIService) Close() error {
	return nil
}

// imageMIME は ImageData の MIME タイプを返す。
// ToJPEG 変換後は mimeType が空になり得るため、フォーマットからフォールバックする。
func imageMIME(i *valueobjects.ImageData) string {
	if mt := i.MimeType(); mt != "" {
		return mt
	}
	if i.Format() == valueobjects.PNG {
		return "image/png"
	}
	return "image/jpeg"
}

func toGenaiPersonGeneration(p valueobjects.PersonGeneration) genai_std.PersonGeneration {
	switch p {
	case valueobjects.AllowAll:
		return genai_std.PersonGenerationAllowAll
	case valueobjects.DontAllow:
		return genai_std.PersonGenerationDontAllow
	default:
		return genai_std.PersonGenerationAllowAdult
	}
}

func toGenaiSafetyFilterLevel(s valueobjects.SafetySetting) genai_std.SafetyFilterLevel {
	switch s {
	case valueobjects.BlockLowAndAbove:
		return genai_std.SafetyFilterLevelBlockLowAndAbove
	case valueobjects.BlockOnlyHigh:
		return genai_std.SafetyFilterLevelBlockOnlyHigh
	case valueobjects.BlockNone:
		return genai_std.SafetyFilterLevelBlockNone
	default:
		return genai_std.SafetyFilterLevelBlockMediumAndAbove
	}
}
