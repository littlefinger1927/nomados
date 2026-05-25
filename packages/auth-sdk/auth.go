package authsdk

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID    string `json:"uid"`
	SessionID string `json:"sid"`
	DeviceID  string `json:"did"`
	jwt.RegisteredClaims
}

type TokenValidator struct {
	secret []byte
}

func NewTokenValidator(secret string) *TokenValidator {
	return &TokenValidator{secret: []byte(secret)}
}

func (v *TokenValidator) GenerateAccessToken(userID, sessionID, deviceID string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:    userID,
		SessionID: sessionID,
		DeviceID:  deviceID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "nomados",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(v.secret)
}

func (v *TokenValidator) ValidateAccessToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return v.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}