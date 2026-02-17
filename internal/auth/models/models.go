package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type OtpCategory string

const (
	OtpEmail OtpCategory = "email"
	OtpSMS   OtpCategory = "sms"
	//OtpWhatsapp OtpCategory = "whatsapp"
)

type AuthMethod string

const (
	AuthEmailOtp      AuthMethod = "email_otp"
	AuthMobileOtp     AuthMethod = "mobile_otp"
	AuthEmailPassword AuthMethod = "email_password"
)

type DeviceType string

const (
	DeviceWeb     DeviceType = "web"
	DeviceMobile  DeviceType = "mobile"
	DeviceDesktop DeviceType = "desktop"
)

type User struct {
	ID              uint      `sql:"id"`
	UserID          uuid.UUID `sql:"user_id"`
	FullName        string    `sql:"full_name"`
	Email           string    `sql:"email"`
	Phone           *string   `sql:"phone"`
	PasswordHash    string    `sql:"password_hash"`
	IsEmailVerified bool      `sql:"is_email_verified"`
	IsPhoneVerified bool      `sql:"is_phone_verified"`
	IsActive        bool      `sql:"is_active"`
	CreatedAt       time.Time `sql:"created_at"`
	UpdatedAt       time.Time `sql:"updated_at"`

	Roles       []string
	Permissions []string
}

type OtpLogin struct {
	ID            uint        `sql:"id"`
	UserID        uuid.UUID   `sql:"user_id"`
	EmailOrMobile string      `sql:"email_or_mobile"`
	DeviceID      string      `sql:"device_id"`
	OtpType       OtpCategory `sql:"otp_type"`
	OtpHash       string      `sql:"otp_hash"`
	ExpiresAt     time.Time   `sql:"expires_at"`
	IsUsed        bool        `sql:"is_used"`
	IsValid       bool        `sql:"is_valid"`
	CreatedAt     time.Time   `sql:"created_at"`
}

type UserSession struct {
	ID                    uint       `sql:"id"`
	SessionID             uuid.UUID  `sql:"session_id"`
	UserID                uuid.UUID  `sql:"user_id"`
	DeviceType            DeviceType `sql:"device_type"`
	DeviceID              string     `sql:"device_id"`
	UserAgent             []byte     `sql:"user_agent"`
	OS                    string     `sql:"os"`
	IPAddress             string     `sql:"ip_address"`
	Geolocation           string     `sql:"geolocation"`
	AppVersion            string     `sql:"app_version"`
	Timezone              string     `sql:"timezone"`
	CreatedAt             time.Time  `sql:"created_at"`
	LastActivityAt        time.Time  `sql:"last_activity_at"`
	SessionExpiredAt      time.Time  `sql:"session_expired_at"`
	RefreshToken          []byte     `sql:"refresh_token"` // hashed
	RefreshTokenExpiresAt time.Time  `sql:"refresh_token_expires_at"`
	AuthMethod            AuthMethod `sql:"auth_method"`
	IsValid               bool       `sql:"is_valid"`
	DeviceSignature       string     `sql:"device_signature"`
}

type Claims struct {
	UserID      uuid.UUID `json:"sub"`
	Email       string    `json:"email"`
	Roles       []string  `json:"roles"`
	Permissions []string  `json:"permissions"`

	jwt.RegisteredClaims
}

// (7) This file defines your database models (structures) for authentication system.

// Models = Blueprint of your database tables
// Each struct usually represents a table