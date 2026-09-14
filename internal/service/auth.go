package service

import (
	"context"
	"fmt"
	"time"

	"CurrencyControl/internal/configs"
	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/ldap"
	"CurrencyControl/internal/service/ports"
	"CurrencyControl/pkg"
)

type authService struct {
	repo ports.AuthRepository
	ldap ldap.Client
}

func NewAuthService(repo ports.AuthRepository, ldap ldap.Client) ports.AuthService {
	return &authService{repo: repo, ldap: ldap}
}

func (s *authService) Login(ctx context.Context, login, password string) (ports.LoginResult, error) {
	ok, err := s.ldap.Authenticate(login, password)
	if err != nil || !ok {
		if err != nil {
			return ports.LoginResult{}, err
		}
		return ports.LoginResult{}, fmt.Errorf("неверный логин или пароль")
	}

	user, _ := s.repo.GetUserByLogin(ctx, login)

	if user != nil {
		result, err := s.createTokenPair(ctx, user.ID, user.Login, user.Role, user.BranchID, ports.StatusActive, "Успешный вход")
		if err != nil {
			return ports.LoginResult{}, fmt.Errorf("ошибка создания сессии: %w", err)
		}
		return result, nil
	}

	result, err := s.createTokenPair(ctx, 0, login, "pre_auth", 0, ports.StatusNoRole, "Укажите ваш филиал и роль для получения доступа")
	if err != nil {
		return ports.LoginResult{}, fmt.Errorf("ошибка создания предварительной сессии: %w", err)
	}
	result.Login = login
	return result, nil
}

func (s *authService) RequestAccess(ctx context.Context, login string, branchID int64, role string) (domain.AccessRequest, error) {
	return s.repo.CreateAccessRequest(ctx, login, branchID, role)
}

func (s *authService) GetRequestStatus(ctx context.Context, requestID int64) (*domain.AccessRequest, error) {
	return s.repo.GetRequestByID(ctx, requestID)
}

func (s *authService) GetPendingRequests(ctx context.Context) ([]domain.AccessRequest, error) {
	return s.repo.GetPendingRequests(ctx)
}

func (s *authService) ApproveRequest(ctx context.Context, requestID int64) (ports.LoginResult, error) {
	user, err := s.repo.ApproveRequest(ctx, requestID)
	if err != nil {
		return ports.LoginResult{}, err
	}
	result, err := s.createTokenPair(ctx, user.ID, user.Login, user.Role, user.BranchID, ports.StatusActive, "Доступ предоставлен")
	if err != nil {
		return ports.LoginResult{}, err
	}
	if err := s.repo.SetAccessRequestSessionToken(ctx, requestID, result.RefreshToken); err != nil {
		return ports.LoginResult{}, err
	}
	return result, nil
}

func (s *authService) RejectRequest(ctx context.Context, requestID int64) error {
	return s.repo.RejectRequest(ctx, requestID)
}

func (s *authService) ValidateSession(ctx context.Context, token string) (*domain.Session, error) {
	claims, err := pkg.ParseToken(token)
	if err != nil {
		return nil, err
	}
	if claims.IsRefresh {
		return nil, fmt.Errorf("refresh token нельзя использовать как access token")
	}
	return &domain.Session{
		Login:     claims.Login,
		Role:      claims.Role,
		BranchID:  claims.BranchID,
		ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}

func (s *authService) Logout(ctx context.Context, token string) error {
	claims, err := pkg.ParseToken(token)
	if err == nil && claims.Login != "" {
		if claims.IsRefresh {
			return s.repo.DeleteSession(ctx, token)
		}
		return s.repo.DeleteSessionsByLogin(ctx, claims.Login)
	}
	return s.repo.DeleteSession(ctx, token)
}

func (s *authService) Refresh(ctx context.Context, refreshToken string) (ports.LoginResult, error) {
	claims, err := pkg.ParseToken(refreshToken)
	if err != nil {
		return ports.LoginResult{}, err
	}
	if !claims.IsRefresh {
		return ports.LoginResult{}, fmt.Errorf("ожидался refresh token")
	}
	sess, err := s.repo.GetSessionByToken(ctx, refreshToken)
	if err != nil || sess == nil {
		return ports.LoginResult{}, fmt.Errorf("refresh token недействителен или отозван")
	}

	accessExpiresAt := time.Now().Add(time.Duration(configs.AppSettings.AuthParams.AccessTokenTtlMinutes) * time.Minute)
	accessToken, err := pkg.GenerateToken(claims.UserID, claims.Login, claims.BranchID, configs.AppSettings.AuthParams.AccessTokenTtlMinutes, claims.Role, false)
	if err != nil {
		return ports.LoginResult{}, err
	}

	return ports.LoginResult{
		Status:              ports.StatusActive,
		AccessToken:         accessToken,
		RefreshToken:        refreshToken,
		AccessTokenExpires:  accessExpiresAt,
		RefreshTokenExpires: sess.ExpiresAt,
		Message:             "Токен обновлён",
		Session:             sess,
	}, nil
}

func (s *authService) createTokenPair(ctx context.Context, userID int64, login, role string, branchID int64, status ports.LoginStatus, message string) (ports.LoginResult, error) {
	accessTTL := configs.AppSettings.AuthParams.AccessTokenTtlMinutes
	refreshTTL := configs.AppSettings.AuthParams.RefreshTokenTtlDays
	now := time.Now()
	accessExpiresAt := now.Add(time.Duration(accessTTL) * time.Minute)
	refreshExpiresAt := now.Add(time.Duration(refreshTTL) * 24 * time.Hour)

	accessToken, err := pkg.GenerateToken(userID, login, branchID, accessTTL, role, false)
	if err != nil {
		return ports.LoginResult{}, err
	}
	refreshToken, err := pkg.GenerateToken(userID, login, branchID, refreshTTL, role, true)
	if err != nil {
		return ports.LoginResult{}, err
	}
	sess, err := s.repo.SaveSession(ctx, refreshToken, login, role, branchID, refreshExpiresAt)
	if err != nil {
		return ports.LoginResult{}, err
	}

	return ports.LoginResult{
		Status:              status,
		AccessToken:         accessToken,
		RefreshToken:        refreshToken,
		AccessTokenExpires:  accessExpiresAt,
		RefreshTokenExpires: refreshExpiresAt,
		Login:               login,
		Message:             message,
		Session:             &sess,
	}, nil
}
