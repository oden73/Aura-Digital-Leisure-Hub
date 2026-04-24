package app

import (
	"fmt"
	"net/http"

	"aura/backend/core-go/internal/config"
	httptransport "aura/backend/core-go/internal/transport/http"
)

func Run() error {
	cfg := config.Load()
	router := httptransport.NewRouter()

	addr := fmt.Sprintf("%s:%d", cfg.HTTPHost, cfg.HTTPPort)
	return http.ListenAndServe(addr, router)
}
