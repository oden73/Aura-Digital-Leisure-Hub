package usecase

import (
	"errors"
	"testing"
	"time"

	"aura/backend/core-go/internal/domain/entities"
	repopostgres "aura/backend/core-go/internal/infrastructure/repository/postgres"
)

type fakeLibraryRepo struct {
	interactions []entities.Interaction
	items        []repopostgres.LibraryItem
	err          error
	gotUserID    string
	gotLimit     int
}

func (f *fakeLibraryRepo) GetUserInteractions(userID string) ([]entities.Interaction, error) {
	f.gotUserID = userID
	if f.err != nil {
		return nil, f.err
	}
	return f.interactions, nil
}

func (f *fakeLibraryRepo) GetUserLibraryItems(userID string, limit int) ([]repopostgres.LibraryItem, error) {
	f.gotUserID = userID
	f.gotLimit = limit
	if f.err != nil {
		return nil, f.err
	}
	return f.items, nil
}

type fakeContentRepo struct {
	item entities.Item
	err  error
	got  string
}

func (f *fakeContentRepo) GetItem(itemID string) (entities.Item, error) {
	f.got = itemID
	if f.err != nil {
		return entities.Item{}, f.err
	}
	return f.item, nil
}

type fakeStatsRepo struct {
	stats entities.UserStats
	err   error
	got   string
}

func (f *fakeStatsRepo) GetStats(userID string) (entities.UserStats, error) {
	f.got = userID
	if f.err != nil {
		return entities.UserStats{}, f.err
	}
	return f.stats, nil
}

type fakeInteractionRepo struct {
	saved        entities.Interaction
	saveErr      error
	saveCalled   bool
	interactions []entities.Interaction
}

func (f *fakeInteractionRepo) Save(interaction entities.Interaction) error {
	f.saveCalled = true
	f.saved = interaction
	return f.saveErr
}

func (f *fakeInteractionRepo) GetUserInteractions(userID string) ([]entities.Interaction, error) {
	return f.interactions, nil
}

type fakeInvalidator struct {
	ids []string
}

func (f *fakeInvalidator) Invalidate(id string) {
	f.ids = append(f.ids, id)
}

func TestListLibraryDelegatesToRepository(t *testing.T) {
	updatedAt := time.Now()
	repo := &fakeLibraryRepo{interactions: []entities.Interaction{{
		UserID:    "u-1",
		ItemID:    "i-1",
		Status:    entities.InteractionStatusCompleted,
		UpdatedAt: updatedAt,
	}}}
	uc := NewListLibrary(repo)

	got, err := uc.Execute("u-1")
	if err != nil {
		t.Fatalf("list library: %v", err)
	}
	if repo.gotUserID != "u-1" {
		t.Fatalf("repo called with user %q", repo.gotUserID)
	}
	if len(got) != 1 || got[0].ItemID != "i-1" || !got[0].UpdatedAt.Equal(updatedAt) {
		t.Fatalf("unexpected interactions: %+v", got)
	}
}

func TestListLibraryItemsMapsRepositoryRows(t *testing.T) {
	repo := &fakeLibraryRepo{items: []repopostgres.LibraryItem{{
		Interaction: entities.Interaction{UserID: "u-1", ItemID: "i-1"},
		Item:        entities.Item{ID: "i-1", Title: "Item"},
	}}}
	uc := NewListLibraryItems(repo)

	got, err := uc.Execute("u-1", 25)
	if err != nil {
		t.Fatalf("list library items: %v", err)
	}
	if repo.gotUserID != "u-1" || repo.gotLimit != 25 {
		t.Fatalf("repo called with user=%q limit=%d", repo.gotUserID, repo.gotLimit)
	}
	if len(got) != 1 || got[0].Interaction.ItemID != "i-1" || got[0].Item.Title != "Item" {
		t.Fatalf("unexpected items: %+v", got)
	}
}

