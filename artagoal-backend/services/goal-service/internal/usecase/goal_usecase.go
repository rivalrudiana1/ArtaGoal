package usecase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"artagoal/goal-service/internal/domain/entity"
	"artagoal/goal-service/internal/repository"
)

// ErrValidation menandai input tidak valid.
// Delivery layer memetakannya ke HTTP 400 Bad Request via errors.Is.
var ErrValidation = errors.New("validation error")

// GoalUsecase adalah kontrak bisnis untuk aggregate Goal.
type GoalUsecase interface {
	CreateGoal(ctx context.Context, goal entity.Goal) (entity.Goal, error)
	GetGoalByID(ctx context.Context, id string) (entity.Goal, error)
	GetGoalsByUserID(ctx context.Context, userID string) ([]entity.Goal, error)
	UpdateGoal(ctx context.Context, goal entity.Goal) (entity.Goal, error)
	DeleteGoal(ctx context.Context, id string) error

	// --- contributions ---
	AddContribution(ctx context.Context, goalID string, amount float64, note *string) (entity.GoalContribution, entity.Goal, error)
	ListContributions(ctx context.Context, goalID string) ([]entity.GoalContribution, error)
	GetContributionByID(ctx context.Context, id string) (entity.GoalContribution, error)
	DeleteContribution(ctx context.Context, contributionID string) (entity.Goal, error)

	// GetHeatmapData mengembalikan agregasi kontribusi per hari
	// dalam 1 tahun terakhir untuk heatmap kontribusi.
	GetHeatmapData(ctx context.Context, userID string) ([]entity.HeatmapData, error)

	// --- notifications ---
	// GetNotifications mengembalikan daftar notifikasi milik user.
	GetNotifications(ctx context.Context, userID string) ([]entity.Notification, error)
	// MarkNotificationRead menandai satu notifikasi milik user sebagai dibaca.
	MarkNotificationRead(ctx context.Context, userID, notificationID string) (entity.Notification, error)
	// RunDeadlineReminderCheck memindai goal active yang tenggatnya <= 30 hari
	// dengan progres < 80% lalu membuat notifikasi peringatan (anti-duplikat
	// 7 hari). Mengembalikan jumlah notifikasi yang dibuat.
	RunDeadlineReminderCheck(ctx context.Context) (int, error)

	// CalculateInflationProjection menghitung proyeksi anti-inflasi:
	//   FV = PV * (1 + i)^n, dengan i = inflationRate/100 (desimal per tahun),
	//   n  = durasi tahun dari sekarang ke targetDate.
	//   monthly = FV / total bulan sisa (dibulatkan ke atas, minimal 1).
	CalculateInflationProjection(targetAmount float64, inflationRate float64, targetDate time.Time) (futureTargetAmount float64, monthlySavingsNeeded float64)
	// GoalProgress menghitung ringkasan progres untuk satu goal.
	GoalProgress(goal entity.Goal) GoalProgress
}

// GoalProgress adalah ringkasan progres + proyeksi satu goal.
type GoalProgress struct {
	PercentComplete    float64 `json:"percent_complete"`
	RemainingAmount    float64 `json:"remaining_amount"`
	FutureTargetAmount float64 `json:"future_target_amount,omitempty"`
	MonthlyNeeded      float64 `json:"monthly_needed,omitempty"`
	MonthsRemaining    int     `json:"months_remaining"`
	IsAchieved         bool    `json:"is_achieved"`
}

// goalUsecase implementasi GoalUsecase.
type goalUsecase struct {
	repo repository.GoalRepository
	now  func() time.Time
}

// NewGoalUsecase membuat usecase baru.
func NewGoalUsecase(repo repository.GoalRepository) GoalUsecase {
	return &goalUsecase{repo: repo, now: time.Now}
}

// validationError membungkus pesan validasi agar terdeteksi errors.Is(err, ErrValidation).
func validationError(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrValidation, fmt.Sprintf(format, args...))
}

