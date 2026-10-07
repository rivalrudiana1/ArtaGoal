package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"

	deliveryhttp "artagoal/auth-service/internal/delivery/http"
	authmw "artagoal/auth-service/internal/delivery/http/middleware"
	"artagoal/auth-service/internal/repository/postgres"
	"artagoal/auth-service/internal/usecase"
	"artagoal/auth-service/migrations"
	"artagoal/auth-service/pkg/database"
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
		log.Fatalf("Gagal terhubung ke Database: %v", err)
	}
	defer db.Close()

	// Migrasi skema otomatis (idempotent, dilacak di schema_migrations).
	if err := database.RunMigrations(db, migrations.FS, log.Default()); err != nil {
		log.Fatalf("Gagal menjalankan migrasi: %v", err)
	}

	// Secret JWT wajib sama dengan yang dipakai goal-service
	// agar token hasil login valid di semua service.
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Println("Peringatan: JWT_SECRET kosong — penerbitan token akan gagal")
	}

	// Wiring Clean Architecture: repository -> usecase -> handler
	repo := postgres.NewUserPostgresRepository(db)
	uc := usecase.NewAuthUsecase(repo, jwtSecret)
	authHandler := deliveryhttp.NewAuthHandler(uc)

	r := chi.NewRouter()
	r.Use(chimiddleware.Logger, chimiddleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Publik: healthcheck tanpa token.
	r.Get("/healthcheck", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "OK",
			"service": "auth-service",
		})
	})

	// Publik: file avatar yang diunggah user agar bisa diakses frontend.
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	// Register & login publik; /me dilindungi middleware JWT di dalam handler.
	authHandler.RegisterAuthRoutes(r, authmw.AuthMiddleware(jwtSecret))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	addr := ":" + port
	log.Printf("auth-service listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server berhenti: %v", err)
	}
}

