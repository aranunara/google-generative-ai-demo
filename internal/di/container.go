package di

import (
	"context"
	"fmt"
	"net/http"

	appservices "tryon-demo/internal/application/services"
	"tryon-demo/internal/application/usecases"
	domainservices "tryon-demo/internal/domain/services"
	"tryon-demo/internal/infrastructure/api"
	"tryon-demo/internal/infrastructure/external"
	"tryon-demo/internal/infrastructure/repositories"
	"tryon-demo/internal/infrastructure/services"
)

// Container は依存関係を組み立てる合成ルート（DIコンテナ）。
type Container struct {
	tryOnHandler      *api.TryOnHandler
	imagenHandler     *api.ImagenHandler
	veoHandler        *api.VeoHandler
	nanobananaHandler *api.NanobananaHandler

	// closeFns はリソース解放処理を defer 登録順の LIFO で保持する。
	closeFns []func()
}

// New は設定をもとにインフラ〜API層を組み立て Container を返す。
// 途中で失敗した場合は生成済みリソースを解放してから error を返す。
func New(ctx context.Context, cfg *Config) (*Container, error) {
	// Client Pool Service初期化
	clientPoolService := services.NewClientPoolService(cfg.ProjectID, cfg.Location)

	// VertexAI Client取得 (TryOn用)
	vertexClient, err := clientPoolService.VertexAIPool().GetVertexAIClient(ctx)
	if err != nil {
		clientPoolService.Close()
		return nil, fmt.Errorf("failed to get Vertex AI client: %w", err)
	}

	// GenAI Client取得 (Imagen/Veo用)
	genaiClient, err := clientPoolService.GenAIPool().GetGenAIClient(ctx, cfg.GeminiAPIKey)
	if err != nil {
		vertexClient.Close()
		clientPoolService.Close()
		return nil, fmt.Errorf("failed to get Gen AI client: %w", err)
	}

	// インフラ層を初期化

	// VertexAI Service初期化
	vertexAIService := external.NewVertexAIService(
		cfg.ProjectID, cfg.Location, cfg.VTOModel, cfg.UseSDK, vertexClient,
	)

	// Imagen AI Service初期化
	imagenAIService := external.NewImagenAIService(genaiClient)

	// Veo AI Service初期化
	veoAIService := external.NewVeoAIService(genaiClient)

	// Nanobanana AI Service初期化
	nanobananaAIService := external.NewNanobananaAIService(genaiClient)

	// リポジトリ層を初期化
	tryOnRepository := repositories.NewMemoryTryOnRepository()

	// ドメイン層を初期化
	textAIService := external.NewGeminiAIService(genaiClient)
	tryOnDomainService := domainservices.NewTryOnDomainService(vertexAIService)
	imagenDomainService := domainservices.NewImagenDomainService(imagenAIService, textAIService)
	veoDomainService := domainservices.NewVeoDomainService(veoAIService, textAIService)
	nanobananaDomainService := domainservices.NewNanobananaDomainService(nanobananaAIService, textAIService)

	// アプリケーション層を初期化
	tryOnUseCase := usecases.NewTryOnUseCase(tryOnRepository, tryOnDomainService)
	imagenUseCase := usecases.NewImagenUseCase(imagenDomainService)
	veoUseCase := usecases.NewVeoUseCase(veoDomainService, imagenDomainService)
	nanobananaUseCase := usecases.NewNanobananaUseCase(nanobananaDomainService)
	parameterService := appservices.NewParameterService()

	return &Container{
		tryOnHandler:      api.NewTryOnHandler(tryOnUseCase, parameterService, cfg.Location),
		imagenHandler:     api.NewImagenHandler(imagenUseCase, cfg.Location),
		veoHandler:        api.NewVeoHandler(veoUseCase, cfg.Location),
		nanobananaHandler: api.NewNanobananaHandler(nanobananaUseCase, cfg.Location),
		// defer 登録順 (clientPool→vertexClient→vertexAI→imagenAI) の
		// 逆順 = LIFO で解放する。
		closeFns: []func(){
			func() { imagenAIService.Close() },
			func() { vertexAIService.Close() },
			func() { vertexClient.Close() },
			func() { clientPoolService.Close() },
		},
	}, nil
}

// Close は組み立てたリソースを LIFO 順で解放する。
func (c *Container) Close() {
	for _, fn := range c.closeFns {
		fn()
	}
}

// Handler はルーティングを構築した http.Handler を返す。
func (c *Container) Handler() http.Handler {
	r := http.NewServeMux()
	r.HandleFunc("GET /{$}", c.tryOnHandler.HandleIndex)
	r.HandleFunc("POST /tryon", c.tryOnHandler.HandleTryOn)
	r.HandleFunc("GET /healthz", c.tryOnHandler.HandleHealth)
	r.HandleFunc("GET /api/sample-images", c.tryOnHandler.HandleSampleImages)
	r.HandleFunc("GET /api/sample-image", c.tryOnHandler.HandleSampleImage)

	// 静的ファイル配信（CloudRunでも動作するように設定）
	r.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))
	// Imagen関連のルート
	r.HandleFunc("GET /imagen", c.imagenHandler.HandleImagenIndex)
	r.HandleFunc("POST /imagen", c.imagenHandler.HandleImagen)
	// Veo関連のルート
	r.HandleFunc("GET /veo", c.veoHandler.HandleVeoIndex)
	r.HandleFunc("POST /veo", c.veoHandler.HandleVeo)

	// Nanobanana関連のルート
	r.HandleFunc("GET /nanobanana/image-editing", c.nanobananaHandler.HandleNanobananaIndex)
	r.HandleFunc("POST /nanobanana/image-editing", c.nanobananaHandler.HandleNanobanana)

	return r
}
