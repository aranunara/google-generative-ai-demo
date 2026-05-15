package main

import (
	"context"
	"log"
	"net/http"

	"tryon-demo/internal/di"
)

func main() {
	cfg, err := di.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("[boot] Using VTO_MODEL=%s", cfg.VTOModel)
	log.Printf("[boot] USE_SDK=%v (false=REST API, true=genai.Client)", cfg.UseSDK)

	ctx := context.Background()

	c, err := di.New(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize container: %v", err)
	}
	defer c.Close()

	log.Printf("Starting server on port %s", cfg.Port)
	log.Printf("Project: %s, Location: %s, Model: %s", cfg.ProjectID, cfg.Location, cfg.VTOModel)
	log.Printf("API Mode: %s", func() string {
		if !cfg.UseSDK {
			return "REST API"
		}
		return "genai.Client"
	}())

	if err := http.ListenAndServe(":"+cfg.Port, c.Handler()); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
