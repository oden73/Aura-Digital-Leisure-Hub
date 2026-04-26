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

// baseItemColumns is the canonical SELECT list for base_items used by
// GetItem, SearchByText, TopRated, fetchBatch and GetUserLibraryItems so
// every read produces the same column order. Media-specific details are
// loaded separately via getBookDetails / getCinemaDetails / getGameDetails.
//
// Nullable text columns are COALESCEd to '' so the scanner can read into
// plain Go strings; the two columns that callers actually need to
// distinguish from "absent" (release_date, average_rating) stay nullable
// and are scanned through pointers in scanBaseItem.
const baseItemColumns = `
	item_id,
	title,
	COALESCE(original_title, ''),
	COALESCE(description, ''),
	release_date,
	COALESCE(cover_image_url, ''),
	average_rating,
	media_type,
	COALESCE(genre, ''),
	COALESCE(setting, ''),
	COALESCE(themes, ''),
	COALESCE(tonality, ''),
	COALESCE(target_audience, '')
`

// scanBaseItem reads one base_items row into the provided Item using the
// column order declared in baseItemColumns. It accepts a pgx.Row so callers
// can pass either Query (rows.Scan) or QueryRow results.
func scanBaseItem(row pgx.Row, it *entities.Item) error {
	var releaseDate *time.Time
	var avgRating *float64
	if err := row.Scan(
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
		return err
	}
	it.ReleaseDate = releaseDate
	if avgRating != nil {
		it.AverageRating = *avgRating
	}
	return nil
}

// GetItem returns the full Item including media-specific details. The
// details table is queried only for the matching media_type so we never
// pay for joins on the other two.
func (r *MetadataRepo) GetItem(itemID string) (entities.Item, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var it entities.Item
	row := r.DB.QueryRow(ctx, `SELECT `+baseItemColumns+` FROM base_items WHERE item_id = $1`, itemID)
	if err := scanBaseItem(row, &it); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Item{}, ErrNotFound
		}
		return entities.Item{}, err
	}

	if err := r.loadDetails(ctx, &it); err != nil {
		return entities.Item{}, err
	}
	return it, nil
}

func (r *MetadataRepo) loadDetails(ctx context.Context, it *entities.Item) error {
	switch it.MediaType {
	case entities.MediaTypeBook:
		d, err := getBookDetails(ctx, r.DB, it.ID)
		if err != nil {
			return err
		}
		it.BookDetails = d
	case entities.MediaTypeCinema:
		d, err := getCinemaDetails(ctx, r.DB, it.ID)
		if err != nil {
			return err
		}
		it.CinemaDetails = d
	case entities.MediaTypeGame:
		d, err := getGameDetails(ctx, r.DB, it.ID)
		if err != nil {
			return err
		}
		it.GameDetails = d
	}
	return nil
}

// SaveItem persists base_items and the matching details row in a single
// transaction. When item.ID is empty the row is inserted and the new
// item_id is written back into item via the RETURNING clause so the
// details upsert can target it. Details are written only for the matching
// media_type (book/cinema/game); supplying a BookDetails alongside
// MediaTypeGame is a no-op for the games table — the book section will
// still be written, callers must keep their objects consistent.
func (r *MetadataRepo) SaveItem(item entities.Item) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := r.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	id, err := upsertBaseItem(ctx, tx, item)
	if err != nil {
		return err
	}

	if item.BookDetails != nil {
		if err := upsertBookDetails(ctx, tx, id, item.BookDetails); err != nil {
			return err
		}
	}
	if item.CinemaDetails != nil {
		if err := upsertCinemaDetails(ctx, tx, id, item.CinemaDetails); err != nil {
			return err
		}
	}
	if item.GameDetails != nil {
		if err := upsertGameDetails(ctx, tx, id, item.GameDetails); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func upsertBaseItem(ctx context.Context, tx pgx.Tx, item entities.Item) (string, error) {
	if item.ID == "" {
		var id string
		err := tx.QueryRow(ctx, `
			INSERT INTO base_items (
				title, original_title, description, release_date, cover_image_url, average_rating, media_type,
				genre, setting, themes, tonality, target_audience
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
			RETURNING item_id
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
		).Scan(&id)
		return id, err
	}

	_, err := tx.Exec(ctx, `
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
	return item.ID, err
}

// TopRated returns the most popular catalog items ordered by
// average_rating DESC. When mediaTypes is non-empty the result is restricted
// to those types. Used by the cold-start fallback to backfill recommendation
// responses for users with no signal.
func (r *MetadataRepo) TopRated(limit int, mediaTypes []entities.MediaType) ([]entities.Item, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if limit <= 0 {
		limit = 20
	}

	var rows pgx.Rows
	var err error
	if len(mediaTypes) == 0 {
		rows, err = r.DB.Query(ctx, `SELECT `+baseItemColumns+` FROM base_items
			ORDER BY average_rating DESC NULLS LAST, updated_at DESC
			LIMIT $1
		`, limit)
	} else {
		typeFilter := make([]string, 0, len(mediaTypes))
		for _, m := range mediaTypes {
			typeFilter = append(typeFilter, string(m))
		}
		rows, err = r.DB.Query(ctx, `SELECT `+baseItemColumns+` FROM base_items
			WHERE media_type::text = ANY($1)
			ORDER BY average_rating DESC NULLS LAST, updated_at DESC
			LIMIT $2
		`, typeFilter, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []entities.Item
	for rows.Next() {
		var it entities.Item
		if err := scanBaseItem(rows, &it); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// IterateAll streams every item in the catalog through the visitor callback.
// It is intentionally batched so memory usage stays flat even for very large
// catalogs. If the visitor returns an error iteration stops and that error
// is returned. Used by the embeddings backfill CLI.
func (r *MetadataRepo) IterateAll(ctx context.Context, batchSize int, visit func(entities.Item) error) error {
	if batchSize <= 0 {
		batchSize = 200
	}

	cursor := ""
	for {
		items, err := r.fetchBatch(ctx, cursor, batchSize)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		for _, it := range items {
			if err := visit(it); err != nil {
				return err
			}
		}
		cursor = items[len(items)-1].ID
		if len(items) < batchSize {
			return nil
		}
	}
}

func (r *MetadataRepo) fetchBatch(ctx context.Context, cursor string, limit int) ([]entities.Item, error) {
	queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := r.DB.Query(queryCtx, `SELECT `+baseItemColumns+` FROM base_items
		WHERE ($1 = '' OR item_id::text > $1)
		ORDER BY item_id::text ASC
		LIMIT $2
	`, cursor, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []entities.Item
	for rows.Next() {
		var it entities.Item
		if err := scanBaseItem(rows, &it); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (r *MetadataRepo) SearchByText(query string, limit int) ([]entities.Item, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if limit <= 0 {
		limit = 20
	}

	rows, err := r.DB.Query(ctx, `SELECT `+baseItemColumns+` FROM base_items
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
		if err := scanBaseItem(rows, &it); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}
