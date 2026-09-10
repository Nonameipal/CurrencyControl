package service

import (
	"context"
	"fmt"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/ldap"
	"CurrencyControl/internal/repository"
)

type LoginStatus string

const (
	StatusActive  LoginStatus = "active"   
	StatusNoRole  LoginStatus = "no_role"  
)

type LoginResult struct {
	Status  LoginStatus     `json:"status"`
	Token   string          `json:"token,omitempty"`  
	Login   string          `json:"login,omitempty"`   
	Message string          `json:"message"`
	Session *domain.Session `json:"session,omitempty"`
}

type AuthService interface {
	Login(ctx context.Context, login, password string) (LoginResult, error)
	RequestAccess(ctx context.Context, login string, branchID int64, role string) (domain.AccessRequest, error)
	GetRequestStatus(ctx context.Context, requestID int64) (*domain.AccessRequest, error)
	GetPendingRequests(ctx context.Context) ([]domain.AccessRequest, error)
	ApproveRequest(ctx context.Context, requestID int64) (domain.Session, error)
	RejectRequest(ctx context.Context, requestID int64) error
	ValidateSession(ctx context.Context, token string) (*domain.Session, error)
	Logout(ctx context.Context, token string) error
}

type authService struct {
	repo repository.AuthRepository
	ldap ldap.Client
}

func NewAuthService(repo repository.AuthRepository, ldap ldap.Client) AuthService {
	return &authService{repo: repo, ldap: ldap}
}

func (s *authService) Login(ctx context.Context, login, password string) (LoginResult, error) {
	ok, err := s.ldap.Authenticate(login, password)
	if err != nil || !ok {
		if err != nil {
			return LoginResult{}, err
		}
		return LoginResult{}, fmt.Errorf("неверный логин или пароль")
	}

	user, _ := s.repo.GetUserByLogin(ctx, login)

	if user != nil {
		sess, err := s.repo.CreateSession(ctx, user.Login, user.Role, user.BranchID)
		if err != nil {
			return LoginResult{}, fmt.Errorf("ошибка создания сессии: %w", err)
		}
		return LoginResult{
			Status:  StatusActive,
			Token:   sess.Token,
			Message: "Успешный вход",
			Session: &sess,
		}, nil
	}

	return LoginResult{
		Status:  StatusNoRole,
		Login:   login,
		Message: "Укажите ваш филиал и роль для получения доступа",
	}, nil
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

func (s *authService) ApproveRequest(ctx context.Context, requestID int64) (domain.Session, error) {
	return s.repo.ApproveRequest(ctx, requestID)
}

func (s *authService) RejectRequest(ctx context.Context, requestID int64) error {
	return s.repo.RejectRequest(ctx, requestID)
}

func (s *authService) ValidateSession(ctx context.Context, token string) (*domain.Session, error) {
	return s.repo.GetSessionByToken(ctx, token)
}

func (s *authService) Logout(ctx context.Context, token string) error {
	return s.repo.DeleteSession(ctx, token)
}
