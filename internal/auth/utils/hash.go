package utils
import "golang.org/x/crypto/bcrypt"

// hashRefreshToken creates bcrypt hash of refresh token
func HashRefreshToken(token string) ([]byte, error) {
    return bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
}

// compareRefreshToken checks plain token against stored hash
func CompareRefreshToken(plain string, hashed []byte) bool {
    err := bcrypt.CompareHashAndPassword(hashed, []byte(plain))
    return err == nil
}