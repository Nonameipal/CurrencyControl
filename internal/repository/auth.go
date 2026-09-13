package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"time"

	"CurrencyControl/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthRepository interface {
	GetUserByLogin(ctx context.Context, login string) (*domain.User, error)


	CreateAccessRequest(ctx context.Context, login string, branchID int64, role string) (domain.AccessRequest, error)


	GetRequestByID(ctx context.Context, requestID int64) (*domain.AccessRequest, error)

	GetPendingRequests(ctx context.Context) ([]domain.AccessRequest, error)

	ApproveRequest(ctx context.Context, requestID int64) (domain.Session, error)


	RejectRequest(ctx context.Context, requestID int64) error

	CreateSession(ctx context.Context, login, role string, branchID int64) (domain.Session, error)
	GetSessionByToken(ctx context.Context, token string) (*domain.Session, error)
	DeleteSession(ctx context.Context, token string) error
}

type authRepo struct{ db *pgxpool.Pool }

func NewAuthRepository(db *pgxpool.Pool) AuthRepository {
	return &authRepo{db: db}
}

func (r *authRepo) GetUserByLogin(ctx context.Context, login string) (*domain.User, error) {
	var u domain.User
	err := r.db.QueryRow(ctx,
		`SELECT id, login, role, branch_id, created_at FROM users WHERE login = $1`, login,
	).Scan(&u.ID, &u.Login, &u.Role, &u.BranchID, &u.CreatedAt)
	if err != nil {
		return nil, nil 
	}
	return &u, nil
}

func (r *authRepo) CreateAccessRequest(ctx context.Context, login string, branchID int64, role string) (domain.AccessRequest, error) {
	var cnt int
	r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM access_requests WHERE login=$1 AND status='pending'`, login,
	).Scan(&cnt)
	if cnt > 0 {
		return domain.AccessRequest{}, fmt.Errorf("запрос уже ожидает подтверждения")
	}

	var req domain.AccessRequest
	err := r.db.QueryRow(ctx,
		`WITH ins AS (
			INSERT INTO access_requests (login, branch_id, role, status)
			VALUES ($1, $2, $3, 'pending')
			RETURNING id, login, branch_id, role, status, session_token, created_at, reviewed_at
		)
		SELECT ins.id, ins.login, ins.branch_id, COALESCE(b.name, ''), ins.role, ins.status, ins.session_token, ins.created_at, ins.reviewed_at
		FROM ins
		LEFT JOIN branches b ON b.id = ins.branch_id`,
		login, branchID, role,
	).Scan(&req.ID, &req.Login, &req.BranchID, &req.BranchName, &req.Role, &req.Status, &req.SessionToken, &req.CreatedAt, &req.ReviewedAt)
	return req, err
}

func (r *authRepo) GetRequestByID(ctx context.Context, requestID int64) (*domain.AccessRequest, error) {
	var req domain.AccessRequest
	err := r.db.QueryRow(ctx,
		`SELECT ar.id, ar.login, ar.branch_id, COALESCE(b.name, ''), ar.role, ar.status, ar.session_token, ar.created_at, ar.reviewed_at
		 FROM access_requests ar
		 LEFT JOIN branches b ON b.id = ar.branch_id
		 WHERE ar.id=$1`,
		requestID,
	).Scan(&req.ID, &req.Login, &req.BranchID, &req.BranchName, &req.Role, &req.Status, &req.SessionToken, &req.CreatedAt, &req.ReviewedAt)
	if err != nil {
		return nil, fmt.Errorf("запрос не найден")
	}
	return &req, nil
}

func (r *authRepo) GetPendingRequests(ctx context.Context) ([]domain.AccessRequest, error) {
	rows, err := r.db.Query(ctx,
		`SELECT ar.id, ar.login, ar.branch_id, COALESCE(b.name, ''), ar.role, ar.status, ar.session_token, ar.created_at, ar.reviewed_at
		 FROM access_requests ar
		 LEFT JOIN branches b ON b.id = ar.branch_id
		 WHERE ar.status = 'pending' ORDER BY ar.created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.AccessRequest
	for rows.Next() {
		var req domain.AccessRequest
		if err := rows.Scan(&req.ID, &req.Login, &req.BranchID, &req.BranchName, &req.Role, &req.Status, &req.SessionToken, &req.CreatedAt, &req.ReviewedAt); err != nil {
			return nil, err
		}
		result = append(result, req)
	}
	if result == nil {
		result = []domain.AccessRequest{}
	}
	return result, nil
}

func (r *authRepo) ApproveRequest(ctx context.Context, requestID int64) (domain.Session, error) {
	var req domain.AccessRequest
	err := r.db.QueryRow(ctx,
		`UPDATE access_requests SET status='approved', reviewed_at=NOW()
		 WHERE id=$1 AND status='pending'
		 RETURNING id, login, branch_id, role`,
		requestID,
	).Scan(&req.ID, &req.Login, &req.BranchID, &req.Role)
	if err != nil {
		return domain.Session{}, fmt.Errorf("запрос не найден или уже обработан")
	}

	r.db.Exec(ctx,
		`INSERT INTO users (login, role, branch_id) VALUES ($1, $2, $3)
		 ON CONFLICT (login) DO UPDATE SET role=$2, branch_id=$3`,
		req.Login, req.Role, req.BranchID,
	)

	sess, err := r.CreateSession(ctx, req.Login, req.Role, req.BranchID)
	if err != nil {
		return domain.Session{}, err
	}

	r.db.Exec(ctx,
		`UPDATE access_requests SET session_token=$1 WHERE id=$2`,
		sess.Token, requestID,
	)

	return sess, nil
}

func (r *authRepo) RejectRequest(ctx context.Context, requestID int64) error {
	result, err := r.db.Exec(ctx,
		`UPDATE access_requests SET status='rejected', reviewed_at=NOW()
		 WHERE id=$1 AND status='pending'`,
		requestID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("запрос не найден или уже обработан")
	}
	return nil
}

func (r *authRepo) CreateSession(ctx context.Context, login, role string, branchID int64) (domain.Session, error) {
	token, err := generateToken()
	if err != nil {
		return domain.Session{}, err
	}

	ttlHours := 8
	if v, err := strconv.Atoi(os.Getenv("SESSION_TTL_HOURS")); err == nil && v > 0 {
		ttlHours = v
	}
	expiresAt := time.Now().Add(time.Duration(ttlHours) * time.Hour)

	r.db.Exec(ctx, `DELETE FROM sessions WHERE login=$1`, login)

	var sess domain.Session
	err = r.db.QueryRow(ctx,
		`INSERT INTO sessions (token, login, role, branch_id, expires_at)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, token, login, role, branch_id, expires_at, created_at`,
		token, login, role, branchID, expiresAt,
	).Scan(&sess.ID, &sess.Token, &sess.Login, &sess.Role, &sess.BranchID, &sess.ExpiresAt, &sess.CreatedAt)
	return sess, err
}

func (r *authRepo) GetSessionByToken(ctx context.Context, token string) (*domain.Session, error) {
	var sess domain.Session
	err := r.db.QueryRow(ctx,
		`SELECT id, token, login, role, branch_id, expires_at, created_at
		 FROM sessions WHERE token=$1 AND expires_at > NOW()`,
		token,
	).Scan(&sess.ID, &sess.Token, &sess.Login, &sess.Role, &sess.BranchID, &sess.ExpiresAt, &sess.CreatedAt)
	if err != nil {
		return nil, nil
	}
	return &sess, nil
}

func (r *authRepo) DeleteSession(ctx context.Context, token string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM sessions WHERE token=$1`, token)
	return err
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
