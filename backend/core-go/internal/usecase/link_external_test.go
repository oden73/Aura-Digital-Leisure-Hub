package usecase

import (
	"errors"
	"testing"

	"aura/backend/core-go/internal/domain/entities"
)

type fakeLinkRepo struct {
	saved entities.ExternalAccount
	err   error
}

func (f *fakeLinkRepo) LinkExternalAccount(a entities.ExternalAccount) (entities.ExternalAccount, error) {
	if f.err != nil {
		return entities.ExternalAccount{}, f.err
	}
	a.AccountID = 42
	f.saved = a
	return a, nil
}

func TestLinkExternalAccount_BindsToCallerAndPersists(t *testing.T) {
	repo := &fakeLinkRepo{}
	uc := NewLinkExternalAccount(repo)

	got, err := uc.Execute("user-1", entities.ExternalAccount{
		UserID:         "ignored-user",
		ServiceName:    entities.ExternalServiceSteam,
		ExternalUserID: "76561198000000000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.saved.UserID != "user-1" {
		t.Fatalf("expected use case to overwrite user_id with caller, got %q", repo.saved.UserID)
	}
	if got.AccountID != 42 {
		t.Fatalf("expected returned account to carry persisted id, got %d", got.AccountID)
	}
}

func TestLinkExternalAccount_RejectsMissingFields(t *testing.T) {
	uc := NewLinkExternalAccount(&fakeLinkRepo{})
	if _, err := uc.Execute("", entities.ExternalAccount{
		ServiceName:    entities.ExternalServiceSteam,
		ExternalUserID: "x",
	}); err == nil {
		t.Fatal("expected error when user id is empty")
	}
	if _, err := uc.Execute("u-1", entities.ExternalAccount{}); err == nil {
		t.Fatal("expected error when service / external id are empty")
	}
}

func TestLinkExternalAccount_PropagatesRepositoryError(t *testing.T) {
	repo := &fakeLinkRepo{err: errors.New("conflict")}
	uc := NewLinkExternalAccount(repo)
	if _, err := uc.Execute("u-1", entities.ExternalAccount{
		ServiceName:    entities.ExternalServiceSteam,
		ExternalUserID: "x",
	}); err == nil {
		t.Fatal("expected error to propagate")
	}
}
