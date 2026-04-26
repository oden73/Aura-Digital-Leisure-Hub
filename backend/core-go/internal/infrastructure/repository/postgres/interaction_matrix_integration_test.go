//go:build integration

package postgres

import (
	"sort"
	"testing"

	"aura/backend/core-go/internal/domain/entities"
	dbpostgres "aura/backend/core-go/internal/infrastructure/db/postgres"
)

// matrixFixture seeds a tiny rating matrix shared by several tests:
//
//	         itemX  itemY  itemZ
//	userA      9      6      —
//	userB      8      —      7
//	userC      —      —      —     (rated nothing)
type matrixFixture struct {
	userA, userB, userC string
	itemX, itemY, itemZ string
}

func seedMatrix(t *testing.T, pool *dbpostgres.Pool) matrixFixture {
	t.Helper()
	users := NewUserRepo(pool)
	items := NewMetadataRepo(pool)
	inter := NewInteractionRepo(pool)

	f := matrixFixture{
		userA: mustCreateUser(t, users, "ann", "ann@example.com"),
		userB: mustCreateUser(t, users, "bea", "bea@example.com"),
		userC: mustCreateUser(t, users, "cee", "cee@example.com"),
		itemX: mustSaveItem(t, items, entities.Item{Title: "x", MediaType: entities.MediaTypeBook, AverageRating: 8.7}),
		itemY: mustSaveItem(t, items, entities.Item{Title: "y", MediaType: entities.MediaTypeBook, AverageRating: 7.4}),
		itemZ: mustSaveItem(t, items, entities.Item{Title: "z", MediaType: entities.MediaTypeBook, AverageRating: 6.1}),
	}
	mustSaveInteraction(t, inter, f.userA, f.itemX, entities.InteractionStatusCompleted, 9)
	mustSaveInteraction(t, inter, f.userA, f.itemY, entities.InteractionStatusCompleted, 6)
	mustSaveInteraction(t, inter, f.userB, f.itemX, entities.InteractionStatusCompleted, 8)
	mustSaveInteraction(t, inter, f.userB, f.itemZ, entities.InteractionStatusCompleted, 7)
	return f
}

func TestInteractionMatrix_GetUserRatings(t *testing.T) {
	pool := mustGetPool(t)
	f := seedMatrix(t, pool)
	matrix := NewInteractionMatrixRepo(pool)

	got, err := matrix.GetUserRatings(f.userA)
	if err != nil {
		t.Fatalf("user ratings: %v", err)
	}
	if len(got) != 2 || got[f.itemX] != 9 || got[f.itemY] != 6 {
		t.Fatalf("unexpected userA ratings: %+v", got)
	}
	empty, err := matrix.GetUserRatings(f.userC)
	if err != nil {
		t.Fatalf("cold user: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected zero ratings for cold user, got %+v", empty)
	}
}

func TestInteractionMatrix_GetItemRatings(t *testing.T) {
	pool := mustGetPool(t)
	f := seedMatrix(t, pool)
	matrix := NewInteractionMatrixRepo(pool)

	got, err := matrix.GetItemRatings(f.itemX)
	if err != nil {
		t.Fatalf("item ratings: %v", err)
	}
	if len(got) != 2 || got[f.userA] != 9 || got[f.userB] != 8 {
		t.Fatalf("unexpected itemX ratings: %+v", got)
	}
}

func TestInteractionMatrix_MeanAndVariance(t *testing.T) {
	pool := mustGetPool(t)
	f := seedMatrix(t, pool)
	matrix := NewInteractionMatrixRepo(pool)

	mean, err := matrix.GetMeanRating(f.userA)
	if err != nil {
		t.Fatalf("mean: %v", err)
	}
	if mean != 7.5 { // (9 + 6) / 2
		t.Fatalf("mean: got %v want 7.5", mean)
	}
	variance, err := matrix.GetVariance(f.userA)
	if err != nil {
		t.Fatalf("variance: %v", err)
	}
	// Sample variance of [9, 6] = ((9-7.5)^2 + (6-7.5)^2) / (2 - 1) = 4.5
	if variance != 4.5 {
		t.Fatalf("variance: got %v want 4.5", variance)
	}

	zero, err := matrix.GetMeanRating(f.userC)
	if err != nil || zero != 0 {
		t.Fatalf("cold user mean: %v err=%v (want 0)", zero, err)
	}
}

func TestInteractionMatrix_GetCommonUsers(t *testing.T) {
	pool := mustGetPool(t)
	f := seedMatrix(t, pool)
	matrix := NewInteractionMatrixRepo(pool)

	common, err := matrix.GetCommonUsers(f.itemX, f.itemY)
	if err != nil {
		t.Fatalf("common: %v", err)
	}
	if len(common) != 1 || common[0] != f.userA {
		t.Fatalf("expected only userA in common(x,y), got %v", common)
	}
}

func TestInteractionMatrix_AllUsers(t *testing.T) {
	pool := mustGetPool(t)
	f := seedMatrix(t, pool)
	matrix := NewInteractionMatrixRepo(pool)

	all, err := matrix.AllUsers()
	if err != nil {
		t.Fatalf("all users: %v", err)
	}
	sort.Strings(all)

	want := []string{f.userA, f.userB}
	sort.Strings(want)
	if len(all) != len(want) {
		t.Fatalf("expected only rated users, got %v", all)
	}
	for i := range want {
		if all[i] != want[i] {
			t.Fatalf("mismatch at %d: %s vs %s", i, all[i], want[i])
		}
	}
	for _, u := range all {
		if u == f.userC {
			t.Fatal("cold user must not appear in AllUsers")
		}
	}
}

func TestInteractionMatrix_CandidateItemsForUser(t *testing.T) {
	pool := mustGetPool(t)
	f := seedMatrix(t, pool)
	matrix := NewInteractionMatrixRepo(pool)

	// userA rated x and y; only z must come back as candidate.
	got, err := matrix.CandidateItemsForUser(f.userA, 10)
	if err != nil {
		t.Fatalf("candidates: %v", err)
	}
	if len(got) != 1 || got[0] != f.itemZ {
		t.Fatalf("expected only itemZ, got %v", got)
	}

	// userC rated nothing → all three items are candidates ordered by rating.
	all, err := matrix.CandidateItemsForUser(f.userC, 10)
	if err != nil {
		t.Fatalf("candidates: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 candidates for cold user, got %d", len(all))
	}
	if all[0] != f.itemX {
		t.Fatalf("expected highest-rated item first (itemX), got %s", all[0])
	}
}
