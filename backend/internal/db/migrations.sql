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

-- Nutrition cache: persists AI/web-fetched nutrition lookups for reuse
CREATE TABLE IF NOT EXISTS nutrition_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    query TEXT NOT NULL,
    name TEXT NOT NULL,
    calories INT NOT NULL DEFAULT 0,
    protein_g NUMERIC(7,2) NOT NULL DEFAULT 0,
    carbs_g NUMERIC(7,2) NOT NULL DEFAULT 0,
    fat_g NUMERIC(7,2) NOT NULL DEFAULT 0,
    fiber_g NUMERIC(7,2) NOT NULL DEFAULT 0,
    serving_size TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT 'ai',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS nutrition_cache_query_idx ON nutrition_cache (lower(query));

-- Push notification subscriptions (Web Push VAPID)
CREATE TABLE IF NOT EXISTS push_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint TEXT NOT NULL UNIQUE,
    p256dh TEXT NOT NULL,
    auth TEXT NOT NULL,
    user_agent TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS push_subscriptions_user_idx ON push_subscriptions(user_id);

-- Notification preferences per user
CREATE TABLE IF NOT EXISTS notification_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    water_reminders BOOLEAN NOT NULL DEFAULT TRUE,
    water_interval_hours INT NOT NULL DEFAULT 2,
    meal_reminders BOOLEAN NOT NULL DEFAULT TRUE,
    if_window_reminders BOOLEAN NOT NULL DEFAULT TRUE,
    streak_reminders BOOLEAN NOT NULL DEFAULT TRUE,
    quiet_start INT NOT NULL DEFAULT 22,
    quiet_end INT NOT NULL DEFAULT 8,
    ntfy_topic TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- v2.5 migrations

-- Foods database (populated via admin import or API fallback)
CREATE TABLE IF NOT EXISTS foods_db (
    id BIGSERIAL PRIMARY KEY,
    barcode TEXT,
    name TEXT NOT NULL,
    brand TEXT NOT NULL DEFAULT '',
    calories INT NOT NULL DEFAULT 0,
    protein_g NUMERIC(7,2) NOT NULL DEFAULT 0,
    carbs_g NUMERIC(7,2) NOT NULL DEFAULT 0,
    fat_g NUMERIC(7,2) NOT NULL DEFAULT 0,
    fiber_g NUMERIC(7,2) NOT NULL DEFAULT 0,
    serving_size TEXT NOT NULL DEFAULT '100g',
    ingredients TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS foods_db_barcode_idx ON foods_db(barcode) WHERE barcode IS NOT NULL AND barcode != '';
CREATE INDEX IF NOT EXISTS foods_db_name_fts_idx ON foods_db USING GIN (to_tsvector('english', name));
CREATE INDEX IF NOT EXISTS foods_db_name_trgm_idx ON foods_db(lower(name));

-- User recipes
CREATE TABLE IF NOT EXISTS recipes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS recipes_user_idx ON recipes(user_id);

CREATE TABLE IF NOT EXISTS recipe_foods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipe_id UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    calories INT NOT NULL DEFAULT 0,
    protein_g NUMERIC(7,2) NOT NULL DEFAULT 0,
    carbs_g NUMERIC(7,2) NOT NULL DEFAULT 0,
    fat_g NUMERIC(7,2) NOT NULL DEFAULT 0,
    fiber_g NUMERIC(7,2) NOT NULL DEFAULT 0,
    serving_size TEXT NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS recipe_foods_recipe_idx ON recipe_foods(recipe_id);

-- Dietary restrictions on user preferences
ALTER TABLE user_preferences ADD COLUMN IF NOT EXISTS dietary_restrictions TEXT[] NOT NULL DEFAULT '{}';

-- Expand food_items.source to include 'db', 'barcode', 'leftover', 'recipe'
ALTER TABLE food_items DROP CONSTRAINT IF EXISTS food_items_source_check;
ALTER TABLE food_items ADD CONSTRAINT food_items_source_check
    CHECK (source IN ('ai', 'manual', 'leftover', 'db', 'barcode', 'recipe'));

-- Intermittent fasting: add eating window and fast tracking to user_goals
ALTER TABLE user_goals ADD COLUMN IF NOT EXISTS eating_window_start TIME DEFAULT '12:00';
ALTER TABLE user_goals ADD COLUMN IF NOT EXISTS current_fast_start TIMESTAMPTZ;
ALTER TABLE user_goals ADD COLUMN IF NOT EXISTS fasting_streak INT DEFAULT 0;

-- Cheat days: user-marked days where calorie overages are intentional
CREATE TABLE IF NOT EXISTS cheat_days (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    PRIMARY KEY (user_id, date)
);

-- Step counter: daily step logs from manual entry or Google Fit sync
CREATE TABLE IF NOT EXISTS step_logs (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    step_count INT NOT NULL DEFAULT 0,
    source TEXT NOT NULL DEFAULT 'manual',
    PRIMARY KEY (user_id, date)
);

-- Google Fit OAuth tokens per user
CREATE TABLE IF NOT EXISTS google_fit_tokens (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    access_token TEXT NOT NULL,
    refresh_token TEXT NOT NULL DEFAULT '',
    expiry TIMESTAMPTZ NOT NULL
);

-- Social groups
CREATE TABLE IF NOT EXISTS groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL DEFAULT 'private' CHECK (type IN ('private', 'public')),
    invite_code TEXT UNIQUE NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Habit tracking / points
CREATE TABLE IF NOT EXISTS user_stats (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    total_points INT NOT NULL DEFAULT 0,
    streak_days INT NOT NULL DEFAULT 0,
    last_active_date DATE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS group_members (
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('admin', 'member')),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, user_id)
);

