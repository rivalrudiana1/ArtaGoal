package httphandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"artagoal/goal-service/internal/domain/entity"
	"artagoal/goal-service/internal/usecase"
)

// dbDownUC menyimulasikan database mati: semua query usecase gagal.
type dbDownUC struct {
	usecase.GoalUsecase
	err error
}

func (s *dbDownUC) GetGoalsByUserID(_ context.Context, _ string, _, _ int) ([]entity.Goal, int, error) {
	return nil, 0, s.err
}

func (s *dbDownUC) GetNotifications(_ context.Context, _ string, _, _ int) ([]entity.Notification, int, error) {
	return nil, 0, s.err
}

func (s *dbDownUC) GetHeatmapData(_ context.Context, _ string) ([]entity.HeatmapData, error) {
	return nil, s.err
}

// TestHandlersReturnSafeJSONOnDBError memastikan saat DB down, ketiga
// handler mengembalikan JSON error 500 (bukan panic / body kosong).
func TestHandlersReturnSafeJSONOnDBError(t *testing.T) {
	h := NewGoalHandler(&dbDownUC{err: errors.New("postgres: connection refused")})
	r := newRouter(h)

	for path, method := range map[string]string{
		"/api/v1/goals":         http.MethodGet,
		"/api/v1/notifications": http.MethodGet,
		"/api/v1/goals/heatmap": http.MethodGet,
		"/goals":                http.MethodGet,
		"/notifications":        http.MethodGet,
		"/goals/heatmap":        http.MethodGet,
	} {
		req := httptest.NewRequest(method, path, nil)
		req = withAuth(req, "u1")
		rec := httptest.NewRecorder()

		// Harus kembali normal (chi Recoverer tidak boleh terpicu).
		func() {
			defer func() {
				if v := recover(); v != nil {
					t.Fatalf("%s panic: %v", path, v)
				}
			}()
			r.ServeHTTP(rec, req)
		}()

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("%s ingin 500, dapat %d: %s", path, rec.Code, rec.Body.String())
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Fatalf("%s Content-Type ingin application/json, dapat %q", path, ct)
		}
		var body map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s body bukan JSON: %s", path, rec.Body.String())
		}
		if body["error"] == "" {
			t.Fatalf("%s JSON harus memuat field error: %s", path, rec.Body.String())
		}
	}
}
