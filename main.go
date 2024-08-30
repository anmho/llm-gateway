package main

import (
	"errors"
	"fmt"
	"github.com/anmho/prism/scope"
	"github.com/joho/godotenv"
	_ "github.com/joho/godotenv/autoload"
	"github.com/labstack/echo/v4"
	"log/slog"
	"net/http"
)

const (
	port = 8080
)

func main() {
	godotenv.Load(".env")
	mux := echo.New()

	mux.HTTPErrorHandler = func(err error, c echo.Context) {
		scope.GetLogger().Error("error: ", slog.Any("error", err))
	}
	mux.GET("/hello", func(c echo.Context) error {

		msg := "hello"

		return c.JSON(http.StatusOK, map[string]any{
			"message": "hello",
		})
	})

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
