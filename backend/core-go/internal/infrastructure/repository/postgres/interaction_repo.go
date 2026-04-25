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

type LibraryItem struct {
	Interaction entities.Interaction
	Item        entities.Item
}

func (r *InteractionRepo) GetUserLibraryItems(userID string, limit int) ([]LibraryItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if limit <= 0 {
		limit = 50
	}

	rows, err := r.DB.Query(ctx, `
		SELECT
			ui.interaction_id,
			ui.user_id,
			ui.item_id,
			ui.status,
			ui.rating,
			ui.is_favorite,
			ui.review_text,
			ui.updated_at,

			bi.item_id,
			bi.title,
			bi.original_title,
			bi.description,
			bi.release_date,
			bi.cover_image_url,
			bi.average_rating,
			bi.media_type,
			bi.genre,
			bi.setting,
			bi.themes,
			bi.tonality,
			bi.target_audience
		FROM user_interactions ui
		JOIN base_items bi ON bi.item_id = ui.item_id
		WHERE ui.user_id = $1
		ORDER BY ui.updated_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LibraryItem
	for rows.Next() {
		var li LibraryItem
		var rating *int
		var review *string
		var releaseDate *time.Time
		var avgRating *float64

		if err := rows.Scan(
			&li.Interaction.ID,
			&li.Interaction.UserID,
			&li.Interaction.ItemID,
			&li.Interaction.Status,
			&rating,
			&li.Interaction.IsFavorite,
			&review,
			&li.Interaction.UpdatedAt,

			&li.Item.ID,
			&li.Item.Title,
			&li.Item.OriginalTitle,
			&li.Item.Description,
			&releaseDate,
			&li.Item.CoverImageURL,
			&avgRating,
			&li.Item.MediaType,
			&li.Item.Criteria.Genre,
			&li.Item.Criteria.Setting,
			&li.Item.Criteria.Themes,
			&li.Item.Criteria.Tonality,
			&li.Item.Criteria.TargetAudience,
		); err != nil {
			return nil, err
		}
		if rating != nil {
			li.Interaction.Rating = *rating
		}
		if review != nil {
			li.Interaction.ReviewText = *review
		}
		li.Item.ReleaseDate = releaseDate
		if avgRating != nil {
			li.Item.AverageRating = *avgRating
		}
		out = append(out, li)
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
