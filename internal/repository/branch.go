package repository

import (
	"context"
	"fmt"

	"CurrencyControl/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type BranchRepository interface {
	Create(ctx context.Context, b domain.Branch) (domain.Branch, error)
	GetByID(ctx context.Context, id int) (*domain.Branch, error)
	GetAll(ctx context.Context) ([]domain.Branch, error)
	Update(ctx context.Context, id int, name string) (*domain.Branch, error)
	SoftDelete(ctx context.Context, id int) error
	CheckExists(ctx context.Context, id int) (bool, error)
	CheckHasRelations(ctx context.Context, id int) (bool, error)
}

func NewBranchRepository(db *pgxpool.Pool) BranchRepository {
	return &branchRepo{db: db}
}

type branchRepo struct {
	db *pgxpool.Pool
}

func (r *branchRepo) Create(ctx context.Context, b domain.Branch) (domain.Branch, error) {
	query := `
		INSERT INTO branches (id, name, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id, name, COALESCE(created_by, ''), created_at, updated_at`

	var res domain.Branch
	err := r.db.QueryRow(ctx, query, b.ID, b.Name, b.CreatedBy).Scan(
		&res.ID,
		&res.Name,
		&res.CreatedBy,
		&res.CreatedAt,
		&res.UpdatedAt,
	)
	if err != nil {
		return domain.Branch{}, err
	}
	return res, nil
}

func (r *branchRepo) GetByID(ctx context.Context, id int) (*domain.Branch, error) {
	query := `
		SELECT id, name, COALESCE(created_by, ''), created_at, updated_at
		FROM branches
		WHERE id = $1 AND deleted_at IS NULL`

	var res domain.Branch
	err := r.db.QueryRow(ctx, query, id).Scan(
		&res.ID,
		&res.Name,
		&res.CreatedBy,
		&res.CreatedAt,
		&res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *branchRepo) GetAll(ctx context.Context) ([]domain.Branch, error) {
	query := `
		SELECT id, name, COALESCE(created_by, ''), created_at, updated_at
		FROM branches
		WHERE deleted_at IS NULL
		ORDER BY id ASC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var branches []domain.Branch
	for rows.Next() {
		var b domain.Branch
		if err := rows.Scan(&b.ID, &b.Name, &b.CreatedBy, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		branches = append(branches, b)
	}
	if branches == nil {
		branches = []domain.Branch{}
	}
	return branches, nil
}

func (r *branchRepo) Update(ctx context.Context, id int, name string) (*domain.Branch, error) {
	query := `
		UPDATE branches
		SET name = $2, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, name, COALESCE(created_by, ''), created_at, updated_at`

	var res domain.Branch
	err := r.db.QueryRow(ctx, query, id, name).Scan(
		&res.ID,
		&res.Name,
		&res.CreatedBy,
		&res.CreatedAt,
		&res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *branchRepo) SoftDelete(ctx context.Context, id int) error {
	result, err := r.db.Exec(ctx, `UPDATE branches SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("филиал не найден или уже удален")
	}
	return nil
}

func (r *branchRepo) CheckExists(ctx context.Context, id int) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM branches WHERE id = $1 AND deleted_at IS NULL)`, id).Scan(&exists)
	return exists, err
}

func (r *branchRepo) CheckHasRelations(ctx context.Context, id int) (bool, error) {
	var count int
	_ = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE branch_id = $1`, id).Scan(&count)
	if count > 0 {
		return true, nil
	}
	_ = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM counterparties WHERE branch_id = $1 AND deleted_at IS NULL`, id).Scan(&count)
	if count > 0 {
		return true, nil
	}
	_ = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM access_requests WHERE branch_id = $1 AND status = 'pending'`, id).Scan(&count)
	if count > 0 {
		return true, nil
	}
	return false, nil
}



