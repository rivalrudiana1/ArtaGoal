package usecase

import (
	"context"
	"strings"
	"testing"
	"time"

	"artagoal/goal-service/internal/domain/entity"
	"artagoal/goal-service/internal/push"
	"artagoal/goal-service/internal/repository"
)

// notifRepo adalah fake untuk jalur notifikasi/worker.
type notifRepo struct {
	repository.GoalRepository
	goals            []entity.Goal
	recent           map[string]bool
	created          []entity.Notification
	notifs           []entity.Notification
	markErr          error
	dates            []time.Time
	total            int
	subs             []entity.PushSubscription
	savedSub         *entity.PushSubscription
	deletedEndpoints []string
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

func (s *notifRepo) GetNotificationsByUserID(_ context.Context, _ string, _, _ int) ([]entity.Notification, error) {
	return s.notifs, nil
}

func (s *notifRepo) MarkNotificationAsRead(_ context.Context, id, _ string) (entity.Notification, error) {
	if s.markErr != nil {
		return entity.Notification{}, s.markErr
	}
	return entity.Notification{ID: id, IsRead: true}, nil
}

func (s *notifRepo) ListContributionDates(_ context.Context, _ string) ([]time.Time, error) {
	return s.dates, nil
}

func (s *notifRepo) CountContributionsByUserID(_ context.Context, _ string) (int, error) {
	return s.total, nil
}

func (s *notifRepo) SavePushSubscription(_ context.Context, sub entity.PushSubscription) (entity.PushSubscription, error) {
	sub.ID = "ps1"
	s.savedSub = &sub
	return sub, nil
}

func (s *notifRepo) ListPushSubscriptionsByUserID(_ context.Context, _ string) ([]entity.PushSubscription, error) {
	return s.subs, nil
}

func (s *notifRepo) DeletePushSubscription(_ context.Context, _, endpoint string) error {
	s.deletedEndpoints = append(s.deletedEndpoints, endpoint)
	return nil
}

func daysAgo(n int) time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.UTC).AddDate(0, 0, -n)
}

func TestGetStatsStreakAndLevel(t *testing.T) {
	// Setoran hari ini, kemarin, 2 hari lalu (streak 3), lalu gap,
	// lalu 5 & 6 hari lalu (run 2) -> longest 3.
	st := &notifRepo{
		dates: []time.Time{daysAgo(0), daysAgo(1), daysAgo(2), daysAgo(5), daysAgo(6)},
		total: 12,
	}
	uc := NewGoalUsecase(st)

	stats, err := uc.GetStats(context.Background(), "u1")
	if err != nil {
		t.Fatalf("gagal: %v", err)
	}
	if stats.CurrentStreakDays != 3 {
		t.Fatalf("streak ingin 3, dapat %d", stats.CurrentStreakDays)
	}
	if stats.LongestStreakDays != 3 {
		t.Fatalf("longest ingin 3, dapat %d", stats.LongestStreakDays)
	}
	if stats.ActiveDays365 != 5 {
		t.Fatalf("active days ingin 5, dapat %d", stats.ActiveDays365)
	}
	if stats.Level != "Penabung Rutin" {
		t.Fatalf("level ingin Penabung Rutin, dapat %q", stats.Level)
	}
	if stats.NextLevelAt == nil || *stats.NextLevelAt != 30 {
		t.Fatalf("next level ingin 30, dapat %+v", stats.NextLevelAt)
	}
}

func TestGetStatsStreakAliveFromYesterday(t *testing.T) {
	st := &notifRepo{dates: []time.Time{daysAgo(1), daysAgo(2)}, total: 2}
	uc := NewGoalUsecase(st)

	stats, err := uc.GetStats(context.Background(), "u1")
	if err != nil {
		t.Fatalf("gagal: %v", err)
	}
	if stats.CurrentStreakDays != 2 {
		t.Fatalf("streak ingin 2, dapat %d", stats.CurrentStreakDays)
	}
	if stats.Level != "Pemula" || stats.NextLevelAt == nil || *stats.NextLevelAt != 10 {
		t.Fatalf("level salah: %+v", stats)
	}
}

