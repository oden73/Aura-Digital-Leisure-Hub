package entities

// BookDetails holds book-specific criteria from docs/predone/data_model.md.
type BookDetails struct {
	Author        string `json:"author,omitempty"`
	Publisher     string `json:"publisher,omitempty"`
	LiteraryForm  string `json:"literary_form,omitempty"`
	VolumeFormat  string `json:"volume_format,omitempty"`
	NarrativeType string `json:"narrative_type,omitempty"`
	ArtisticStyle string `json:"artistic_style,omitempty"`
	PageCount     int    `json:"page_count,omitempty"`
}

// CinemaDetails holds cinema/series-specific criteria.
type CinemaDetails struct {
	Director         string `json:"director,omitempty"`
	Cast             string `json:"cast,omitempty"`
	Format           string `json:"format,omitempty"`
	ProductionMethod string `json:"production_method,omitempty"`
	VisualStyle      string `json:"visual_style,omitempty"`
	PlotStructure    string `json:"plot_structure,omitempty"`
	DurationMins     int    `json:"duration_mins,omitempty"`
}

// GameDetails holds game-specific criteria.
type GameDetails struct {
	Developer      string `json:"developer,omitempty"`
	GameplayGenre  string `json:"gameplay_genre,omitempty"`
	Platforms      string `json:"platforms,omitempty"`
	PlayerCount    string `json:"player_count,omitempty"`
	Perspective    string `json:"perspective,omitempty"`
	PlotGenre      string `json:"plot_genre,omitempty"`
	WorldStructure string `json:"world_structure,omitempty"`
	Monetization   string `json:"monetization,omitempty"`
}
