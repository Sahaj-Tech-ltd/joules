CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    verification_code TEXT,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_admin BOOLEAN NOT NULL DEFAULT FALSE,
    approved BOOLEAN NOT NULL DEFAULT TRUE,
    must_change_password BOOLEAN NOT NULL DEFAULT FALSE,
    verification_code_expires_at TIMESTAMPTZ,
    plan TEXT NOT NULL DEFAULT 'free',
    plan_expires_at TIMESTAMPTZ,
    trial_started_at TIMESTAMPTZ
);

CREATE TABLE user_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL DEFAULT '',
    age INT,
    sex TEXT CHECK (sex IN ('male', 'female', 'other')),
    height_cm NUMERIC(5,1),
    weight_kg NUMERIC(5,1),
    target_weight_kg NUMERIC(5,1),
    activity_level TEXT CHECK (activity_level IN ('sedentary', 'light', 'moderate', 'active', 'very_active')),
    avatar_url TEXT,
    coach_notes TEXT NOT NULL DEFAULT '',
    onboarding_complete BOOLEAN NOT NULL DEFAULT FALSE,
    identity_aspiration TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_goals (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    objective TEXT NOT NULL CHECK (objective IN ('cut_fat', 'feel_better', 'maintain', 'build_muscle')),
    diet_plan TEXT NOT NULL CHECK (diet_plan IN ('calorie_deficit', 'keto', 'intermittent_fasting', 'paleo', 'mediterranean', 'balanced')),
    fasting_window TEXT CHECK (fasting_window IN ('16:8', '18:6', '20:4', 'omad')),
    daily_calorie_target INT NOT NULL DEFAULT 2000,
    daily_protein_g INT NOT NULL DEFAULT 150,
    daily_carbs_g INT NOT NULL DEFAULT 200,
    daily_fat_g INT NOT NULL DEFAULT 65,
    eating_window_start TIME DEFAULT '12:00',
    current_fast_start TIMESTAMPTZ,
    fasting_streak INT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE meals (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    meal_type TEXT NOT NULL CHECK (meal_type IN ('breakfast', 'lunch', 'dinner', 'snack')),
    photo_path TEXT,
    note TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE food_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    meal_id UUID NOT NULL REFERENCES meals(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    calories INT NOT NULL DEFAULT 0,
    protein_g NUMERIC(8,1) NOT NULL DEFAULT 0,
    carbs_g NUMERIC(8,1) NOT NULL DEFAULT 0,
    fat_g NUMERIC(8,1) NOT NULL DEFAULT 0,
    fiber_g NUMERIC(8,1) NOT NULL DEFAULT 0,
    serving_size TEXT DEFAULT '',
    source TEXT NOT NULL DEFAULT 'manual' CHECK (source IN ('ai', 'manual', 'leftover', 'db', 'barcode', 'recipe'))
);

CREATE TABLE weight_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    weight_kg NUMERIC(5,1) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, date)
);

CREATE TABLE water_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    amount_ml INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE exercises (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    name TEXT NOT NULL,
    duration_min INT NOT NULL,
    calories_burned INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE coach_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('user', 'assistant')),
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE achievements (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT 'general',
    progress_current INT NOT NULL DEFAULT 0,
    progress_target INT NOT NULL DEFAULT 0,
    unlocked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, type)
);



CREATE INDEX idx_meals_user_date ON meals(user_id, timestamp);
CREATE INDEX idx_food_items_meal ON food_items(meal_id);
CREATE INDEX idx_weight_logs_user ON weight_logs(user_id, date);
CREATE INDEX idx_water_logs_user ON water_logs(user_id, date);
CREATE INDEX idx_exercises_user ON exercises(user_id, timestamp);
CREATE INDEX idx_coach_messages_user ON coach_messages(user_id, created_at);
CREATE INDEX idx_achievements_user ON achievements(user_id);

CREATE TABLE step_logs (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    step_count INT NOT NULL DEFAULT 0,
    source TEXT NOT NULL DEFAULT 'manual' CHECK (source IN ('manual', 'google_fit')),
    PRIMARY KEY (user_id, date)
);

