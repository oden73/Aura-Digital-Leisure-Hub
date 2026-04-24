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
type Interaction struct {
	ID         int64
	UserID     string
	ItemID     string
	Status     InteractionStatus
	Rating     int
	IsFavorite bool
	ReviewText string
	UpdatedAt  time.Time
}
