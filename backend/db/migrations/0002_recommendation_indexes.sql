-- Indexes that make the collaborative-filtering and cold-start queries
-- defined in backend/core-go/internal/infrastructure/repository/postgres/
-- run in O(log n) instead of full sequential scans.
--
-- Mapping (see interaction_matrix.go and metadata_repo.go):
--   * GetUserRatings / GetMeanRating / GetVariance  →  idx_interactions_user_rated
--   * AllUsers (DISTINCT user_id WHERE rating IS NOT NULL)
--                                                   →  idx_interactions_user_rated
--   * GetItemRatings / GetCommonUsers (self-join)   →  idx_interactions_item_rated
--   * CandidateItemsForUser (anti-join + ORDER BY popularity)
--                                                   →  idx_interactions_user_rated
--                                                      + idx_base_items_popularity
--   * cold-start top-popular per media_type         →  idx_base_items_popularity_by_media

BEGIN;

-- Covering partial index. (user_id, item_id) drives the anti-join used by
-- CandidateItemsForUser and every per-user fetch; the rating is INCLUDEd so
-- aggregation queries (AVG/VAR_SAMP) can be answered with an index-only scan.
CREATE INDEX IF NOT EXISTS idx_interactions_user_rated
    ON user_interactions (user_id, item_id)
    INCLUDE (rating)
    WHERE rating IS NOT NULL;

-- Covering partial index for item-side lookups: GetItemRatings hits this
-- directly, and the GetCommonUsers self-join probes it twice (once per item).
CREATE INDEX IF NOT EXISTS idx_interactions_item_rated
    ON user_interactions (item_id, user_id)
    INCLUDE (rating)
    WHERE rating IS NOT NULL;

-- Popularity ordering used by CandidateItemsForUser and the cold-start
-- fallback (top-N items globally).
CREATE INDEX IF NOT EXISTS idx_base_items_popularity
    ON base_items (average_rating DESC NULLS LAST, updated_at DESC);

-- Same ordering, but partitioned by media_type for cold-start queries that
-- restrict the response to specific media types (RecommendationFilters).
CREATE INDEX IF NOT EXISTS idx_base_items_popularity_by_media
    ON base_items (media_type, average_rating DESC NULLS LAST, updated_at DESC);

COMMIT;
