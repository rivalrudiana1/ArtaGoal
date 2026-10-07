package httphandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"artagoal/goal-service/internal/delivery/http/middleware"
	"artagoal/goal-service/internal/domain/entity"
	"artagoal/goal-service/internal/usecase"
)

// GoalHandler melayani endpoint REST untuk aggregate Goal.
// Bergantung pada interface usecase (bukan implementasi konkret).
type GoalHandler struct {
	uc usecase.GoalUsecase
}

// NewGoalHandler membuat handler baru.
func NewGoalHandler(uc usecase.GoalUsecase) *GoalHandler {
	return &GoalHandler{uc: uc}
}

// RegisterGoalRoutes mendaftarkan endpoint goal pada router chi yang diberikan.
// Pemanggil wajib memasang middleware.AuthMiddleware pada grup route ini
// (lihat cmd/api/main.go); /healthcheck tetap publik di luar grup tersebut.
func (h *GoalHandler) RegisterGoalRoutes(r chi.Router) {
	r.Post("/api/v1/goals", h.CreateGoal)
	r.Get("/api/v1/goals", h.GetMyGoals)
	// Daftarkan sebelum /{id} agar "heatmap" tidak ditangkap sebagai id.
	r.Get("/api/v1/goals/heatmap", h.HandleGetHeatmap)
	r.Get("/api/v1/goals/{id}", h.GetGoalByID)
	r.Get("/api/v1/users/{userID}/goals", h.GetGoalsByUser)
	r.Put("/api/v1/goals/{id}", h.UpdateGoal)
	r.Delete("/api/v1/goals/{id}", h.DeleteGoal)

	// Next feature: contributions + progress/projection.
	r.Post("/api/v1/goals/{id}/contributions", h.CreateContribution)
	r.Get("/api/v1/goals/{id}/contributions", h.ListContributions)
	r.Delete("/api/v1/goals/{id}/contributions/{contributionID}", h.DeleteContribution)
	r.Get("/api/v1/goals/{id}/progress", h.GetProgress)
	r.Get("/api/v1/goals/{id}/projection", h.GetProjection)

	// In-app notifications (dibuat oleh reminder worker).
	r.Get("/api/v1/notifications", h.GetNotifications)
	r.Put("/api/v1/notifications/{id}/read", h.MarkNotificationRead)
}

// ---- DTO ----

// CreateGoalRequest adalah payload POST /api/v1/goals.
// user_id TIDAK lagi diterima dari body — diambil dari JWT context.
// target_date menerima "2006-01-02" atau RFC3339.
type CreateGoalRequest struct {
	Title                 string  `json:"title"`
	Category              string  `json:"category"`
	TargetAmount          float64 `json:"target_amount"`
	CurrentAmount         float64 `json:"current_amount"`
	TargetDate            string  `json:"target_date"`
	ExpectedInflationRate float64 `json:"expected_inflation_rate"`
	Status                string  `json:"status"`
}

// UpdateGoalRequest adalah payload PUT /api/v1/goals/{id} (full update).
// user_id imutabel sehingga tidak termasuk di sini.
type UpdateGoalRequest struct {
	Title                 string  `json:"title"`
	Category              string  `json:"category"`
	TargetAmount          float64 `json:"target_amount"`
	CurrentAmount         float64 `json:"current_amount"`
	TargetDate            string  `json:"target_date"`
	ExpectedInflationRate float64 `json:"expected_inflation_rate"`
	Status                string  `json:"status"`
}

// ProjectionDTO adalah hasil kalkulasi proyeksi anti-inflasi terbaru.
type ProjectionDTO struct {
	FutureTargetAmount   float64 `json:"future_target_amount"`
	MonthlySavingsNeeded float64 `json:"monthly_savings_needed"`
	MonthsRemaining      int     `json:"months_remaining"`
}