// normalizeStatus mengisi status default dan memvalidasi nilainya.
func normalizeStatus(status string) (string, error) {
	if status == "" {
		return entity.GoalStatusActive, nil
	}
	switch status {
	case entity.GoalStatusActive, entity.GoalStatusAchieved, entity.GoalStatusCancelled:
		return status, nil
	default:
		return "", validationError("status %q tidak valid (active|achieved|cancelled)", status)
	}
}

// validateGoalFields memeriksa field bisnis yang berlaku untuk create & update.
func validateGoalFields(goal *entity.Goal, now time.Time) error {
	if goal.UserID == "" {
		return validationError("user_id wajib diisi")
	}
	if goal.Title == "" {
		return validationError("title wajib diisi")
	}
	if goal.TargetAmount <= 0 {
		return validationError("target_amount harus > 0")
	}
	if goal.CurrentAmount < 0 {
		return validationError("current_amount tidak boleh negatif")
	}
	if goal.ExpectedInflationRate < 0 {
		return validationError("expected_inflation_rate tidak boleh negatif")
	}
	if goal.TargetDate != nil && !goal.TargetDate.After(now) {
		return validationError("target_date harus di masa depan")
	}
	status, err := normalizeStatus(goal.Status)
	if err != nil {
		return err
	}
	goal.Status = status
	return nil
}

// CreateGoal memvalidasi input lalu mendelegasikan penyimpanan ke repository.
func (u *goalUsecase) CreateGoal(ctx context.Context, goal entity.Goal) (entity.Goal, error) {
	if err := validateGoalFields(&goal, u.now()); err != nil {
		return entity.Goal{}, err
	}
	created, err := u.repo.CreateGoal(ctx, goal)
	if err != nil {
		return entity.Goal{}, fmt.Errorf("usecase: CreateGoal: %w", err)
	}
	return created, nil
}

// GetGoalByID mengambil satu goal berdasarkan id.
func (u *goalUsecase) GetGoalByID(ctx context.Context, id string) (entity.Goal, error) {
	if id == "" {
		return entity.Goal{}, validationError("id wajib diisi")
	}
	goal, err := u.repo.GetGoalByID(ctx, id)
	if err != nil {
		return entity.Goal{}, fmt.Errorf("usecase: GetGoalByID: %w", err)
	}
	return goal, nil
}

// GetGoalsByUserID mengambil semua goal milik satu user.
func (u *goalUsecase) GetGoalsByUserID(ctx context.Context, userID string) ([]entity.Goal, error) {
	if userID == "" {
		return nil, validationError("user_id wajib diisi")
	}
	goals, err := u.repo.GetGoalsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("usecase: GetGoalsByUserID: %w", err)
	}
	return goals, nil
}

// UpdateGoal memuat entity lama (agar field imutabel seperti user_id
// dan created_at terjaga), menimpa field mutable, memvalidasi,
// lalu menyimpan via repository.
func (u *goalUsecase) UpdateGoal(ctx context.Context, goal entity.Goal) (entity.Goal, error) {
	if goal.ID == "" {
		return entity.Goal{}, validationError("id wajib diisi")
	}

	existing, err := u.repo.GetGoalByID(ctx, goal.ID)
	if err != nil {
		return entity.Goal{}, fmt.Errorf("usecase: UpdateGoal load: %w", err)
	}

	existing.Title = goal.Title
	existing.Category = goal.Category
	existing.TargetAmount = goal.TargetAmount
	existing.CurrentAmount = goal.CurrentAmount
	existing.TargetDate = goal.TargetDate
	existing.ExpectedInflationRate = goal.ExpectedInflationRate
	if goal.Status != "" {
		existing.Status = goal.Status
	}

	if err := validateGoalFields(&existing, u.now()); err != nil {
		return entity.Goal{}, err
	}
	updated, err := u.repo.UpdateGoal(ctx, existing)
	if err != nil {
		return entity.Goal{}, fmt.Errorf("usecase: UpdateGoal: %w", err)
	}
	return updated, nil
}

