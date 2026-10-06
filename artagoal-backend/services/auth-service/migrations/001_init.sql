-- ArtaGoal auth-service baseline schema.
-- Terapkan sekali: psql "$DATABASE_URL" -f 001_init.sql
-- Aman dijalankan ulang (IF NOT EXISTS).
-- Email disimpan lowercase oleh aplikasi; UNIQUE menjamin satu akun per email.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 100),
    email TEXT NOT NULL UNIQUE CHECK (char_length(email) BETWEEN 3 AND 255),
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);
