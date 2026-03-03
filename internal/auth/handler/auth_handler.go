package handler

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/render"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"volumetric-backend/internal/auth/jwt"
	
	"volumetric-backend/internal/auth/models"
	"volumetric-backend/internal/auth/repo"
	"volumetric-backend/internal/auth/utils"
)

type AuthHandler struct {
	Repo *repo.AuthRepo
}

func NewAuthHandler(repo *repo.AuthRepo) *AuthHandler {
	return &AuthHandler{Repo: repo}
}

// SendOTPRequest — what client sends
type SendOTPRequest struct {
	Email string `json:"email"`
}

// SendOTPResponse
type SendOTPResponse struct {
	Message   string `json:"message"`
	DeviceID  string `json:"device_id,omitempty"`
	ExpiresIn int    `json:"expires_in,omitempty"`
}
type VerifyOTPRequest struct {
	Email    string `json:"email"`
	OTP      string `json:"otp"`
	DeviceID string `json:"device_id"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // seconds
	Message      string `json:"message"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token,omitempty"` 
}


// POST /auth/otp/send
func (h *AuthHandler) SendOTP(w http.ResponseWriter, r *http.Request) {
	var req SendOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("SendOTP: invalid JSON: %v", err)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid JSON"})
		return
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" || !strings.Contains(email, "@") {
		log.Printf("SendOTP: invalid email: %q", email)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Valid email is required"})
		return
	}

	// Find user
	user, err := h.Repo.GetUserByEmail(email)
	if err != nil {
		log.Printf("SendOTP: GetUserByEmail failed for %q: %v", email, err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal error"})
		return
	}
	if user == nil || !user.IsActive {
		log.Printf("SendOTP: user not found or inactive for %q", email)
		render.Status(r, http.StatusOK)
		render.JSON(w, r, map[string]string{
			"message": "If the email exists, OTP has been sent",
		})
		return
	}

	// Generate 6-digit OTP
	otp, err := generateNumericOTP(6)
	if err != nil {
		log.Printf("SendOTP: generate OTP failed: %v", err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to generate OTP"})
		return
	}

	// Device ID
	deviceID := r.Header.Get("X-Device-ID")
	if deviceID == "" {
		deviceID = uuid.New().String()
	}

	// Store OTP
	err = h.Repo.StoreOTP(user.UserID, email, otp, models.OtpEmail, deviceID)
	if err != nil {
		log.Printf("SendOTP: StoreOTP failed for user %s: %v", user.UserID, err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to store OTP"})
		return
	}

	// In production: send via email/SMS (Twilio, SendGrid, etc.)
	// For now: print to console/logs
	fmt.Printf("\n=== OTP GENERATED ===\n")
	fmt.Printf("To: %s\n", email)
	fmt.Printf("OTP: %s\n", otp)
	fmt.Printf("DeviceID: %s\n", deviceID)
	fmt.Printf("Expires in 10 minutes\n")
	fmt.Printf("===================\n")

	// Set device_id in response (client should store it)
	render.Status(r, http.StatusOK)
	render.JSON(w, r, SendOTPResponse{
		Message:   "OTP sent successfully (check server logs/console for OTP in dev)",
		DeviceID:  deviceID,
		ExpiresIn: 600, // seconds
	})

	// Optional: set cookie for web clients
	http.SetCookie(w, &http.Cookie{
		Name:     "device_id",
		Value:    deviceID,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // true in prod with HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	})
}

// Helper: generate random 6-digit string
func generateNumericOTP(length int) (string, error) {
	const digits = "0123456789"
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		b[i] = digits[n.Int64()]
	}
	return string(b), nil
}

func (h *AuthHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req VerifyOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("VerifyOTP: invalid JSON: %v", err)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid JSON"})
		return
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))
	otp := strings.TrimSpace(req.OTP)
	deviceID := strings.TrimSpace(req.DeviceID)

	if email == "" || otp == "" || deviceID == "" {
		log.Printf("VerifyOTP: missing required fields")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Email, OTP and device_id are required"})
		return
	}

	// 1. Get latest valid OTP
	otpRecord, err := h.Repo.GetLatestValidOTP(email, deviceID)
	if err != nil {
		log.Printf("VerifyOTP: GetLatestValidOTP error: %v", err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal error"})
		return
	}
	if otpRecord == nil {
		log.Printf("VerifyOTP: no valid OTP found for %s / %s", email, deviceID)
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "Invalid or expired OTP"})
		return
	}

	// 2. Compare OTP
	if err := bcrypt.CompareHashAndPassword([]byte(otpRecord.OtpHash), []byte(otp)); err != nil {
		log.Printf("VerifyOTP: OTP mismatch for %s", email)
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "Invalid OTP"})
		return
	}

	// 3. Mark as used
	if err := h.Repo.MarkOTPUsed(otpRecord.ID); err != nil {
		log.Printf("VerifyOTP: failed to mark OTP used: %v", err)
		// not fatal — continue
	}

	// 4. Load user again
	user, err := h.Repo.GetUserByEmail(email)
	if err != nil || user == nil {
		log.Printf("VerifyOTP: user disappeared: %v", err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "User not found"})
		return
	}

	// 5. Load roles & permissions
	roles, permissions, err := h.Repo.GetUserRolesAndPermissions(user.UserID)
	if err != nil {
		log.Printf("VerifyOTP: failed to load roles: %v", err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to load permissions"})
		return
	}

	// 6. Generate access token
	accessToken, err := jwt.GenerateAccessToken(user, roles, permissions)
	if err != nil {
		log.Printf("VerifyOTP: failed to generate access token: %v", err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to generate token"})
		return
	}

	// 7. Generate refresh token (plain random string)
	refreshTokenPlain := uuid.New().String()

	// Hash it before storing
	refreshTokenHash, err := utils.HashRefreshToken(refreshTokenPlain)
	if err != nil {
		log.Printf("VerifyOTP: failed to hash refresh token: %v", err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create session"})
		return
	}

	// 8. Create session
	session := &models.UserSession{
		SessionID:             uuid.New(),
		UserID:                user.UserID,
		DeviceType:            models.DeviceWeb,
		DeviceID:              deviceID,
		UserAgent:             []byte(r.UserAgent()),
		OS:                    "unknown",
		IPAddress:             extractIP(r.RemoteAddr),
		Geolocation:           "",
		Timezone:              "",
		SessionExpiredAt:      time.Now().Add(7 * 24 * time.Hour),
		RefreshToken:          refreshTokenHash,
		RefreshTokenExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		AuthMethod:            models.AuthEmailOtp,
		DeviceSignature:       deviceID + r.UserAgent(),
	}
	if err := h.Repo.CreateSession(session); err != nil {
		log.Printf("VerifyOTP: failed to create session: %v", err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create session"})
		return
	}

	// 9. Success
	log.Printf("VerifyOTP: success for %s", email)

	render.Status(r, http.StatusOK)
	render.JSON(w, r, AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenPlain,
		ExpiresIn:    600, // 60  minutes
		Message:      "Login successful",
	})
}

func extractIP(remoteAddr string) string {
	if remoteAddr == "" {
		return "unknown"
	}

	ip, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {

		return ip
	}

	cleaned := strings.Trim(remoteAddr, "[]")
	if strings.Contains(cleaned, ":") {

		parts := strings.Split(cleaned, ":")
		if len(parts) > 1 {
			return strings.Join(parts[:len(parts)-1], ":")
		}
	}

	return cleaned
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("RefreshToken: invalid JSON: %v", err)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid JSON"})
		return
	}

	refreshToken := strings.TrimSpace(req.RefreshToken)
	if refreshToken == "" {
		log.Printf("RefreshToken: missing refresh_token")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Refresh token required"})
		return
	}

	// 1. Look up session
	session, err := h.Repo.GetSessionByRefreshToken(refreshToken)
	if err != nil {
		log.Printf("RefreshToken: DB lookup failed: %v", err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal error"})
		return
	}
	if session == nil || !session.IsValid || time.Now().After(session.RefreshTokenExpiresAt) {
		log.Printf("RefreshToken: invalid/expired session")
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "Invalid or expired refresh token"})
		return
	}

	// 2. Load roles & permissions
	roles, permissions, err := h.Repo.GetUserRolesAndPermissions(session.UserID)
	if err != nil {
		log.Printf("RefreshToken: failed to load roles: %v", err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to load permissions"})
		return
	}

	// 3. Load user by UUID
	user, err := h.Repo.GetUserByUUID(session.UserID)
	if err != nil || user == nil {
		log.Printf("RefreshToken: user not found: %v", err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "User not found"})
		return
	}

	// 4. Generate new access token
	newAccessToken, err := jwt.GenerateAccessToken(user, roles, permissions)
	if err != nil {
		log.Printf("RefreshToken: failed to generate token: %v", err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to generate token"})
		return
	}

	// 5. Update last activity (via repo — clean)
	if err := h.Repo.UpdateSessionLastActivity(session.ID); err != nil {
		log.Printf("RefreshToken: failed to update activity: %v", err)
	
	}

	// 6. Success
	log.Printf("RefreshToken: success for user %s", session.UserID)
	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]interface{}{
		"access_token": newAccessToken,
		"expires_in":   600,
		"message":      "Token refreshed successfully",
	})
}




