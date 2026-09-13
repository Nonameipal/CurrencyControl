package pkg

import (
	"fmt"
	"time"

	"CurrencyControl/internal/configs"
	"github.com/golang-jwt/jwt/v5"
)

type CustomClaims struct {
	jwt.RegisteredClaims
	UserID    int    `json:"user_id"`
	Role      string `json:"role"`
	IsRefresh bool   `json:"is_refresh"`
}

func GenerateToken(userID int, ttl int, role string, isRefresh bool) (string, error) {
	claims := CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{},
		UserID:           userID,
		IsRefresh:        isRefresh,
		Role:             role,
	}

	if isRefresh {
		claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Duration(ttl) * 24 * time.Hour))
	} else {
		claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Duration(ttl) * time.Minute))
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(configs.AppSettings.AuthParams.JwtSecret))
}

func ParseToken(tokenString string) (int, bool, string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(configs.AppSettings.AuthParams.JwtSecret), nil
	})
	if err != nil {
		return 0, false, "", err
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return 0, false, "", fmt.Errorf("invalid token")
	}

	return claims.UserID, claims.IsRefresh, claims.Role, nil
}