CREATE TABLE IF NOT EXISTS group_challenges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    metric TEXT NOT NULL DEFAULT 'meals' CHECK (metric IN ('meals', 'calories', 'steps', 'protein')),
    target_value INT NOT NULL DEFAULT 7,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Coach memory: auto-extracted facts about users
CREATE TABLE IF NOT EXISTS coach_memory (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category TEXT NOT NULL,
    content TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT 'agent',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_coach_memory_user ON coach_memory(user_id, category);

-- Coach reminders
CREATE TABLE IF NOT EXISTS coach_reminders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type TEXT NOT NULL DEFAULT 'custom',
    message TEXT NOT NULL,
    reminder_time TIME NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_coach_reminders_user ON coach_reminders(user_id);

-- Coach notes column on user_profiles
ALTER TABLE user_profiles ADD COLUMN IF NOT EXISTS coach_notes TEXT NOT NULL DEFAULT '';

-- routing_model app setting
INSERT INTO app_settings (key, value) VALUES ('routing_model', '') ON CONFLICT (key) DO NOTHING;

INSERT INTO app_settings (key, value) VALUES ('vision_model', '') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('ocr_model', '') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('custom_base_url', '') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('custom_api_key', '') ON CONFLICT (key) DO NOTHING;

-- Food favorites
CREATE TABLE IF NOT EXISTS food_favorites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    calories INT NOT NULL DEFAULT 0,
    protein_g NUMERIC(8,1) NOT NULL DEFAULT 0,
    carbs_g NUMERIC(8,1) NOT NULL DEFAULT 0,
    fat_g NUMERIC(8,1) NOT NULL DEFAULT 0,
    fiber_g NUMERIC(8,1) NOT NULL DEFAULT 0,
    serving_size TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT 'manual',
    use_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS food_favorites_user_idx ON food_favorites(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS food_favorites_user_name_idx ON food_favorites(user_id, lower(name));

-- Achievement categories + progress
ALTER TABLE achievements ADD COLUMN IF NOT EXISTS category TEXT NOT NULL DEFAULT 'general';
ALTER TABLE achievements ADD COLUMN IF NOT EXISTS progress_current INT NOT NULL DEFAULT 0;
ALTER TABLE achievements ADD COLUMN IF NOT EXISTS progress_target INT NOT NULL DEFAULT 0;