// GoalResponse adalah representasi goal + proyeksi terbarunya.
type GoalResponse struct {
	ID                    string         `json:"id"`
	UserID                string         `json:"user_id"`
	Title                 string         `json:"title"`
	Category              string         `json:"category"`
	TargetAmount          float64        `json:"target_amount"`
	CurrentAmount         float64        `json:"current_amount"`
	TargetDate            *time.Time     `json:"target_date,omitempty"`
	ExpectedInflationRate float64        `json:"expected_inflation_rate"`
	Status                string         `json:"status"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	Projection            *ProjectionDTO `json:"projection,omitempty"`
}

// ErrorResponse adalah amplop error JSON yang konsisten.
type ErrorResponse struct {
	Error string `json:"error"`
}

// ---- helpers ----

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}

// statusForError memetakan error domain/usecase ke HTTP status.
func statusForError(err error) int {
	switch {
	case errors.Is(err, entity.ErrGoalNotFound),
		errors.Is(err, entity.ErrContributionNotFound),
		errors.Is(err, entity.ErrNotificationNotFound):
		return http.StatusNotFound
	case errors.Is(err, usecase.ErrValidation):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// decodeJSON membaca body (maks 1 MiB) menjadi v.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "body JSON tidak valid: "+err.Error())
		return false
	}
	return true
}

// ---- auth helpers ----

// currentUserID mengambil user_id terautentikasi dari request context
// (disuntikkan oleh middleware.AuthMiddleware).
func currentUserID(r *http.Request) (string, bool) {
	return middleware.UserIDFromContext(r.Context())
}

// requireAuth memastikan request terautentikasi; menulis 401 bila tidak.
func requireAuth(w http.ResponseWriter, r *http.Request) (string, bool) {
	uid, ok := currentUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized: token tidak ada atau tidak valid")
		return "", false
	}
	return uid, true
}

// ensureOwner memuat goal dan memastikan pemanggil adalah pemiliknya.
// Mengembalikan goal + user_id bila lolos; menulis respons error bila tidak.
func (h *GoalHandler) ensureOwner(w http.ResponseWriter, r *http.Request, goalID string) (entity.Goal, string, bool) {
	uid, ok := requireAuth(w, r)
	if !ok {
		return entity.Goal{}, "", false
	}
	g, err := h.uc.GetGoalByID(r.Context(), goalID)
	if err != nil {
		status := statusForError(err)
		if status == http.StatusInternalServerError {
			writeError(w, status, "gagal mengambil goal")
			return entity.Goal{}, "", false
		}
		writeError(w, status, err.Error())
		return entity.Goal{}, "", false
	}
	if g.UserID != uid {
		writeError(w, http.StatusForbidden, "forbidden: bukan pemilik goal")
		return entity.Goal{}, "", false
	}
	return g, uid, true
}

// dateLayouts adalah format tanggal yang diterima pada input.
var dateLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02",
}

// parseDateFlexible mencoba beberapa format tanggal umum.
func parseDateFlexible(s string) (time.Time, error) {
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("format tanggal tidak dikenali (gunakan YYYY-MM-DD atau RFC3339)")
}

// parseOptionalDate mengembalikan nil untuk string kosong.
func parseOptionalDate(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := parseDateFlexible(s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// toResponse memetakan entity ke DTO respons termasuk proyeksi terbaru.
func (h *GoalHandler) toResponse(g entity.Goal) GoalResponse {
	resp := GoalResponse{
		ID:                    g.ID,
		UserID:                g.UserID,
		Title:                 g.Title,
		Category:              g.Category,
		TargetAmount:          g.TargetAmount,
		CurrentAmount:         g.CurrentAmount,
		TargetDate:            g.TargetDate,
		ExpectedInflationRate: g.ExpectedInflationRate,
		Status:                g.Status,
		CreatedAt:             g.CreatedAt,
		UpdatedAt:             g.UpdatedAt,
	}
	if g.TargetDate != nil {
		fv, monthly := h.uc.CalculateInflationProjection(g.TargetAmount, g.ExpectedInflationRate, *g.TargetDate)
		resp.Projection = &ProjectionDTO{
			FutureTargetAmount:   fv,
			MonthlySavingsNeeded: monthly,
			MonthsRemaining:      usecase.MonthsRemaining(*g.TargetDate, time.Now()),
		}
	}
	return resp
}

// ---- handlers ----

// CreateGoal menangani POST /api/v1/goals.
// user_id diambil dari JWT context (bukan body), lalu disimpan ke DB.
// Mengembalikan 201 Created beserta proyeksinya.
func (h *GoalHandler) CreateGoal(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireAuth(w, r)
	if !ok {
		return
	}

	var req CreateGoalRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	targetDate, err := parseOptionalDate(req.TargetDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "target_date: "+err.Error())
		return
	}

	created, err := h.uc.CreateGoal(r.Context(), entity.Goal{
		UserID:                userID,
		Title:                 req.Title,
		Category:              req.Category,
		TargetAmount:          req.TargetAmount,
		CurrentAmount:         req.CurrentAmount,
		TargetDate:            targetDate,
		ExpectedInflationRate: req.ExpectedInflationRate,
		Status:                req.Status,
	})
	if err != nil {
		status := statusForError(err)
		if status == http.StatusInternalServerError {
			writeError(w, status, "gagal menyimpan goal")
			return
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, h.toResponse(created))
}

// GetGoalByID menangani GET /api/v1/goals/{id}.
// Hanya pemilik goal yang boleh mengakses (403 bila bukan pemilik).
func (h *GoalHandler) GetGoalByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id wajib diisi")
		return
	}

	goal, _, ok := h.ensureOwner(w, r, id)
	if !ok {
		return
	}

	writeJSON(w, http.StatusOK, h.toResponse(goal))
}

// GetMyGoals menangani GET /api/v1/goals.
// Mengembalikan semua goal milik user terautentikasi (dari JWT context).
func (h *GoalHandler) GetMyGoals(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireAuth(w, r)
	if !ok {
		return
	}

	goals, err := h.uc.GetGoalsByUserID(r.Context(), userID)
	if err != nil {
		status := statusForError(err)
		if status == http.StatusInternalServerError {
			writeError(w, status, "gagal mengambil daftar goal")
			return
		}
		writeError(w, status, err.Error())
		return
	}

	resp := make([]GoalResponse, 0, len(goals))
	for _, g := range goals {
		resp = append(resp, h.toResponse(g))
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetGoalsByUser menangani GET /api/v1/users/{userID}/goals.
// Path userID harus sama dengan user terautentikasi (403 bila beda)
// untuk mencegah akses data milik user lain (IDOR).
func (h *GoalHandler) GetGoalsByUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "user_id wajib diisi")
		return
	}

	callerID, ok := requireAuth(w, r)
	if !ok {
		return
	}
	if userID != callerID {
		writeError(w, http.StatusForbidden, "forbidden: hanya boleh mengakses goal milik sendiri")
		return
	}

	goals, err := h.uc.GetGoalsByUserID(r.Context(), userID)
	if err != nil {
		status := statusForError(err)
		if status == http.StatusInternalServerError {
			writeError(w, status, "gagal mengambil daftar goal")
			return
		}
		writeError(w, status, err.Error())
		return
	}

	resp := make([]GoalResponse, 0, len(goals))
	for _, g := range goals {
		resp = append(resp, h.toResponse(g))
	}
	writeJSON(w, http.StatusOK, resp)
}

// UpdateGoal menangani PUT /api/v1/goals/{id}.
// Hanya pemilik yang boleh mengubah; user_id imutabel (dijaga usecase).
func (h *GoalHandler) UpdateGoal(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id wajib diisi")
		return
	}

	if _, _, ok := h.ensureOwner(w, r, id); !ok {
		return
	}

	var req UpdateGoalRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	targetDate, err := parseOptionalDate(req.TargetDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "target_date: "+err.Error())
		return
	}

	updated, err := h.uc.UpdateGoal(r.Context(), entity.Goal{
		ID:                    id,
		Title:                 req.Title,
		Category:              req.Category,
		TargetAmount:          req.TargetAmount,
		CurrentAmount:         req.CurrentAmount,
		TargetDate:            targetDate,
		ExpectedInflationRate: req.ExpectedInflationRate,
		Status:                req.Status,
	})
	if err != nil {
		status := statusForError(err)
		if status == http.StatusInternalServerError {
			writeError(w, status, "gagal memperbarui goal")
			return
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, h.toResponse(updated))
}

// DeleteGoal menangani DELETE /api/v1/goals/{id}.
// Hanya pemilik yang boleh menghapus.
func (h *GoalHandler) DeleteGoal(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id wajib diisi")
		return
	}

	if _, _, ok := h.ensureOwner(w, r, id); !ok {
		return
	}

	if err := h.uc.DeleteGoal(r.Context(), id); err != nil {
		status := statusForError(err)
		if status == http.StatusInternalServerError {
			writeError(w, status, "gagal menghapus goal")
			return
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "goal deleted"})
}

// ---- contributions & progress DTO ----

// CreateContributionRequest adalah payload POST /goals/{id}/contributions.
type CreateContributionRequest struct {
	Amount float64 `json:"amount"`
	Note   string  `json:"note"`
}

// ContributionResponse merepresentasikan satu setoran.
type ContributionResponse struct {
	ID        string    `json:"id"`
	GoalID    string    `json:"goal_id"`
	Amount    float64   `json:"amount"`
	Note      *string   `json:"note,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// ListContributionsResponse membungkus daftar + ringkasan.
type ListContributionsResponse struct {
	Data             []ContributionResponse `json:"data"`
	Count            int                    `json:"count"`
	TotalContributed float64                `json:"total_contributed"`
}

// ProgressResponse menggabungkan snapshot goal + progres terhitung.
type ProgressResponse struct {
	Goal     GoalResponse         `json:"goal"`
	Progress usecase.GoalProgress `json:"progress"`
}

// ProjectionResponse adalah detail proyeksi untuk satu goal.
type ProjectionResponse struct {
	GoalID             string     `json:"goal_id"`
	CurrentAmount      float64    `json:"current_amount"`
	TargetAmount       float64    `json:"target_amount"`
	FutureTargetAmount float64    `json:"future_target_amount,omitempty"`
	MonthlyNeeded      float64    `json:"monthly_needed,omitempty"`
	MonthsRemaining    int        `json:"months_remaining"`
	PercentComplete    float64    `json:"percent_complete"`
	RemainingAmount    float64    `json:"remaining_amount"`
	IsAchieved         bool       `json:"is_achieved"`
	TargetDate         *time.Time `json:"target_date,omitempty"`
}

func toContribution(c entity.GoalContribution) ContributionResponse {
	return ContributionResponse{
		ID: c.ID, GoalID: c.GoalID, Amount: c.Amount, Note: c.Note, CreatedAt: c.CreatedAt,
	}
}

// CreateContribution menangani POST /api/v1/goals/{id}/contributions.
// Body: {"amount": 500000, "note": "Gaji Jan"}. Mengembalikan 201 + goal terbaru.
// Hanya pemilik goal yang boleh menabung.
func (h *GoalHandler) CreateContribution(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id wajib diisi")
		return
	}
	if _, _, ok := h.ensureOwner(w, r, id); !ok {
		return
	}
	var req CreateContributionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	var note *string
	if req.Note != "" {
		n := req.Note
		note = &n
	}
	c, g, err := h.uc.AddContribution(r.Context(), id, req.Amount, note)
	if err != nil {
		status := statusForError(err)
		if status == http.StatusInternalServerError {
			writeError(w, status, "gagal menyimpan kontribusi")
			return
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"contribution": toContribution(c),
		"goal":         h.toResponse(g),
	})
}

