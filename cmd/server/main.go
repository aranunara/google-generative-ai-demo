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

	ctx := context.Background()

	c, err := di.New(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize container: %v", err)
	}
	defer c.Close()

	log.Printf("Starting server on port %s", cfg.Port)
	log.Printf("Project: %s, Location: %s, Model: %s", cfg.ProjectID, cfg.Location, cfg.VTOModel)

	if err := http.ListenAndServe(":"+cfg.Port, c.Handler()); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
