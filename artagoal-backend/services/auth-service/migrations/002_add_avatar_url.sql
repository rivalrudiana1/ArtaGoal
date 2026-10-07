-- ArtaGoal auth-service: kolom avatar profil user.
-- Terapkan: psql "$DATABASE_URL" -f 002_add_avatar_url.sql
-- Aman dijalankan ulang (IF NOT EXISTS); baris lama terisi '' via DEFAULT.

ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url VARCHAR(255) DEFAULT '';
