package entities

import "time"

// InteractionStatus captures where a user is with a particular item.
type InteractionStatus string

const (
	InteractionStatusPlanned    InteractionStatus = "planned"
	InteractionStatusInProgress InteractionStatus = "in_progress"
	InteractionStatusCompleted  InteractionStatus = "completed"
	InteractionStatusDropped    InteractionStatus = "dropped"
)

// Interaction is one row of the Rui (user/item/rating) matrix.
//
// Rating uses `omitempty` so a zero value (= "no rating yet") collapses
// rather than serialising as 0, which would be ambiguous with a real
// rating of 0. The OpenAPI contract marks rating as nullable; clients
// should treat a missing field the same as null.
type Interaction struct {
	ID         int64             `json:"id"`
	UserID     string            `json:"user_id"`
	ItemID     string            `json:"item_id"`
	Status     InteractionStatus `json:"status"`
	Rating     int               `json:"rating,omitempty"`
	IsFavorite bool              `json:"is_favorite"`
	ReviewText string            `json:"review_text,omitempty"`
	UpdatedAt  time.Time         `json:"updated_at"`
}
