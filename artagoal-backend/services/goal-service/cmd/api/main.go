package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"

	deliveryhttp "artagoal/goal-service/internal/delivery/http"
	authmw "artagoal/goal-service/internal/delivery/http/middleware"
	"artagoal/goal-service/internal/push"
	"artagoal/goal-service/internal/repository/postgres"
	"artagoal/goal-service/internal/usecase"
	"artagoal/goal-service/internal/worker"
	"artagoal/goal-service/migrations"
	"artagoal/goal-service/pkg/database"
)

func main() {
	// Memuat variabel dari berkas .env
	if err := godotenv.Load(); err != nil {
		log.Println("Peringatan: Berkas .env tidak ditemukan, menggunakan environment OS")
	}

	// Koneksi ke PostgreSQL (Supabase)
	dsn := os.Getenv("DATABASE_URL")
	db, err := database.NewPostgresConnection(dsn)
	if err != nil {
		log.Fatalf("Gagal terhubung ke Database Supabase: %v", err)
	}
	defer db.Close()

	// Migrasi skema otomatis (idempotent, dilacak di schema_migrations).
	if err := database.RunMigrations(db, "goal-service", migrations.FS, log.Default()); err != nil {
		log.Fatalf("Gagal menjalankan migrasi: %v", err)
	}

	// Wiring Clean Architecture: repository -> usecase -> handler
	repo := postgres.NewGoalPostgresRepository(db)
	uc := usecase.NewGoalUsecase(repo)
	goalHandler := deliveryhttp.NewGoalHandler(uc)

	// Web Push (VAPID): nonaktif bila kunci belum dikonfigurasi.
	contact := os.Getenv("VAPID_CONTACT")
	if contact == "" {
		contact = "mailto:admin@artagoal.local"
	}
	pushSender := push.NewSender(
		os.Getenv("VAPID_PUBLIC_KEY"),
		os.Getenv("VAPID_PRIVATE_KEY"),
		contact,
	)
	if pushSender.Enabled() {
		uc.SetPushSender(pushSender)
	} else {
		log.Println("Peringatan: VAPID_PUBLIC_KEY/VAPID_PRIVATE_KEY kosong — Web Push nonaktif")
	}

	// Background worker: cek pengingat tenggat segera saat boot,
	// lalu berulang setiap 24 jam.
	reminder := worker.NewReminderWorker(uc, 24*time.Hour, log.Default())
	go reminder.Start(context.Background())

	r := chi.NewRouter()
	r.Use(chimiddleware.Logger, chimiddleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://arta-goal.vercel.app", "http://localhost:5173", "http://127.0.0.1:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "bypass-tunnel-reminder"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Publik: healthcheck tanpa token.
	r.Get("/healthcheck", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "OK",
			"service": "goal-service",
		})
	})

	// Terproteksi: semua endpoint goal wajib JWT valid.
	// user_id diambil dari klaim token ke request context oleh middleware.
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Println("Peringatan: JWT_SECRET kosong — semua request terproteksi akan ditolak (401)")
	} else if len(jwtSecret) < 32 {
		log.Println("Peringatan: JWT_SECRET kurang dari 32 karakter — gunakan nilai acak yang lebih panjang")
	}
	r.Group(func(r chi.Router) {
		r.Use(authmw.AuthMiddleware(jwtSecret))
		goalHandler.RegisterGoalRoutes(r)
	})

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("goal-service listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server berhenti: %v", err)
	}
}
