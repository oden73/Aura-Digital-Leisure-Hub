package entities

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// These tests pin the JSON wire contract for entities returned to clients.
// The OpenAPI schema in backend/contracts/openapi/core-api.yaml is the
// source of truth; if a tag changes here update it there too.

func TestUser_JSONHidesPasswordHash(t *testing.T) {
	u := User{
		ID:           "u-1",
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: "argon2id$$$secret",
		CreatedAt:    time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC),
	}
	b, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(b)
	if strings.Contains(got, "argon2id") || strings.Contains(got, "password_hash") {
		t.Fatalf("password hash leaked into json: %s", got)
	}
	for _, want := range []string{`"id":"u-1"`, `"username":"alice"`, `"email":"alice@example.com"`, `"created_at":"2025-01-02T03:04:05Z"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in json, got %s", want, got)
		}
	}
}

func TestItem_JSONUsesSnakeCaseAndNullableReleaseDate(t *testing.T) {
	it := Item{
		ID:        "i-1",
		Title:     "Hyperion",
		MediaType: MediaTypeBook,
		Criteria:  BaseItemCriteria{Genre: "sci-fi"},
	}
	b, err := json.Marshal(it)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(b)
	for _, want := range []string{
		`"id":"i-1"`,
		`"title":"Hyperion"`,
		`"media_type":"book"`,
		`"release_date":null`,
		`"criteria":{"genre":"sci-fi"}`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in json, got %s", want, got)
		}
	}
	if strings.Contains(got, `"book_details"`) {
		t.Fatalf("nil book_details should be omitted, got %s", got)
	}
}

func TestItem_BookDetailsRoundTrip(t *testing.T) {
	in := Item{
		ID:        "b-1",
		Title:     "X",
		MediaType: MediaTypeBook,
		BookDetails: &BookDetails{
			Author:    "K. Vonnegut",
			PageCount: 240,
		},
	}
	b, _ := json.Marshal(in)
	var out Item
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.BookDetails == nil || out.BookDetails.Author != "K. Vonnegut" || out.BookDetails.PageCount != 240 {
		t.Fatalf("book details lost in round-trip: %+v", out.BookDetails)
	}
}

func TestInteraction_JSONContract(t *testing.T) {
	i := Interaction{
		ID:         42,
		UserID:     "u-1",
		ItemID:     "i-1",
		Status:     InteractionStatusCompleted,
		Rating:     8,
		IsFavorite: true,
		ReviewText: "great",
		UpdatedAt:  time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
	}
	b, _ := json.Marshal(i)
	got := string(b)
	for _, want := range []string{
		`"id":42`,
		`"user_id":"u-1"`,
		`"item_id":"i-1"`,
		`"status":"completed"`,
		`"rating":8`,
		`"is_favorite":true`,
		`"review_text":"great"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in json, got %s", want, got)
		}
	}

	// Zero rating must collapse so a 0 in the response is never confused
	// with a real rating.
	zero := Interaction{ID: 1, UserID: "u", ItemID: "i", Status: InteractionStatusPlanned}
	zb, _ := json.Marshal(zero)
	if strings.Contains(string(zb), `"rating"`) {
		t.Fatalf("zero rating should be omitted, got %s", zb)
	}
}

func TestExternalAccount_JSONContract(t *testing.T) {
	a := ExternalAccount{
		AccountID:      7,
		UserID:         "u-1",
		ServiceName:    ExternalServiceSteam,
		ExternalUserID: "76561",
	}
	b, _ := json.Marshal(a)
	got := string(b)
	for _, want := range []string{
		`"account_id":7`,
		`"user_id":"u-1"`,
		`"service_name":"steam"`,
		`"external_user_id":"76561"`,
		`"last_synced_at":null`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in json, got %s", want, got)
		}
	}
}
