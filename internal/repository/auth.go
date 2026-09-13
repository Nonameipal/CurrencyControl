package repository

import (
	"context"
	"fmt"
	"time"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"

	"github.com/jackc/pgx/v5/pgxpool"
)

type authRepo struct{ db *pgxpool.Pool }

func NewAuthRepository(db *pgxpool.Pool) ports.AuthRepository {
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

func (r *authRepo) ApproveRequest(ctx context.Context, requestID int64) (domain.User, error) {
	var req domain.AccessRequest
	err := r.db.QueryRow(ctx,
		`UPDATE access_requests SET status='approved', reviewed_at=NOW()
		 WHERE id=$1 AND status='pending'
		 RETURNING id, login, branch_id, role`,
		requestID,
	).Scan(&req.ID, &req.Login, &req.BranchID, &req.Role)
	if err != nil {
		return domain.User{}, fmt.Errorf("запрос не найден или уже обработан")
	}

	var user domain.User
	err = r.db.QueryRow(ctx,
		`INSERT INTO users (login, role, branch_id) VALUES ($1, $2, $3)
		 ON CONFLICT (login) DO UPDATE SET role=$2, branch_id=$3
		 RETURNING id, login, role, branch_id, created_at`,
		req.Login, req.Role, req.BranchID,
	).Scan(&user.ID, &user.Login, &user.Role, &user.BranchID, &user.CreatedAt)
	if err != nil {
		return domain.User{}, err
	}

	return user, nil
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

func (r *authRepo) SaveSession(ctx context.Context, token, login, role string, branchID int64, expiresAt time.Time) (domain.Session, error) {
	r.db.Exec(ctx, `DELETE FROM sessions WHERE login=$1`, login)

	var sess domain.Session
	err := r.db.QueryRow(ctx,
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

func (r *authRepo) DeleteSessionsByLogin(ctx context.Context, login string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM sessions WHERE login=$1`, login)
	return err
}

func (r *authRepo) SetAccessRequestSessionToken(ctx context.Context, requestID int64, token string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE access_requests SET session_token=$1 WHERE id=$2`,
		token, requestID,
	)
	return err
}
