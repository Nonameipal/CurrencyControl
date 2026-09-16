package repository

import (
	"context"
	"fmt"
	"time"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type authRepo struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) ports.AuthRepository {
	return &authRepo{db: db}
}

func (r *authRepo) GetUserByLogin(ctx context.Context, login string) (*domain.User, error) {
	var u domain.User
	if err := r.db.WithContext(ctx).Where("login = ?", login).First(&u).Error; err != nil {
		return nil, nil
	}
	return &u, nil
}

func (r *authRepo) CreateAccessRequest(ctx context.Context, login string, branchID int64, role string) (domain.AccessRequest, error) {
	var cnt int64
	r.db.WithContext(ctx).Model(&domain.AccessRequest{}).
		Where("login = ? AND status = ?", login, "pending").
		Count(&cnt)
	if cnt > 0 {
		return domain.AccessRequest{}, fmt.Errorf("запрос уже ожидает подтверждения")
	}

	req := domain.AccessRequest{
		Login:    login,
		BranchID: branchID,
		Role:     role,
		Status:   "pending",
	}
	if err := r.db.WithContext(ctx).Create(&req).Error; err != nil {
		return domain.AccessRequest{}, err
	}

	var b domain.Branch
	if err := r.db.WithContext(ctx).Select("name").First(&b, branchID).Error; err == nil {
		req.BranchName = b.Name
	}

	return req, nil
}

func (r *authRepo) GetRequestByID(ctx context.Context, requestID int64) (*domain.AccessRequest, error) {
	var req domain.AccessRequest
	err := r.db.WithContext(ctx).Table("access_requests ar").
		Select("ar.id, ar.login, ar.branch_id, COALESCE(b.name, '') as branch_name, ar.role, ar.status, ar.session_token, ar.created_at, ar.reviewed_at").
		Joins("LEFT JOIN branches b ON b.id = ar.branch_id").
		Where("ar.id = ?", requestID).
		Scan(&req).Error
	if err != nil || req.ID == 0 {
		return nil, fmt.Errorf("запрос не найден")
	}
	return &req, nil
}

func (r *authRepo) GetPendingRequests(ctx context.Context) ([]domain.AccessRequest, error) {
	var result []domain.AccessRequest
	err := r.db.WithContext(ctx).Table("access_requests ar").
		Select("ar.id, ar.login, ar.branch_id, COALESCE(b.name, '') as branch_name, ar.role, ar.status, ar.session_token, ar.created_at, ar.reviewed_at").
		Joins("LEFT JOIN branches b ON b.id = ar.branch_id").
		Where("ar.status = ?", "pending").
		Order("ar.created_at ASC").
		Scan(&result).Error
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = []domain.AccessRequest{}
	}
	return result, nil
}

func (r *authRepo) ApproveRequest(ctx context.Context, requestID int64) (domain.User, error) {
	var req domain.AccessRequest
	now := time.Now()
	res := r.db.WithContext(ctx).Model(&domain.AccessRequest{}).
		Where("id = ? AND status = ?", requestID, "pending").
		Updates(map[string]interface{}{
			"status":      "approved",
			"reviewed_at": &now,
		})
	if res.Error != nil {
		return domain.User{}, res.Error
	}
	if res.RowsAffected == 0 {
		return domain.User{}, fmt.Errorf("запрос не найден или уже обработан")
	}

	if err := r.db.WithContext(ctx).First(&req, requestID).Error; err != nil {
		return domain.User{}, err
	}

	user := domain.User{
		Login:    req.Login,
		Role:     req.Role,
		BranchID: req.BranchID,
	}
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "login"}},
		DoUpdates: clause.AssignmentColumns([]string{"role", "branch_id"}),
	}).Create(&user).Error
	if err != nil {
		return domain.User{}, err
	}

	_ = r.db.WithContext(ctx).Where("login = ?", req.Login).First(&user)
	return user, nil
}

func (r *authRepo) RejectRequest(ctx context.Context, requestID int64) error {
	now := time.Now()
	res := r.db.WithContext(ctx).Model(&domain.AccessRequest{}).
		Where("id = ? AND status = ?", requestID, "pending").
		Updates(map[string]interface{}{
			"status":      "rejected",
			"reviewed_at": &now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("запрос не найден или уже обработан")
	}
	return nil
}

func (r *authRepo) SaveSession(ctx context.Context, token, login, role string, branchID int64, expiresAt time.Time) (domain.Session, error) {
	r.db.WithContext(ctx).Where("login = ?", login).Delete(&domain.Session{})

	sess := domain.Session{
		Token:     token,
		Login:     login,
		Role:      role,
		BranchID:  branchID,
		ExpiresAt: expiresAt,
	}
	err := r.db.WithContext(ctx).Create(&sess).Error
	return sess, err
}

func (r *authRepo) GetSessionByToken(ctx context.Context, token string) (*domain.Session, error) {
	var sess domain.Session
	err := r.db.WithContext(ctx).
		Where("token = ? AND expires_at > ?", token, time.Now()).
		First(&sess).Error
	if err != nil {
		return nil, nil
	}
	return &sess, nil
}

func (r *authRepo) DeleteSession(ctx context.Context, token string) error {
	return r.db.WithContext(ctx).Where("token = ?", token).Delete(&domain.Session{}).Error
}

func (r *authRepo) DeleteSessionsByLogin(ctx context.Context, login string) error {
	return r.db.WithContext(ctx).Where("login = ?", login).Delete(&domain.Session{}).Error
}

func (r *authRepo) SetAccessRequestSessionToken(ctx context.Context, requestID int64, token string) error {
	return r.db.WithContext(ctx).Model(&domain.AccessRequest{}).
		Where("id = ?", requestID).
		Update("session_token", token).Error
}