// ListContributions menangani GET /api/v1/goals/{id}/contributions.
// Hanya pemilik goal yang boleh melihat.
func (h *GoalHandler) ListContributions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id wajib diisi")
		return
	}
	if _, _, ok := h.ensureOwner(w, r, id); !ok {
		return
	}
	items, err := h.uc.ListContributions(r.Context(), id)
	if err != nil {
		status := statusForError(err)
		if status == http.StatusInternalServerError {
			writeError(w, status, "gagal mengambil kontribusi")
			return
		}
		writeError(w, status, err.Error())
		return
	}
	resp := ListContributionsResponse{Data: make([]ContributionResponse, 0, len(items))}
	for _, c := range items {
		resp.Data = append(resp.Data, toContribution(c))
		resp.TotalContributed += c.Amount
	}
	resp.Count = len(resp.Data)
	writeJSON(w, http.StatusOK, resp)
}

// DeleteContribution menangani DELETE /api/v1/goals/{id}/contributions/{contributionID}.
// Saldo goal diturunkan otomatis; mengembalikan goal terbaru.
// Kepemilikan diverifikasi via goal induk kontribusi sebelum menghapus.
func (h *GoalHandler) DeleteContribution(w http.ResponseWriter, r *http.Request) {
	contributionID := chi.URLParam(r, "contributionID")
	if contributionID == "" {
		writeError(w, http.StatusBadRequest, "contributionID wajib diisi")
		return
	}
	userID, ok := requireAuth(w, r)
	if !ok {
		return
	}
	c, err := h.uc.GetContributionByID(r.Context(), contributionID)
	if err != nil {
		status := statusForError(err)
		if status == http.StatusInternalServerError {
			writeError(w, status, "gagal mengambil kontribusi")
			return
		}
		writeError(w, status, err.Error())
		return
	}
	g, err := h.uc.GetGoalByID(r.Context(), c.GoalID)
	if err != nil {
		status := statusForError(err)
		if status == http.StatusInternalServerError {
			writeError(w, status, "gagal mengambil goal")
			return
		}
		writeError(w, status, err.Error())
		return
	}
	if g.UserID != userID {
		writeError(w, http.StatusForbidden, "forbidden: bukan pemilik goal")
		return
	}
	updated, err := h.uc.DeleteContribution(r.Context(), contributionID)
	if err != nil {
		status := statusForError(err)
		if status == http.StatusInternalServerError {
			writeError(w, status, "gagal menghapus kontribusi")
			return
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, h.toResponse(updated))
}

// GetProgress menangani GET /api/v1/goals/{id}/progress.
// Hanya pemilik goal yang boleh melihat.
func (h *GoalHandler) GetProgress(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id wajib diisi")
		return
	}
	g, _, ok := h.ensureOwner(w, r, id)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, ProgressResponse{
		Goal:     h.toResponse(g),
		Progress: h.uc.GoalProgress(g),
	})
}

