package pkg

import (
	"fmt"
	"time"

	"CurrencyControl/internal/configs"
	"github.com/golang-jwt/jwt/v5"
)

type CustomClaims struct {
	jwt.RegisteredClaims
	UserID    int64  `json:"user_id"`
	Login     string `json:"login"`
	LastName  string `json:"last_name,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	Email     string `json:"email,omitempty"`
	Role      string `json:"role"`
	BranchID  int64  `json:"branch_id"`
	IsRefresh bool   `json:"is_refresh"`
}

func GenerateToken(userID int64, login, lastName, firstName, email string, branchID int64, ttl int, role string, isRefresh bool) (string, error) {
	claims := CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{},
		UserID:           userID,
		Login:            login,
		LastName:         lastName,
		FirstName:        firstName,
		Email:            email,
		IsRefresh:        isRefresh,
		Role:             role,
		BranchID:         branchID,
	}

	if isRefresh {
		claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Duration(ttl) * 24 * time.Hour))
	} else {
		claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Duration(ttl) * time.Minute))
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(configs.AppSettings.AuthParams.JwtSecret))
}

func ParseToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(configs.AppSettings.AuthParams.JwtSecret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}