// DeleteGoal menghapus goal berdasarkan id.
func (u *goalUsecase) DeleteGoal(ctx context.Context, id string) error {
	if id == "" {
		return validationError("id wajib diisi")
	}
	if err := u.repo.DeleteGoal(ctx, id); err != nil {
		return fmt.Errorf("usecase: DeleteGoal: %w", err)
	}
	return nil
}

// meanDaysPerMonth adalah rata-rata hari per bulan Gregorian (365.25/12).
const meanDaysPerMonth = 30.4375

// InflationProjection adalah versi murni (pure function) dari kalkulasi
// proyeksi anti-inflasi, menerima waktu "sekarang" sebagai parameter
// agar deterministik dan mudah diuji.
func InflationProjection(targetAmount float64, inflationRate float64, targetDate time.Time, now time.Time) (futureTargetAmount float64, monthlySavingsNeeded float64) {
	days := targetDate.Sub(now).Hours() / 24
	if days < 0 {
		days = 0
	}

	years := days / 365
	futureTargetAmount = targetAmount * math.Pow(1+inflationRate/100, years)

	months := int(math.Ceil(days / meanDaysPerMonth))
	if months < 1 {
		// Tidak ada waktu tersisa: dana harus tersedia penuh sekaligus.
		months = 1
	}
	monthlySavingsNeeded = futureTargetAmount / float64(months)
	return futureTargetAmount, monthlySavingsNeeded
}

// MonthsRemaining menghitung sisa bulan kalender ke targetDate
// (dibulatkan ke atas, 0 jika sudah lewat). Murni untuk kebutuhan tampilan.
func MonthsRemaining(targetDate time.Time, now time.Time) int {
	days := targetDate.Sub(now).Hours() / 24
	if days <= 0 {
		return 0
	}
	return int(math.Ceil(days / meanDaysPerMonth))
}

// CalculateInflationProjection mengimplementasikan kontrak interface
// dengan jam sistem sebagai acuan "sekarang".
func (u *goalUsecase) CalculateInflationProjection(targetAmount float64, inflationRate float64, targetDate time.Time) (float64, float64) {
	return InflationProjection(targetAmount, inflationRate, targetDate, u.now())
}

// validateContributionInput memeriksa nominal & catatan kontribusi.
func validateContributionInput(goalID string, amount float64, note *string) error {
	if goalID == "" {
		return validationError("goal_id wajib diisi")
	}
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		return validationError("amount harus angka valid")
	}
	if amount <= 0 {
		return validationError("amount harus > 0")
	}
	if amount > 1e15 {
		return validationError("amount terlalu besar")
	}
	if note != nil && len(*note) > 500 {
		return validationError("note maksimal 500 karakter")
	}
	return nil
}

// AddContribution memvalidasi bisnis lalu mendelegasikan ke repository
// (insert + bump current_amount atomik di dalam transaksi).
func (u *goalUsecase) AddContribution(ctx context.Context, goalID string, amount float64, note *string) (entity.GoalContribution, entity.Goal, error) {
	if err := validateContributionInput(goalID, amount, note); err != nil {
		return entity.GoalContribution{}, entity.Goal{}, err
	}

	existing, err := u.repo.GetGoalByID(ctx, goalID)
	if err != nil {
		return entity.GoalContribution{}, entity.Goal{}, fmt.Errorf("usecase: AddContribution load goal: %w", err)
	}
	if existing.Status == entity.GoalStatusCancelled {
		return entity.GoalContribution{}, entity.Goal{}, validationError("tidak bisa menabung ke goal yang cancelled")
	}

	c, g, err := u.repo.AddContribution(ctx, goalID, amount, note)
	if err != nil {
		return entity.GoalContribution{}, entity.Goal{}, fmt.Errorf("usecase: AddContribution: %w", err)
	}
	return c, g, nil
}

