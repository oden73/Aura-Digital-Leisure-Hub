//go:build integration

package postgres

import (
	"testing"
	"time"

	"aura/backend/core-go/internal/domain/entities"
)

func TestInteractionRepo_SaveAndGetUserInteractions(t *testing.T) {
	pool := mustGetPool(t)
	users := NewUserRepo(pool)
	items := NewMetadataRepo(pool)
	repo := NewInteractionRepo(pool)

	uid := mustCreateUser(t, users, "ivy", "ivy@example.com")
	itemID := mustSaveItem(t, items, entities.Item{Title: "Solaris", MediaType: entities.MediaTypeBook})

	now := time.Now().UTC()
	if err := repo.Save(entities.Interaction{
		UserID:     uid,
		ItemID:     itemID,
		Status:     entities.InteractionStatusInProgress,
		Rating:     7,
		IsFavorite: true,
		ReviewText: "loving it",
		UpdatedAt:  now,
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	rows, err := repo.GetUserInteractions(uid)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	r := rows[0]
	if r.Status != entities.InteractionStatusInProgress || r.Rating != 7 ||
		!r.IsFavorite || r.ReviewText != "loving it" {
		t.Fatalf("round-trip mismatch: %+v", r)
	}
}

func TestInteractionRepo_Save_UpsertsOnConflict(t *testing.T) {
	pool := mustGetPool(t)
	users := NewUserRepo(pool)
	items := NewMetadataRepo(pool)
	repo := NewInteractionRepo(pool)

	uid := mustCreateUser(t, users, "jay", "jay@example.com")
	itemID := mustSaveItem(t, items, entities.Item{Title: "Solaris", MediaType: entities.MediaTypeBook})

	now := time.Now().UTC()
	if err := repo.Save(entities.Interaction{
		UserID: uid, ItemID: itemID, Status: entities.InteractionStatusPlanned, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("first save: %v", err)
	}
	if err := repo.Save(entities.Interaction{
		UserID: uid, ItemID: itemID, Status: entities.InteractionStatusCompleted, Rating: 8, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("second save: %v", err)
	}

	rows, err := repo.GetUserInteractions(uid)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected upsert (1 row), got %d", len(rows))
	}
	if rows[0].Status != entities.InteractionStatusCompleted || rows[0].Rating != 8 {
		t.Fatalf("upsert did not refresh row: %+v", rows[0])
	}
}

func TestInteractionRepo_GetUserLibraryItems_JoinsItemAndOrdersByUpdated(t *testing.T) {
	pool := mustGetPool(t)
	users := NewUserRepo(pool)
	items := NewMetadataRepo(pool)
	repo := NewInteractionRepo(pool)

	uid := mustCreateUser(t, users, "kim", "kim@example.com")
	itemA := mustSaveItem(t, items, entities.Item{Title: "Solaris", MediaType: entities.MediaTypeBook})
	itemB := mustSaveItem(t, items, entities.Item{Title: "Stalker", MediaType: entities.MediaTypeCinema})

	older := time.Now().UTC().Add(-2 * time.Hour)
	newer := time.Now().UTC()

	mustSaveAt(t, repo, entities.Interaction{
		UserID: uid, ItemID: itemA, Status: entities.InteractionStatusCompleted, Rating: 7, UpdatedAt: older,
	})
	mustSaveAt(t, repo, entities.Interaction{
		UserID: uid, ItemID: itemB, Status: entities.InteractionStatusInProgress, Rating: 9, UpdatedAt: newer,
	})

	library, err := repo.GetUserLibraryItems(uid, 10)
	if err != nil {
		t.Fatalf("library: %v", err)
	}
	if len(library) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(library))
	}
	if library[0].Item.Title != "Stalker" || library[1].Item.Title != "Solaris" {
		t.Fatalf("expected newest-first ordering, got %q then %q",
			library[0].Item.Title, library[1].Item.Title)
	}
	if library[0].Item.MediaType != entities.MediaTypeCinema {
		t.Fatalf("media_type lost in join: %s", library[0].Item.MediaType)
	}
	if library[0].Interaction.Rating != 9 {
		t.Fatalf("interaction rating lost: %d", library[0].Interaction.Rating)
	}
}

func TestInteractionRepo_GetUserLibraryItems_LimitClamps(t *testing.T) {
	pool := mustGetPool(t)
	users := NewUserRepo(pool)
	items := NewMetadataRepo(pool)
	repo := NewInteractionRepo(pool)

	uid := mustCreateUser(t, users, "lou", "lou@example.com")
	for i := 0; i < 5; i++ {
		id := mustSaveItem(t, items, entities.Item{
			Title:     "lib-" + string(rune('a'+i)),
			MediaType: entities.MediaTypeBook,
		})
		mustSaveInteraction(t, repo, uid, id, entities.InteractionStatusPlanned, 0)
	}
	got, err := repo.GetUserLibraryItems(uid, 3)
	if err != nil {
		t.Fatalf("library: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected limit=3 to clamp, got %d", len(got))
	}
}

// mustSaveAt persists an interaction preserving the supplied UpdatedAt
// so ordering tests can pin row recency without sleeping.
func mustSaveAt(t *testing.T, repo *InteractionRepo, in entities.Interaction) {
	t.Helper()
	if err := repo.Save(in); err != nil {
		t.Fatalf("save (%s,%s): %v", in.UserID, in.ItemID, err)
	}
}
