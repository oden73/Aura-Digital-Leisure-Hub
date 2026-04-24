package main

import (
	"log"

	"aura/backend/core-go/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
