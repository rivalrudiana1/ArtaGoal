package httphandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"artagoal/goal-service/internal/domain/entity"
)

// notifStubUC memperluas stubUC dengan jalur notifikasi.
type notifStubUC struct {
	stubUC
	notifs []entity.Notification
	marked entity.Notification
}

func (s *notifStubUC) GetNotifications(_ context.Context, _ string) ([]entity.Notification, error) {
	return s.notifs, nil
}

func (s *notifStubUC) MarkNotificationRead(_ context.Context, _, _ string) (entity.Notification, error) {
	s.marked.IsRead = true
	return s.marked, nil
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
