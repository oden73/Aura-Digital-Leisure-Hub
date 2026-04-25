package postgres

import (
	"context"
	"time"

	"aura/backend/core-go/internal/domain/entities"
	dbpostgres "aura/backend/core-go/internal/infrastructure/db/postgres"
)

type InteractionRepo struct {
	DB *dbpostgres.Pool
}

func NewInteractionRepo(db *dbpostgres.Pool) *InteractionRepo { return &InteractionRepo{DB: db} }

func (r *InteractionRepo) Save(i entities.Interaction) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := r.DB.Exec(ctx, `
		INSERT INTO user_interactions (user_id, item_id, status, rating, is_favorite, review_text, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, item_id)
		DO UPDATE SET
			status = EXCLUDED.status,
			rating = EXCLUDED.rating,
			is_favorite = EXCLUDED.is_favorite,
			review_text = EXCLUDED.review_text,
			updated_at = EXCLUDED.updated_at
	`, i.UserID, i.ItemID, i.Status, nullInt(i.Rating), i.IsFavorite, nullString(i.ReviewText), i.UpdatedAt)
	return err
}

func (r *InteractionRepo) GetUserInteractions(userID string) ([]entities.Interaction, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := r.DB.Query(ctx, `
		SELECT interaction_id, user_id, item_id, status, rating, is_favorite, review_text, updated_at
		FROM user_interactions
		WHERE user_id = $1
		ORDER BY updated_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []entities.Interaction
	for rows.Next() {
		var i entities.Interaction
		var rating *int
		var review *string
		if err := rows.Scan(&i.ID, &i.UserID, &i.ItemID, &i.Status, &rating, &i.IsFavorite, &review, &i.UpdatedAt); err != nil {
			return nil, err
		}
		if rating != nil {
			i.Rating = *rating
		}
		if review != nil {
			i.ReviewText = *review
		}
		out = append(out, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullInt(v int) any {
	if v == 0 {
		return nil
	}
	return v
}

var _ interface {
	Save(entities.Interaction) error
	GetUserInteractions(string) ([]entities.Interaction, error)
} = (*InteractionRepo)(nil)
