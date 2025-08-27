package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	appservices "tryon-demo/internal/application/services"
	"tryon-demo/internal/application/usecases"
	domainservices "tryon-demo/internal/domain/services"
	"tryon-demo/internal/infrastructure/api"
	"tryon-demo/internal/infrastructure/external"
	"tryon-demo/internal/infrastructure/repositories"
	"tryon-demo/internal/infrastructure/services"
)

func main() {
	geminiApiKey := os.Getenv("GEMINI_API_KEY")
	if geminiApiKey == "" {
		log.Fatal("環境変数 GEMINI_API_KEY が未設定です")
	}

	// gcsUri := os.Getenv("GCS_URI")
	// if gcsUri == "" {
	// 	log.Fatal("環境変数 GCS_URI が未設定です")
	// }

	// 環境変数から設定を取得
	projectID := os.Getenv("PROJECT_ID")
	if projectID == "" {
		log.Fatal("環境変数 PROJECT_ID が未設定です")
	}

	location := os.Getenv("LOCATION")
	if location == "" {
		location = "us-central1"
	}

	vtoModel := os.Getenv("VTO_MODEL")
	if vtoModel == "" {
		vtoModel = "virtual-try-on-preview-08-04"
	}

	useSDK := os.Getenv("USE_SDK") == "true"

	log.Printf("[boot] Using VTO_MODEL=%s", vtoModel)
	log.Printf("[boot] USE_SDK=%v (false=REST API, true=genai.Client)", useSDK)

	ctx := context.Background()

	// Client Pool Service初期化
	clientPoolService := services.NewClientPoolService(projectID, location)
	defer clientPoolService.Close()

	// VertexAI Client取得 (TryOn用)
	vertexClient, err := clientPoolService.VertexAIPool().GetVertexAIClient(ctx)
	if err != nil {
		log.Fatalf("Failed to get Vertex AI client: %v", err)
	}

	defer vertexClient.Close()

	// GenAI Client取得 (Imagen/Veo用)
	genaiClient, err := clientPoolService.GenAIPool().GetGenAIClient(ctx, geminiApiKey)
	if err != nil {
		log.Fatalf("Failed to get Gen AI client: %v", err)
	}

	// インフラ層を初期化

	// VertexAI Service初期化
	vertexAIService := external.NewVertexAIService(
		projectID, location, vtoModel, useSDK, vertexClient,
	)
	defer vertexAIService.Close()

	// Imagen AI Service初期化
	imagenAIService := external.NewImagenAIService(genaiClient)
	defer imagenAIService.Close()

	// Veo AI Service初期化
	veoAIService := external.NewVeoAIService(genaiClient)

	// リポジトリ層を初期化
	tryOnRepository := repositories.NewMemoryTryOnRepository()

	// ドメイン層を初期化
	textAIService := external.NewGeminiAIService(genaiClient)
	tryOnDomainService := domainservices.NewTryOnDomainService(vertexAIService)
	imagenDomainService := domainservices.NewImagenDomainService(imagenAIService, textAIService)
	veoDomainService := domainservices.NewVeoDomainService(veoAIService, textAIService)

	// アプリケーション層を初期化
	tryOnUseCase := usecases.NewTryOnUseCase(tryOnRepository, tryOnDomainService)
	imagenUseCase := usecases.NewImagenUseCase(imagenDomainService)
	veoUseCase := usecases.NewVeoUseCase(veoDomainService, imagenDomainService)
	parameterService := appservices.NewParameterService()

	// API層を初期化
	handler := api.NewTryOnHandler(tryOnUseCase, parameterService, location)
	imagenHandler := api.NewImagenHandler(imagenUseCase, location)
	veoHandler := api.NewVeoHandler(veoUseCase, location)

	// ルートを設定
	r := mux.NewRouter()
	r.HandleFunc("/", handler.HandleIndex).Methods("GET")
	r.HandleFunc("/tryon", handler.HandleTryOn).Methods("POST")
	r.HandleFunc("/healthz", handler.HandleHealth).Methods("GET")
	r.HandleFunc("/api/sample-images", handler.HandleSampleImages).Methods("GET")
	r.HandleFunc("/api/sample-image", handler.HandleSampleImage).Methods("GET")

	// 静的ファイル配信（CloudRunでも動作するように設定）
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))
	// Imagen関連のルート
	r.HandleFunc("/imagen", imagenHandler.HandleImagenIndex).Methods("GET")
	r.HandleFunc("/imagen", imagenHandler.HandleImagen).Methods("POST")
	// Veo関連のルート
	r.HandleFunc("/veo", veoHandler.HandleVeoIndex).Methods("GET")
	r.HandleFunc("/veo", veoHandler.HandleVeo).Methods("POST")

	// サーバーを起動
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on port %s", port)
	log.Printf("Project: %s, Location: %s, Model: %s", projectID, location, vtoModel)
	log.Printf("API Mode: %s", func() string {
		if !useSDK {
			return "REST API"
		}
		return "genai.Client"
	}())

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
