package httphandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"artagoal/goal-service/internal/domain/entity"
	"artagoal/goal-service/internal/usecase"
)

// notifStubUC memperluas stubUC dengan jalur notifikasi.
type notifStubUC struct {
	stubUC
	notifs   []entity.Notification
	marked   entity.Notification
	vapidKey string
}

func (s *notifStubUC) GetNotifications(_ context.Context, _ string, _, _ int) ([]entity.Notification, int, error) {
	return s.notifs, len(s.notifs), nil
}

func (s *notifStubUC) MarkNotificationRead(_ context.Context, _, _ string) (entity.Notification, error) {
	s.marked.IsRead = true
	return s.marked, nil
}

func (s *notifStubUC) GetStats(_ context.Context, _ string) (entity.GoalStats, error) {
	return entity.GoalStats{
		TotalContributions: 12, ActiveDays365: 5,
		CurrentStreakDays: 3, LongestStreakDays: 3,
		Level: "Penabung Rutin", LevelProgress: 10, NextLevelAt: &[]int{30}[0],
	}, nil
}

func (s *notifStubUC) PushPublicKey() (string, error) {
	if s.vapidKey == "" {
		return "", usecase.ErrValidation
	}
	return s.vapidKey, nil
}

func (s *notifStubUC) SavePushSubscription(_ context.Context, userID, endpoint, p256dh, auth string) (entity.PushSubscription, error) {
	return entity.PushSubscription{ID: "ps1", UserID: userID, Endpoint: endpoint, P256dh: p256dh, Auth: auth}, nil
}

func (s *notifStubUC) DeletePushSubscription(_ context.Context, _, _ string) error {
	return nil
}

func TestGetNotifications200(t *testing.T) {
	st := &notifStubUC{
		stubUC: stubUC{goal: entity.Goal{ID: "g1", UserID: "u1"}},
		notifs: []entity.Notification{
			{ID: "n1", UserID: "u1", Title: "🚨 Peringatan Tenggat Target", Message: "Tes", CreatedAt: time.Now()},
		},
	}
	h := NewGoalHandler(st)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
	req = withAuth(req, "u1")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ingin 200, dapat %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Peringatan Tenggat Target") {
		t.Fatalf("respons harus memuat notifikasi: %s", rec.Body.String())
	}
}

func TestMarkNotificationRead200(t *testing.T) {
	st := &notifStubUC{
		stubUC: stubUC{goal: entity.Goal{ID: "g1", UserID: "u1"}},
		marked: entity.Notification{ID: "n1", UserID: "u1", Title: "T"},
	}
	h := NewGoalHandler(st)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/notifications/n1/read", nil)
	req = withAuth(req, "u1")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ingin 200, dapat %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"is_read":true`) {
		t.Fatalf("respons harus bertanda dibaca: %s", rec.Body.String())
	}
}

func TestGetStats200(t *testing.T) {
	st := &notifStubUC{stubUC: stubUC{goal: entity.Goal{ID: "g1", UserID: "u1"}}}
	h := NewGoalHandler(st)
	r := newRouter(h)

	// "stats" harus ditangani handler stats, bukan GetGoalByID.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/goals/stats", nil)
	req = withAuth(req, "u1")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ingin 200, dapat %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Penabung Rutin") {
		t.Fatalf("respons harus memuat level: %s", rec.Body.String())
	}
}

func TestNotificationsUnauthenticated401(t *testing.T) {
	st := &notifStubUC{stubUC: stubUC{goal: entity.Goal{ID: "g1", UserID: "u1"}}}
	h := NewGoalHandler(st)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("GET tanpa auth ingin 401, dapat %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPut, "/api/v1/notifications/n1/read", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("PUT tanpa auth ingin 401, dapat %d", rec.Code)
	}
}

func TestPushEndpoints(t *testing.T) {
	st := &notifStubUC{
		stubUC:   stubUC{goal: entity.Goal{ID: "g1", UserID: "u1"}},
		vapidKey: "PUBKEY-TEST",
	}
	h := NewGoalHandler(st)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/push/vapid-public-key", nil)
	req = withAuth(req, "u1")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "PUBKEY-TEST") {
		t.Fatalf("vapid key ingin 200 + kunci, dapat %d: %s", rec.Code, rec.Body.String())
	}

	body := `{"endpoint":"https://push.example/e1","keys":{"p256dh":"p","auth":"a"}}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/push/subscriptions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withAuth(req, "u1")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("save subscription ingin 201, dapat %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/push/subscriptions", strings.NewReader(`{"endpoint":"https://push.example/e1"}`))
	req.Header.Set("Content-Type", "application/json")
	req = withAuth(req, "u1")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete subscription ingin 200, dapat %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPushPublicKeyUnavailable503(t *testing.T) {
	st := &notifStubUC{stubUC: stubUC{goal: entity.Goal{ID: "g1", UserID: "u1"}}}
	h := NewGoalHandler(st)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/push/vapid-public-key", nil)
	req = withAuth(req, "u1")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("tanpa kunci ingin 503, dapat %d: %s", rec.Code, rec.Body.String())
	}
}
