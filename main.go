package main

import (
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
)

func main() {
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

	// 生成をスキップするかどうか
	isSkipGenerate := false

	// インフラ層を初期化
	vertexAIService, err := external.NewVertexAIService(projectID, location, vtoModel, useSDK)
	if err != nil {
		log.Fatalf("Failed to create Vertex AI service: %v", err)
	}
	defer vertexAIService.Close()

	tryOnRepository := repositories.NewMemoryTryOnRepository()

	// ドメイン層を初期化
	tryOnDomainService := domainservices.NewTryOnDomainService(vertexAIService)

	// アプリケーション層を初期化
	tryOnUseCase := usecases.NewTryOnUseCase(tryOnRepository, tryOnDomainService)
	parameterService := appservices.NewParameterService()

	// API層を初期化
	handler := api.NewTryOnHandler(tryOnUseCase, parameterService, isSkipGenerate, location)

	// ルートを設定
	r := mux.NewRouter()
	r.HandleFunc("/", handler.HandleIndex).Methods("GET")
	r.HandleFunc("/tryon", handler.HandleTryOn).Methods("POST")
	r.HandleFunc("/healthz", handler.HandleHealth).Methods("GET")
	r.HandleFunc("/api/sample-images", handler.HandleSampleImages).Methods("GET")

	// 静的ファイル配信 (サンプル画像用)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))

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
