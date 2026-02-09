package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/anamika/zenact-web/server/agent"
	"github.com/anamika/zenact-web/server/api"
	"github.com/anamika/zenact-web/server/config"
	"github.com/anamika/zenact-web/server/llm"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	llmClient := llm.NewClient(cfg.OpenRouterAPIKey, cfg.OpenRouterModel)
	ag := agent.New(cfg, llmClient)
	router := api.NewRouter(ag)

	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("Zenact server starting on %s", addr)
	log.Printf("Model: %s | Browser: headless=%v %dx%d | Max iterations: %d",
		cfg.OpenRouterModel, cfg.BrowserHeadless, cfg.BrowserWidth, cfg.BrowserHeight, cfg.MaxIterations)

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