func TestUseCasesPropagateRepositoryErrors(t *testing.T) {
	repoErr := errors.New("repo down")

	if _, err := NewListLibrary(&fakeLibraryRepo{err: repoErr}).Execute("u-1"); !errors.Is(err, repoErr) {
		t.Fatalf("list library error = %v", err)
	}
	if _, err := NewListLibraryItems(&fakeLibraryRepo{err: repoErr}).Execute("u-1", 10); !errors.Is(err, repoErr) {
		t.Fatalf("list library items error = %v", err)
	}
	if _, err := NewGetContent(&fakeContentRepo{err: repoErr}).Execute("i-1"); !errors.Is(err, repoErr) {
		t.Fatalf("get content error = %v", err)
	}
	if _, err := NewGetUserStats(&fakeStatsRepo{err: repoErr}).Execute("u-1"); !errors.Is(err, repoErr) {
		t.Fatalf("get stats error = %v", err)
	}
}

func TestGetContentDelegatesToRepository(t *testing.T) {
	repo := &fakeContentRepo{item: entities.Item{ID: "i-1", Title: "Item"}}
	got, err := NewGetContent(repo).Execute("i-1")
	if err != nil {
		t.Fatalf("get content: %v", err)
	}
	if repo.got != "i-1" || got.Title != "Item" {
		t.Fatalf("repo got %q, item %+v", repo.got, got)
	}
}

func TestGetUserStatsDelegatesToRepository(t *testing.T) {
	repo := &fakeStatsRepo{stats: entities.UserStats{TotalInteractions: 3, FavoriteCount: 1}}
	got, err := NewGetUserStats(repo).Execute("u-1")
	if err != nil {
		t.Fatalf("get stats: %v", err)
	}
	if repo.got != "u-1" || got.TotalInteractions != 3 || got.FavoriteCount != 1 {
		t.Fatalf("repo got %q, stats %+v", repo.got, got)
	}
}

func TestUpdateInteractionSavesAndInvalidatesCaches(t *testing.T) {
	repo := &fakeInteractionRepo{}
	userCache := &fakeInvalidator{}
	itemCache := &fakeInvalidator{}
	uc := NewUpdateInteraction(repo).WithCacheInvalidation(userCache, itemCache)

	err := uc.Execute("u-1", "i-1", InteractionData{
		Status:     entities.InteractionStatusCompleted,
		Rating:     9,
		IsFavorite: true,
		ReviewText: "great",
	})
	if err != nil {
		t.Fatalf("update interaction: %v", err)
	}
	if !repo.saveCalled {
		t.Fatal("expected interaction to be saved")
	}
	if repo.saved.UserID != "u-1" || repo.saved.ItemID != "i-1" ||
		repo.saved.Status != entities.InteractionStatusCompleted ||
		repo.saved.Rating != 9 || !repo.saved.IsFavorite ||
		repo.saved.ReviewText != "great" {
		t.Fatalf("saved interaction = %+v", repo.saved)
	}
	if repo.saved.UpdatedAt.IsZero() {
		t.Fatal("expected UpdatedAt to be set")
	}
	if len(userCache.ids) != 1 || userCache.ids[0] != "u-1" {
		t.Fatalf("user cache invalidations = %+v", userCache.ids)
	}
	if len(itemCache.ids) != 1 || itemCache.ids[0] != "i-1" {
		t.Fatalf("item cache invalidations = %+v", itemCache.ids)
	}
}

func TestUpdateInteractionDoesNotInvalidateOnSaveError(t *testing.T) {
	repoErr := errors.New("save failed")
	repo := &fakeInteractionRepo{saveErr: repoErr}
	userCache := &fakeInvalidator{}
	itemCache := &fakeInvalidator{}
	uc := NewUpdateInteraction(repo).WithCacheInvalidation(userCache, itemCache)

	err := uc.Execute("u-1", "i-1", InteractionData{})
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected save error, got %v", err)
	}
	if len(userCache.ids) != 0 || len(itemCache.ids) != 0 {
		t.Fatalf("unexpected invalidations: user=%+v item=%+v", userCache.ids, itemCache.ids)
	}
}
