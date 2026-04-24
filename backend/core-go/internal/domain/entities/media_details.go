package entities

// BookDetails holds book-specific criteria from docs/predone/data_model.md.
type BookDetails struct {
	Author        string
	Publisher     string
	LiteraryForm  string
	VolumeFormat  string
	NarrativeType string
	ArtisticStyle string
	PageCount     int
}

// CinemaDetails holds cinema/series-specific criteria.
type CinemaDetails struct {
	Director         string
	Cast             string
	Format           string
	ProductionMethod string
	VisualStyle      string
	PlotStructure    string
	DurationMins     int
}

// GameDetails holds game-specific criteria.
type GameDetails struct {
	Developer      string
	GameplayGenre  string
	Platforms      string
	PlayerCount    string
	Perspective    string
	PlotGenre      string
	WorldStructure string
	Monetization   string
}
