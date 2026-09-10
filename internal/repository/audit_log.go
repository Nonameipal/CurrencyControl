package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"CurrencyControl/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditLogFilter struct {
	UserLogin string
	Action    string
	Entity    string
	BranchID  *int64
	FromDate  *time.Time
	ToDate    *time.Time
	Limit     int
	Offset    int
}

type AuditLogRepository interface {
	Create(ctx context.Context, log domain.AuditLog) error
	List(ctx context.Context, filter AuditLogFilter) ([]domain.AuditLog, int64, error)
}

type auditLogRepo struct {
	db *pgxpool.Pool
}

func NewAuditLogRepository(db *pgxpool.Pool) AuditLogRepository {
	return &auditLogRepo{db: db}
}

func (r *auditLogRepo) Create(ctx context.Context, log domain.AuditLog) error {
	query := `
		INSERT INTO audit_logs (user_login, role, branch_id, action, entity, entity_id, details, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())`

	_, err := r.db.Exec(ctx, query,
		log.UserLogin,
		log.Role,
		log.BranchID,
		log.Action,
		log.Entity,
		log.EntityID,
		log.Details,
		log.IPAddress,
	)
	return err
}

func (r *auditLogRepo) List(ctx context.Context, filter AuditLogFilter) ([]domain.AuditLog, int64, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if strings.TrimSpace(filter.UserLogin) != "" {
		conditions = append(conditions, fmt.Sprintf("user_login ILIKE $%d", argIdx))
		args = append(args, "%"+strings.TrimSpace(filter.UserLogin)+"%")
		argIdx++
	}

	if strings.TrimSpace(filter.Action) != "" {
		conditions = append(conditions, fmt.Sprintf("action = $%d", argIdx))
		args = append(args, strings.TrimSpace(filter.Action))
		argIdx++
	}

	if strings.TrimSpace(filter.Entity) != "" {
		conditions = append(conditions, fmt.Sprintf("entity = $%d", argIdx))
		args = append(args, strings.TrimSpace(filter.Entity))
		argIdx++
	}

	if filter.BranchID != nil && *filter.BranchID > 0 {
		conditions = append(conditions, fmt.Sprintf("branch_id = $%d", argIdx))
		args = append(args, *filter.BranchID)
		argIdx++
	}

	if filter.FromDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *filter.FromDate)
		argIdx++
	}

	if filter.ToDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *filter.ToDate)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs %s", whereClause)
	var total int64
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	dataQuery := fmt.Sprintf(`
		SELECT id, user_login, role, branch_id, action, entity, entity_id, COALESCE(details, ''), COALESCE(ip_address, ''), created_at
		FROM audit_logs
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []domain.AuditLog
	for rows.Next() {
		var l domain.AuditLog
		if err := rows.Scan(
			&l.ID,
			&l.UserLogin,
			&l.Role,
			&l.BranchID,
			&l.Action,
			&l.Entity,
			&l.EntityID,
			&l.Details,
			&l.IPAddress,
			&l.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}

	return logs, total, nil
}
