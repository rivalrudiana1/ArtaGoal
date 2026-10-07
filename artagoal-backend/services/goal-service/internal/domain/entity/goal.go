package entity

import (
	"errors"
	"time"
)

// Status goal yang dikenali aplikasi.
const (
	GoalStatusActive    = "active"
	GoalStatusAchieved  = "achieved"
	GoalStatusCancelled = "cancelled"
)

// Sentinel errors domain.
var (
	ErrGoalNotFound         = errors.New("goal not found")
	ErrContributionNotFound = errors.New("goal contribution not found")
)

// Goal memetakan tabel goals di PostgreSQL (Supabase).
type Goal struct {
	ID                    string     `json:"id"`
	UserID                string     `json:"user_id"`
	Title                 string     `json:"title"`
	Category              string     `json:"category"`
	TargetAmount          float64    `json:"target_amount"`
	CurrentAmount         float64    `json:"current_amount"`
	TargetDate            *time.Time `json:"target_date,omitempty"`
	ExpectedInflationRate float64    `json:"expected_inflation_rate"`
	Status                string     `json:"status"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

// GoalContribution memetakan tabel goal_contributions di PostgreSQL (Supabase).
type GoalContribution struct {
	ID        string    `json:"id"`
	GoalID    string    `json:"goal_id"`
	Amount    float64   `json:"amount"`
	Note      *string   `json:"note,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// HeatmapData adalah agregasi kontribusi per hari untuk heatmap 365 hari.
// Date berformat YYYY-MM-DD.
type HeatmapData struct {
	Date        string  `json:"date"`
	Count       int     `json:"count"`
	TotalAmount float64 `json:"total_amount"`
}
