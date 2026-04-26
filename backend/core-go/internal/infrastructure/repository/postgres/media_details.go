package postgres

import (
	"context"
	"errors"

	"aura/backend/core-go/internal/domain/entities"
	dbpostgres "aura/backend/core-go/internal/infrastructure/db/postgres"

	"github.com/jackc/pgx/v5"
)

// dbReader narrows the connection pool to just the methods used by the
// read-only details helpers below.
type dbReader = *dbpostgres.Pool

// ---------- book_details ----------

func getBookDetails(ctx context.Context, db dbReader, itemID string) (*entities.BookDetails, error) {
	var d entities.BookDetails
	var pageCount *int
	err := db.QueryRow(ctx, `
		SELECT
			COALESCE(author, ''),
			COALESCE(publisher, ''),
			COALESCE(literary_form, ''),
			COALESCE(volume_format, ''),
			COALESCE(narrative_type, ''),
			COALESCE(artistic_style, ''),
			page_count
		FROM book_details
		WHERE item_id = $1
	`, itemID).Scan(
		&d.Author,
		&d.Publisher,
		&d.LiteraryForm,
		&d.VolumeFormat,
		&d.NarrativeType,
		&d.ArtisticStyle,
		&pageCount,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if pageCount != nil {
		d.PageCount = *pageCount
	}
	return &d, nil
}

func upsertBookDetails(ctx context.Context, tx pgx.Tx, itemID string, d *entities.BookDetails) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO book_details
			(item_id, author, publisher, literary_form, volume_format, narrative_type, artistic_style, page_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (item_id) DO UPDATE SET
			author         = EXCLUDED.author,
			publisher      = EXCLUDED.publisher,
			literary_form  = EXCLUDED.literary_form,
			volume_format  = EXCLUDED.volume_format,
			narrative_type = EXCLUDED.narrative_type,
			artistic_style = EXCLUDED.artistic_style,
			page_count     = EXCLUDED.page_count
	`,
		itemID,
		nullString(d.Author),
		nullString(d.Publisher),
		nullString(d.LiteraryForm),
		nullString(d.VolumeFormat),
		nullString(d.NarrativeType),
		nullString(d.ArtisticStyle),
		nullInt(d.PageCount),
	)
	return err
}

// ---------- cinema_details ----------

func getCinemaDetails(ctx context.Context, db dbReader, itemID string) (*entities.CinemaDetails, error) {
	var d entities.CinemaDetails
	var duration *int
	err := db.QueryRow(ctx, `
		SELECT
			COALESCE(director, ''),
			COALESCE(cast_list, ''),
			COALESCE(format, ''),
			COALESCE(production_method, ''),
			COALESCE(visual_style, ''),
			COALESCE(plot_structure, ''),
			duration_mins
		FROM cinema_details
		WHERE item_id = $1
	`, itemID).Scan(
		&d.Director,
		&d.Cast,
		&d.Format,
		&d.ProductionMethod,
		&d.VisualStyle,
		&d.PlotStructure,
		&duration,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if duration != nil {
		d.DurationMins = *duration
	}
	return &d, nil
}

func upsertCinemaDetails(ctx context.Context, tx pgx.Tx, itemID string, d *entities.CinemaDetails) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO cinema_details
			(item_id, director, cast_list, format, production_method, visual_style, plot_structure, duration_mins)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (item_id) DO UPDATE SET
			director          = EXCLUDED.director,
			cast_list         = EXCLUDED.cast_list,
			format            = EXCLUDED.format,
			production_method = EXCLUDED.production_method,
			visual_style      = EXCLUDED.visual_style,
			plot_structure    = EXCLUDED.plot_structure,
			duration_mins     = EXCLUDED.duration_mins
	`,
		itemID,
		nullString(d.Director),
		nullString(d.Cast),
		nullString(d.Format),
		nullString(d.ProductionMethod),
		nullString(d.VisualStyle),
		nullString(d.PlotStructure),
		nullInt(d.DurationMins),
	)
	return err
}

// ---------- game_details ----------

func getGameDetails(ctx context.Context, db dbReader, itemID string) (*entities.GameDetails, error) {
	var d entities.GameDetails
	err := db.QueryRow(ctx, `
		SELECT
			COALESCE(developer, ''),
			COALESCE(gameplay_genre, ''),
			COALESCE(platforms, ''),
			COALESCE(player_count, ''),
			COALESCE(perspective, ''),
			COALESCE(plot_genre, ''),
			COALESCE(world_structure, ''),
			COALESCE(monetization, '')
		FROM game_details
		WHERE item_id = $1
	`, itemID).Scan(
		&d.Developer,
		&d.GameplayGenre,
		&d.Platforms,
		&d.PlayerCount,
		&d.Perspective,
		&d.PlotGenre,
		&d.WorldStructure,
		&d.Monetization,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func upsertGameDetails(ctx context.Context, tx pgx.Tx, itemID string, d *entities.GameDetails) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO game_details
			(item_id, developer, gameplay_genre, platforms, player_count, perspective, plot_genre, world_structure, monetization)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (item_id) DO UPDATE SET
			developer       = EXCLUDED.developer,
			gameplay_genre  = EXCLUDED.gameplay_genre,
			platforms       = EXCLUDED.platforms,
			player_count    = EXCLUDED.player_count,
			perspective     = EXCLUDED.perspective,
			plot_genre      = EXCLUDED.plot_genre,
			world_structure = EXCLUDED.world_structure,
			monetization    = EXCLUDED.monetization
	`,
		itemID,
		nullString(d.Developer),
		nullString(d.GameplayGenre),
		nullString(d.Platforms),
		nullString(d.PlayerCount),
		nullString(d.Perspective),
		nullString(d.PlotGenre),
		nullString(d.WorldStructure),
		nullString(d.Monetization),
	)
	return err
}
