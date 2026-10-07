package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"artagoal/goal-service/internal/domain/entity"
	"artagoal/goal-service/internal/repository"
)

// GoalPostgresRepository adalah implementasi repository.GoalRepository
// memakai database/sql + driver github.com/lib/pq.
type GoalPostgresRepository struct {
	db *sql.DB
}

// Pastikan kontrak terpenuhi saat compile.
var _ repository.GoalRepository = (*GoalPostgresRepository)(nil)

// NewGoalPostgresRepository membuat instance baru.
func NewGoalPostgresRepository(db *sql.DB) *GoalPostgresRepository {
	return &GoalPostgresRepository{db: db}
}

// scanner adalah abstraksi atas *sql.Row dan *sql.Rows agar logika
// scan terpusat di satu tempat.
type scanner interface {
	Scan(dest ...any) error
}

// scanGoal memetakan satu baris goals ke entity.Goal,
// menangani kolom NULL (target_date) secara aman.
func scanGoal(s scanner) (entity.Goal, error) {
	var g entity.Goal
	var targetDate sql.NullTime

	if err := s.Scan(
		&g.ID,
		&g.UserID,
		&g.Title,
		&g.Category,
		&g.TargetAmount,
		&g.CurrentAmount,
		&targetDate,
		&g.ExpectedInflationRate,
		&g.Status,
		&g.CreatedAt,
		&g.UpdatedAt,
	); err != nil {
		return entity.Goal{}, err
	}

	if targetDate.Valid {
		t := targetDate.Time
		g.TargetDate = &t
	}

	return g, nil
}

const goalColumns = `id, user_id, title, category, target_amount, current_amount,
	target_date, expected_inflation_rate, status, created_at, updated_at`

// CreateGoal menyimpan goal baru. Kolom id diserahkan ke default
// database (gen_random_uuid()), lalu dikembalikan via RETURNING.
func (r *GoalPostgresRepository) CreateGoal(ctx context.Context, goal entity.Goal) (entity.Goal, error) {
	const query = `
		INSERT INTO goals
			(user_id, title, category, target_amount, current_amount,
			 target_date, expected_inflation_rate, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`

	var targetDate any
	if goal.TargetDate != nil {
		targetDate = *goal.TargetDate
	}

	if err := r.db.QueryRowContext(ctx, query,
		goal.UserID,
		goal.Title,
		goal.Category,
		goal.TargetAmount,
		goal.CurrentAmount,
		targetDate,
		goal.ExpectedInflationRate,
		goal.Status,
	).Scan(&goal.ID, &goal.CreatedAt, &goal.UpdatedAt); err != nil {
		return entity.Goal{}, fmt.Errorf("postgres: CreateGoal: %w", err)
	}

	return goal, nil
}

// GetGoalByID mengambil satu goal berdasarkan id.
func (r *GoalPostgresRepository) GetGoalByID(ctx context.Context, id string) (entity.Goal, error) {
	query := `SELECT ` + goalColumns + ` FROM goals WHERE id = $1`

	goal, err := scanGoal(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.Goal{}, fmt.Errorf("postgres: GetGoalByID %s: %w", id, entity.ErrGoalNotFound)
		}
		return entity.Goal{}, fmt.Errorf("postgres: GetGoalByID %s: %w", id, err)
	}

	return goal, nil
}

