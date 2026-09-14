package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type gtdExtensionRepo struct {
	db *pgxpool.Pool
}

func NewGTDExtensionRepository(db *pgxpool.Pool) ports.GTDExtensionRepository {
	return &gtdExtensionRepo{db: db}
}

const gtdExtensionSelectCols = `
	id, gtd_id, contract_id, invoice_id, current_deadline,
	requested_deadline, document_path, status, created_by,
	reviewed_by, reviewed_at, COALESCE(comment, ''), created_at, updated_at
`

func scanGTDExtension(row pgx.Row) (*domain.GTDExtensionRequest, error) {
	var req domain.GTDExtensionRequest
	err := row.Scan(
		&req.ID,
		&req.GTDID,
		&req.ContractID,
		&req.InvoiceID,
		&req.CurrentDeadline,
		&req.RequestedDeadline,
		&req.DocumentPath,
		&req.Status,
		&req.CreatedBy,
		&req.ReviewedBy,
		&req.ReviewedAt,
		&req.Comment,
		&req.CreatedAt,
		&req.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *gtdExtensionRepo) CreateRequest(ctx context.Context, req domain.GTDExtensionRequest) (domain.GTDExtensionRequest, error) {
	var contractID int64
	var invoiceID int64
	var currentDeadline *time.Time

	err := r.db.QueryRow(ctx, `
		SELECT contract_id, invoice_id, delivery_deadline 
		FROM gtd 
		WHERE id = $1 AND deleted_at IS NULL`, req.GTDID,
	).Scan(&contractID, &invoiceID, &currentDeadline)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.GTDExtensionRequest{}, fmt.Errorf("ГТД с ID %d не найдена", req.GTDID)
		}
		return domain.GTDExtensionRequest{}, fmt.Errorf("ошибка проверки ГТД: %w", err)
	}

	// Защита от дублей: нельзя создать новую заявку, пока висит активная
	var pendingCount int
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) 
		FROM gtd_extension_requests 
		WHERE gtd_id = $1 AND status = 'pending' AND deleted_at IS NULL`, req.GTDID,
	).Scan(&pendingCount)
	if err != nil {
		return domain.GTDExtensionRequest{}, fmt.Errorf("ошибка проверки существующих заявок: %w", err)
	}
	if pendingCount > 0 {
		return domain.GTDExtensionRequest{}, fmt.Errorf("по данной ГТД уже есть активная заявка на увеличение срока, ожидающая рассмотрения валютным контролем")
	}

	req.ContractID = contractID
	req.InvoiceID = invoiceID
	req.CurrentDeadline = currentDeadline
	req.Status = domain.ExtensionStatusPending

	query := fmt.Sprintf(`
		INSERT INTO gtd_extension_requests (
			gtd_id, contract_id, invoice_id, current_deadline,
			requested_deadline, document_path, status, created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING %s`, gtdExtensionSelectCols)

	row := r.db.QueryRow(ctx, query,
		req.GTDID, req.ContractID, req.InvoiceID, req.CurrentDeadline,
		req.RequestedDeadline, req.DocumentPath, req.Status, req.CreatedBy,
	)

	created, err := scanGTDExtension(row)
	if err != nil {
		return domain.GTDExtensionRequest{}, fmt.Errorf("ошибка создания заявки на продление ГТД: %w", err)
	}
	return *created, nil
}

func (r *gtdExtensionRepo) GetByID(ctx context.Context, id int64) (*domain.GTDExtensionRequest, error) {
	query := fmt.Sprintf(`SELECT %s FROM gtd_extension_requests WHERE id = $1 AND deleted_at IS NULL`, gtdExtensionSelectCols)
	row := r.db.QueryRow(ctx, query, id)
	req, err := scanGTDExtension(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("заявка на продление не найдена")
		}
		return nil, err
	}
	return req, nil
}

func (r *gtdExtensionRepo) GetByGTDID(ctx context.Context, gtdID int64) ([]domain.GTDExtensionRequest, error) {
	query := fmt.Sprintf(`
		SELECT %s 
		FROM gtd_extension_requests 
		WHERE gtd_id = $1 AND deleted_at IS NULL 
		ORDER BY created_at DESC`, gtdExtensionSelectCols)

	rows, err := r.db.Query(ctx, query, gtdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.GTDExtensionRequest
	for rows.Next() {
		item, err := scanGTDExtension(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *item)
	}
	if list == nil {
		list = []domain.GTDExtensionRequest{}
	}
	return list, nil
}

func (r *gtdExtensionRepo) GetPendingRequests(ctx context.Context, branchID *int, page, pageSize int) ([]domain.GTDExtensionRequest, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	baseWhere := "r.status = 'pending' AND r.deleted_at IS NULL"
	args := []interface{}{}
	if branchID != nil && *branchID > 0 {
		args = append(args, *branchID)
		baseWhere += fmt.Sprintf(" AND c.branch_id = $%d", len(args))
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM gtd_extension_requests r
		JOIN contracts c ON c.id = r.contract_id
		WHERE %s`, baseWhere)

	var total int
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT r.id, r.gtd_id, r.contract_id, r.invoice_id, r.current_deadline,
		       r.requested_deadline, r.document_path, r.status, r.created_by,
		       r.reviewed_by, r.reviewed_at, COALESCE(r.comment, ''), r.created_at, r.updated_at
		FROM gtd_extension_requests r
		JOIN contracts c ON c.id = r.contract_id
		WHERE %s
		ORDER BY r.created_at ASC
		LIMIT %d OFFSET %d`, baseWhere, pageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []domain.GTDExtensionRequest
	for rows.Next() {
		item, err := scanGTDExtension(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, *item)
	}
	if list == nil {
		list = []domain.GTDExtensionRequest{}
	}
	return list, total, nil
}

func (r *gtdExtensionRepo) ReviewRequest(ctx context.Context, id int64, decision string, approvedDeadline *time.Time, comment, reviewer string) (*domain.GTDExtensionRequest, error) {
	var gtdID int64
	var requestedDeadline time.Time
	var currentStatus string

	err := r.db.QueryRow(ctx, `
		SELECT gtd_id, requested_deadline, status 
		FROM gtd_extension_requests 
		WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&gtdID, &requestedDeadline, &currentStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("заявка на продление не найдена")
		}
		return nil, err
	}

	if currentStatus != domain.ExtensionStatusPending {
		return nil, fmt.Errorf("заявка уже рассмотрена (текущий статус: %s)", currentStatus)
	}

	normDecision := strings.ToLower(strings.TrimSpace(decision))
	switch normDecision {
	case "approve", "approved", "одобрить", "одобрено":
		targetDeadline := requestedDeadline
		if approvedDeadline != nil && !approvedDeadline.IsZero() {
			targetDeadline = *approvedDeadline
		}

		// 1. Обновляем статус заявки на approved
		now := time.Now()
		updateReqQuery := `
			UPDATE gtd_extension_requests
			SET status = $1,
			    approved_deadline = $2,
			    reviewed_by = $3,
			    reviewed_at = $4,
			    comment = $5,
			    updated_at = NOW()
			WHERE id = $6`
		_, err = r.db.Exec(ctx, updateReqQuery, domain.ExtensionStatusApproved, targetDeadline, reviewer, now, comment, id)
		if err != nil {
			return nil, fmt.Errorf("ошибка утверждения заявки: %w", err)
		}

		// 2. Обновляем дедлайн ГТД и пересчитываем просрочку
		var docType string
		var subDate *time.Time
		err = r.db.QueryRow(ctx, `SELECT COALESCE(document_type, 'gtd'), submission_date FROM gtd WHERE id = $1 AND deleted_at IS NULL`, gtdID).Scan(&docType, &subDate)
		if err == nil {
			actualDate := now
			if subDate != nil && !subDate.IsZero() {
				actualDate = *subDate
			}
			diffDays, status, notice := domain.CalculateDeliveryComparison(docType, actualDate, &targetDeadline)
			noticeWithNotice := fmt.Sprintf("Срок продлен Валютным контролем до %s. %s", targetDeadline.Format("02.01.2006"), notice)

			updateGTDQuery := `
				UPDATE gtd
				SET delivery_deadline = $1,
				    days_difference = $2,
				    delivery_status = $3,
				    delivery_notice = $4,
				    updated_at = NOW()
				WHERE id = $5`
			_, _ = r.db.Exec(ctx, updateGTDQuery, targetDeadline, diffDays, status, noticeWithNotice, gtdID)
		}

		return r.GetByID(ctx, id)

	case "reject", "rejected", "отклонить", "отказ":
		if strings.TrimSpace(comment) == "" {
			return nil, fmt.Errorf("причина отказа обязательна для заполнения")
		}

		now := time.Now()
		updateReqQuery := `
			UPDATE gtd_extension_requests
			SET status = $1,
			    reviewed_by = $2,
			    reviewed_at = $3,
			    comment = $4,
			    updated_at = NOW()
			WHERE id = $5`
		_, err = r.db.Exec(ctx, updateReqQuery, domain.ExtensionStatusRejected, reviewer, now, comment, id)
		if err != nil {
			return nil, fmt.Errorf("ошибка отклонения заявки: %w", err)
		}

		return r.GetByID(ctx, id)

	default:
		return nil, fmt.Errorf("недопустимое решение: %s (допустимы: approve, reject)", decision)
	}
}
