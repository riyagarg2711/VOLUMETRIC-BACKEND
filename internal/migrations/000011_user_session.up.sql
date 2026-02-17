CREATE TABLE IF NOT EXISTS user_session (
    id BIGSERIAL PRIMARY KEY,
    session_id UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    user_id CHAR(36) REFERENCES users(user_id) ON DELETE CASCADE,
    device_type auth_device_type,
    device_id VARCHAR(256),
    user_agent BYTEA,
    os VARCHAR(150),
    ip_address INET,
    geolocation VARCHAR(255), 
    app_version VARCHAR(50),
    timezone VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT now(),
    last_activity_at TIMESTAMPTZ DEFAULT now(),
    session_expired_at TIMESTAMPTZ NOT NULL,
    refresh_token BYTEA NOT NULL, 
    refresh_token_expires_at TIMESTAMPTZ NOT NULL,
    auth_method auth_method_type,
    is_valid BOOLEAN DEFAULT true,
    device_signature VARCHAR(256) NOT NULL
);

-- Indexes
CREATE INDEX idx_sessions_user_id ON user_session(user_id);
CREATE INDEX idx_sessions_refresh_token_expires_at ON user_session(refresh_token_expires_at);
CREATE INDEX idx_sessions_is_valid ON user_session(is_valid);
CREATE INDEX idx_sessions_created_at ON user_session(created_at);
CREATE INDEX idx_sessions_refresh_token ON user_session(refresh_token);
CREATE INDEX idx_sessions_device_signature ON user_session(device_signature);
CREATE INDEX idx_sessions_last_activity_at ON user_session(last_activity_at);
CREATE INDEX idx_sessions_session_expired_at ON user_session(session_expired_at);