// GetGoalsByUserID mengambil semua goal milik satu user,
// diurutkan dari yang terbaru.
func (r *GoalPostgresRepository) GetGoalsByUserID(ctx context.Context, userID string) ([]entity.Goal, error) {
	query := `SELECT ` + goalColumns + ` FROM goals WHERE user_id = $1 ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("postgres: GetGoalsByUserID %s: %w", userID, err)
	}
	defer rows.Close()

	goals := make([]entity.Goal, 0)
	for rows.Next() {
		goal, err := scanGoal(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres: GetGoalsByUserID %s scan: %w", userID, err)
		}
		goals = append(goals, goal)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: GetGoalsByUserID %s rows: %w", userID, err)
	}

	return goals, nil
}

// UpdateGoal memperbarui field mutable sebuah goal dan
// mengembalikan entity terbaru (updated_at dari database).
func (r *GoalPostgresRepository) UpdateGoal(ctx context.Context, goal entity.Goal) (entity.Goal, error) {
	const query = `
		UPDATE goals SET
			title = $1,
			category = $2,
			target_amount = $3,
			current_amount = $4,
			target_date = $5,
			expected_inflation_rate = $6,
			status = $7,
			updated_at = NOW()
		WHERE id = $8
		RETURNING updated_at`

	var targetDate any
	if goal.TargetDate != nil {
		targetDate = *goal.TargetDate
	}

	if err := r.db.QueryRowContext(ctx, query,
		goal.Title,
		goal.Category,
		goal.TargetAmount,
		goal.CurrentAmount,
		targetDate,
		goal.ExpectedInflationRate,
		goal.Status,
		goal.ID,
	).Scan(&goal.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.Goal{}, fmt.Errorf("postgres: UpdateGoal %s: %w", goal.ID, entity.ErrGoalNotFound)
		}
		return entity.Goal{}, fmt.Errorf("postgres: UpdateGoal %s: %w", goal.ID, err)
	}

	return goal, nil
}

// DeleteGoal menghapus goal. Mengembalikan ErrGoalNotFound
// jika tidak ada baris yang terhapus.
func (r *GoalPostgresRepository) DeleteGoal(ctx context.Context, id string) error {
	const query = `DELETE FROM goals WHERE id = $1`

	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("postgres: DeleteGoal %s: %w", id, err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres: DeleteGoal %s rows affected: %w", id, err)
	}
	if affected == 0 {
		return fmt.Errorf("postgres: DeleteGoal %s: %w", id, entity.ErrGoalNotFound)
	}

	return nil
}

// scanContribution memetakan satu baris goal_contributions ke entity.
func scanContribution(s scanner) (entity.GoalContribution, error) {
	var c entity.GoalContribution
	var note sql.NullString
	if err := s.Scan(&c.ID, &c.GoalID, &c.Amount, &note, &c.CreatedAt); err != nil {
		return entity.GoalContribution{}, err
	}
	if note.Valid {
		n := note.String
		c.Note = &n
	}
	return c, nil
}

// AddContribution menyisipkan kontribusi + menaikkan current_amount atomik.
// Jika saldo baru >= target_amount, status otomatis menjadi achieved.
func (r *GoalPostgresRepository) AddContribution(ctx context.Context, goalID string, amount float64, note *string) (entity.GoalContribution, entity.Goal, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return entity.GoalContribution{}, entity.Goal{}, fmt.Errorf("postgres: AddContribution begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var noteParam any
	if note != nil {
		noteParam = *note
	}

	var c entity.GoalContribution
	var noteOut sql.NullString
	const insertQ = `
		INSERT INTO goal_contributions (goal_id, amount, note)
		VALUES ($1, $2, $3)
		RETURNING id, goal_id, amount, note, created_at`
	if err := tx.QueryRowContext(ctx, insertQ, goalID, amount, noteParam).
		Scan(&c.ID, &c.GoalID, &c.Amount, &noteOut, &c.CreatedAt); err != nil {
		return entity.GoalContribution{}, entity.Goal{}, fmt.Errorf("postgres: AddContribution insert: %w", err)
	}
	if noteOut.Valid {
		n := noteOut.String
		c.Note = &n
	}

	// Kunci baris goal agar increment konsisten di bawah konkurensi,
	// sekaligus pastikan goal ada sebelum update.
	var g entity.Goal
	const lockQ = `SELECT ` + goalColumns + ` FROM goals WHERE id = $1 FOR UPDATE`
	g, err = scanGoal(tx.QueryRowContext(ctx, lockQ, goalID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.GoalContribution{}, entity.Goal{}, fmt.Errorf("postgres: AddContribution goal %s: %w", goalID, entity.ErrGoalNotFound)
		}
		return entity.GoalContribution{}, entity.Goal{}, fmt.Errorf("postgres: AddContribution lock goal %s: %w", goalID, err)
	}

	const updateQ = `
		UPDATE goals SET
			current_amount = current_amount + $1,
			status = CASE WHEN current_amount + $1 >= target_amount THEN 'achieved' ELSE status END,
			updated_at = NOW()
		WHERE id = $2
		RETURNING current_amount, status, updated_at`
	if err := tx.QueryRowContext(ctx, updateQ, amount, goalID).
		Scan(&g.CurrentAmount, &g.Status, &g.UpdatedAt); err != nil {
		return entity.GoalContribution{}, entity.Goal{}, fmt.Errorf("postgres: AddContribution bump goal %s: %w", goalID, err)
	}

	if err := tx.Commit(); err != nil {
		return entity.GoalContribution{}, entity.Goal{}, fmt.Errorf("postgres: AddContribution commit: %w", err)
	}
	return c, g, nil
}

// ListContributionsByGoalID mengembalikan kontribusi dari yang terbaru.
func (r *GoalPostgresRepository) ListContributionsByGoalID(ctx context.Context, goalID string) ([]entity.GoalContribution, error) {
	const query = `SELECT id, goal_id, amount, note, created_at
		FROM goal_contributions WHERE goal_id = $1 ORDER BY created_at DESC`

	// Pastikan goal ada agar 404 konsisten (bukan list kosong untuk id ngawur).
	var exists bool
	if err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM goals WHERE id = $1)`, goalID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("postgres: ListContributions exists check %s: %w", goalID, err)
	}
	if !exists {
		return nil, fmt.Errorf("postgres: ListContributions goal %s: %w", goalID, entity.ErrGoalNotFound)
	}

	rows, err := r.db.QueryContext(ctx, query, goalID)
	if err != nil {
		return nil, fmt.Errorf("postgres: ListContributions %s: %w", goalID, err)
	}
	defer rows.Close()

	out := make([]entity.GoalContribution, 0)
	for rows.Next() {
		c, err := scanContribution(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres: ListContributions %s scan: %w", goalID, err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: ListContributions %s rows: %w", goalID, err)
	}
	return out, nil
}