// ListContributions mengembalikan daftar setoran satu goal (terbaru dulu).
func (u *goalUsecase) ListContributions(ctx context.Context, goalID string) ([]entity.GoalContribution, error) {
	if goalID == "" {
		return nil, validationError("goal_id wajib diisi")
	}
	out, err := u.repo.ListContributionsByGoalID(ctx, goalID)
	if err != nil {
		return nil, fmt.Errorf("usecase: ListContributions: %w", err)
	}
	return out, nil
}

// GetContributionByID mengambil satu setoran berdasarkan id.
func (u *goalUsecase) GetContributionByID(ctx context.Context, id string) (entity.GoalContribution, error) {
	if id == "" {
		return entity.GoalContribution{}, validationError("contribution id wajib diisi")
	}
	c, err := u.repo.GetContributionByID(ctx, id)
	if err != nil {
		return entity.GoalContribution{}, fmt.Errorf("usecase: GetContributionByID: %w", err)
	}
	return c, nil
}

// DeleteContribution menghapus satu setoran dan menyesuaikan saldo goal.
func (u *goalUsecase) DeleteContribution(ctx context.Context, contributionID string) (entity.Goal, error) {
	if contributionID == "" {
		return entity.Goal{}, validationError("contribution id wajib diisi")
	}
	g, err := u.repo.DeleteContribution(ctx, contributionID)
	if err != nil {
		return entity.Goal{}, fmt.Errorf("usecase: DeleteContribution: %w", err)
	}
	return g, nil
}

// GetHeatmapData memvalidasi user lalu mendelegasikan agregasi
// heatmap ke repository.
func (u *goalUsecase) GetHeatmapData(ctx context.Context, userID string) ([]entity.HeatmapData, error) {
	if userID == "" {
		return nil, validationError("user_id wajib diisi")
	}
	out, err := u.repo.GetContributionHeatmap(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("usecase: GetHeatmapData: %w", err)
	}
	if out == nil {
		out = []entity.HeatmapData{}
	}
	return out, nil
}

// Batas kriteria pengingat tenggat oleh reminder worker.
const (
	// reminderDueDays adalah jarak tenggat maksimal agar diingatkan.
	reminderDueDays = 30
	// reminderMinProgress adalah progres minimal agar TIDAK diingatkan.
	reminderMinProgress = 80.0
	// reminderDedupDays adalah jendela anti-duplikat notifikasi serupa.
	reminderDedupDays = 7
)

// reminderTitle adalah judul notifikasi peringatan tenggat.
const reminderTitle = "🚨 Peringatan Tenggat Target"

// shortPercent memformat persen ringkas: bulat tanpa desimal,
// selain itu satu angka desimal (mis. 45% atau 45.5%).
func shortPercent(p float64) string {
	if p == math.Trunc(p) {
		return fmt.Sprintf("%.0f", p)
	}
	return fmt.Sprintf("%.1f", p)
}

// GetNotifications mengembalikan daftar notifikasi milik satu user.
func (u *goalUsecase) GetNotifications(ctx context.Context, userID string) ([]entity.Notification, error) {
	if userID == "" {
		return nil, validationError("user_id wajib diisi")
	}
	out, err := u.repo.GetNotificationsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("usecase: GetNotifications: %w", err)
	}
	if out == nil {
		out = []entity.Notification{}
	}
	return out, nil
}

// MarkNotificationRead menandai satu notifikasi milik user sebagai dibaca.
func (u *goalUsecase) MarkNotificationRead(ctx context.Context, userID, notificationID string) (entity.Notification, error) {
	if userID == "" {
		return entity.Notification{}, validationError("user_id wajib diisi")
	}
	if notificationID == "" {
		return entity.Notification{}, validationError("notification id wajib diisi")
	}
	n, err := u.repo.MarkNotificationAsRead(ctx, notificationID, userID)
	if err != nil {
		return entity.Notification{}, fmt.Errorf("usecase: MarkNotificationRead: %w", err)
	}
	return n, nil
}

