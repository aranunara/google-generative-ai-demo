package external

import (
	"context"
	"log"
	"os"
	"time"

	genai_std "google.golang.org/genai"

	"tryon-demo/internal/domain/entities"
	"tryon-demo/internal/domain/repositories"
	"tryon-demo/internal/domain/valueobjects"
)

type VeoAIService struct {
	genAIClient *genai_std.Client
}

func NewVeoAIService(genAIClient *genai_std.Client) repositories.VeoAIService {
	return &VeoAIService{
		genAIClient: genAIClient,
	}
}

func (s *VeoAIService) GenerateVideo(
	ctx context.Context,
	request *entities.VeoRequest,
) (*entities.VeoResult, error) {
	// 画像をgenai_std.GeneratedImageに変換
	image := &genai_std.Image{
		ImageBytes: request.Images().Data(),
		MIMEType:   request.Images().MimeType(),
	}

	// 動画生成
	operation, err := s.genAIClient.Models.GenerateVideos(
		ctx,
		request.VeoModel(),
		request.VideoPrompt(),
		image,
		nil, // GenerateVideosConfig
	)
	if err != nil {
		log.Fatal(err)
	}

	// 動画生成が完了するまで待つ
	for !operation.Done {
		log.Println("Waiting for video generation to complete...")
		time.Sleep(10 * time.Second)
		operation, _ = s.genAIClient.Operations.GetVideosOperation(ctx, operation, nil)
	}

	// 動画をダウンロード
	video := operation.Response.GeneratedVideos[0]
	s.genAIClient.Files.Download(ctx, video.Video, nil)
	fname := "veo3_with_image_input.mp4"
	_ = os.WriteFile(fname, video.Video.VideoBytes, 0644)
	log.Printf("Generated video saved to %s\n", fname)

	videoData := valueobjects.NewVideoData(video.Video.VideoBytes)

	return entities.NewVeoResult(videoData), nil
}

func (s *VeoAIService) Close() error {
	if s.genAIClient != nil {
		// GenAI Clientはリソースクリーンアップ不要
		s.genAIClient = nil
	}
	return nil
}
