package postgres

import (
	"context"
	"errors"
	"time"

	dbpostgres "aura/backend/core-go/internal/infrastructure/db/postgres"

	"github.com/jackc/pgx/v5"
)

// InteractionMatrixRepo is a Postgres-backed implementation of
// cf.InteractionMatrix and cf.UserStatisticsRepository. It reads from the
// user_interactions table, treating the rating column as the Rui value.
type InteractionMatrixRepo struct {
	DB *dbpostgres.Pool
}

// NewInteractionMatrixRepo builds a repository over the connection pool.
func NewInteractionMatrixRepo(db *dbpostgres.Pool) *InteractionMatrixRepo {
	return &InteractionMatrixRepo{DB: db}
}

func (r *InteractionMatrixRepo) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 3*time.Second)
}

// GetUserRatings returns the map of item_id -> rating for the given user.
// Only rated rows (rating IS NOT NULL) are returned.
func (r *InteractionMatrixRepo) GetUserRatings(userID string) (map[string]float64, error) {
	ctx, cancel := r.ctx()
	defer cancel()

	rows, err := r.DB.Query(ctx, `
		SELECT item_id, rating
		FROM user_interactions
		WHERE user_id = $1 AND rating IS NOT NULL
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]float64)
	for rows.Next() {
		var itemID string
		var rating int
		if err := rows.Scan(&itemID, &rating); err != nil {
			return nil, err
		}
		out[itemID] = float64(rating)
	}
	return out, rows.Err()
}

// GetItemRatings returns the map of user_id -> rating for the given item.
func (r *InteractionMatrixRepo) GetItemRatings(itemID string) (map[string]float64, error) {
	ctx, cancel := r.ctx()
	defer cancel()

	rows, err := r.DB.Query(ctx, `
		SELECT user_id, rating
		FROM user_interactions
		WHERE item_id = $1 AND rating IS NOT NULL
	`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]float64)
	for rows.Next() {
		var userID string
		var rating int
		if err := rows.Scan(&userID, &rating); err != nil {
			return nil, err
		}
		out[userID] = float64(rating)
	}
	return out, rows.Err()
}

// GetMeanRating returns AVG(rating) for the user.
// When the user has no ratings the function returns 0 without error so the
// caller can treat it as "no signal" instead of a hard failure.
func (r *InteractionMatrixRepo) GetMeanRating(userID string) (float64, error) {
	ctx, cancel := r.ctx()
	defer cancel()

	var mean *float64
	err := r.DB.QueryRow(ctx, `
		SELECT AVG(rating)::float8
		FROM user_interactions
		WHERE user_id = $1 AND rating IS NOT NULL
	`, userID).Scan(&mean)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if mean == nil {
		return 0, nil
	}
	return *mean, nil
}

// GetVariance returns sample variance of the user's ratings.
func (r *InteractionMatrixRepo) GetVariance(userID string) (float64, error) {
	ctx, cancel := r.ctx()
	defer cancel()

	var variance *float64
	err := r.DB.QueryRow(ctx, `
		SELECT VAR_SAMP(rating)::float8
		FROM user_interactions
		WHERE user_id = $1 AND rating IS NOT NULL
	`, userID).Scan(&variance)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if variance == nil {
		return 0, nil
	}
	return *variance, nil
}

// GetCommonUsers returns user_ids that rated both items.
func (r *InteractionMatrixRepo) GetCommonUsers(itemI string, itemJ string) ([]string, error) {
	ctx, cancel := r.ctx()
	defer cancel()

	rows, err := r.DB.Query(ctx, `
		SELECT a.user_id
		FROM user_interactions a
		JOIN user_interactions b ON b.user_id = a.user_id
		WHERE a.item_id = $1 AND a.rating IS NOT NULL
		  AND b.item_id = $2 AND b.rating IS NOT NULL
	`, itemI, itemJ)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		out = append(out, uid)
	}
	return out, rows.Err()
}

// AllUsers returns every user_id that has rated at least one item.
// Used by the user-based recommender to enumerate similarity candidates.
func (r *InteractionMatrixRepo) AllUsers() ([]string, error) {
	ctx, cancel := r.ctx()
	defer cancel()

	rows, err := r.DB.Query(ctx, `
		SELECT DISTINCT user_id
		FROM user_interactions
		WHERE rating IS NOT NULL
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		out = append(out, uid)
	}
	return out, rows.Err()
}

// CandidateItemsForUser returns items the user has not rated yet, optionally
// filtered by media_type. Implements the candidate generation step that
// precedes scoring.
func (r *InteractionMatrixRepo) CandidateItemsForUser(userID string, limit int) ([]string, error) {
	ctx, cancel := r.ctx()
	defer cancel()

	if limit <= 0 {
		limit = 200
	}

	rows, err := r.DB.Query(ctx, `
		SELECT bi.item_id
		FROM base_items bi
		WHERE NOT EXISTS (
			SELECT 1 FROM user_interactions ui
			WHERE ui.user_id = $1 AND ui.item_id = bi.item_id
		)
		ORDER BY bi.average_rating DESC NULLS LAST, bi.updated_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
