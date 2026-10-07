package usecase

import (
	"context"
	"strings"
	"testing"
	"time"

	"artagoal/goal-service/internal/domain/entity"
	"artagoal/goal-service/internal/repository"
)

// notifRepo adalah fake untuk jalur notifikasi/worker.
type notifRepo struct {
	repository.GoalRepository
	goals   []entity.Goal
	recent  map[string]bool
	created []entity.Notification
	notifs  []entity.Notification
	markErr error
}

func (s *notifRepo) ListActiveGoalsDueWithin(_ context.Context, _ int) ([]entity.Goal, error) {
	return s.goals, nil
}

func (s *notifRepo) HasRecentNotification(_ context.Context, goalID, _ string, _ int) (bool, error) {
	return s.recent[goalID], nil
}

func (s *notifRepo) CreateNotification(_ context.Context, n entity.Notification) (entity.Notification, error) {
	n.ID = "n1"
	n.CreatedAt = time.Now()
	s.created = append(s.created, n)
	return n, nil
}

func (s *notifRepo) GetNotificationsByUserID(_ context.Context, _ string) ([]entity.Notification, error) {
	return s.notifs, nil
}

func (s *notifRepo) MarkNotificationAsRead(_ context.Context, id, _ string) (entity.Notification, error) {
	if s.markErr != nil {
		return entity.Notification{}, s.markErr
	}
	return entity.Notification{ID: id, IsRead: true}, nil
}

func goalDueIn(days, target, current int) entity.Goal {
	td := time.Now().AddDate(0, 0, days)
	return entity.Goal{
		UserID: "u1", Title: "DP Rumah",
		TargetAmount: float64(target), CurrentAmount: float64(current),
		TargetDate: &td, Status: entity.GoalStatusActive,
	}
}

func TestRunDeadlineReminderCheck(t *testing.T) {
	soon := goalDueIn(20, 1000, 500) // 50% -> wajib diingatkan
	soon.ID = "g-soon"
	rich := goalDueIn(10, 1000, 900) // 90% -> dilewati
	rich.ID = "g-rich"
	dup := goalDueIn(5, 1000, 100) // duplikat 7 hari -> dilewati
	dup.ID = "g-dup"

	st := &notifRepo{
		goals:  []entity.Goal{soon, rich, dup},
		recent: map[string]bool{"g-dup": true},
	}
	uc := NewGoalUsecase(st)

	created, err := uc.RunDeadlineReminderCheck(context.Background())
	if err != nil {
		t.Fatalf("gagal: %v", err)
	}
	if created != 1 {
		t.Fatalf("ingin 1 notifikasi, dapat %d (%+v)", created, st.created)
	}
	n := st.created[0]
	if n.Title != "🚨 Peringatan Tenggat Target" {
		t.Fatalf("title salah: %q", n.Title)
	}
	if !strings.Contains(n.Message, "DP Rumah") ||
		!strings.Contains(n.Message, "20 hari") ||
		!strings.Contains(n.Message, "50%") {
		t.Fatalf("message salah: %q", n.Message)
	}
	if n.UserID != "u1" || n.GoalID == nil || *n.GoalID != "g-soon" {
		t.Fatalf("target notifikasi salah: %+v", n)
	}
}

func TestRunDeadlineReminderCheckSkipsInvalid(t *testing.T) {
	past := goalDueIn(-2, 1000, 100) // sudah lewat -> dilewati
	past.ID = "g-past"
	zero := goalDueIn(10, 0, 0) // target 0 -> dilewati
	zero.ID = "g-zero"

	st := &notifRepo{goals: []entity.Goal{past, zero}, recent: map[string]bool{}}
	uc := NewGoalUsecase(st)

	created, err := uc.RunDeadlineReminderCheck(context.Background())
	if err != nil {
		t.Fatalf("gagal: %v", err)
	}
	if created != 0 {
		t.Fatalf("ingin 0 notifikasi, dapat %d", created)
	}
}

func TestGetAndMarkNotificationValidation(t *testing.T) {
	st := &notifRepo{}
	uc := NewGoalUsecase(st)
	ctx := context.Background()

	if _, err := uc.GetNotifications(ctx, ""); err == nil {
		t.Fatal("user kosong harus ditolak")
	}
	if _, err := uc.MarkNotificationRead(ctx, "", "n1"); err == nil {
		t.Fatal("user kosong harus ditolak")
	}
	if _, err := uc.MarkNotificationRead(ctx, "u1", ""); err == nil {
		t.Fatal("id kosong harus ditolak")
	}

	st.markErr = entity.ErrNotificationNotFound
	if _, err := uc.MarkNotificationRead(ctx, "u1", "ngawur"); err == nil {
		t.Fatal("notifikasi tak ada harus error")
	}
}
