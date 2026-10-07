// Package worker berisi background worker goal-service.
// ReminderWorker memindai goal bertenggat dekat secara berkala
// dan membuat notifikasi peringatan in-app via usecase.
package worker

import (
	"context"
	"log"
	"time"

	"artagoal/goal-service/internal/usecase"
)

// ReminderWorker menjalankan pengecekan pengingat tenggat setiap interval.
type ReminderWorker struct {
	uc       usecase.GoalUsecase
	interval time.Duration
	logger   *log.Logger
}

// NewReminderWorker membuat worker baru dengan interval pengecekan.
func NewReminderWorker(uc usecase.GoalUsecase, interval time.Duration, logger *log.Logger) *ReminderWorker {
	if logger == nil {
		logger = log.Default()
	}
	return &ReminderWorker{uc: uc, interval: interval, logger: logger}
}

// Start menjalankan satu pengecekan segera saat boot, lalu berulang
// setiap interval memakai time.Ticker hingga ctx dibatalkan.
func (w *ReminderWorker) Start(ctx context.Context) {
	w.runOnce(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Println("worker: reminder dihentikan")
			return
		case <-ticker.C:
			w.runOnce(ctx)
		}
	}
}

// runOnce menjalankan satu siklus pengecekan; kegagalan hanya dicatat
// agar tidak menghentikan siklus berikutnya.
func (w *ReminderWorker) runOnce(ctx context.Context) {
	created, err := w.uc.RunDeadlineReminderCheck(ctx)
	if err != nil {
		w.logger.Printf("worker: reminder gagal: %v", err)
		return
	}
	if created > 0 {
		w.logger.Printf("worker: reminder membuat %d notifikasi", created)
	}
}
