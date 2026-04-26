package entities

import "time"

// User is a registered account in Aura. PasswordHash is intentionally
// dropped from JSON (`json:"-"`) so it never leaks through any handler
// that decides to serialise a User directly.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
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
	AccountID          int64           `json:"account_id"`
	UserID             string          `json:"user_id"`
	ServiceName        ExternalService `json:"service_name"`
	ExternalUserID     string          `json:"external_user_id"`
	ExternalProfileURL string          `json:"external_profile_url,omitempty"`
	LastSyncedAt       *time.Time      `json:"last_synced_at"`
}

// UserProfile is the aggregated preferences snapshot used by ranking rules.
type UserProfile struct {
	UserID              string      `json:"user_id"`
	MeanRating          float64     `json:"mean_rating"`
	RatingVariance      float64     `json:"rating_variance"`
	PreferredGenres     []string    `json:"preferred_genres,omitempty"`
	PreferredMediaTypes []MediaType `json:"preferred_media_types,omitempty"`
}
