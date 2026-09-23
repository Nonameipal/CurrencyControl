package repository

import (
	"context"
	"errors"
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *authRepo) GetUserByID(ctx context.Context, id int64) (*domain.User, error) {
	var u domain.User
	err := r.db.WithContext(ctx).Table("users u").
		Select("u.id, u.login, u.last_name, u.first_name, u.email, u.role, u.branch_id, COALESCE(b.name, '') as branch_name, u.created_at").
		Joins("LEFT JOIN branches b ON b.id = u.branch_id").
		Where("u.id = ?", id).
		Scan(&u).Error
	if err != nil || u.ID == 0 {
		return nil, fmt.Errorf("пользователь не найден")
	}
	return &u, nil
}

func (r *authRepo) GetAllUsers(ctx context.Context) ([]domain.User, error) {
	var users []domain.User
	err := r.db.WithContext(ctx).Table("users u").
		Select("u.id, u.login, u.last_name, u.first_name, u.email, u.role, u.branch_id, COALESCE(b.name, '') as branch_name, u.created_at").
		Joins("LEFT JOIN branches b ON b.id = u.branch_id").
		Order("u.id ASC").
		Scan(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (r *authRepo) CreateUser(ctx context.Context, user domain.User) (domain.User, error) {
	var count int64
	r.db.WithContext(ctx).Model(&domain.User{}).Where("login = ?", user.Login).Count(&count)
	if count > 0 {
		return domain.User{}, fmt.Errorf("пользователь с логином '%s' уже существует", user.Login)
	}

	if err := r.db.WithContext(ctx).Create(&user).Error; err != nil {
		return domain.User{}, err
	}

	var b domain.Branch
	if err := r.db.WithContext(ctx).Select("name").First(&b, user.BranchID).Error; err == nil {
		user.BranchName = b.Name
	}
	return user, nil
}

func (r *authRepo) UpdateUserInfo(ctx context.Context, login, lastName, firstName, email string) error {
	updates := map[string]interface{}{}
	if lastName != "" {
		updates["last_name"] = lastName
	}
	if firstName != "" {
		updates["first_name"] = firstName
	}
	if email != "" {
		updates["email"] = email
	}
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&domain.User{}).
		Where("login = ?", login).
		Updates(updates).Error
}

func (r *authRepo) UpdateUser(ctx context.Context, id int64, role string, branchID int64) (domain.User, error) {
	updates := map[string]interface{}{}
	if role != "" {
		updates["role"] = role
	}
	if branchID > 0 {
		updates["branch_id"] = branchID
	}
	if len(updates) == 0 {
		u, err := r.GetUserByID(ctx, id)
		if err != nil {
			return domain.User{}, err
		}
		return *u, nil
	}

	res := r.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", id).Updates(updates)
	if res.Error != nil {
		return domain.User{}, res.Error
	}
	if res.RowsAffected == 0 {
		return domain.User{}, fmt.Errorf("пользователь не найден")
	}

	u, err := r.GetUserByID(ctx, id)
	if err != nil {
		return domain.User{}, err
	}
	return *u, nil
}

func (r *authRepo) DeleteUser(ctx context.Context, id int64) error {
	u, err := r.GetUserByID(ctx, id)
	if err != nil {
		return err
	}

	res := r.db.WithContext(ctx).Delete(&domain.User{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("пользователь не найден")
	}

	_ = r.DeleteSessionsByLogin(ctx, u.Login)
	return nil
}

func (r *authRepo) CreateAccessRequest(ctx context.Context, login, lastName, firstName, email string, branchID int64, role string) (domain.AccessRequest, error) {
	var cnt int64
	r.db.WithContext(ctx).Model(&domain.AccessRequest{}).
		Where("login = ? AND status = ?", login, "pending").
		Count(&cnt)
	if cnt > 0 {
		return domain.AccessRequest{}, fmt.Errorf("запрос уже ожидает подтверждения")
	}

	req := domain.AccessRequest{
		Login:     login,
		LastName:  lastName,
		FirstName: firstName,
		Email:     email,
		BranchID:  branchID,
		Role:      role,
		Status:    "pending",
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
		Select("ar.id, ar.login, ar.last_name, ar.first_name, ar.email, ar.branch_id, COALESCE(b.name, '') as branch_name, ar.role, ar.status, ar.session_token, ar.created_at, ar.reviewed_at, COALESCE(ar.reviewed_by, '') as reviewed_by").
		Joins("LEFT JOIN branches b ON b.id = ar.branch_id").
		Where("ar.id = ?", requestID).
		Scan(&req).Error
	if err != nil || req.ID == 0 {
		return nil, fmt.Errorf("запрос не найден")
	}
	list := []domain.AccessRequest{req}
	_ = r.enrichAccessRequests(ctx, list)
	return &list[0], nil
}

func (r *authRepo) GetPendingRequests(ctx context.Context) ([]domain.AccessRequest, error) {
	var result []domain.AccessRequest
	err := r.db.WithContext(ctx).Table("access_requests ar").
		Select("ar.id, ar.login, ar.last_name, ar.first_name, ar.email, ar.branch_id, COALESCE(b.name, '') as branch_name, ar.role, ar.status, ar.session_token, ar.created_at, ar.reviewed_at, COALESCE(ar.reviewed_by, '') as reviewed_by").
		Joins("LEFT JOIN branches b ON b.id = ar.branch_id").
		Where("ar.status = ?", "pending").
		Order("ar.created_at ASC").
		Scan(&result).Error
	if err != nil {
		return nil, err
	}
	_ = r.enrichAccessRequests(ctx, result)
	return result, nil
}

func (r *authRepo) GetAccessRequestsHistory(ctx context.Context) ([]domain.AccessRequest, error) {
	var result []domain.AccessRequest
	err := r.db.WithContext(ctx).Table("access_requests ar").
		Select("ar.id, ar.login, ar.last_name, ar.first_name, ar.email, ar.branch_id, COALESCE(b.name, '') as branch_name, ar.role, ar.status, ar.session_token, ar.created_at, ar.reviewed_at, COALESCE(ar.reviewed_by, '') as reviewed_by").
		Joins("LEFT JOIN branches b ON b.id = ar.branch_id").
		Where("ar.status IN ('approved', 'rejected')").
		Order("ar.reviewed_at DESC, ar.id DESC").
		Scan(&result).Error
	if err != nil {
		return nil, err
	}
	_ = r.enrichAccessRequests(ctx, result)
	return result, nil
}

func (r *authRepo) enrichAccessRequests(ctx context.Context, list []domain.AccessRequest) error {
	loginsMap := make(map[string]bool)
	for _, ar := range list {
		if ar.ReviewedBy != "" {
			loginsMap[ar.ReviewedBy] = true
		}
	}
	var users []domain.User
	if len(loginsMap) > 0 {
		logins := make([]string, 0, len(loginsMap))
		for l := range loginsMap {
			logins = append(logins, l)
		}
		_ = r.db.WithContext(ctx).Table("users").
			Select("login, first_name, last_name, email").
			Where("login IN ?", logins).
			Find(&users).Error
	}
	userMap := make(map[string]domain.UserBrief, len(users))
	for _, u := range users {
		userMap[u.Login] = domain.UserBrief{
			Login:     u.Login,
			FirstName: u.FirstName,
			LastName:  u.LastName,
			Email:     u.Email,
		}
	}

	for i := range list {
		list[i].Applicant = &domain.UserBrief{
			Login:     list[i].Login,
			FirstName: list[i].FirstName,
			LastName:  list[i].LastName,
			Email:     list[i].Email,
		}
		if list[i].ReviewedBy != "" {
			if rev, ok := userMap[list[i].ReviewedBy]; ok {
				list[i].Reviewer = &rev
			} else {
				list[i].Reviewer = &domain.UserBrief{Login: list[i].ReviewedBy}
			}
		}
	}
	return nil
}

func (r *authRepo) ApproveRequest(ctx context.Context, requestID int64, reviewer string) (domain.User, error) {
	var req domain.AccessRequest
	now := time.Now()
	res := r.db.WithContext(ctx).Model(&domain.AccessRequest{}).
		Where("id = ? AND status = ?", requestID, "pending").
		Updates(map[string]interface{}{
			"status":      "approved",
			"reviewed_at": &now,
			"reviewed_by": reviewer,
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
		Login:     req.Login,
		LastName:  req.LastName,
		FirstName: req.FirstName,
		Email:     req.Email,
		Role:      req.Role,
		BranchID:  req.BranchID,
	}
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "login"}},
		DoUpdates: clause.AssignmentColumns([]string{"role", "branch_id", "last_name", "first_name", "email"}),
	}).Create(&user).Error
	if err != nil {
		return domain.User{}, err
	}

	if err := r.db.WithContext(ctx).Where("login = ?", req.Login).First(&user).Error; err != nil {
		return domain.User{}, err
	}
	return user, nil
}

func (r *authRepo) RejectRequest(ctx context.Context, requestID int64, reviewer string) error {
	now := time.Now()
	res := r.db.WithContext(ctx).Model(&domain.AccessRequest{}).
		Where("id = ? AND status = ?", requestID, "pending").
		Updates(map[string]interface{}{
			"status":      "rejected",
			"reviewed_at": &now,
			"reviewed_by": reviewer,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("запрос не найден или уже обработан")
	}
	return nil
}

func (r *authRepo) SaveSession(ctx context.Context, token, login, lastName, firstName, email, role string, branchID int64, expiresAt time.Time) (domain.Session, error) {
	r.db.WithContext(ctx).Where("login = ?", login).Delete(&domain.Session{})

	sess := domain.Session{
		Token:     token,
		Login:     login,
		LastName:  lastName,
		FirstName: firstName,
		Email:     email,
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
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
