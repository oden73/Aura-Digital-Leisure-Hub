package main

import (
	"log/slog"
	"os"

	"aura/backend/core-go/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		slog.Error("server_exit", "error", err)
		os.Exit(1)
	}
}
