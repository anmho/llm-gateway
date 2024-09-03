package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/anmho/prism/api"
	"github.com/anmho/prism/scope"
	"github.com/caarlos0/env/v11"
	"github.com/google/generative-ai-go/genai"
	_ "github.com/joho/godotenv/autoload"
	"github.com/sashabaranov/go-openai"
	"google.golang.org/api/option"
	"log"
	"log/slog"
	"net/http"
)

const (
	port = 8080
)

type Config struct {
	OpenAIKey   string `env:"OPENAI_KEY"`
	GoogleAIKey string `env:"GOOGLE_AI_KEY"`
}

func main() {
	var config Config

	err := env.Parse(&config)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("config: %+v", config)
	ctx := context.Background()
	openaiClient := openai.NewClient(config.OpenAIKey)

	googleClient, err := genai.NewClient(ctx, option.WithAPIKey(config.GoogleAIKey))

	if err != nil {
		log.Fatalln(err)
	}

	mux := api.MakeServer(openaiClient, googleClient)

	srv := http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	scope.GetLogger().Info("server is listening", slog.Int("port", port))
	if err := srv.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}
}
