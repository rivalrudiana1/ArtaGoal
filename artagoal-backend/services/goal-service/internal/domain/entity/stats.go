package entity

// GoalStats adalah ringkasan gamifikasi menabung milik satu user:
// streak harian, total setoran, dan level konsistensi.
type GoalStats struct {
	TotalContributions int     `json:"total_contributions"`
	ActiveDays365      int     `json:"active_days_365"`
	CurrentStreakDays  int     `json:"current_streak_days"`
	LongestStreakDays  int     `json:"longest_streak_days"`
	Level              string  `json:"level"`
	LevelProgress      float64 `json:"level_progress"`
	NextLevelAt        *int    `json:"next_level_at,omitempty"`
}
