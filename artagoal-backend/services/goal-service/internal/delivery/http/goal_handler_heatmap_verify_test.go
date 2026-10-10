package httphandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"artagoal/goal-service/internal/delivery/http/middleware"
	"artagoal/goal-service/internal/domain/entity"
)

type heatmapStubUC struct {
	stubUC
	heatmap []entity.HeatmapData
}

func (s *heatmapStubUC) GetHeatmapData(_ context.Context, _ string) ([]entity.HeatmapData, error) {
	return s.heatmap, nil
}

func TestHeatmapRouteNotShadowedByID(t *testing.T) {
	st := &heatmapStubUC{
		stubUC:  stubUC{goal: entity.Goal{ID: "g1", UserID: "u1", Title: "T", TargetAmount: 1000}},
		heatmap: []entity.HeatmapData{{Date: "2026-10-07", Count: 2, TotalAmount: 50000}},
	}
	h := NewGoalHandler(st)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/goals/heatmap", nil)
	req = req.WithContext(middleware.ContextWithUserID(req.Context(), "u1"))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ingin 200, dapat %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "2026-10-07") {
		t.Fatalf("respons harus memuat data heatmap: %s", rec.Body.String())
	}

	// Tanpa auth harus 401, bukan 403/404 (bukti route heatmap yang menangani).
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/goals/heatmap", nil)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusUnauthorized {
		t.Fatalf("tanpa auth ingin 401, dapat %d: %s", rec2.Code, rec2.Body.String())
	}
}

func TestHeatmapRouteWithoutPrefix(t *testing.T) {
	st := &heatmapStubUC{
		stubUC:  stubUC{goal: entity.Goal{ID: "g1", UserID: "u1", Title: "T", TargetAmount: 1000}},
		heatmap: []entity.HeatmapData{{Date: "2026-10-07", Count: 1, TotalAmount: 25000}},
	}
	h := NewGoalHandler(st)
	r := newRouter(h)

	// Varian tanpa /api/v1 harus ditangani handler yang sama, bukan {id}.
	req := httptest.NewRequest(http.MethodGet, "/goals/heatmap", nil)
	req = req.WithContext(middleware.ContextWithUserID(req.Context(), "u1"))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ingin 200, dapat %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "2026-10-07") {
		t.Fatalf("respons harus memuat data heatmap: %s", rec.Body.String())
	}
}

func TestHeatmapEmptyReturnsArrayNotNull(t *testing.T) {
	// Simulasi user dengan 0 setoran (usecase mengembalikan nil).
	st := &heatmapStubUC{
		stubUC:  stubUC{goal: entity.Goal{ID: "g1", UserID: "u1", Title: "T", TargetAmount: 1000}},
		heatmap: nil,
	}
	h := NewGoalHandler(st)
	r := newRouter(h)

	for _, path := range []string{"/api/v1/goals/heatmap", "/goals/heatmap"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req = req.WithContext(middleware.ContextWithUserID(req.Context(), "u1"))
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("%s ingin 200, dapat %d: %s", path, rec.Code, rec.Body.String())
		}
		if strings.TrimSpace(rec.Body.String()) != "[]" {
			t.Fatalf("%s harus mengembalikan [], dapat: %s", path, rec.Body.String())
		}
	}
}
