package repository

import (
	"context"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"

	"github.com/jackc/pgx/v5/pgxpool"
)

type permissionRepo struct {
	db *pgxpool.Pool
}

func NewPermissionRepository(db *pgxpool.Pool) ports.PermissionRepository {
	return &permissionRepo{db: db}
}

func (r *permissionRepo) GetCurrencyControlPermissions(ctx context.Context) ([]domain.CurrencyControlPermission, error) {
	rows, err := r.db.Query(ctx, `
		SELECT login, can_edit, can_delete, granted_by, granted_at
		FROM currency_control_permissions
		ORDER BY granted_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []domain.CurrencyControlPermission
	for rows.Next() {
		var p domain.CurrencyControlPermission
		if err := rows.Scan(&p.Login, &p.CanEdit, &p.CanDelete, &p.GrantedBy, &p.GrantedAt); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	if perms == nil {
		perms = []domain.CurrencyControlPermission{}
	}
	return perms, nil
}

func (r *permissionRepo) CheckCurrencyControlPermission(ctx context.Context, login string, action string) (bool, error) {
	var canEdit, canDelete bool
	err := r.db.QueryRow(ctx, `
		SELECT can_edit, can_delete
		FROM currency_control_permissions
		WHERE login = $1
	`, login).Scan(&canEdit, &canDelete)
	if err == nil {
		if action == "edit" {
			return canEdit, nil
		}
		if action == "delete" {
			return canDelete, nil
		}
		return false, nil
	}

	err = r.db.QueryRow(ctx, `
		SELECT can_edit, can_delete
		FROM currency_control_permissions
		WHERE login = '*'
	`).Scan(&canEdit, &canDelete)
	if err == nil {
		if action == "edit" {
			return canEdit, nil
		}
		if action == "delete" {
			return canDelete, nil
		}
		return false, nil
	}

	return false, nil
}

func (r *permissionRepo) GrantCurrencyControlPermission(ctx context.Context, perm domain.CurrencyControlPermission) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO currency_control_permissions (login, can_edit, can_delete, granted_by, granted_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (login) DO UPDATE
		SET can_edit = $2, can_delete = $3, granted_by = $4, granted_at = NOW()
	`, perm.Login, perm.CanEdit, perm.CanDelete, perm.GrantedBy)
	return err
}

func (r *permissionRepo) RevokeCurrencyControlPermission(ctx context.Context, login string) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM currency_control_permissions
		WHERE login = $1
	`, login)
	return err
}
