package postgres

import (
	"context"
	"errors"
	"time"

	"aura/backend/core-go/internal/domain/entities"
	dbpostgres "aura/backend/core-go/internal/infrastructure/db/postgres"

	"github.com/jackc/pgx/v5"
)

type MetadataRepo struct {
	DB *dbpostgres.Pool
}

func NewMetadataRepo(db *dbpostgres.Pool) *MetadataRepo { return &MetadataRepo{DB: db} }

func (r *MetadataRepo) GetItem(itemID string) (entities.Item, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var it entities.Item
	var releaseDate *time.Time
	var avgRating *float64
	err := r.DB.QueryRow(ctx, `
		SELECT
			item_id,
			title,
			original_title,
			description,
			release_date,
			cover_image_url,
			average_rating,
			media_type,
			genre,
			setting,
			themes,
			tonality,
			target_audience
		FROM base_items
		WHERE item_id = $1
	`, itemID).Scan(
		&it.ID,
		&it.Title,
		&it.OriginalTitle,
		&it.Description,
		&releaseDate,
		&it.CoverImageURL,
		&avgRating,
		&it.MediaType,
		&it.Criteria.Genre,
		&it.Criteria.Setting,
		&it.Criteria.Themes,
		&it.Criteria.Tonality,
		&it.Criteria.TargetAudience,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return entities.Item{}, ErrNotFound
	}
	if err != nil {
		return entities.Item{}, err
	}
	it.ReleaseDate = releaseDate
	if avgRating != nil {
		it.AverageRating = *avgRating
	}
	return it, nil
}

func (r *MetadataRepo) SaveItem(item entities.Item) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if item.ID == "" {
		_, err := r.DB.Exec(ctx, `
			INSERT INTO base_items (
				title, original_title, description, release_date, cover_image_url, average_rating, media_type,
				genre, setting, themes, tonality, target_audience
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		`,
			item.Title,
			nullString(item.OriginalTitle),
			nullString(item.Description),
			item.ReleaseDate,
			nullString(item.CoverImageURL),
			nullFloat(item.AverageRating),
			item.MediaType,
			nullString(item.Criteria.Genre),
			nullString(item.Criteria.Setting),
			nullString(item.Criteria.Themes),
			nullString(item.Criteria.Tonality),
			nullString(item.Criteria.TargetAudience),
		)
		return err
	}

	_, err := r.DB.Exec(ctx, `
		INSERT INTO base_items (
			item_id,
			title, original_title, description, release_date, cover_image_url, average_rating, media_type,
			genre, setting, themes, tonality, target_audience
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (item_id)
		DO UPDATE SET
			title = EXCLUDED.title,
			original_title = EXCLUDED.original_title,
			description = EXCLUDED.description,
			release_date = EXCLUDED.release_date,
			cover_image_url = EXCLUDED.cover_image_url,
			average_rating = EXCLUDED.average_rating,
			media_type = EXCLUDED.media_type,
			genre = EXCLUDED.genre,
			setting = EXCLUDED.setting,
			themes = EXCLUDED.themes,
			tonality = EXCLUDED.tonality,
			target_audience = EXCLUDED.target_audience,
			updated_at = now()
	`,
		item.ID,
		item.Title,
		nullString(item.OriginalTitle),
		nullString(item.Description),
		item.ReleaseDate,
		nullString(item.CoverImageURL),
		nullFloat(item.AverageRating),
		item.MediaType,
		nullString(item.Criteria.Genre),
		nullString(item.Criteria.Setting),
		nullString(item.Criteria.Themes),
		nullString(item.Criteria.Tonality),
		nullString(item.Criteria.TargetAudience),
	)
	return err
}

func (r *MetadataRepo) SearchByText(query string, limit int) ([]entities.Item, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if limit <= 0 {
		limit = 20
	}

	rows, err := r.DB.Query(ctx, `
		SELECT
			item_id,
			title,
			original_title,
			description,
			release_date,
			cover_image_url,
			average_rating,
			media_type,
			genre,
			setting,
			themes,
			tonality,
			target_audience
		FROM base_items
		WHERE
			($1 = '' OR
			 title ILIKE '%' || $1 || '%' OR
			 original_title ILIKE '%' || $1 || '%' OR
			 description ILIKE '%' || $1 || '%'
			)
		ORDER BY updated_at DESC
		LIMIT $2
	`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []entities.Item
	for rows.Next() {
		var it entities.Item
		var releaseDate *time.Time
		var avgRating *float64
		if err := rows.Scan(
			&it.ID,
			&it.Title,
			&it.OriginalTitle,
			&it.Description,
			&releaseDate,
			&it.CoverImageURL,
			&avgRating,
			&it.MediaType,
			&it.Criteria.Genre,
			&it.Criteria.Setting,
			&it.Criteria.Themes,
			&it.Criteria.Tonality,
			&it.Criteria.TargetAudience,
		); err != nil {
			return nil, err
		}
		it.ReleaseDate = releaseDate
		if avgRating != nil {
			it.AverageRating = *avgRating
		}
		out = append(out, it)
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

func nullFloat(v float64) any {
	if v == 0 {
		return nil
	}
	return v
}

