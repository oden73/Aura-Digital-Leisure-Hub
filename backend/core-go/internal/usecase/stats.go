package usecase

import "aura/backend/core-go/internal/domain/entities"

// UserStatsRepository is the minimum repo surface GetUserStats needs.
type UserStatsRepository interface {
	GetStats(userID string) (entities.UserStats, error)
}

// GetUserStats fetches aggregated interaction stats for a user.
type GetUserStats struct {
	Repo UserStatsRepository
}

func NewGetUserStats(repo UserStatsRepository) *GetUserStats {
	return &GetUserStats{Repo: repo}
}

func (u *GetUserStats) Execute(userID string) (entities.UserStats, error) {
	return u.Repo.GetStats(userID)
}
