# 🎯 ArtaGoal — Smart Financial Goal Tracker & Inflation Projection

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![React](https://img.shields.io/badge/React-19-61DAFB?style=for-the-badge&logo=react&logoColor=black)
![Vite](https://img.shields.io/badge/Vite-8-646CFF?style=for-the-badge&logo=vite&logoColor=white)
![TailwindCSS](https://img.shields.io/badge/Tailwind-4-06B6D4?style=for-the-badge&logo=tailwindcss&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?style=for-the-badge&logo=docker&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-4169E1?style=for-the-badge&logo=postgresql&logoColor=white)

**ArtaGoal** adalah aplikasi *full-stack* manajemen target keuangan berbasis *microservices* yang dilengkapi kalkulasi proyeksi masa depan yang disesuaikan terhadap laju inflasi (*Inflation-Adjusted Target Projection*). Backend dibangun dengan prinsip *Clean Architecture* di Go, frontend memakai React 19 yang modern.

---

## 🌟 Fitur Utama

- 🔐 **Authentication & Security** — autentikasi berbasis JWT, *password hashing* dengan `bcrypt`, dan otorisasi anti-IDOR di tingkat layanan.
- 📈 **Kalkulasi Proyeksi Inflasi Real-time** — menghitung *Future Value* (`FV = PV × (1 + i/100)^tahun`) dan kebutuhan setoran bulanan otomatis, plus slider simulasi interaktif di halaman Target Baru.
- 📊 **Visualisasi Grafik (Recharts)** — grafik gabungan Area + Line yang membandingkan dana terkumpul vs nilai target terinflasi per target.
- 💳 **Manajemen & Riwayat Setoran** — pencatatan kontribusi tabungan berkala, modal riwayat transaksi, dan pembatalan setoran (saldo diperbarui atomik via transaksi DB).
- ✏️ **Ubah Target** — sunting judul, kategori, nominal, tenggat, inflasi, dan status langsung dari kartu target (saldo terkumpul read-only, hanya berubah lewat setoran).
- 🟩 **Heatmap Kontribusi ala GitHub** — agregasi setoran 365 hari (`GET /goals/heatmap`), level warna 0–4, tooltip nominal, me-refresh otomatis setelah setor/ubah/hapus.
- 🔔 **Pengingat Tenggat & Notifikasi In-App** — background worker (cek saat boot + tiap 24 jam) membuat peringatan untuk target ≤30 hari dengan progres <80% (anti-duplikat 7 hari); lonceng + badge + dropdown "Tandai Dibaca" di header.
- 👤 **Profil & Avatar** — ubah nama + upload avatar (JPG/PNG ≤2MB) via `PUT /auth/profile`, disajikan sebagai file statis `/uploads/*`.
- 📲 **PWA** — manifest + service worker (auto-update) + ikon 192/512, bisa diinstal sebagai aplikasi.
- 🔍 **Pencarian, Filter & Sorting** — cari berdasar judul, filter kategori & status (`ACTIVE` / `ACHIEVED`), urutkan berdasar tenggat, progres, nominal, atau terbaru.
- 🖨️ **Export Laporan (.csv)** — unduh ringkasan portofolio target (PV, FV, terkumpul, sisa, progres, status, tenggat) ke format CSV.
- 🐳 **Full Containerization** — backend + Postgres + Redis siap jalan dalam satu perintah via Docker Compose (*multi-stage Go builds*).

---

## 🏗️ Arsitektur Sistem & Tech Stack

Monorepo yang memisahkan *backend microservices* dan *frontend SPA*:

```text
ArtaGoal/
├── README.md
├── apitest.http                  # HTTP Client Test Suite (VS Code REST Client)
├── artagoal-backend/
│   ├── docker-compose.yml        # postgres:15, redis:7, auth-service, goal-service
│   ├── .env.example              # Contoh JWT_SECRET untuk compose (salin jadi .env)
│   └── services/
│       ├── auth-service/         # Port :8081 — register/login/me/profile
│       │   ├── cmd/api/main.go
│       │   ├── internal/{delivery,usecase,repository,domain}
│       │   ├── migrations/{001_init.sql,002_add_avatar_url.sql,embed.go}
│       │   └── pkg/database/     # koneksi + auto-migrate
│       └── goal-service/         # Port :8080 — goals, contributions, heatmap, notifications
│           ├── cmd/api/main.go
│           ├── internal/{delivery,usecase,repository,domain,worker}
│           ├── migrations/{001_init.sql,002_create_notifications.sql,embed.go}
│           └── pkg/database/     # koneksi + auto-migrate
└── artagoal-frontend/            # Port :5173 (Vite dev) — React SPA + PWA
    ├── vite.config.js            # plugin VitePWA + manifest
    ├── public/pwa-{192,512}.png  # ikon PWA (dibangkitkan via scripts/generate-pwa-icons.mjs)
    └── src/
        ├── pages/                # Login, Register, Dashboard, NewGoal, Profile
        ├── components/           # GoalCard, GoalProjectionChart, ContributionHistoryModal,
        │                         # ContributionHeatmap, NotificationBell, AuthLayout
        ├── context/AuthContext.jsx
        ├── services/api.js       # axios clients + goalService/authService
        └── utils/{format.js,exportCsv.js}
```

### Backend Stack

| Aspek | Teknologi |
|---|---|
| Language | Go 1.25 |
| Router & Middleware | `go-chi/chi/v5`, `go-chi/cors` |
| Database | PostgreSQL 15 & Redis 7 |
| Authentication | `golang-jwt/jwt/v5`, `golang.org/x/crypto` (bcrypt) |
| Arsitektur | Clean Architecture (Handler / Usecase / Repository) |

### Frontend Stack

| Aspek | Teknologi |
|---|---|
| Framework | React 19 + Vite 8 |
| Styling | Tailwind CSS v4 |
| Routing | react-router-dom v7 |
| HTTP Client | Axios |
| Charts | Recharts (ComposedChart: Area + Line) |
| Icons | lucide-react |
| Lint | oxlint |

---

## 🚀 Panduan Memulai (Quick Start)

### Prasyarat

- Docker Desktop & Docker Compose
- Node.js (v20+)
- Go (v1.25+) — opsional, hanya jika menjalankan backend native tanpa Docker

### Opsi 1: Backend via Docker Compose (Rekomendasi)

1. **Siapkan secret bersama (sekali saja):**

   ```powershell
   cd artagoal-backend
   Copy-Item .env.example .env
   # Isi JWT_SECRET di .env dengan nilai acak (mis. hasil `openssl rand -base64 48`).
   # File .env TIDAK masuk git — jangan pernah commit kredensial asli.
   ```

2. **Jalankan stack backend (migrasi DB berjalan otomatis saat boot):**

   ```powershell
   docker compose up -d --build
   ```

   Masing-masing service menerapkan `migrations/*.sql` yang belum diterapkan
   (dilacak di tabel `schema_migrations`) setiap kali dinyalakan — tidak perlu
   `psql -f` manual lagi. File upload avatar dipersist di volume `auth_uploads`.

3. **Jalankan frontend React:**

   ```powershell
   cd ../artagoal-frontend
   npm install
   npm run dev
   ```

4. Akses aplikasi di browser pada **http://localhost:5173**.

### Opsi 2: Backend Native (Go Run)

Salin contoh env tiap service lalu isi nilai asli (cukup untuk dev lokal):

```powershell
Copy-Item services/auth-service/.env.example services/auth-service/.env
Copy-Item services/goal-service/.env.example services/goal-service/.env
```

Pastikan Postgres lokal berjalan, lalu:

```powershell
# Terminal 1 — Auth Service (:8081)
cd artagoal-backend/services/auth-service
go run ./cmd/api/main.go

# Terminal 2 — Goal Service (:8080)
cd ../goal-service
go run ./cmd/api/main.go
```

### Konfigurasi Environment Frontend

Salin `.env.example` menjadi `.env` untuk override lokal:

```powershell
cd artagoal-frontend
Copy-Item .env.example .env
```

| Variabel | Default | Keterangan |
|---|---|---|
| `VITE_AUTH_API_URL` | `http://localhost:8081/api/v1` | Base URL auth-service |
| `VITE_GOAL_API_URL` | `http://localhost:8080/api/v1` | Base URL goal-service |

### Perintah Frontend

| Perintah | Kegunaan |
|---|---|
| `npm run dev` | Menjalankan Vite dev server |
| `npm run lint` | Cek lint dengan oxlint |
| `npm run build` | Build produksi ke `dist/` |
| `npm run preview` | Pratinjau hasil build |

---

## 📡 Ringkasan API Endpoints

### Auth Service (`http://localhost:8081`)

| Method | Endpoint | Deskripsi | Otorisasi |
|---|---|---|---|
| `GET` | `/healthcheck` | Cek status layanan | Publik |
| `POST` | `/api/v1/auth/register` | Registrasi akun baru | Publik |
| `POST` | `/api/v1/auth/login` | Login & penerbitan token JWT | Publik |
| `GET` | `/api/v1/auth/me` | Profil pengguna aktif | Bearer Token |
| `PUT` | `/api/v1/auth/profile` | Ubah nama + upload avatar (multipart, JPG/PNG ≤2MB) | Bearer Token |
| `GET` | `/uploads/avatars/{file}` | File avatar statis | Publik |

### Goal Service (`http://localhost:8080`)

| Method | Endpoint | Deskripsi | Otorisasi |
|---|---|---|---|
| `GET` | `/healthcheck` | Cek status layanan | Publik |
| `GET` | `/api/v1/goals` | Daftar target milik user | Bearer Token |
| `POST` | `/api/v1/goals` | Membuat target baru | Bearer Token |
| `GET` | `/api/v1/goals/{id}` | Detail target spesifik | Bearer Token (pemilik) |
| `PUT` | `/api/v1/goals/{id}` | Memperbarui target | Bearer Token (pemilik) |
| `DELETE` | `/api/v1/goals/{id}` | Menghapus target | Bearer Token (pemilik) |
| `POST` | `/api/v1/goals/{id}/contributions` | Menambah setoran | Bearer Token (pemilik) |
| `GET` | `/api/v1/goals/{id}/contributions` | Riwayat setoran | Bearer Token (pemilik) |
| `DELETE` | `/api/v1/goals/{id}/contributions/{cId}` | Hapus/batal setoran | Bearer Token (pemilik) |
| `GET` | `/api/v1/goals/{id}/progress` | Snapshot progres + proyeksi | Bearer Token (pemilik) |
| `GET` | `/api/v1/goals/{id}/projection` | Detail proyeksi inflasi | Bearer Token (pemilik) |
| `GET` | `/api/v1/goals/heatmap` | Agregasi setoran per hari 365 hari terakhir | Bearer Token |
| `GET` | `/api/v1/notifications` | Daftar notifikasi milik user | Bearer Token |
| `PUT` | `/api/v1/notifications/{id}/read` | Tandai notifikasi dibaca | Bearer Token |

> Koleksi pengujian manual tersedia di `apitest.http` (kompatibel dengan ekstensi REST Client VS Code). Alur: register → login (token otomatis dipakai) → create goal → contributions → progress/projection.

### Contoh Body

**Register** (`POST /api/v1/auth/register`):

```json
{
  "name": "Budi",
  "email": "budi@example.com",
  "password": "rahasia123"
}
```

**Buat target** (`POST /api/v1/goals`):

```json
{
  "title": "DP Rumah",
  "category": "Rumah",
  "target_amount": 100000000,
  "current_amount": 0,
  "target_date": "2027-12-31",
  "expected_inflation_rate": 3,
  "status": "active"
}
```

**Tambah setoran** (`POST /api/v1/goals/{id}/contributions`):

```json
{
  "amount": 500000,
  "note": "Gaji Jan"
}
```

---

## 🧪 Testing

```powershell
# Unit test auth-service
cd artagoal-backend/services/auth-service
go test ./...

# Unit test goal-service
cd ../goal-service
go test ./...
```

---

## 🔑 Rute Frontend

| Rute | Halaman | Akses |
|---|---|---|
| `/` | Redirect ke `/dashboard` | — |
| `/login`, `/register` | Autentikasi | Guest only |
| `/dashboard` | Ringkasan, heatmap, grafik, filter, daftar target | Login |
| `/goals/new` | Form target + simulasi inflasi | Login |
| `/profile` | Ubah nama + foto profil | Login |

---

## 📝 Catatan

- `target_date` menerima format `YYYY-MM-DD` atau RFC3339. Pada **update**, goal yang sudah lewat tenggat tetap bisa disunting.
- `current_amount` bersifat read-only di endpoint update — saldo hanya berubah lewat setoran/kontribusi.
- Status goal yang valid: `active`, `achieved`, `cancelled`.
- JWT dikirim via header `Authorization: Bearer <token>`; frontend menyimpan token di `localStorage` dan otomatis redirect ke `/login` saat 401.
- **Keamanan secret:** `JWT_SECRET` harus sama di semua service dan minimal 32 karakter acak. Jangan pernah commit file `.env` berisi kredensial asli (sudah masuk `.gitignore`); gunakan `*.env.example` sebagai acuan.
