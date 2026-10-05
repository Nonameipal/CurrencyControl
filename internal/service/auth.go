package service

import (
	"context"
	"fmt"
	"strings"
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
	info, err := s.ldap.Authenticate(login, password)
	if err != nil {
		return ports.LoginResult{}, err
	}

	user, _ := s.repo.GetUserByLogin(ctx, login)

	if user != nil {
		// Всегда обновляем ФИО и email из AD при входе
		if info != nil {
			_ = s.repo.UpdateUserInfo(ctx, user.Login, info.LastName, info.FirstName, info.Email)
		}

		lastName := info.LastName
		firstName := info.FirstName
		email := info.Email
		if lastName == "" {
			lastName = user.LastName
		}
		if firstName == "" {
			firstName = user.FirstName
		}
		if email == "" {
			email = user.Email
		}

		result, err := s.createTokenPair(ctx, user.ID, user.Login, lastName, firstName, email, user.Role, user.BranchID, ports.StatusActive, "Успешный вход")
		if err != nil {
			return ports.LoginResult{}, fmt.Errorf("ошибка создания сессии: %w", err)
		}
		result.LastName = lastName
		result.FirstName = firstName
		result.Email = email
		return result, nil
	}

	result, err := s.createTokenPair(ctx, 0, login, info.LastName, info.FirstName, info.Email, "pre_auth", 0, ports.StatusNoRole, "Укажите ваш филиал и роль для получения доступа")
	if err != nil {
		return ports.LoginResult{}, fmt.Errorf("ошибка создания предварительной сессии: %w", err)
	}
	result.Login = login
	result.LastName = info.LastName
	result.FirstName = info.FirstName
	result.Email = info.Email
	return result, nil
}

func (s *authService) RequestAccess(ctx context.Context, login, lastName, firstName, email string, branchID int64, role string) (domain.AccessRequest, error) {
	role = strings.TrimSpace(role)
	if role == domain.RoleAdmin {
		return domain.AccessRequest{}, fmt.Errorf("роль 'admin' нельзя запросить через интерфейс. Она назначается только напрямую в базе данных")
	}
	if !domain.IsAssignableRole(role) {
		return domain.AccessRequest{}, fmt.Errorf("недопустимая роль: %s", role)
	}
	return s.repo.CreateAccessRequest(ctx, login, lastName, firstName, email, branchID, role)
}

func (s *authService) GetRequestStatus(ctx context.Context, requestID int64) (*domain.AccessRequest, error) {
	return s.repo.GetRequestByID(ctx, requestID)
}

func (s *authService) GetPendingRequests(ctx context.Context) ([]domain.AccessRequest, error) {
	return s.repo.GetPendingRequests(ctx)
}

func (s *authService) GetAccessRequestsHistory(ctx context.Context) ([]domain.AccessRequest, error) {
	return s.repo.GetAccessRequestsHistory(ctx)
}

func (s *authService) ApproveRequest(ctx context.Context, requestID int64, reviewer string) (ports.LoginResult, error) {
	user, err := s.repo.ApproveRequest(ctx, requestID, reviewer)
	if err != nil {
		return ports.LoginResult{}, err
	}
	result, err := s.createTokenPair(ctx, user.ID, user.Login, user.LastName, user.FirstName, user.Email, user.Role, user.BranchID, ports.StatusActive, "Доступ предоставлен")
	if err != nil {
		return ports.LoginResult{}, err
	}
	if err := s.repo.SetAccessRequestSessionToken(ctx, requestID, result.RefreshToken); err != nil {
		return ports.LoginResult{}, err
	}
	return result, nil
}

func (s *authService) RejectRequest(ctx context.Context, requestID int64, reviewer string) error {
	return s.repo.RejectRequest(ctx, requestID, reviewer)
}

func (s *authService) GetAllUsers(ctx context.Context) ([]domain.User, error) {
	return s.repo.GetAllUsers(ctx)
}

func (s *authService) CreateUser(ctx context.Context, req domain.CreateUserRequest) (domain.User, error) {
	login := strings.TrimSpace(req.Login)
	if login == "" {
		return domain.User{}, fmt.Errorf("логин обязателен")
	}
	role := strings.TrimSpace(req.Role)
	if role == domain.RoleAdmin {
		return domain.User{}, fmt.Errorf("роль 'admin' нельзя назначить")
	}
	if !domain.IsAssignableRole(role) {
		return domain.User{}, fmt.Errorf("недопустимая роль: %s", role)
	}
	if req.BranchID <= 0 {
		return domain.User{}, fmt.Errorf("branch_id обязателен и должен быть больше 0")
	}

	user := domain.User{
		Login:    login,
		Role:     role,
		BranchID: req.BranchID,
	}
	return s.repo.CreateUser(ctx, user)
}

func (s *authService) UpdateUser(ctx context.Context, id int64, req domain.UpdateUserRequest) (domain.User, error) {
	if id <= 0 {
		return domain.User{}, fmt.Errorf("некорректный ID пользователя")
	}
	role := strings.TrimSpace(req.Role)
	if role == domain.RoleAdmin {
		return domain.User{}, fmt.Errorf("роль 'admin' нельзя назначить")
	}
	if role != "" && !domain.IsAssignableRole(role) {
		return domain.User{}, fmt.Errorf("недопустимая роль: %s", role)
	}
	if req.BranchID < 0 {
		return domain.User{}, fmt.Errorf("некорректный branch_id")
	}

	return s.repo.UpdateUser(ctx, id, role, req.BranchID)
}

func (s *authService) DeleteUser(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("некорректный ID пользователя")
	}
	return s.repo.DeleteUser(ctx, id)
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
		LastName:  claims.LastName,
		FirstName: claims.FirstName,
		Email:     claims.Email,
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
	accessToken, err := pkg.GenerateToken(claims.UserID, claims.Login, claims.LastName, claims.FirstName, claims.Email, claims.BranchID, configs.AppSettings.AuthParams.AccessTokenTtlMinutes, claims.Role, false)
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

func (s *authService) createTokenPair(ctx context.Context, userID int64, login, lastName, firstName, email, role string, branchID int64, status ports.LoginStatus, message string) (ports.LoginResult, error) {
	accessTTL := configs.AppSettings.AuthParams.AccessTokenTtlMinutes
	refreshTTL := configs.AppSettings.AuthParams.RefreshTokenTtlDays
	now := time.Now()
	accessExpiresAt := now.Add(time.Duration(accessTTL) * time.Minute)
	refreshExpiresAt := now.Add(time.Duration(refreshTTL) * 24 * time.Hour)

	accessToken, err := pkg.GenerateToken(userID, login, lastName, firstName, email, branchID, accessTTL, role, false)
	if err != nil {
		return ports.LoginResult{}, err
	}
	refreshToken, err := pkg.GenerateToken(userID, login, lastName, firstName, email, branchID, refreshTTL, role, true)
	if err != nil {
		return ports.LoginResult{}, err
	}
	sess, err := s.repo.SaveSession(ctx, refreshToken, login, lastName, firstName, email, role, branchID, refreshExpiresAt)
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
