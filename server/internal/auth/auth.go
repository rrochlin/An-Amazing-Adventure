package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type RefreshToken struct {
	Token     string     `json:"token"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	UserID    uuid.UUID  `json:"user_id"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

type User struct {
	ID             uuid.UUID `json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Email          string    `json:"email"`
	HashedPassword string    `json:"hashed_password"`
}

func HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), 4)
	if err != nil {
		return "", err
	}
	return string(hashed), err
}

func CheckPasswordHash(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func MakeJWT(userID uuid.UUID, tokenSecret string) (string, error) {
	expiresIn, _ := time.ParseDuration("1h")
	JWT := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.RegisteredClaims{
			Issuer:    "maze",
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
			Subject:   userID.String(),
		})
	signature, err := JWT.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}
	return signature, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claimsStruct := jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		&claimsStruct,
		func(t *jwt.Token) (any, error) {
			return []byte(tokenSecret), nil
		})
	if err != nil {
		fmt.Println("error parsing jwt")
		return uuid.UUID{}, err
	}
	id, err := token.Claims.GetSubject()
	if err != nil {
		fmt.Println("error retrieving claims")
		return uuid.UUID{}, err
	}

	issuer, err := token.Claims.GetIssuer()
	if err != nil {
		fmt.Println("error retrieving claims")
		return uuid.UUID{}, err
	}
	if issuer != string("maze") {
		return uuid.UUID{}, fmt.Errorf("invalid issuer")
	}

	parsedId, err := uuid.Parse(id)
	if err != nil {
		fmt.Println("error parsing uuid")
		return uuid.UUID{}, err
	}

	return parsedId, nil

}

func GetBearerToken(headers http.Header) (string, error) {
	auth := headers.Get("Authorization")
	if auth == "" {
		fmt.Println("auth header missing")
		return "", fmt.Errorf("auth header missing")
	}
	return strings.TrimPrefix(auth, "Bearer "), nil
}

func MakeRefreshToken() (string, error) {
	key := make([]byte, 32)
	rand.Read(key)
	return hex.EncodeToString(key), nil
}

func GetAPIKey(headers http.Header) (string, error) {
	auth := headers.Get("Authorization")
	if auth == "" {
		fmt.Println("auth header missing")
		return "", fmt.Errorf("auth header missing")
	}
	return strings.TrimPrefix(auth, "ApiKey "), nil

}
