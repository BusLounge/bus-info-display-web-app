CREATE TABLE IF NOT EXISTS broadcast_messages (
	id UUID PRIMARY KEY,
	message TEXT NOT NULL,
	priority TEXT NOT NULL DEFAULT 'normal',
	display_duration_seconds INTEGER NOT NULL CHECK (display_duration_seconds > 0),
	frequency_seconds INTEGER NOT NULL CHECK (frequency_seconds > 0),
	start_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	end_at TIMESTAMPTZ NULL,
	is_active BOOLEAN NOT NULL DEFAULT TRUE,
	show_on_lounge_tv BOOLEAN NOT NULL DEFAULT TRUE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_broadcast_messages_active_window
	ON broadcast_messages (is_active, start_at, end_at);

CREATE TABLE IF NOT EXISTS lounge_ads (
	id UUID PRIMARY KEY,
	lounge_id UUID NULL REFERENCES lounges(id) ON DELETE CASCADE,
	advertisement_name TEXT NOT NULL,
	media_url TEXT NOT NULL,
	media_type TEXT NOT NULL,
	duration_seconds INTEGER NOT NULL CHECK (duration_seconds > 0),
	priority TEXT NOT NULL DEFAULT 'normal',
	is_active BOOLEAN NOT NULL DEFAULT TRUE,
	is_default_for_all BOOLEAN NOT NULL DEFAULT FALSE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_lounge_ads_lounge_active
	ON lounge_ads (lounge_id, is_active);

CREATE INDEX IF NOT EXISTS idx_lounge_ads_default_active
	ON lounge_ads (is_default_for_all, is_active);
