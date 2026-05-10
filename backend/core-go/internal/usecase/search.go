package usecase

import (
	"errors"
	"log"
	"strings"

	"aura/backend/core-go/internal/domain/entities"
	"aura/backend/core-go/internal/domain/services/embeddings"
	"aura/backend/core-go/internal/infrastructure/external"
	"aura/backend/core-go/internal/infrastructure/repository/postgres"
)

// SearchContent implements SearchContentUseCase.
type SearchContent struct {
	Metadata  postgres.MetadataRepository
	Adapters  map[entities.ExternalService]external.Adapter
	Publisher *embeddings.Publisher
}

// NewSearchContent wires the dependencies.
func NewSearchContent(metadata postgres.MetadataRepository) *SearchContent {
	return &SearchContent{Metadata: metadata}
}

// WithExternalSources enables best-effort catalog expansion during search.
// Local results are always returned even if an external provider is down.
func (u *SearchContent) WithExternalSources(
	adapters map[entities.ExternalService]external.Adapter,
	publisher *embeddings.Publisher,
) *SearchContent {
	u.Adapters = adapters
	u.Publisher = publisher
	return u
}

// Execute searches the local catalog first, then uses external providers to
// lazily fill missing game metadata and discover games that are not stored yet.
func (u *SearchContent) Execute(query SearchQuery) ([]entities.Item, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	local, err := u.Metadata.SearchByText(query.Text, limit)
	if err != nil {
		return nil, err
	}

	local = u.enrichMissingLocalGames(local)
	if strings.TrimSpace(query.Text) == "" || len(local) >= limit {
		return local, nil
	}

	return u.appendSteamResults(local, query.Text, limit), nil
}

func (u *SearchContent) enrichMissingLocalGames(items []entities.Item) []entities.Item {
	steam := u.Adapters[entities.ExternalServiceSteam]
	if steam == nil {
		return items
	}
	out := make([]entities.Item, len(items))
	copy(out, items)
	for i, item := range out {
		if item.MediaType != entities.MediaTypeGame || item.CoverImageURL != "" {
			continue
		}
		found, err := steam.Search(item.Title, 5)
		if err != nil || len(found) == 0 {
			if err != nil {
				log.Printf("search: steam lookup failed for %q: %v", item.Title, err)
			}
			continue
		}
		match := bestExternalTitleMatch(item.Title, found)
		full, err := steam.FetchMetadata(match.ExternalID)
		if err != nil {
			log.Printf("search: steam metadata failed for %q: %v", item.Title, err)
			continue
		}
		if !isSteamGame(full) {
			continue
		}
		merged := mergeExternalItem(item, full.ToItemMetadata())
		if err := u.saveAndPublish(&merged); err != nil {
			log.Printf("search: saving enriched game %q failed: %v", item.Title, err)
			continue
		}
		out[i] = merged
	}
	return out
}

func (u *SearchContent) appendSteamResults(local []entities.Item, text string, limit int) []entities.Item {
	steam := u.Adapters[entities.ExternalServiceSteam]
	if steam == nil {
		return local
	}
	seen := make(map[string]struct{}, len(local))
	for _, item := range local {
		seen[normaliseTitle(item.Title)] = struct{}{}
	}

	externalHits, err := steam.Search(text, limit)
	if err != nil {
		log.Printf("search: steam search failed for %q: %v", text, err)
		return local
	}

	out := local
	for _, hit := range externalHits {
		if len(out) >= limit {
			break
		}
		key := normaliseTitle(hit.Title)
		if _, ok := seen[key]; ok {
			continue
		}
		full, err := steam.FetchMetadata(hit.ExternalID)
		if err != nil {
			log.Printf("search: steam metadata failed for app %q: %v", hit.ExternalID, err)
			continue
		}
		if !isSteamGame(full) {
			continue
		}
		item := full.ToItemMetadata()
		if err := u.saveAndPublish(&item); err != nil {
			log.Printf("search: saving steam game %q failed: %v", item.Title, err)
			continue
		}
		out = append(out, item)
		seen[key] = struct{}{}
	}
	return out
}

func (u *SearchContent) saveAndPublish(item *entities.Item) error {
	if err := u.Metadata.SaveItem(item); err != nil {
		return err
	}
	if u.Publisher != nil {
		if err := u.Publisher.Publish(*item); err != nil && !errors.Is(err, embeddings.ErrNoText) {
			log.Printf("search: embedding publish failed for %q: %v", item.ID, err)
		}
	}
	return nil
}

func mergeExternalItem(existing entities.Item, external entities.Item) entities.Item {
	external.ID = existing.ID
	external.Title = existing.Title
	if external.OriginalTitle == "" {
		external.OriginalTitle = existing.OriginalTitle
	}
	if external.Description == "" {
		external.Description = existing.Description
	}
	if external.ReleaseDate == nil {
		external.ReleaseDate = existing.ReleaseDate
	}
	if external.CoverImageURL == "" {
		external.CoverImageURL = existing.CoverImageURL
	}
	if external.AverageRating == 0 {
		external.AverageRating = existing.AverageRating
	}
	if external.MediaType == "" {
		external.MediaType = existing.MediaType
	}
	external.Criteria = mergeCriteria(existing.Criteria, external.Criteria)
	if external.GameDetails == nil {
		external.GameDetails = existing.GameDetails
	}
	return external
}

func mergeCriteria(existing, incoming entities.BaseItemCriteria) entities.BaseItemCriteria {
	if incoming.Genre == "" {
		incoming.Genre = existing.Genre
	}
	if incoming.Setting == "" {
		incoming.Setting = existing.Setting
	}
	if incoming.Themes == "" {
		incoming.Themes = existing.Themes
	}
	if incoming.Tonality == "" {
		incoming.Tonality = existing.Tonality
	}
	if incoming.TargetAudience == "" {
		incoming.TargetAudience = existing.TargetAudience
	}
	return incoming
}

func normaliseTitle(title string) string {
	return strings.ToLower(strings.TrimSpace(title))
}

func bestExternalTitleMatch(title string, hits []external.ExternalData) external.ExternalData {
	if len(hits) == 0 {
		return external.ExternalData{}
	}
	needle := normaliseTitle(title)
	for _, hit := range hits {
		if normaliseTitle(hit.Title) == needle {
			return hit
		}
	}
	return hits[0]
}

func isSteamGame(data external.ExternalData) bool {
	appType, _ := data.RawData["steam_app_type"].(string)
	return appType == "" || appType == "game"
}