CREATE TABLE google_fit_tokens (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    access_token TEXT NOT NULL,
    refresh_token TEXT NOT NULL DEFAULT '',
    expiry TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS user_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    diet_type TEXT NOT NULL DEFAULT 'omnivore',
    allergies TEXT[] NOT NULL DEFAULT '{}',
    food_notes TEXT NOT NULL DEFAULT '',
    eating_context TEXT NOT NULL DEFAULT '',
    height_unit TEXT NOT NULL DEFAULT 'cm',
    weight_unit TEXT NOT NULL DEFAULT 'kg',
    energy_unit TEXT NOT NULL DEFAULT 'kcal',
    dietary_restrictions TEXT[] NOT NULL DEFAULT '{}',
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
INSERT INTO app_settings (key, value) VALUES ('ai_provider', '') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('ai_model', '') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('routing_model', '') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('vision_model', '') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('ocr_model', '') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('custom_base_url', '') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('custom_api_key', '') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('vision_provider', '') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('vision_api_key', '') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('vision_base_url', '') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('ocr_provider', '') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('ocr_api_key', '') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('ocr_base_url', '') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('classifier_provider', '') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('classifier_api_key', '') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('classifier_base_url', '') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('classifier_model', '') ON CONFLICT (key) DO NOTHING;

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

-- Cheat days
CREATE TABLE IF NOT EXISTS cheat_days (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    PRIMARY KEY (user_id, date)
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
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Admin banners
CREATE TABLE IF NOT EXISTS admin_banners (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL DEFAULT '',
    message TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'info',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- System logs
CREATE TABLE IF NOT EXISTS system_logs (
    id BIGSERIAL PRIMARY KEY,
    level TEXT NOT NULL DEFAULT 'info',
    category TEXT NOT NULL DEFAULT 'general',
    message TEXT NOT NULL,
    details JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

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
CREATE TABLE IF NOT EXISTS user_stats (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    total_points INT NOT NULL DEFAULT 0,
    streak_days INT NOT NULL DEFAULT 0,
    last_active_date DATE,
    current_phase TEXT NOT NULL DEFAULT 'scaffolding',
    phase_updated_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

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

-- Per-user food memory (correction learning)
CREATE TABLE IF NOT EXISTS user_food_memory (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    food_name TEXT NOT NULL,
    canonical_name TEXT,
    calories REAL NOT NULL,
    protein REAL DEFAULT 0,
    carbs REAL DEFAULT 0,
    fat REAL DEFAULT 0,
    fiber REAL DEFAULT 0,
    serving_size REAL,
    serving_unit TEXT,
    correction_count INT DEFAULT 1,
    source TEXT DEFAULT 'user_correction',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, food_name)
);
CREATE INDEX IF NOT EXISTS idx_user_food_memory_user ON user_food_memory(user_id);

-- Identity quote history (deduplication)
CREATE TABLE IF NOT EXISTS identity_quotes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    quote TEXT NOT NULL,
    date DATE NOT NULL DEFAULT CURRENT_DATE,
    context_type TEXT DEFAULT 'daily',
    UNIQUE(user_id, date)
);

-- Grace days tracking
CREATE TABLE IF NOT EXISTS grace_days (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    week_start DATE NOT NULL,
    days_used INT DEFAULT 0,
    max_per_week INT DEFAULT 2,
    PRIMARY KEY (user_id, week_start)
);

-- Implementation intentions (if-then plans)
CREATE TABLE IF NOT EXISTS implementation_intentions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    meal_type TEXT NOT NULL,
    trigger_text TEXT NOT NULL,
    action_text TEXT NOT NULL,
    notification_time TIME,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_implementation_intentions_user ON implementation_intentions(user_id);

-- Stripe subscriptions (cloud only)
CREATE TABLE IF NOT EXISTS subscriptions (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    stripe_customer_id TEXT,
    stripe_subscription_id TEXT,
    status TEXT DEFAULT 'trialing',
    current_period_end TIMESTAMPTZ,
    plan TEXT DEFAULT 'free'
);

-- Expo push tokens
CREATE TABLE IF NOT EXISTS expo_push_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL,
    platform TEXT NOT NULL CHECK (platform IN ('ios', 'android')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, token)
);
CREATE INDEX IF NOT EXISTS idx_expo_push_tokens_user ON expo_push_tokens(user_id);

-- Behavioral insights (computed nightly by scheduler)
CREATE TABLE IF NOT EXISTS behavioral_insights (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    patterns JSONB NOT NULL DEFAULT '{}',
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Notification deduplication log (used by fresh-start scheduler)
CREATE TABLE IF NOT EXISTS notification_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tag TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_notification_log_user_tag ON notification_log(user_id, tag, created_at);

-- Plan-tier model overrides
INSERT INTO app_settings (key, value) VALUES ('model_vision_free', '') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('model_primary_free', '') ON CONFLICT (key) DO NOTHING;
