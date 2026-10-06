-- artagoal goal-service baseline schema.
-- Terapkan sekali: psql "$DATABASE_URL" -f 001_init.sql
-- Aman dijalankan ulang (IF NOT EXISTS).

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS goals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    title TEXT NOT NULL,
    category TEXT NOT NULL DEFAULT '',
    target_amount DOUBLE PRECISION NOT NULL CHECK (target_amount > 0),
    current_amount DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (current_amount >= 0),
    target_date TIMESTAMPTZ NULL,
    expected_inflation_rate DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (expected_inflation_rate >= 0),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','achieved','cancelled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_goals_user_id ON goals (user_id);

CREATE TABLE IF NOT EXISTS goal_contributions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    goal_id UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    amount DOUBLE PRECISION NOT NULL CHECK (amount > 0),
    note TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_goal_contributions_goal_id ON goal_contributions (goal_id);