// GetProjection menangani GET /api/v1/goals/{id}/projection.
// Ringkas dan stabil untuk konsumsi frontend (tanpa monthly breakdown 600 baris).
// Hanya pemilik goal yang boleh melihat.
func (h *GoalHandler) GetProjection(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id wajib diisi")
		return
	}
	g, _, ok := h.ensureOwner(w, r, id)
	if !ok {
		return
	}
	p := h.uc.GoalProgress(g)
	writeJSON(w, http.StatusOK, ProjectionResponse{
		GoalID: id, CurrentAmount: g.CurrentAmount, TargetAmount: g.TargetAmount,
		FutureTargetAmount: p.FutureTargetAmount, MonthlyNeeded: p.MonthlyNeeded,
		MonthsRemaining: p.MonthsRemaining, PercentComplete: p.PercentComplete,
		RemainingAmount: p.RemainingAmount, IsAchieved: p.IsAchieved, TargetDate: g.TargetDate,
	})
}

// ---- heatmap ----

// HeatmapResponse adalah satu titik agregasi harian untuk heatmap kontribusi.
type HeatmapResponse struct {
	Date        string  `json:"date"`
	Count       int     `json:"count"`
	TotalAmount float64 `json:"total_amount"`
}

// HandleGetHeatmap menangani GET /api/v1/goals/heatmap.
// Mengembalikan agregasi jumlah dan total setoran per hari dalam
// 365 hari terakhir milik user terautentikasi (dari JWT context).
func (h *GoalHandler) HandleGetHeatmap(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireAuth(w, r)
	if !ok {
		return
	}

	items, err := h.uc.GetHeatmapData(r.Context(), userID)
	if err != nil {
		status := statusForError(err)
		if status == http.StatusInternalServerError {
			writeError(w, status, "gagal mengambil data heatmap")
			return
		}
		writeError(w, status, err.Error())
		return
	}

	resp := make([]HeatmapResponse, 0, len(items))
	for _, it := range items {
		resp = append(resp, HeatmapResponse{
			Date:        it.Date,
			Count:       it.Count,
			TotalAmount: it.TotalAmount,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

// ---- notifications ----

// NotificationResponse merepresentasikan satu notifikasi in-app.
type NotificationResponse struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	GoalID    *string   `json:"goal_id,omitempty"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}

func toNotification(n entity.Notification) NotificationResponse {
	return NotificationResponse{
		ID: n.ID, UserID: n.UserID, GoalID: n.GoalID,
		Title: n.Title, Message: n.Message,
		IsRead: n.IsRead, CreatedAt: n.CreatedAt,
	}
}

// GetNotifications menangani GET /api/v1/notifications.
// Mengembalikan daftar notifikasi milik user terautentikasi (terbaru dulu).
func (h *GoalHandler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireAuth(w, r)
	if !ok {
		return
	}

	items, err := h.uc.GetNotifications(r.Context(), userID)
	if err != nil {
		status := statusForError(err)
		if status == http.StatusInternalServerError {
			writeError(w, status, "gagal mengambil notifikasi")
			return
		}
		writeError(w, status, err.Error())
		return
	}

	resp := make([]NotificationResponse, 0, len(items))
	for _, n := range items {
		resp = append(resp, toNotification(n))
	}
	writeJSON(w, http.StatusOK, resp)
}

// MarkNotificationRead menangani PUT /api/v1/notifications/{id}/read.
// Menandai notifikasi milik user terautentikasi sebagai telah dibaca
// (404 bila id tidak ada atau bukan milik user).
func (h *GoalHandler) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireAuth(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id wajib diisi")
		return
	}

	updated, err := h.uc.MarkNotificationRead(r.Context(), userID, id)
	if err != nil {
		status := statusForError(err)
		if status == http.StatusInternalServerError {
			writeError(w, status, "gagal menandai notifikasi")
			return
		}
		if status == http.StatusNotFound {
			writeError(w, status, "notifikasi tidak ditemukan")
			return
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toNotification(updated))
}

