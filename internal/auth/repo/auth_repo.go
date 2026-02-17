package repo

import (
	"database/sql"
	"fmt"

	"time"

	"volumetric-backend/internal/auth/models"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"volumetric-backend/internal/auth/utils"
)

type AuthRepo struct {
	db *sql.DB
}

func NewAuthRepo(db *sql.DB) *AuthRepo {
	return &AuthRepo{db: db}
}

// GetUserByEmail — used during OTP send / login
func (r *AuthRepo) GetUserByEmail(email string) (*models.User, error) {
	user := &models.User{}
	err := r.db.QueryRow(`
		SELECT id, user_id, full_name, email, phone, password_hash,
		       is_email_verified, is_phone_verified, is_active,
		       created_at, updated_at
		FROM users
		WHERE email = $1
	`, email).Scan(
		&user.ID, &user.UserID, &user.FullName, &user.Email, &user.Phone,
		&user.PasswordHash, &user.IsEmailVerified, &user.IsPhoneVerified,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

// StoreOTP — save OTP hash + metadata
func (r *AuthRepo) StoreOTP(userID uuid.UUID, emailOrMobile string, otp string, otpType models.OtpCategory, deviceID string) error {
	hashedOTP, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	expiresAt := time.Now().Add(10 * time.Minute)

	_, err = r.db.Exec(`
		INSERT INTO otp_login (
			user_id, email_or_mobile, device_id, otp_type, otp_hash, expires_at,
			is_used, is_valid, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, false, true, now())
	`, userID, emailOrMobile, deviceID, otpType, string(hashedOTP), expiresAt)
	return err
}

// GetLatestValidOTP — find the most recent non-expired OTP for this email + device
func (r *AuthRepo) GetLatestValidOTP(emailOrMobile, deviceID string) (*models.OtpLogin, error) {
	otp := &models.OtpLogin{}
	err := r.db.QueryRow(`
		SELECT id, user_id, email_or_mobile, device_id, otp_type, otp_hash,
		       expires_at, is_used, is_valid, created_at
		FROM otp_login
		WHERE email_or_mobile = $1
		  AND device_id = $2
		  AND is_valid = true
		  AND is_used = false
		  AND expires_at > now()
		ORDER BY created_at DESC
		LIMIT 1
	`, emailOrMobile, deviceID).Scan(
		&otp.ID, &otp.UserID, &otp.EmailOrMobile, &otp.DeviceID,
		&otp.OtpType, &otp.OtpHash, &otp.ExpiresAt,
		&otp.IsUsed, &otp.IsValid, &otp.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return otp, nil
}

// MarkOTPUsed — after successful verification
func (r *AuthRepo) MarkOTPUsed(otpID uint) error {
	_, err := r.db.Exec("UPDATE otp_login SET is_used = true, is_valid = false WHERE id = $1", otpID)
	return err
}

// CreateSession — store refresh token + session info
func (r *AuthRepo) CreateSession(session *models.UserSession) error {
	_, err := r.db.Exec(`
		INSERT INTO user_session (
			session_id, user_id, device_type, device_id, user_agent, os,
			ip_address, geolocation, app_version, timezone,
			created_at, last_activity_at, session_expired_at,
			refresh_token, refresh_token_expires_at,
			auth_method, is_valid, device_signature
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now(), now(), $11, $12, $13, $14, true, $15)
	`,
		session.SessionID, session.UserID, session.DeviceType, session.DeviceID,
		session.UserAgent, session.OS, session.IPAddress, session.Geolocation,
		session.AppVersion, session.Timezone,
		session.SessionExpiredAt, session.RefreshToken,
		session.RefreshTokenExpiresAt, session.AuthMethod,
		session.DeviceSignature,
	)
	return err
}

// GetUserRolesAndPermissions — load roles + perms when user logs in
func (r *AuthRepo) GetUserRolesAndPermissions(userID uuid.UUID) ([]string, []string, error) {
	rows, err := r.db.Query(`
		SELECT DISTINCT r.user_role, p.permission
		FROM user_role ur
		JOIN role r ON ur.role_id = r.id
		LEFT JOIN role_permission rp ON r.id = rp.role_id
		LEFT JOIN permission p ON rp.permission_id = p.id
		WHERE ur.user_id = $1
	`, userID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	roleMap := make(map[string]bool)
	permMap := make(map[string]bool)

	for rows.Next() {
		var role, perm sql.NullString
		if err := rows.Scan(&role, &perm); err != nil {
			return nil, nil, err
		}
		if role.Valid {
			roleMap[role.String] = true
		}
		if perm.Valid {
			permMap[perm.String] = true
		}
	}

	var roles, perms []string
	for r := range roleMap {
		roles = append(roles, r)
	}
	for p := range permMap {
		perms = append(perms, p)
	}

	return roles, perms, nil
}


// GetSessionByRefreshToken finds a valid session by comparing hash of plain token
func (r *AuthRepo) GetSessionByRefreshToken(refreshTokenPlain string) (*models.UserSession, error) {
	rows, err := r.db.Query(`
		SELECT id, session_id, user_id, device_type, device_id, user_agent, os,
		       ip_address, geolocation, app_version, timezone, created_at,
		       last_activity_at, session_expired_at, refresh_token, refresh_token_expires_at,
		       auth_method, is_valid, device_signature
		FROM user_session
		WHERE is_valid = true 
		  AND refresh_token_expires_at > now()
		  AND created_at > now() - INTERVAL '30 days'  -- limit to recent sessions
		ORDER BY created_at DESC
		LIMIT 10  -- small limit — most users have few sessions
	`)
	if err != nil {
		return nil, fmt.Errorf("query sessions failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		session := &models.UserSession{}
		err := rows.Scan(
			&session.ID, &session.SessionID, &session.UserID, &session.DeviceType, &session.DeviceID,
			&session.UserAgent, &session.OS, &session.IPAddress, &session.Geolocation,
			&session.AppVersion, &session.Timezone, &session.CreatedAt,
			&session.LastActivityAt, &session.SessionExpiredAt, &session.RefreshToken,
			&session.RefreshTokenExpiresAt, &session.AuthMethod, &session.IsValid,
			&session.DeviceSignature,
		)
		if err != nil {
			return nil, fmt.Errorf("scan session failed: %w", err)
		}

		// Compare plain token with stored hash
		if utils.CompareRefreshToken(refreshTokenPlain, session.RefreshToken) {
			return session, nil // match found
		}
	}

	return nil, nil // no match — invalid/expired
}



// GetUserByUUID fetches user by user_id (UUID string from JWT/sub)
func (r *AuthRepo) GetUserByUUID(userUUID uuid.UUID) (*models.User, error) {
    user := &models.User{}
    err := r.db.QueryRow(`
        SELECT id, user_id, full_name, email, phone,
               password_hash, is_email_verified, is_phone_verified,
               is_active, created_at, updated_at
        FROM users
        WHERE user_id = $1
    `, userUUID).Scan(
        &user.ID, &user.UserID, &user.FullName, &user.Email, &user.Phone,
        &user.PasswordHash, &user.IsEmailVerified, &user.IsPhoneVerified,
        &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
    )
    if err == sql.ErrNoRows {
        return nil, nil // not found
    }
    if err != nil {
        return nil, fmt.Errorf("GetUserByUUID failed: %w", err)
    }
    return user, nil
}
// UpdateSessionLastActivity updates last_activity_at for a session
func (r *AuthRepo) UpdateSessionLastActivity(sessionID uint) error {
	_, err := r.db.Exec(`
		UPDATE user_session 
		SET last_activity_at = now()
		WHERE id = $1 AND is_valid = true
	`, sessionID)
	if err != nil {
		return fmt.Errorf("UpdateSessionLastActivity failed: %w", err)
	}
	return nil
}

// InvalidateSessionByRefreshToken invalidates by matching hash
func (r *AuthRepo) InvalidateSessionByRefreshToken(refreshTokenPlain string) error {
	rows, err := r.db.Query(`
		SELECT id, refresh_token
		FROM user_session
		WHERE is_valid = true AND refresh_token_expires_at > now()
		LIMIT 10
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id uint
		var storedHash []byte
		if err := rows.Scan(&id, &storedHash); err != nil {
			continue
		}

		if utils.CompareRefreshToken(refreshTokenPlain, storedHash) {
			_, err := r.db.Exec("UPDATE user_session SET is_valid = false WHERE id = $1", id)
			if err != nil {
				return err
			}
			return nil // success
		}
	}
	return nil 
}

// InvalidateSessionByUserAndDevice invalidates by user + device
func (r *AuthRepo) InvalidateSessionByUserAndDevice(userID uuid.UUID, deviceID string) error {
	_, err := r.db.Exec(`
		UPDATE user_session 
		SET is_valid = false
		WHERE user_id = $1 AND device_id = $2 AND is_valid = true
	`, userID, deviceID)
	return err
}

func (r *AuthRepo) GetSessionByUserAndDevice(userID uuid.UUID, deviceID string) (*models.UserSession, error) {
	session := &models.UserSession{}
	err := r.db.QueryRow(`
		SELECT id, session_id, user_id, device_type, device_id, user_agent, os,
		       ip_address, geolocation, app_version, timezone, created_at,
		       last_activity_at, session_expired_at, refresh_token, refresh_token_expires_at,
		       auth_method, is_valid, device_signature
		FROM user_session
		WHERE user_id = $1 AND device_id = $2
		  AND is_valid = true
		  AND refresh_token_expires_at > now()
		ORDER BY created_at DESC
		LIMIT 1
	`, userID, deviceID).Scan(
		&session.ID, &session.SessionID, &session.UserID, &session.DeviceType, &session.DeviceID,
		&session.UserAgent, &session.OS, &session.IPAddress, &session.Geolocation,
		&session.AppVersion, &session.Timezone, &session.CreatedAt,
		&session.LastActivityAt, &session.SessionExpiredAt, &session.RefreshToken,
		&session.RefreshTokenExpiresAt, &session.AuthMethod, &session.IsValid,
		&session.DeviceSignature,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return session, nil
}

// (8) This file is your Auth Repository (Database Layer).

// It runs SQL queries
// Repository = Layer that talks directly to the database
// It does NOT handle HTTP

// This file:

// Reads users from DB
// Stores OTP
// Verifies OTP
// Creates sessions
// Checks sessions
// Invalidates sessions
// Loads roles & permissions