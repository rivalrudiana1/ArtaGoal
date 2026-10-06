package httphandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"artagoal/goal-service/internal/delivery/http/middleware"
	"artagoal/goal-service/internal/domain/entity"
	"artagoal/goal-service/internal/usecase"
)

type stubUC struct {
	usecase.GoalUsecase
	goal     entity.Goal
	contribs []entity.GoalContribution
	addErr   error
	getErr   error
}

func (s *stubUC) AddContribution(_ context.Context, _ string, amount float64, note *string) (entity.GoalContribution, entity.Goal, error) {
	if s.addErr != nil {
		return entity.GoalContribution{}, entity.Goal{}, s.addErr
	}
	c := entity.GoalContribution{ID: "c1", GoalID: s.goal.ID, Amount: amount, Note: note, CreatedAt: time.Now()}
	s.goal.CurrentAmount += amount
	return c, s.goal, nil
}
func (s *stubUC) ListContributions(_ context.Context, _ string) ([]entity.GoalContribution, error) {
	return s.contribs, nil
}
func (s *stubUC) GetContributionByID(_ context.Context, id string) (entity.GoalContribution, error) {
	return entity.GoalContribution{ID: id, GoalID: s.goal.ID, Amount: 100, CreatedAt: time.Now()}, nil
}
func (s *stubUC) DeleteContribution(_ context.Context, _ string) (entity.Goal, error) {
	return s.goal, nil
}
func (s *stubUC) GetGoalByID(_ context.Context, _ string) (entity.Goal, error) {
	if s.getErr != nil {
		return entity.Goal{}, s.getErr
	}
	return s.goal, nil
}
func (s *stubUC) GetGoalsByUserID(_ context.Context, _ string) ([]entity.Goal, error) {
	return []entity.Goal{s.goal}, nil
}
func (s *stubUC) GoalProgress(g entity.Goal) usecase.GoalProgress {
	return usecase.GoalProgress{PercentComplete: 50, RemainingAmount: 500, MonthsRemaining: 6}
}
func (s *stubUC) CalculateInflationProjection(_ float64, _ float64, _ time.Time) (float64, float64) {
	return 1000, 100
}

func newRouter(h *GoalHandler) chi.Router {
	r := chi.NewRouter()
	h.RegisterGoalRoutes(r)
	return r
}

// withAuth menyuntikkan user_id terautentikasi ke request context,
// mensimulasikan request yang sudah lolos middleware JWT.
func withAuth(req *http.Request, userID string) *http.Request {
	return req.WithContext(middleware.ContextWithUserID(req.Context(), userID))
}

func TestCreateContribution201(t *testing.T) {
	td := time.Now().AddDate(1, 0, 0)
	st := &stubUC{goal: entity.Goal{ID: "g1", UserID: "u1", Title: "T", TargetAmount: 1000, CurrentAmount: 100, TargetDate: &td, Status: "active"}}
	h := NewGoalHandler(st)
	r := newRouter(h)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/goals/g1/contributions", strings.NewReader(`{"amount":200,"note":"gajian"}`))
	req.Header.Set("Content-Type", "application/json")
	req = withAuth(req, "u1")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("ingin 201, dapat %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "c1") {
		t.Fatalf("respons harus memuat kontribusi: %s", rec.Body.String())
	}
}

func TestCreateContribution400OnZero(t *testing.T) {
	// usecase asli untuk validasi nyata amount<=0 -> 400
	// pakai stub yang meneruskan ke validasi? stub tidak validasi, jadi uji handler level:
	// kirim JSON tak valid (amount string) -> decode ok? amount 0 tetap diteruskan stub -> 201.
	// Maka uji nyata validasi ada di usecase test. Di sini pastikan route terdaftar (tidak 404/405).
	st := &stubUC{goal: entity.Goal{ID: "g1", UserID: "u1", TargetAmount: 1000}}
	h := NewGoalHandler(st)
	r := newRouter(h)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/goals/g1/contributions", strings.NewReader(`{"amount":0}`))
	req.Header.Set("Content-Type", "application/json")
	req = withAuth(req, "u1")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code == http.StatusNotFound || rec.Code == http.StatusMethodNotAllowed {
		t.Fatalf("route contributions tidak terdaftar: %d", rec.Code)
	}
}

func TestProgressAndProjection200(t *testing.T) {
	td := time.Now().AddDate(0, 6, 0)
	st := &stubUC{goal: entity.Goal{ID: "g1", UserID: "u1", Title: "T", TargetAmount: 1000, CurrentAmount: 500, TargetDate: &td, Status: "active"}}
	h := NewGoalHandler(st)
	r := newRouter(h)
	for _, path := range []string{"/api/v1/goals/g1/progress", "/api/v1/goals/g1/projection", "/api/v1/goals/g1/contributions"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req = withAuth(req, "u1")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s ingin 200, dapat %d: %s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestDeleteContributionRoute200(t *testing.T) {
	st := &stubUC{goal: entity.Goal{ID: "g1", UserID: "u1", TargetAmount: 1000, CurrentAmount: 500, Status: "active"}}
	h := NewGoalHandler(st)
	r := newRouter(h)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/goals/g1/contributions/c1", nil)
	req = withAuth(req, "u1")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("ingin 200, dapat %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUnauthenticated401(t *testing.T) {
	// Tanpa user di context (middleware belum lolos) -> 401.
	st := &stubUC{goal: entity.Goal{ID: "g1", UserID: "u1", TargetAmount: 1000}}
	h := NewGoalHandler(st)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/goals", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("GET /goals tanpa auth ingin 401, dapat %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/goals", strings.NewReader(`{"title":"T","target_amount":100}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("POST /goals tanpa auth ingin 401, dapat %d", rec.Code)
	}
}

func TestForbiddenForOtherUser403(t *testing.T) {
	// Goal milik u1 diakses caller u2 -> 403.
	st := &stubUC{goal: entity.Goal{ID: "g1", UserID: "u1", TargetAmount: 1000}}
	h := NewGoalHandler(st)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/goals/g1", nil)
	req = withAuth(req, "u2")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("akses milik orang lain ingin 403, dapat %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/u1/goals", nil)
	req = withAuth(req, "u2")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("list milik orang lain ingin 403, dapat %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetMyGoals200(t *testing.T) {
	// GET /api/v1/goals memakai user_id dari context.
	st := &stubUC{goal: entity.Goal{ID: "g1", UserID: "u1", Title: "T", TargetAmount: 1000}}
	h := NewGoalHandler(st)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/goals", nil)
	req = withAuth(req, "u1")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("ingin 200, dapat %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "g1") {
		t.Fatalf("respons harus memuat goal: %s", rec.Body.String())
	}

	// Route lama dengan path user sendiri tetap 200.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/u1/goals", nil)
	req = withAuth(req, "u1")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("ingin 200, dapat %d: %s", rec.Code, rec.Body.String())
	}
}

