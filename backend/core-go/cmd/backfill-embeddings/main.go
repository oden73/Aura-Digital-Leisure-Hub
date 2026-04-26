// backfill-embeddings re-publishes every item from base_items to the AI
// engine so the vector store can be populated for an existing catalog.
//
// Usage:
//
//	go run ./cmd/backfill-embeddings              # all items, default batch
//	BATCH_SIZE=500 go run ./cmd/backfill-embeddings
//
// Configuration (DATABASE_URL, AI_ENGINE_URL, AI_ENGINE_TIMEOUT_MS) is read
// from the same environment variables as the API server.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"strconv"
	"time"

	"aura/backend/core-go/internal/config"
	"aura/backend/core-go/internal/domain/entities"
	"aura/backend/core-go/internal/domain/services/embeddings"
	"aura/backend/core-go/internal/infrastructure/clients/ai_engine"
	dbpostgres "aura/backend/core-go/internal/infrastructure/db/postgres"
	repopostgres "aura/backend/core-go/internal/infrastructure/repository/postgres"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "log the items that would be embedded without calling the AI engine")
	flag.Parse()

	cfg := config.Load()

	batchSize := 200
	if v := os.Getenv("BATCH_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			batchSize = n
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := dbpostgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer db.Close()

	metadata := repopostgres.NewMetadataRepo(db)

	var publisher *embeddings.Publisher
	if !*dryRun {
		client := ai_engine.NewHTTPClient(cfg.AIEngineURL, cfg.AIEngineTimeout)
		publisher = embeddings.New(client)
	}

	processed, succeeded, failed, skipped := 0, 0, 0, 0
	start := time.Now()

	err = metadata.IterateAll(ctx, batchSize, func(item entities.Item) error {
		processed++
		text := embeddings.BuildText(item)
		if text == "" {
			skipped++
			log.Printf("skip %s: no text", item.ID)
			return nil
		}
		if *dryRun {
			succeeded++
			return nil
		}
		if err := publisher.Publish(item); err != nil {
			if errors.Is(err, embeddings.ErrNoText) {
				skipped++
				return nil
			}
			failed++
			log.Printf("publish %s: %v", item.ID, err)
			return nil
		}
		succeeded++
		if processed%100 == 0 {
			log.Printf("progress: processed=%d succeeded=%d failed=%d skipped=%d elapsed=%s",
				processed, succeeded, failed, skipped, time.Since(start))
		}
		return nil
	})
	if err != nil {
		log.Fatalf("iterate: %v", err)
	}

	log.Printf("done: processed=%d succeeded=%d failed=%d skipped=%d elapsed=%s",
		processed, succeeded, failed, skipped, time.Since(start))
}