// GetContributionByID mengambil satu kontribusi.
func (r *GoalPostgresRepository) GetContributionByID(ctx context.Context, id string) (entity.GoalContribution, error) {
	const query = `SELECT id, goal_id, amount, note, created_at FROM goal_contributions WHERE id = $1`
	c, err := scanContribution(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.GoalContribution{}, fmt.Errorf("postgres: GetContribution %s: %w", id, entity.ErrContributionNotFound)
		}
		return entity.GoalContribution{}, fmt.Errorf("postgres: GetContribution %s: %w", id, err)
	}
	return c, nil
}

// DeleteContribution menghapus kontribusi dan menurunkan current_amount (floor 0).
// Jika goal sebelumnya achieved dan saldo turun di bawah target, status kembali active.
func (r *GoalPostgresRepository) DeleteContribution(ctx context.Context, contributionID string) (entity.Goal, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return entity.Goal{}, fmt.Errorf("postgres: DeleteContribution begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var goalID string
	var amount float64
	if err := tx.QueryRowContext(ctx,
		`SELECT goal_id, amount FROM goal_contributions WHERE id = $1 FOR UPDATE`, contributionID).
		Scan(&goalID, &amount); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.Goal{}, fmt.Errorf("postgres: DeleteContribution %s: %w", contributionID, entity.ErrContributionNotFound)
		}
		return entity.Goal{}, fmt.Errorf("postgres: DeleteContribution load %s: %w", contributionID, err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM goal_contributions WHERE id = $1`, contributionID); err != nil {
		return entity.Goal{}, fmt.Errorf("postgres: DeleteContribution delete %s: %w", contributionID, err)
	}

	var g entity.Goal
	const updateQ = `
		UPDATE goals SET
			current_amount = GREATEST(0, current_amount - $1),
			status = CASE
				WHEN status = 'achieved' AND GREATEST(0, current_amount - $1) < target_amount THEN 'active'
				ELSE status END,
			updated_at = NOW()
		WHERE id = $2
		RETURNING ` + goalColumns
	row := tx.QueryRowContext(ctx, updateQ, amount, goalID)
	// RETURNING memakai urutan goalColumns -> scanGoal.
	g, err = scanGoal(row)
	if err != nil {
		return entity.Goal{}, fmt.Errorf("postgres: DeleteContribution bump goal %s: %w", goalID, err)
	}

	if err := tx.Commit(); err != nil {
		return entity.Goal{}, fmt.Errorf("postgres: DeleteContribution commit: %w", err)
	}
	return g, nil
}

// GetContributionHeatmap mengambil agregasi jumlah dan total setoran
// per hari dalam 1 tahun terakhir untuk satu user.
func (r *GoalPostgresRepository) GetContributionHeatmap(ctx context.Context, userID string) ([]entity.HeatmapData, error) {
	const query = `
		SELECT c.created_at::date as date, COUNT(c.id) as count, COALESCE(SUM(c.amount), 0) as total_amount
		FROM goal_contributions c
		JOIN goals g ON c.goal_id = g.id
		WHERE g.user_id = $1 AND c.created_at >= NOW() - INTERVAL '365 days'
		GROUP BY c.created_at::date
		ORDER BY date ASC;`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("postgres: GetContributionHeatmap %s: %w", userID, err)
	}
	defer rows.Close()

	out := make([]entity.HeatmapData, 0)
	for rows.Next() {
		var day time.Time
		var h entity.HeatmapData
		var total sql.NullFloat64
		if err := rows.Scan(&day, &h.Count, &total); err != nil {
			return nil, fmt.Errorf("postgres: GetContributionHeatmap %s scan: %w", userID, err)
		}
		h.Date = day.Format("2006-01-02")
		if total.Valid {
			h.TotalAmount = total.Float64
		}
		out = append(out, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: GetContributionHeatmap %s rows: %w", userID, err)
	}

	return out, nil
}

// ---- notifications ----

const notificationColumns = `id, user_id, goal_id, title, message, is_read, created_at`

// scanNotification memetakan satu baris notifications ke entity,
// menangani goal_id yang NULL secara aman.
func scanNotification(s scanner) (entity.Notification, error) {
	var n entity.Notification
	var goalID sql.NullString
	if err := s.Scan(
		&n.ID,
		&n.UserID,
		&goalID,
		&n.Title,
		&n.Message,
		&n.IsRead,
		&n.CreatedAt,
	); err != nil {
		return entity.Notification{}, err
	}
	if goalID.Valid {
		g := goalID.String
		n.GoalID = &g
	}
	return n, nil
}

// ListActiveGoalsDueWithin mengembalikan goal active yang deadline-nya
// (target_date) berada di antara sekarang dan days hari ke depan.
func (r *GoalPostgresRepository) ListActiveGoalsDueWithin(ctx context.Context, days int) ([]entity.Goal, error) {
	query := `SELECT ` + goalColumns + ` FROM goals
		WHERE status = 'active'
		  AND target_date IS NOT NULL
		  AND target_date >= NOW()
		  AND target_date <= NOW() + ($1 * INTERVAL '1 day')
		ORDER BY target_date ASC`

	rows, err := r.db.QueryContext(ctx, query, days)
	if err != nil {
		return nil, fmt.Errorf("postgres: ListActiveGoalsDueWithin: %w", err)
	}
	defer rows.Close()

	goals := make([]entity.Goal, 0)
	for rows.Next() {
		g, err := scanGoal(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres: ListActiveGoalsDueWithin scan: %w", err)
		}
		goals = append(goals, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: ListActiveGoalsDueWithin rows: %w", err)
	}

	return goals, nil
}

// CreateNotification menyimpan satu notifikasi in-app.
func (r *GoalPostgresRepository) CreateNotification(ctx context.Context, n entity.Notification) (entity.Notification, error) {
	const query = `
		INSERT INTO notifications (user_id, goal_id, title, message)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`

	var goalID any
	if n.GoalID != nil {
		goalID = *n.GoalID
	}

	if err := r.db.QueryRowContext(ctx, query,
		n.UserID,
		goalID,
		n.Title,
		n.Message,
	).Scan(&n.ID, &n.CreatedAt); err != nil {
		return entity.Notification{}, fmt.Errorf("postgres: CreateNotification: %w", err)
	}

	return n, nil
}

// HasRecentNotification melaporkan apakah sudah ada notifikasi berjudul
// title untuk goal tersebut dalam days hari terakhir.
func (r *GoalPostgresRepository) HasRecentNotification(ctx context.Context, goalID, title string, days int) (bool, error) {
	const query = `SELECT EXISTS(
		SELECT 1 FROM notifications
		WHERE goal_id = $1 AND title = $2
		  AND created_at >= NOW() - ($3 * INTERVAL '1 day')
	)`

	var exists bool
	if err := r.db.QueryRowContext(ctx, query, goalID, title, days).Scan(&exists); err != nil {
		return false, fmt.Errorf("postgres: HasRecentNotification %s: %w", goalID, err)
	}

	return exists, nil
}

// GetNotificationsByUserID mengembalikan notifikasi milik user (terbaru dulu).
func (r *GoalPostgresRepository) GetNotificationsByUserID(ctx context.Context, userID string) ([]entity.Notification, error) {
	query := `SELECT ` + notificationColumns + ` FROM notifications WHERE user_id = $1 ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("postgres: GetNotificationsByUserID %s: %w", userID, err)
	}
	defer rows.Close()

	out := make([]entity.Notification, 0)
	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres: GetNotificationsByUserID %s scan: %w", userID, err)
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: GetNotificationsByUserID %s rows: %w", userID, err)
	}

	return out, nil
}

// MarkNotificationAsRead menandai notifikasi milik user sebagai dibaca.
// Klausa user_id mencegah user menandai notifikasi milik user lain;
// 0 baris terpengaruh dipetakan ke ErrNotificationNotFound.
func (r *GoalPostgresRepository) MarkNotificationAsRead(ctx context.Context, id, userID string) (entity.Notification, error) {
	const query = `UPDATE notifications SET is_read = TRUE
		WHERE id = $1 AND user_id = $2
		RETURNING ` + notificationColumns

	n, err := scanNotification(r.db.QueryRowContext(ctx, query, id, userID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.Notification{}, fmt.Errorf("postgres: MarkNotificationAsRead %s: %w", id, entity.ErrNotificationNotFound)
		}
		return entity.Notification{}, fmt.Errorf("postgres: MarkNotificationAsRead %s: %w", id, err)
	}

	return n, nil
}

