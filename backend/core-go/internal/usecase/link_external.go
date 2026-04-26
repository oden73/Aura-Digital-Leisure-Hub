package usecase

import (
	"errors"

	"aura/backend/core-go/internal/domain/entities"
)

// LinkExternalAccountUseCase associates an external service profile (Steam,
// Goodreads, ...) with the authenticated Aura user.
type LinkExternalAccountUseCase interface {
	Execute(userID string, account entities.ExternalAccount) (entities.ExternalAccount, error)
}

// LinkExternalAccount is the production implementation backed by a user
// repository.
type LinkExternalAccount struct {
	Users interface {
		LinkExternalAccount(account entities.ExternalAccount) (entities.ExternalAccount, error)
	}
}

// NewLinkExternalAccount wires the repository.
func NewLinkExternalAccount(users interface {
	LinkExternalAccount(account entities.ExternalAccount) (entities.ExternalAccount, error)
}) *LinkExternalAccount {
	return &LinkExternalAccount{Users: users}
}

// Execute validates the request and persists the link. The userID parameter
// always wins over any user_id sent in the body — the link must belong to
// the authenticated caller.
func (u *LinkExternalAccount) Execute(
	userID string,
	account entities.ExternalAccount,
) (entities.ExternalAccount, error) {
	if userID == "" {
		return entities.ExternalAccount{}, errors.New("missing user id")
	}
	if account.ServiceName == "" || account.ExternalUserID == "" {
		return entities.ExternalAccount{}, errors.New("missing service or external user id")
	}
	account.UserID = userID
	return u.Users.LinkExternalAccount(account)
}
