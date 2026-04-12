package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/NdumLab/noso/internal/llm"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:15321", "listen address for the local noso llm service")
	provider := flag.String("provider", "heuristic", "backend provider: heuristic or ollama")
	model := flag.String("model", "qwen2.5:7b-instruct", "model name when using an upstream LLM backend")
	ollamaEndpoint := flag.String("ollama-endpoint", "http://127.0.0.1:11434/api/chat", "Ollama chat API endpoint")
	timeout := flag.Duration("timeout", 10*time.Second, "upstream provider timeout")
	logPath := flag.String("log-path", "", "optional JSONL path for noso-llm request logging")
	flag.Parse()
	if *logPath == "" {
		*logPath = os.Getenv("NOSO_LLM_LOG_PATH")
	}

	metrics := llm.NewMetrics()
	requestLogger, err := llm.NewRequestLogger(*logPath)
	if err != nil {
		log.Fatalf("request logger init failed: %v", err)
	}

	var backend llm.Provider
	switch *provider {
	case "heuristic":
		backend = llm.NewHeuristicProvider()
	case "ollama":
		backend = llm.NewOllamaProvider(*ollamaEndpoint, *model, *timeout)
	default:
		log.Fatalf("unsupported provider %q", *provider)
	}
	if ollamaProvider, ok := backend.(*llm.OllamaProvider); ok {
		ollamaProvider.SetMetrics(metrics)
	}

	server := &http.Server{
		Addr:    *listen,
		Handler: llm.NewHandlerWithOptions(backend, metrics, requestLogger),
	}

	log.Printf("noso-llm listening on %s provider=%s model=%s", *listen, backend.Name(), backend.Model())
	log.Fatal(server.ListenAndServe())
}
