package external

import (
	"context"
	"fmt"
	"log"
	"log/slog"
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
) ([]*entities.VeoResult, error) {
	slog.Info("GenerateVideo", "request", request)

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
		// 2025/08/28時点で、対応していないらしい：　generateAudio parameter is not supported in Gemini API
		// GenerateAudio: request.GenerateAudio(),
		&genai_std.GenerateVideosConfig{
			NumberOfVideos: 1,
		},
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

	if operation.Error != nil {
		return nil, fmt.Errorf("video generation failed: %v", operation.Error)
	}

	slog.Info("operation.Metadata", "value", operation.Metadata)
	slog.Info("operation.Response.GeneratedVideos", "counts", len(operation.Response.GeneratedVideos))

	// 動画をダウンロード
	generatedVideos := make([]*genai_std.Video, len(operation.Response.GeneratedVideos))
	veoResults := make([]*entities.VeoResult, len(operation.Response.GeneratedVideos))
	for i, video := range operation.Response.GeneratedVideos {
		generatedVideos[i] = video.Video

		// 動画をダウンロードする: genai_std.Videoはgenai_std.DownloadURIの実装を満たす。渡すことでsetVideoBytes()を通じてダウンロードされる。
		s.genAIClient.Files.Download(ctx, generatedVideos[i], nil)

		// fname := fmt.Sprintf("veo3_with_image_input_%s.mp4", time.Now().Format("20060102150405"))
		// _ = os.WriteFile(fname, generatedVideos[i].VideoBytes, 0644)
		// log.Printf("Generated video saved to %s\n", fname)

		veoResults[i] = entities.NewVeoResult(valueobjects.NewVideoData(generatedVideos[i].VideoBytes))
	}

	if len(generatedVideos) == 0 {
		return nil, fmt.Errorf("no video generated")
	}

	return veoResults, nil
}

func (s *VeoAIService) Close() error {
	if s.genAIClient != nil {
		// GenAI Clientはリソースクリーンアップ不要
		s.genAIClient = nil
	}
	return nil
}