// POST /auth/logout — invalidate current session
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
    // 1. Get token from header (manual check — no middleware here)
    authHeader := r.Header.Get("Authorization")
    if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
        render.Status(r, http.StatusUnauthorized)
        render.JSON(w, r, map[string]string{"error": "Missing or invalid token"})
        return
    }

    tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

    // 2. Validate token (same as middleware)
    claims, err := jwt.ValidateAccessToken(tokenStr)
    if err != nil {
        log.Printf("Logout: token invalid: %v", err)
        render.Status(r, http.StatusUnauthorized)
        render.JSON(w, r, map[string]string{"error": "Invalid or expired token"})
        return
    }

    userID := claims.UserID

    // 3. Optional: use device_id if sent
    deviceID := r.Header.Get("X-Device-ID")
    if deviceID == "" {
        deviceID = "unknown"
    }

    // 4. Invalidate the session
    err = h.Repo.InvalidateSessionByUserAndDevice(userID, deviceID)
    if err != nil {
        log.Printf("Logout: failed to invalidate for user %s: %v", userID, err)
        render.Status(r, http.StatusInternalServerError)
        render.JSON(w, r, map[string]string{"error": "Failed to logout"})
        return
    }

    // 5. Success
    log.Printf("Logout: success for user %s", userID)
    render.Status(r, http.StatusOK)
    render.JSON(w, r, map[string]string{"message": "Logged out successfully"})
}

// (9) This file handles all authentication HTTP requests.

// It connects:
// Client request → Repo (DB) → JWT → Response