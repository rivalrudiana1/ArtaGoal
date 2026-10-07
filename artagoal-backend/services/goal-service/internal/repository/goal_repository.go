package repository

import (
	"context"
	"time"

	"artagoal/goal-service/internal/domain/entity"
)

// GoalRepository adalah kontrak persistensi untuk aggregate Goal.
// Implementasi konkret (mis. PostgreSQL) tinggal di subpaket postgres.
type GoalRepository interface {
	CreateGoal(ctx context.Context, goal entity.Goal) (entity.Goal, error)
	GetGoalByID(ctx context.Context, id string) (entity.Goal, error)
	// GetGoalsByUserID mengambil halaman goal milik user (limit/offset sudah divalidasi).
	GetGoalsByUserID(ctx context.Context, userID string, limit, offset int) ([]entity.Goal, error)
	// CountGoalsByUserID menghitung total goal milik user (untuk X-Total-Count).
	CountGoalsByUserID(ctx context.Context, userID string) (int, error)
	UpdateGoal(ctx context.Context, goal entity.Goal) (entity.Goal, error)
	DeleteGoal(ctx context.Context, id string) error

	// --- contributions (next feature) ---
	// AddContribution menyisipkan satu kontribusi dan menaikkan
	// goals.current_amount secara atomik, mengembalikan kontribusi
	// yang tersimpan dan goal terbaru (termasuk auto-achieved).
	AddContribution(ctx context.Context, goalID string, amount float64, note *string) (entity.GoalContribution, entity.Goal, error)
	ListContributionsByGoalID(ctx context.Context, goalID string, limit, offset int) ([]entity.GoalContribution, error)
	// CountContributionsByGoalID menghitung (jumlah, total nominal) setoran satu goal.
	CountContributionsByGoalID(ctx context.Context, goalID string) (count int, total float64, err error)
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
	// GetNotificationsByUserID mengembalikan halaman notifikasi milik user.
	GetNotificationsByUserID(ctx context.Context, userID string, limit, offset int) ([]entity.Notification, error)
	// CountNotificationsByUserID menghitung total notifikasi milik user.
	CountNotificationsByUserID(ctx context.Context, userID string) (int, error)
	// MarkNotificationAsRead menandai notifikasi milik user sebagai dibaca.
	// Mengembalikan ErrNotificationNotFound bila id tidak ada / bukan milik user.
	MarkNotificationAsRead(ctx context.Context, id, userID string) (entity.Notification, error)

	// --- stats (gamifikasi) ---
	// ListContributionDates mengembalikan tanggal-tanggal unik (tengah malam)
	// saat user melakukan setoran, diurutkan dari yang terbaru.
	ListContributionDates(ctx context.Context, userID string) ([]time.Time, error)
	// CountContributionsByUserID menghitung total setoran milik user.
	CountContributionsByUserID(ctx context.Context, userID string) (int, error)

	// --- web push ---
	// SavePushSubscription menyimpan langganan push (upsert per endpoint).
	SavePushSubscription(ctx context.Context, sub entity.PushSubscription) (entity.PushSubscription, error)
	// ListPushSubscriptionsByUserID mengembalikan langganan push milik user.
	ListPushSubscriptionsByUserID(ctx context.Context, userID string) ([]entity.PushSubscription, error)
	// DeletePushSubscription menghapus langganan push milik user.
	DeletePushSubscription(ctx context.Context, userID, endpoint string) error
}
