package usecase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"artagoal/goal-service/internal/domain/entity"
	"artagoal/goal-service/internal/push"
	"artagoal/goal-service/internal/repository"
)

// ErrValidation menandai input tidak valid.
// Delivery layer memetakannya ke HTTP 400 Bad Request via errors.Is.
var ErrValidation = errors.New("validation error")

// GoalUsecase adalah kontrak bisnis untuk aggregate Goal.
type GoalUsecase interface {
	CreateGoal(ctx context.Context, goal entity.Goal) (entity.Goal, error)
	GetGoalByID(ctx context.Context, id string) (entity.Goal, error)
	// GetGoalsByUserID mengembalikan satu halaman goal + total keseluruhan.
	GetGoalsByUserID(ctx context.Context, userID string, page, limit int) ([]entity.Goal, int, error)
	UpdateGoal(ctx context.Context, goal entity.Goal) (entity.Goal, error)
	DeleteGoal(ctx context.Context, id string) error

	// --- contributions ---
	AddContribution(ctx context.Context, goalID string, amount float64, note *string) (entity.GoalContribution, entity.Goal, error)
	// ListContributions mengembalikan satu halaman setoran + jumlah dan
	// total nominal keseluruhan (untuk envelope respons).
	ListContributions(ctx context.Context, goalID string, page, limit int) (items []entity.GoalContribution, count int, total float64, err error)
	GetContributionByID(ctx context.Context, id string) (entity.GoalContribution, error)
	DeleteContribution(ctx context.Context, contributionID string) (entity.Goal, error)

	// GetHeatmapData mengembalikan agregasi kontribusi per hari
	// dalam 1 tahun terakhir untuk heatmap kontribusi.
	GetHeatmapData(ctx context.Context, userID string) ([]entity.HeatmapData, error)

	// --- notifications ---
	// GetNotifications mengembalikan satu halaman notifikasi + total.
	GetNotifications(ctx context.Context, userID string, page, limit int) ([]entity.Notification, int, error)
	// MarkNotificationRead menandai satu notifikasi milik user sebagai dibaca.
	MarkNotificationRead(ctx context.Context, userID, notificationID string) (entity.Notification, error)
	// RunDeadlineReminderCheck memindai goal active yang tenggatnya <= 30 hari
	// dengan progres < 80% lalu membuat notifikasi peringatan (anti-duplikat
	// 7 hari). Mengembalikan jumlah notifikasi yang dibuat.
	RunDeadlineReminderCheck(ctx context.Context) (int, error)

	// GetStats menghitung ringkasan gamifikasi (streak & level) milik user.
	GetStats(ctx context.Context, userID string) (entity.GoalStats, error)

	// --- web push ---
	// SetPushSender memasang pengirim push (nil = push nonaktif).
	SetPushSender(sender PushSender)
	// PushPublicKey mengembalikan kunci publik VAPID untuk klien.
	PushPublicKey() (string, error)
	// SavePushSubscription mendaftarkan langganan push milik user.
	SavePushSubscription(ctx context.Context, userID, endpoint, p256dh, auth string) (entity.PushSubscription, error)
	// DeletePushSubscription menghapus langganan push milik user.
	DeletePushSubscription(ctx context.Context, userID, endpoint string) error

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
	repo       repository.GoalRepository
	now        func() time.Time
	pushSender PushSender
}

