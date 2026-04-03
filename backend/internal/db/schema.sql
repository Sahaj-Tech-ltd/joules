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
    approved BOOLEAN NOT NULL DEFAULT TRUE
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
    onboarding_complete BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_goals (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    objective TEXT NOT NULL CHECK (objective IN ('cut_fat', 'feel_better', 'maintain', 'build_muscle')),
    diet_plan TEXT NOT NULL CHECK (diet_plan IN ('calorie_deficit', 'keto', 'intermittent_fasting', 'paleo', 'mediterranean')),
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
    source TEXT NOT NULL DEFAULT 'manual' CHECK (source IN ('ai', 'manual'))
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
    source TEXT NOT NULL DEFAULT 'manual',
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
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Habit tracking / points
CREATE TABLE IF NOT EXISTS user_stats (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    total_points INT NOT NULL DEFAULT 0,
    streak_days INT NOT NULL DEFAULT 0,
    last_active_date DATE,
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
