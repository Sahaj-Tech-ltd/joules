-- Idempotent migrations — safe to run on existing installs
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_admin BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS approved BOOLEAN NOT NULL DEFAULT TRUE;

CREATE TABLE IF NOT EXISTS user_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    diet_type TEXT NOT NULL DEFAULT 'omnivore',
    allergies TEXT[] NOT NULL DEFAULT '{}',
    food_notes TEXT NOT NULL DEFAULT '',
    eating_context TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS app_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO app_settings (key, value) VALUES ('require_approval', 'false')
ON CONFLICT (key) DO NOTHING;

-- Expand diet_plan constraint to include 'balanced'
ALTER TABLE user_goals DROP CONSTRAINT IF EXISTS user_goals_diet_plan_check;
ALTER TABLE user_goals ADD CONSTRAINT user_goals_diet_plan_check
    CHECK (diet_plan IN ('calorie_deficit', 'keto', 'intermittent_fasting', 'paleo', 'mediterranean', 'balanced'));

-- Avatar URL for user profiles
ALTER TABLE user_profiles ADD COLUMN IF NOT EXISTS avatar_url TEXT;

-- AI settings stored in app_settings
INSERT INTO app_settings (key, value) VALUES ('ai_provider', '') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('ai_model', '') ON CONFLICT (key) DO NOTHING;

-- Admin banners table
CREATE TABLE IF NOT EXISTS admin_banners (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL DEFAULT '',
    message TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'info',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Unit preferences
ALTER TABLE user_preferences ADD COLUMN IF NOT EXISTS height_unit TEXT NOT NULL DEFAULT 'cm';
ALTER TABLE user_preferences ADD COLUMN IF NOT EXISTS weight_unit TEXT NOT NULL DEFAULT 'kg';
ALTER TABLE user_preferences ADD COLUMN IF NOT EXISTS energy_unit TEXT NOT NULL DEFAULT 'kcal';

-- Force password change flag (for default admin accounts)
ALTER TABLE users ADD COLUMN IF NOT EXISTS must_change_password BOOLEAN NOT NULL DEFAULT FALSE;

-- System logs table
CREATE TABLE IF NOT EXISTS system_logs (
    id BIGSERIAL PRIMARY KEY,
    level TEXT NOT NULL DEFAULT 'info',
    category TEXT NOT NULL DEFAULT 'general',
    message TEXT NOT NULL,
    details JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
