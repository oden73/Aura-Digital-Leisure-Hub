package entities

import "time"

// User is a registered account in Aura.
type User struct {
	ID           string
	Username     string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

// ExternalService enumerates third-party providers we sync profiles with.
type ExternalService string

const (
	ExternalServiceSteam       ExternalService = "steam"
	ExternalServiceEpicGames   ExternalService = "epic_games"
	ExternalServiceKinopoisk   ExternalService = "kinopoisk"
	ExternalServiceNetflix     ExternalService = "netflix"
	ExternalServiceGoodreads   ExternalService = "goodreads"
	ExternalServiceYandexBooks ExternalService = "yandex_books"
)

// ExternalAccount links an Aura user to an external service profile.
type ExternalAccount struct {
	AccountID          int64
	UserID             string
	ServiceName        ExternalService
	ExternalUserID     string
	ExternalProfileURL string
	LastSyncedAt       *time.Time
}

// UserProfile is the aggregated preferences snapshot used by ranking rules.
type UserProfile struct {
	UserID              string
	MeanRating          float64
	RatingVariance      float64
	PreferredGenres     []string
	PreferredMediaTypes []MediaType
}
