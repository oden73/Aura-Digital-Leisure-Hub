package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"aura/backend/core-go/internal/domain/entities"
	dbpostgres "aura/backend/core-go/internal/infrastructure/db/postgres"

	"github.com/jackc/pgx/v5"
)

var ErrNotFound = errors.New("not found")

type UserRepo struct {
	DB *dbpostgres.Pool
}

func NewUserRepo(db *dbpostgres.Pool) *UserRepo { return &UserRepo{DB: db} }

func (r *UserRepo) Create(user entities.User) (entities.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	email := strings.ToLower(strings.TrimSpace(user.Email))
	username := strings.TrimSpace(user.Username)
	if email == "" || username == "" || user.PasswordHash == "" {
		return entities.User{}, errors.New("invalid user")
	}

	row := r.DB.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING user_id, created_at
	`, username, email, user.PasswordHash)

	var id string
	var createdAt time.Time
	if err := row.Scan(&id, &createdAt); err != nil {
		return entities.User{}, err
	}
	user.ID = id
	user.Email = email
	user.Username = username
	user.CreatedAt = createdAt
	return user, nil
}

func (r *UserRepo) GetByID(userID string) (entities.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var u entities.User
	err := r.DB.QueryRow(ctx, `
		SELECT user_id, username, email, password_hash, created_at
		FROM users
		WHERE user_id = $1
	`, userID).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return entities.User{}, ErrNotFound
	}
	return u, err
}

func (r *UserRepo) GetByEmail(email string) (entities.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var u entities.User
	err := r.DB.QueryRow(ctx, `
		SELECT user_id, username, email, password_hash, created_at
		FROM users
		WHERE email = $1
	`, strings.ToLower(strings.TrimSpace(email))).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return entities.User{}, ErrNotFound
	}
	return u, err
}

// GetProfile aggregates per-user ranking signals from the Rui matrix:
// average rating, sample variance, the genres and media types the user
// rates highest. The result is consumed by hybrid.RankingContext (for
// preference-aware ranking rules) and by cf.DefaultCoordinator.SelectStrategy
// to decide between user-based, item-based or hybrid pipelines based on
// profile density.
//
// "High" rating is defined as rating >= 7 on the 1..10 scale, which loosely
// matches the "liked" bucket regardless of the user's personal mean.
func (r *UserRepo) GetProfile(userID string) (entities.UserProfile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	profile := entities.UserProfile{UserID: userID}

	var mean, variance *float64
	err := r.DB.QueryRow(ctx, `
		SELECT AVG(rating)::float8, VAR_SAMP(rating)::float8
		FROM user_interactions
		WHERE user_id = $1 AND rating IS NOT NULL
	`, userID).Scan(&mean, &variance)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return entities.UserProfile{}, err
	}
	if mean != nil {
		profile.MeanRating = *mean
	}
	if variance != nil {
		profile.RatingVariance = *variance
	}

	rows, err := r.DB.Query(ctx, `
		SELECT bi.genre
		FROM user_interactions ui
		JOIN base_items bi ON bi.item_id = ui.item_id
		WHERE ui.user_id = $1
		  AND ui.rating IS NOT NULL
		  AND ui.rating >= 7
		  AND bi.genre IS NOT NULL
		  AND bi.genre <> ''
		GROUP BY bi.genre
		ORDER BY COUNT(*) DESC, bi.genre ASC
		LIMIT 5
	`, userID)
	if err != nil {
		return entities.UserProfile{}, err
	}
	for rows.Next() {
		var g string
		if err := rows.Scan(&g); err != nil {
			rows.Close()
			return entities.UserProfile{}, err
		}
		profile.PreferredGenres = append(profile.PreferredGenres, g)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return entities.UserProfile{}, err
	}
	rows.Close()

	rows, err = r.DB.Query(ctx, `
		SELECT bi.media_type::text
		FROM user_interactions ui
		JOIN base_items bi ON bi.item_id = ui.item_id
		WHERE ui.user_id = $1
		  AND ui.rating IS NOT NULL
		  AND ui.rating >= 7
		GROUP BY bi.media_type
		ORDER BY COUNT(*) DESC, bi.media_type ASC
		LIMIT 3
	`, userID)
	if err != nil {
		return entities.UserProfile{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var mt string
		if err := rows.Scan(&mt); err != nil {
			return entities.UserProfile{}, err
		}
		profile.PreferredMediaTypes = append(profile.PreferredMediaTypes, entities.MediaType(mt))
	}
	if err := rows.Err(); err != nil {
		return entities.UserProfile{}, err
	}

	return profile, nil
}

// LinkExternalAccount upserts the (user_id, service_name, external_user_id)
// triple into external_accounts. The (service_name, external_user_id) tuple
// is globally unique per the schema, so re-linking the same external profile
// to a different aura user re-points the row to that user (the previous
// owner loses the link). The function returns the persisted account with
// account_id and last_synced_at populated.
func (r *UserRepo) LinkExternalAccount(account entities.ExternalAccount) (entities.ExternalAccount, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if account.UserID == "" || account.ServiceName == "" || account.ExternalUserID == "" {
		return entities.ExternalAccount{}, errors.New("invalid external account")
	}

	row := r.DB.QueryRow(ctx, `
		INSERT INTO external_accounts
			(user_id, service_name, external_user_id, external_profile_url, last_synced_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (service_name, external_user_id)
		DO UPDATE SET
			user_id              = EXCLUDED.user_id,
			external_profile_url = EXCLUDED.external_profile_url,
			last_synced_at       = now()
		RETURNING account_id, last_synced_at
	`,
		account.UserID,
		account.ServiceName,
		account.ExternalUserID,
		nullString(account.ExternalProfileURL),
	)

	var lastSynced time.Time
	if err := row.Scan(&account.AccountID, &lastSynced); err != nil {
		return entities.ExternalAccount{}, err
	}
	account.LastSyncedAt = &lastSynced
	return account, nil
}

