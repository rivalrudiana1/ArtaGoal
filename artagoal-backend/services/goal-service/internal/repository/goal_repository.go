package repository

import (
	"context"

	"artagoal/goal-service/internal/domain/entity"
)

// GoalRepository adalah kontrak persistensi untuk aggregate Goal.
// Implementasi konkret (mis. PostgreSQL) tinggal di subpaket postgres.
type GoalRepository interface {
	CreateGoal(ctx context.Context, goal entity.Goal) (entity.Goal, error)
	GetGoalByID(ctx context.Context, id string) (entity.Goal, error)
	GetGoalsByUserID(ctx context.Context, userID string) ([]entity.Goal, error)
	UpdateGoal(ctx context.Context, goal entity.Goal) (entity.Goal, error)
	DeleteGoal(ctx context.Context, id string) error

	// --- contributions (next feature) ---
	// AddContribution menyisipkan satu kontribusi dan menaikkan
	// goals.current_amount secara atomik, mengembalikan kontribusi
	// yang tersimpan dan goal terbaru (termasuk auto-achieved).
	AddContribution(ctx context.Context, goalID string, amount float64, note *string) (entity.GoalContribution, entity.Goal, error)
	ListContributionsByGoalID(ctx context.Context, goalID string) ([]entity.GoalContribution, error)
	GetContributionByID(ctx context.Context, id string) (entity.GoalContribution, error)
	// DeleteContribution menghapus kontribusi dan menurunkan
	// goals.current_amount sebesar nominalnya (floor 0).
	// Mengembalikan goal terbaru setelah penyesuaian.
	DeleteContribution(ctx context.Context, contributionID string) (entity.Goal, error)
}