// RunDeadlineReminderCheck memindai goal active bertenggat <= 30 hari yang
// progresnya masih < 80%, lalu membuat satu notifikasi peringatan per goal
// (dilewati bila notifikasi serupa sudah ada dalam 7 hari terakhir).
func (u *goalUsecase) RunDeadlineReminderCheck(ctx context.Context) (int, error) {
	goals, err := u.repo.ListActiveGoalsDueWithin(ctx, reminderDueDays)
	if err != nil {
		return 0, fmt.Errorf("usecase: RunDeadlineReminderCheck list: %w", err)
	}

	now := u.now()
	created := 0
	for _, g := range goals {
		if g.TargetDate == nil || g.TargetAmount <= 0 {
			continue
		}
		daysLeft := int(math.Ceil(g.TargetDate.Sub(now).Hours() / 24))
		if daysLeft <= 0 {
			continue
		}
		percent := g.CurrentAmount / g.TargetAmount * 100
		if percent >= reminderMinProgress {
			continue
		}

		recent, err := u.repo.HasRecentNotification(ctx, g.ID, reminderTitle, reminderDedupDays)
		if err != nil {
			return created, fmt.Errorf("usecase: RunDeadlineReminderCheck dedup %s: %w", g.ID, err)
		}
		if recent {
			continue
		}

		message := fmt.Sprintf(
			"Target '%s' tinggal %d hari lagi, namun saldo Anda baru mencapai %s%%. Segera tambah setoran!",
			g.Title, daysLeft, shortPercent(percent),
		)
		goalID := g.ID
		if _, err := u.repo.CreateNotification(ctx, entity.Notification{
			UserID:  g.UserID,
			GoalID:  &goalID,
			Title:   reminderTitle,
			Message: message,
		}); err != nil {
			return created, fmt.Errorf("usecase: RunDeadlineReminderCheck create %s: %w", g.ID, err)
		}
		created++
	}

	return created, nil
}

// GoalProgress menghitung persen tercapai, sisa, dan proyeksi inflasi.
func (u *goalUsecase) GoalProgress(goal entity.Goal) GoalProgress {
	now := u.now()
	p := GoalProgress{IsAchieved: goal.Status == entity.GoalStatusAchieved}
	if goal.TargetAmount > 0 {
		p.PercentComplete = goal.CurrentAmount / goal.TargetAmount * 100
		if p.PercentComplete < 0 {
			p.PercentComplete = 0
		}
		if p.PercentComplete > 100 && !p.IsAchieved {
			// Cap tampilan 100% kecuali over-fund; biarkan apa adanya tapi tidak negatif.
			// Tidak di-cap keras agar UI bisa menampilkan 112% dsb.
		}
		p.RemainingAmount = goal.TargetAmount - goal.CurrentAmount
		if p.RemainingAmount < 0 {
			p.RemainingAmount = 0
		}
		if goal.CurrentAmount >= goal.TargetAmount {
			p.IsAchieved = true
		}
	}
	if goal.TargetDate != nil {
		p.MonthsRemaining = MonthsRemaining(*goal.TargetDate, now)
		fv, monthly := InflationProjection(goal.TargetAmount, goal.ExpectedInflationRate, *goal.TargetDate, now)
		p.FutureTargetAmount = fv
		// Kebutuhan bulanan dari kekurangan FV (bukan full FV) agar realistis.
		shortfall := fv - goal.CurrentAmount
		if shortfall < 0 {
			shortfall = 0
		}
		months := p.MonthsRemaining
		if months < 1 {
			months = 1
		}
		_ = monthly // nilai full-FV/div; kita pakai shortfall-based:
		p.MonthlyNeeded = shortfall / float64(months)
	}
	return p
}