func TestGetStatsEmpty(t *testing.T) {
	st := &notifRepo{}
	uc := NewGoalUsecase(st)

	stats, err := uc.GetStats(context.Background(), "u1")
	if err != nil {
		t.Fatalf("gagal: %v", err)
	}
	if stats.CurrentStreakDays != 0 || stats.LongestStreakDays != 0 || stats.Level != "Pemula" {
		t.Fatalf("stats kosong salah: %+v", stats)
	}
	if _, err := uc.GetStats(context.Background(), ""); err == nil {
		t.Fatal("user kosong harus ditolak")
	}
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

	if _, _, err := uc.GetNotifications(ctx, "", 1, 50); err == nil {
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

// fakePushSender mencatat push yang dikirim worker.
type fakePushSender struct {
	key  string
	sent []push.Subscription
	gone bool
}

func (f *fakePushSender) PublicKey() string { return f.key }

func (f *fakePushSender) Send(sub push.Subscription, _, _, _ string) (bool, error) {
	f.sent = append(f.sent, sub)
	return f.gone, nil
}

func TestSavePushSubscriptionValidation(t *testing.T) {
	st := &notifRepo{}
	uc := NewGoalUsecase(st)
	ctx := context.Background()

	sub, err := uc.SavePushSubscription(ctx, "u1", "https://push.example/e1", "p256dh-abc", "auth-abc")
	if err != nil {
		t.Fatalf("gagal: %v", err)
	}
	if sub.ID != "ps1" || st.savedSub == nil || st.savedSub.Endpoint != "https://push.example/e1" {
		t.Fatalf("subscription tidak tersimpan: %+v", st.savedSub)
	}
	if _, err := uc.SavePushSubscription(ctx, "", "e", "p", "a"); err == nil {
		t.Fatal("user kosong harus ditolak")
	}
	if _, err := uc.SavePushSubscription(ctx, "u1", "", "p", "a"); err == nil {
		t.Fatal("endpoint kosong harus ditolak")
	}
	if err := uc.DeletePushSubscription(ctx, "u1", ""); err == nil {
		t.Fatal("endpoint kosong harus ditolak")
	}
	if err := uc.DeletePushSubscription(ctx, "u1", "https://push.example/e1"); err != nil {
		t.Fatalf("hapus gagal: %v", err)
	}
}

func TestPushPublicKeyNeedsSender(t *testing.T) {
	uc := NewGoalUsecase(&notifRepo{})
	if _, err := uc.PushPublicKey(); err == nil {
		t.Fatal("tanpa sender harus error")
	}
	uc.SetPushSender(&fakePushSender{key: "PUBKEY"})
	key, err := uc.PushPublicKey()
	if err != nil || key != "PUBKEY" {
		t.Fatalf("ingin PUBKEY, dapat %q (%v)", key, err)
	}
}

func TestReminderSendsPushAndPrunesGone(t *testing.T) {
	soon := goalDueIn(20, 1000, 500)
	soon.ID = "g-soon"
	sender := &fakePushSender{key: "PUBKEY", gone: true}
	st := &notifRepo{
		goals:  []entity.Goal{soon},
		recent: map[string]bool{},
		subs: []entity.PushSubscription{
			{ID: "ps1", UserID: "u1", Endpoint: "https://push.example/mati"},
		},
	}
	uc := NewGoalUsecase(st)
	uc.SetPushSender(sender)

	created, err := uc.RunDeadlineReminderCheck(context.Background())
	if err != nil {
		t.Fatalf("gagal: %v", err)
	}
	if created != 1 {
		t.Fatalf("ingin 1 notifikasi, dapat %d", created)
	}
	if len(sender.sent) != 1 || sender.sent[0].Endpoint != "https://push.example/mati" {
		t.Fatalf("push tidak terkirim: %+v", sender.sent)
	}
	if len(st.deletedEndpoints) != 1 || st.deletedEndpoints[0] != "https://push.example/mati" {
		t.Fatalf("endpoint mati harus dibersihkan: %+v", st.deletedEndpoints)
	}
}
