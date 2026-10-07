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

	// GetContributionHeatmap mengembalikan agregasi jumlah dan total
	// setoran per hari dalam 1 tahun terakhir untuk satu user.
	GetContributionHeatmap(ctx context.Context, userID string) ([]entity.HeatmapData, error)

	// --- notifications (reminder worker + in-app) ---
	// ListActiveGoalsDueWithin mengembalikan goal berstatus active yang
	// memiliki target_date di antara sekarang dan days hari ke depan.
	ListActiveGoalsDueWithin(ctx context.Context, days int) ([]entity.Goal, error)
	// CreateNotification menyimpan satu notifikasi dan mengembalikannya.
	CreateNotification(ctx context.Context, n entity.Notification) (entity.Notification, error)
	// HasRecentNotification melaporkan apakah sudah ada notifikasi berjudul
	// title untuk goal tersebut dalam days hari terakhir (anti-duplikat).
	HasRecentNotification(ctx context.Context, goalID, title string, days int) (bool, error)
	// GetNotificationsByUserID mengembalikan notifikasi milik user (terbaru dulu).
	GetNotificationsByUserID(ctx context.Context, userID string) ([]entity.Notification, error)
	// MarkNotificationAsRead menandai notifikasi milik user sebagai dibaca.
	// Mengembalikan ErrNotificationNotFound bila id tidak ada / bukan milik user.
	MarkNotificationAsRead(ctx context.Context, id, userID string) (entity.Notification, error)
}

