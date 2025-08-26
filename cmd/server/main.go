package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	"tryon-demo/internal/application/services"
	"tryon-demo/internal/application/usecases"
	domainservices "tryon-demo/internal/domain/services"
	"tryon-demo/internal/infrastructure/api"
	"tryon-demo/internal/infrastructure/external"
	"tryon-demo/internal/infrastructure/repositories"
)

func main() {
	// Get configuration from environment variables
	projectID := os.Getenv("PROJECT_ID")
	if projectID == "" {
		projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}
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

	// Initialize infrastructure layer
	vertexAIService, err := external.NewVertexAIService(projectID, location, vtoModel, useSDK)
	if err != nil {
		log.Fatalf("Failed to create Vertex AI service: %v", err)
	}
	defer vertexAIService.Close()

	tryOnRepository := repositories.NewMemoryTryOnRepository()

	// Initialize domain layer
	tryOnDomainService := domainservices.NewTryOnDomainService(vertexAIService)

	// Initialize application layer
	tryOnUseCase := usecases.NewTryOnUseCase(tryOnRepository, tryOnDomainService)
	parameterService := services.NewParameterService()

	// Initialize API layer
	handler := api.NewTryOnHandler(tryOnUseCase, parameterService)

	// Setup routes
	r := mux.NewRouter()
	r.HandleFunc("/", handler.HandleIndex).Methods("GET")
	r.HandleFunc("/tryon", handler.HandleTryOn).Methods("POST")
	r.HandleFunc("/healthz", handler.HandleHealth).Methods("GET")

	// Start server
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
