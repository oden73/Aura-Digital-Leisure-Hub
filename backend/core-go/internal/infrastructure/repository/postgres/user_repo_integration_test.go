//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"

	"aura/backend/core-go/internal/domain/entities"
)

func TestUserRepo_CreateAndGetByID(t *testing.T) {
	pool := mustGetPool(t)
	repo := NewUserRepo(pool)

	created, err := repo.Create(entities.User{
		Username:     "alice",
		Email:        "Alice@Example.COM",
		PasswordHash: "deadbeef$cafebabe",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected RETURNING to populate id")
	}
	if created.Email != "alice@example.com" {
		t.Fatalf("expected email lowercased on insert, got %q", created.Email)
	}
	if created.CreatedAt.IsZero() {
		t.Fatal("expected created_at to be populated")
	}

	round, err := repo.GetByID(created.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if round.Email != "alice@example.com" || round.Username != "alice" {
		t.Fatalf("round-trip mismatch: %+v", round)
	}
	if round.PasswordHash != "deadbeef$cafebabe" {
		t.Fatalf("password_hash not persisted: %q", round.PasswordHash)
	}
}

func TestUserRepo_GetByEmail_LowercasesAndTrims(t *testing.T) {
	pool := mustGetPool(t)
	repo := NewUserRepo(pool)
	mustCreateUser(t, repo, "bob", "bob@example.com")

	got, err := repo.GetByEmail("  BOB@Example.com ")
	if err != nil {
		t.Fatalf("get by email: %v", err)
	}
	if got.Username != "bob" {
		t.Fatalf("expected lookup by case-insensitive email to find bob, got %q", got.Username)
	}
}

func TestUserRepo_GetByID_NotFound(t *testing.T) {
	pool := mustGetPool(t)
	repo := NewUserRepo(pool)
	_, err := repo.GetByID("00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUserRepo_Create_RejectsDuplicateEmail(t *testing.T) {
	pool := mustGetPool(t)
	repo := NewUserRepo(pool)
	mustCreateUser(t, repo, "carol", "carol@example.com")

	_, err := repo.Create(entities.User{
		Username:     "carol2",
		Email:        "carol@example.com",
		PasswordHash: "x$y",
	})
	if err == nil {
		t.Fatal("expected unique-violation on duplicate email")
	}
}

func TestUserRepo_GetProfile_AggregatesRatingsAndPreferences(t *testing.T) {
	pool := mustGetPool(t)
	users := NewUserRepo(pool)
	items := NewMetadataRepo(pool)
	interactions := NewInteractionRepo(pool)

	uid := mustCreateUser(t, users, "dora", "dora@example.com")

	bookID := mustSaveItem(t, items, entities.Item{
		Title:     "Solaris",
		MediaType: entities.MediaTypeBook,
		Criteria:  entities.BaseItemCriteria{Genre: "sci-fi"},
	})
	cinemaID := mustSaveItem(t, items, entities.Item{
		Title:     "Stalker",
		MediaType: entities.MediaTypeCinema,
		Criteria:  entities.BaseItemCriteria{Genre: "sci-fi"},
	})
	gameID := mustSaveItem(t, items, entities.Item{
		Title:     "Disco Elysium",
		MediaType: entities.MediaTypeGame,
		Criteria:  entities.BaseItemCriteria{Genre: "rpg"},
	})

	// Rating >= 7 counts toward preferred genres/media types.
	mustSaveInteraction(t, interactions, uid, bookID, entities.InteractionStatusCompleted, 9)
	mustSaveInteraction(t, interactions, uid, cinemaID, entities.InteractionStatusCompleted, 8)
	// Rating < 7 must NOT bump preferences.
	mustSaveInteraction(t, interactions, uid, gameID, entities.InteractionStatusCompleted, 5)

	profile, err := users.GetProfile(uid)
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}

	if profile.UserID != uid {
		t.Fatalf("user_id mismatch: %q vs %q", profile.UserID, uid)
	}
	wantMean := (9.0 + 8.0 + 5.0) / 3.0
	if abs(profile.MeanRating-wantMean) > 1e-6 {
		t.Fatalf("mean rating: got %v want %v", profile.MeanRating, wantMean)
	}
	if profile.RatingVariance <= 0 {
		t.Fatalf("expected positive sample variance, got %v", profile.RatingVariance)
	}

	if len(profile.PreferredGenres) != 1 || profile.PreferredGenres[0] != "sci-fi" {
		t.Fatalf("expected preferred genres [sci-fi], got %v", profile.PreferredGenres)
	}
	if len(profile.PreferredMediaTypes) != 2 {
		t.Fatalf("expected 2 preferred media types, got %v", profile.PreferredMediaTypes)
	}
}

func TestUserRepo_GetProfile_NoRatingsReturnsZeroes(t *testing.T) {
	pool := mustGetPool(t)
	users := NewUserRepo(pool)
	uid := mustCreateUser(t, users, "ed", "ed@example.com")

	profile, err := users.GetProfile(uid)
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if profile.MeanRating != 0 || profile.RatingVariance != 0 {
		t.Fatalf("expected zero-valued profile for cold user, got %+v", profile)
	}
	if len(profile.PreferredGenres)+len(profile.PreferredMediaTypes) != 0 {
		t.Fatalf("expected empty preferences, got %+v", profile)
	}
}

func TestUserRepo_LinkExternalAccount_UpsertsAndRebinds(t *testing.T) {
	pool := mustGetPool(t)
	users := NewUserRepo(pool)

	uidA := mustCreateUser(t, users, "fox", "fox@example.com")
	uidB := mustCreateUser(t, users, "gull", "gull@example.com")

	first, err := users.LinkExternalAccount(entities.ExternalAccount{
		UserID:             uidA,
		ServiceName:        entities.ExternalServiceSteam,
		ExternalUserID:     "76561198000000001",
		ExternalProfileURL: "https://steamcommunity.com/profiles/76561198000000001",
	})
	if err != nil {
		t.Fatalf("link first: %v", err)
	}
	if first.AccountID == 0 || first.LastSyncedAt == nil {
		t.Fatalf("expected RETURNING to populate id and last_synced_at, got %+v", first)
	}

	// Same external profile, different aura user — should re-bind.
	second, err := users.LinkExternalAccount(entities.ExternalAccount{
		UserID:         uidB,
		ServiceName:    entities.ExternalServiceSteam,
		ExternalUserID: "76561198000000001",
	})
	if err != nil {
		t.Fatalf("link second: %v", err)
	}
	if second.AccountID != first.AccountID {
		t.Fatalf("expected same account_id on conflict, got %d vs %d",
			second.AccountID, first.AccountID)
	}
	// Verify ownership transferred to uidB.
	var owner string
	if err := pool.QueryRow(
		context.Background(),
		`SELECT user_id::text FROM external_accounts WHERE account_id = $1`, second.AccountID,
	).Scan(&owner); err != nil {
		t.Fatalf("verify owner: %v", err)
	}
	if owner != uidB {
		t.Fatalf("expected ownership transferred to %s, still %s", uidB, owner)
	}
}

func TestUserRepo_LinkExternalAccount_RejectsMissingFields(t *testing.T) {
	pool := mustGetPool(t)
	users := NewUserRepo(pool)
	uid := mustCreateUser(t, users, "hen", "hen@example.com")

	cases := []entities.ExternalAccount{
		{UserID: uid, ServiceName: entities.ExternalServiceSteam},                             // no external user id
		{UserID: uid, ExternalUserID: "x"},                                                    // no service name
		{ServiceName: entities.ExternalServiceSteam, ExternalUserID: "x"},                     // no user id
	}
	for i, a := range cases {
		if _, err := users.LinkExternalAccount(a); err == nil {
			t.Fatalf("case %d: expected validation error, got nil", i)
		}
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