// PushSender mengirim satu push Web Push.
// gone=true berarti endpoint sudah mati dan langganannya dibersihkan.
type PushSender interface {
	PublicKey() string
	Send(sub push.Subscription, title, body, url string) (gone bool, err error)
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
// allowPastTargetDate=true untuk update agar goal yang sudah lewat tenggat
// tetap bisa disunting (mis. ubah nama) tanpa dipaksa memajukan tanggal.
func validateGoalFields(goal *entity.Goal, now time.Time, allowPastTargetDate bool) error {
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
	if goal.TargetDate != nil && !allowPastTargetDate && !goal.TargetDate.After(now) {
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
	if err := validateGoalFields(&goal, u.now(), false); err != nil {
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

// Batas pagination daftar: default 50, maksimal 100 per halaman.
const (
	DefaultPageLimit = 50
	MaxPageLimit     = 100
)

// normalizePagination menjepit page/limit ke rentang valid
// dan mengembalikan offset SQL.
func normalizePagination(page, limit int) (offset int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = DefaultPageLimit
	}
	if limit > MaxPageLimit {
		limit = MaxPageLimit
	}
	return (page - 1) * limit
}

// normalizeLimit mengembalikan limit yang sudah dijepit (untuk query).
func normalizeLimit(limit int) int {
	if limit < 1 {
		return DefaultPageLimit
	}
	if limit > MaxPageLimit {
		return MaxPageLimit
	}
	return limit
}

// GetGoalsByUserID mengambil satu halaman goal milik satu user + totalnya.
func (u *goalUsecase) GetGoalsByUserID(ctx context.Context, userID string, page, limit int) ([]entity.Goal, int, error) {
	if userID == "" {
		return nil, 0, validationError("user_id wajib diisi")
	}
	limit = normalizeLimit(limit)
	goals, err := u.repo.GetGoalsByUserID(ctx, userID, limit, normalizePagination(page, limit))
	if err != nil {
		return nil, 0, fmt.Errorf("usecase: GetGoalsByUserID: %w", err)
	}
	total, err := u.repo.CountGoalsByUserID(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase: GetGoalsByUserID count: %w", err)
	}
	return goals, total, nil
}

// UpdateGoal memuat entity lama (agar field imutabel seperti user_id
// dan created_at terjaga), menimpa field mutable, memvalidasi,
// lalu menyimpan via repository.
// current_amount SENGAJA tidak diambil dari input: saldo hanya berubah
// lewat kontribusi (Add/DeleteContribution) agar konsisten dengan ledger.
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
	existing.TargetDate = goal.TargetDate
	existing.ExpectedInflationRate = goal.ExpectedInflationRate
	if goal.Status != "" {
		existing.Status = goal.Status
	}

	if err := validateGoalFields(&existing, u.now(), true); err != nil {
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

// ListContributions mengembalikan satu halaman setoran satu goal beserta
// jumlah dan total nominal keseluruhan.
func (u *goalUsecase) ListContributions(ctx context.Context, goalID string, page, limit int) ([]entity.GoalContribution, int, float64, error) {
	if goalID == "" {
		return nil, 0, 0, validationError("goal_id wajib diisi")
	}
	limit = normalizeLimit(limit)
	out, err := u.repo.ListContributionsByGoalID(ctx, goalID, limit, normalizePagination(page, limit))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("usecase: ListContributions: %w", err)
	}
	count, total, err := u.repo.CountContributionsByGoalID(ctx, goalID)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("usecase: ListContributions count: %w", err)
	}
	return out, count, total, nil
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

// GetNotifications mengembalikan satu halaman notifikasi milik user + total.
func (u *goalUsecase) GetNotifications(ctx context.Context, userID string, page, limit int) ([]entity.Notification, int, error) {
	if userID == "" {
		return nil, 0, validationError("user_id wajib diisi")
	}
	limit = normalizeLimit(limit)
	out, err := u.repo.GetNotificationsByUserID(ctx, userID, limit, normalizePagination(page, limit))
	if err != nil {
		return nil, 0, fmt.Errorf("usecase: GetNotifications: %w", err)
	}
	total, err := u.repo.CountNotificationsByUserID(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase: GetNotifications count: %w", err)
	}
	if out == nil {
		out = []entity.Notification{}
	}
	return out, total, nil
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
		u.sendPushBestEffort(ctx, g.UserID, reminderTitle, message)
	}

	return created, nil
}

// sendPushBestEffort meneruskan pengingat ke semua browser terdaftar.
// Kegagalan tidak menggagalkan siklus worker; endpoint mati dibersihkan.
func (u *goalUsecase) sendPushBestEffort(ctx context.Context, userID, title, message string) {
	if u.pushSender == nil {
		return
	}
	subs, err := u.repo.ListPushSubscriptionsByUserID(ctx, userID)
	if err != nil {
		return
	}
	for _, s := range subs {
		gone, err := u.pushSender.Send(push.Subscription{
			Endpoint: s.Endpoint,
			P256dh:   s.P256dh,
			Auth:     s.Auth,
		}, title, message, "/")
		if err != nil || !gone {
			continue
		}
		_ = u.repo.DeletePushSubscription(ctx, userID, s.Endpoint)
	}
}

// levelTier adalah ambang total setoran untuk tiap level konsistensi.
type levelTier struct {
	min  int
	name string
}

// levelTiers terurut menaik; tier tertinggi yang min-nya terpenuhi menang.
var levelTiers = []levelTier{
	{min: 0, name: "Pemula"},
	{min: 10, name: "Penabung Rutin"},
	{min: 30, name: "Pejuang Konsisten"},
	{min: 100, name: "Legenda Menabung"},
}

// truncateDay membuang komponen jam agar perbandingan hari kalender tepat.
func truncateDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// GetStats menghitung streak harian (berjalan & terpanjang), hari aktif
// setahun terakhir, dan level konsistensi berdasarkan total setoran.
// Streak dianggap hidup bila setoran terakhir hari ini atau kemarin.
func (u *goalUsecase) GetStats(ctx context.Context, userID string) (entity.GoalStats, error) {
	if userID == "" {
		return entity.GoalStats{}, validationError("user_id wajib diisi")
	}
	dates, err := u.repo.ListContributionDates(ctx, userID)
	if err != nil {
		return entity.GoalStats{}, fmt.Errorf("usecase: GetStats dates: %w", err)
	}
	total, err := u.repo.CountContributionsByUserID(ctx, userID)
	if err != nil {
		return entity.GoalStats{}, fmt.Errorf("usecase: GetStats count: %w", err)
	}

	today := truncateDay(u.now())
	days := make([]time.Time, 0, len(dates))
	for _, d := range dates {
		days = append(days, truncateDay(d))
	}

	stats := entity.GoalStats{TotalContributions: total}
	day := 24 * time.Hour

	// Hari aktif dalam 365 hari terakhir.
	cutoff := today.Add(-364 * day)
	for _, d := range days {
		if !d.Before(cutoff) {
			stats.ActiveDays365++
		}
	}

	// Streak berjalan dari belakang (terbaru dulu).
	if len(days) > 0 && (days[0].Equal(today) || days[0].Equal(today.Add(-day))) {
		expected := days[0]
		for _, d := range days {
			if d.Equal(expected) {
				stats.CurrentStreakDays++
				expected = expected.Add(-day)
			} else if d.Before(expected) {
				break
			}
		}
	}

	// Streak terpanjang sepanjang masa (diurut menaik).
	if len(days) > 0 {
		asc := make([]time.Time, len(days))
		copy(asc, days)
		for i, j := 0, len(asc)-1; i < j; i, j = i+1, j-1 {
			asc[i], asc[j] = asc[j], asc[i]
		}
		run, best := 1, 1
		for i := 1; i < len(asc); i++ {
			if asc[i].Equal(asc[i-1].Add(day)) {
				run++
				if run > best {
					best = run
				}
			} else {
				run = 1
			}
		}
		stats.LongestStreakDays = best
	}

	// Level dari total setoran + progres ke level berikut.
	tier := levelTiers[0]
	var next *levelTier
	for i := range levelTiers {
		if total >= levelTiers[i].min {
			tier = levelTiers[i]
			if i+1 < len(levelTiers) {
				next = &levelTiers[i+1]
			} else {
				next = nil
			}
		}
	}
	stats.Level = tier.name
	if next == nil {
		stats.LevelProgress = 100
	} else {
		span := float64(next.min - tier.min)
		stats.LevelProgress = float64(total-tier.min) / span * 100
		n := next.min
		stats.NextLevelAt = &n
	}

	return stats, nil
}

// SetPushSender memasang pengirim Web Push (nil menonaktifkannya).
func (u *goalUsecase) SetPushSender(sender PushSender) {
	u.pushSender = sender
}

// PushPublicKey mengembalikan kunci publik VAPID untuk klien browser.
func (u *goalUsecase) PushPublicKey() (string, error) {
	if u.pushSender == nil || u.pushSender.PublicKey() == "" {
		return "", validationError("web push belum dikonfigurasi di server")
	}
	return u.pushSender.PublicKey(), nil
}

// SavePushSubscription memvalidasi lalu menyimpan langganan push milik user.
func (u *goalUsecase) SavePushSubscription(ctx context.Context, userID, endpoint, p256dh, auth string) (entity.PushSubscription, error) {
	if userID == "" {
		return entity.PushSubscription{}, validationError("user_id wajib diisi")
	}
	if endpoint == "" || p256dh == "" || auth == "" {
		return entity.PushSubscription{}, validationError("endpoint, p256dh, dan auth wajib diisi")
	}
	if len(endpoint) > 2048 || len(p256dh) > 255 || len(auth) > 255 {
		return entity.PushSubscription{}, validationError("data langganan push terlalu panjang")
	}
	sub, err := u.repo.SavePushSubscription(ctx, entity.PushSubscription{
		UserID:   userID,
		Endpoint: endpoint,
		P256dh:   p256dh,
		Auth:     auth,
	})
	if err != nil {
		return entity.PushSubscription{}, fmt.Errorf("usecase: SavePushSubscription: %w", err)
	}
	return sub, nil
}

// DeletePushSubscription menghapus langganan push milik user (idempotent).
func (u *goalUsecase) DeletePushSubscription(ctx context.Context, userID, endpoint string) error {
	if userID == "" {
		return validationError("user_id wajib diisi")
	}
	if endpoint == "" {
		return validationError("endpoint wajib diisi")
	}
	if err := u.repo.DeletePushSubscription(ctx, userID, endpoint); err != nil {
		return fmt.Errorf("usecase: DeletePushSubscription: %w", err)
	}
	return nil
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
