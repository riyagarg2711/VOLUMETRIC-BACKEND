CREATE TYPE  otp_category AS ENUM ('email', 'sms', 'whatsapp');

CREATE TYPE  auth_method_type AS ENUM (
    'username_password', 'mobile_password', 'email_password',
    'username_otp', 'mobile_otp', 'email_otp'
);

CREATE TYPE auth_device_type AS ENUM ('desktop', 'mobile', 'web');


CREATE TABLE IF NOT EXISTS otp_login (
    id BIGSERIAL PRIMARY KEY,
    user_id CHAR(36) NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    email_or_mobile VARCHAR(128) NOT NULL,
    device_id VARCHAR(256),
    otp_type otp_category NOT NULL,
    otp_hash VARCHAR(256) NOT NULL,  
    expires_at TIMESTAMPTZ NOT NULL,
    is_used BOOLEAN NOT NULL DEFAULT FALSE,
    is_valid BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes
CREATE INDEX idx_otp_login_user_id ON otp_login(user_id);
CREATE INDEX idx_otp_login_email_created ON otp_login(email_or_mobile, created_at);