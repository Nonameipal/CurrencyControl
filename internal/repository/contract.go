package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/errs"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ContractRepository interface {
	Create(ctx context.Context, c domain.Contract) (domain.Contract, error)
	GetByID(ctx context.Context, id int64) (domain.Contract, error)
	GetExpiringContracts(ctx context.Context, branchID int) ([]dto.NotificationResponse, error)
	GetByClientID(ctx context.Context, clientID int64) ([]domain.Contract, error)
	GetArchived(ctx context.Context, branchID int, page, pageSize int) ([]domain.Contract, int, error)
	GetArchivedByClientID(ctx context.Context, clientID int64) ([]domain.Contract, error)
	RestoreContract(ctx context.Context, id int64) error
	SearchDashboard(ctx context.Context, req dto.DashboardSearchRequest) ([]dto.DashboardSearchResult, error)
	CheckCountry(ctx context.Context, name string) (bool, error)
	CheckCurrency(ctx context.Context, code string) (bool, error)
	Update(ctx context.Context, id int64, c domain.Contract) (domain.Contract, error)
	SoftDelete(ctx context.Context, id int64) error
}

type contractRepo struct {
	db *pgxpool.Pool
}

func NewContractRepository(db *pgxpool.Pool) ContractRepository {
	return &contractRepo{db: db}
}

func (r *contractRepo) Create(ctx context.Context, c domain.Contract) (domain.Contract, error) {
	if c.RemainingAmount == 0 {
		c.RemainingAmount = c.TotalAmount
	}
	query := `
		INSERT INTO contracts
			(client_id, branch_id, contract_number, contract_date, delivery_date, return_term_days, return_date, total_amount, remaining_amount, contract_currency, subject, contract_end_date, document_path, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, client_id, branch_id, contract_number, contract_date, delivery_date, return_term_days, return_date, total_amount, remaining_amount, contract_currency, subject, contract_end_date, document_path, COALESCE(created_by, ''), created_at, updated_at`

	var result domain.Contract
	err := r.db.QueryRow(ctx, query,
		c.ClientID, c.BranchID, c.ContractNumber, c.ContractDate, c.DeliveryDate,
		c.ReturnTermDays, c.ReturnDate, c.TotalAmount, c.RemainingAmount, c.ContractCurrency,
		c.Subject, c.ContractEndDate, c.DocumentPath, c.CreatedBy,
	).Scan(
		&result.ID, &result.ClientID, &result.BranchID, &result.ContractNumber,
		&result.ContractDate, &result.DeliveryDate, &result.ReturnTermDays, &result.ReturnDate,
		&result.TotalAmount, &result.RemainingAmount, &result.ContractCurrency,
		&result.Subject, &result.ContractEndDate, &result.DocumentPath,
		&result.CreatedBy, &result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		return domain.Contract{}, err
	}
	return result, nil
}

func (r *contractRepo) GetByID(ctx context.Context, id int64) (domain.Contract, error) {
	query := `
		SELECT id, client_id, branch_id, contract_number, contract_date, delivery_date,
		       return_term_days, return_date, total_amount, remaining_amount,
		       contract_currency, subject, contract_end_date, document_path,
		       COALESCE(created_by, ''), created_at, updated_at
		FROM contracts
		WHERE id = $1 AND deleted_at IS NULL`

	var result domain.Contract
	err := r.db.QueryRow(ctx, query, id).Scan(
		&result.ID,
		&result.ClientID,
		&result.BranchID,
		&result.ContractNumber,
		&result.ContractDate,
		&result.DeliveryDate,
		&result.ReturnTermDays,
		&result.ReturnDate,
		&result.TotalAmount,
		&result.RemainingAmount,
		&result.ContractCurrency,
		&result.Subject,
		&result.ContractEndDate,
		&result.Status,
		&result.ArchivedAt,
		&result.ExtendDateTo,
		&result.DocumentPath,
		&result.CreatedBy,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Contract{}, errs.ErrContractNotFound
		}
		return domain.Contract{}, err
	}
	return result, nil
}

func (r *contractRepo) GetByClientID(ctx context.Context, clientID int64) ([]domain.Contract, error) {
	query := `
		SELECT id, client_id, branch_id, contract_number, contract_date, delivery_date,
		       return_term_days, return_date, total_amount, remaining_amount,
		       contract_currency, subject, contract_end_date, document_path,
		       COALESCE(created_by, ''), created_at, updated_at
		FROM contracts
		WHERE client_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contracts []domain.Contract
	for rows.Next() {
		var c domain.Contract
		if err := rows.Scan(
			&c.ID,
			&c.ClientID,
			&c.BranchID,
			&c.ContractNumber,
			&c.ContractDate,
			&c.DeliveryDate,
			&c.ReturnTermDays,
			&c.ReturnDate,
			&c.TotalAmount,
			&c.RemainingAmount,
			&c.ContractCurrency,
			&c.Subject,
			&c.ContractEndDate,
			&c.Status,
			&c.ArchivedAt,
			&c.ExtendDateTo,
			&c.DocumentPath,
			&c.CreatedBy,
			&c.CreatedAt,
			&c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		contracts = append(contracts, c)
	}
	if contracts == nil {
		contracts = []domain.Contract{}
	}
	return contracts, nil
}





func (r *contractRepo) SearchDashboard(ctx context.Context, req dto.DashboardSearchRequest) ([]dto.DashboardSearchResult, error) {
	query := `
		SELECT DISTINCT cp.id, COALESCE(cp.name, ''), COALESCE(cp.inn, '')
		FROM counterparties cp
		LEFT JOIN contracts c ON c.client_id = cp.id AND c.deleted_at IS NULL
		WHERE 1=1 AND cp.deleted_at IS NULL`

	var args []interface{}
	var conditions []string
	argId := 1

	if req.Amount > 0 {
		conditions = append(conditions, fmt.Sprintf(`c.total_amount = $%d`, argId))
		args = append(args, req.Amount)
		argId++
	}
	if req.INN != "" {
		conditions = append(conditions, fmt.Sprintf(`cp.inn ILIKE $%d`, argId))
		args = append(args, "%"+req.INN+"%")
		argId++
	}
	if req.CompanyName != "" {
		conditions = append(conditions, fmt.Sprintf(`cp.name ILIKE $%d`, argId))
		args = append(args, "%"+req.CompanyName+"%")
		argId++
	}
	if req.BranchID > 0 {
		conditions = append(conditions, fmt.Sprintf(`cp.branch_id = $%d`, argId))
		args = append(args, req.BranchID)
		argId++
	}

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY cp.id DESC LIMIT 100"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []dto.DashboardSearchResult
	counter := 1 

	for rows.Next() {
		var res dto.DashboardSearchResult
		if err := rows.Scan(&res.CompanyID, &res.CompanyName, &res.INN); err != nil {
			return nil, err
		}
		res.Number = fmt.Sprintf("№ %d", counter)
		counter++
		
		results = append(results, res)
	}
	if results == nil {
		results = make([]dto.DashboardSearchResult, 0)
	}
	return results, nil
}

func (r *contractRepo) CheckCountry(ctx context.Context, name string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM countries WHERE name_ru = $1)", name).Scan(&exists)
	return exists, err
}

func (r *contractRepo) CheckCurrency(ctx context.Context, code string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM currencies WHERE code = $1)", code).Scan(&exists)
	return exists, err
}
func (r *contractRepo) GetExpiringContracts(ctx context.Context, branchID int) ([]dto.NotificationResponse, error) {
	var result []dto.NotificationResponse
	now := time.Now().Truncate(24 * time.Hour)

	// 1. Будильник по времени предоставления ГТД (поставка товаров по инвойсам)
	gtdQuery := `
		SELECT 
			comp.id AS company_id,
			comp.name AS company_name,
			c.id AS contract_id,
			c.contract_number,
			i.id AS invoice_id,
			i.invoice_number,
			i.amount AS invoice_amount,
			COALESCE((SELECT SUM(g.closes_amount) FROM gtd g WHERE g.invoice_id = i.id AND g.deleted_at IS NULL), 0) AS closed_amount,
			i.currency,
			c.delivery_date
		FROM invoices i
		JOIN contracts c ON i.contract_id = c.id
		JOIN counterparties comp ON c.client_id = comp.id
		WHERE comp.branch_id = $1 
		  AND i.deleted_at IS NULL 
		  AND c.deleted_at IS NULL 
		  AND comp.deleted_at IS NULL
		  AND c.delivery_date IS NOT NULL
		  AND c.delivery_date <= CURRENT_DATE + INTERVAL '10 days'
		  AND COALESCE((SELECT SUM(g.closes_amount) FROM gtd g WHERE g.invoice_id = i.id AND g.deleted_at IS NULL), 0) < i.amount
		ORDER BY c.delivery_date ASC
	`
	gtdRows, err := r.db.Query(ctx, gtdQuery, branchID)
	if err == nil {
		defer gtdRows.Close()
		for gtdRows.Next() {
			var n dto.NotificationResponse
			var invID int64
			var deliveryDate time.Time
			if err := gtdRows.Scan(
				&n.CompanyID,
				&n.CompanyName,
				&n.ContractID,
				&n.ContractNumber,
				&invID,
				&n.InvoiceNumber,
				&n.InvoiceAmount,
				&n.ClosedAmount,
				&n.Currency,
				&deliveryDate,
			); err == nil {
				n.InvoiceID = &invID
				n.Type = "gtd_deadline"
				n.DeadlineDate = deliveryDate
				n.EffectiveEndDate = deliveryDate
				n.UnclosedAmount = n.InvoiceAmount - n.ClosedAmount
				daysLeft := int(deliveryDate.Sub(now).Hours() / 24)
				n.DaysLeft = daysLeft
				if daysLeft < 0 {
					n.Status = "overdue"
					n.Title = fmt.Sprintf("Просрочено предоставление ГТД по инвойсу №%s (просрочка %d дн.)", n.InvoiceNumber, -daysLeft)
				} else {
					n.Status = "approaching"
					n.Title = fmt.Sprintf("Истекает срок предоставления ГТД по инвойсу №%s (осталось %d дн.)", n.InvoiceNumber, daysLeft)
				}
				result = append(result, n)
			}
		}
	}

	// 2. Будильник по истечению срока действия договоров
	expiryQuery := `
		SELECT 
			comp.id AS company_id,
			comp.name AS company_name,
	    	c.id AS contract_id,
			c.contract_number,
			GREATEST(c.contract_end_date, COALESCE(MAX(aa.extend_date_to), c.contract_end_date)) AS effective_end_date
		FROM contracts c
		JOIN counterparties comp ON c.client_id = comp.id
		LEFT JOIN additional_agreements aa ON aa.contract_id = c.id AND aa.deleted_at IS NULL
		WHERE comp.branch_id = $1 AND c.deleted_at IS NULL AND comp.deleted_at IS NULL
        GROUP BY c.id, comp.id
		HAVING GREATEST(c.contract_end_date, COALESCE(MAX(aa.extend_date_to), c.contract_end_date)) <= CURRENT_DATE + INTERVAL '10 days'
		ORDER BY effective_end_date ASC
	`
	expiryRows, err := r.db.Query(ctx, expiryQuery, branchID)
	if err != nil {
		if len(result) > 0 {
			return result, nil
		}
		return nil, err
	}
	defer expiryRows.Close()

	for expiryRows.Next() {
		var n dto.NotificationResponse
		if err := expiryRows.Scan(&n.CompanyID, &n.CompanyName, &n.ContractID, &n.ContractNumber, &n.EffectiveEndDate); err != nil {
			return nil, err
		}
		n.Type = "contract_expiry"
		n.DeadlineDate = n.EffectiveEndDate
		daysLeft := int(n.EffectiveEndDate.Sub(now).Hours() / 24)
		n.DaysLeft = daysLeft
		if daysLeft < 0 {
			n.Status = "overdue"
			n.Title = fmt.Sprintf("Истек срок действия контракта №%s (просрочка %d дн.)", n.ContractNumber, -daysLeft)
		} else {
			n.Status = "approaching"
			n.Title = fmt.Sprintf("Истекает срок действия контракта №%s (осталось %d дн.)", n.ContractNumber, daysLeft)
		}
		result = append(result, n)
	}

	if result == nil {
		result = []dto.NotificationResponse{}
	}
	return result, nil
}

func (r *contractRepo) Update(ctx context.Context, id int64, c domain.Contract) (domain.Contract, error) {
	query := `
		UPDATE contracts SET
			contract_number = $2, contract_date = $3, delivery_date = $4,
			return_term_days = $5, return_date = $6,
			total_amount = $7, remaining_amount = $8, contract_currency = $9,
			subject = $10, contract_end_date = $11,
			document_path = $12, updated_at = NOW()
		WHERE id = $1
		RETURNING id, client_id, branch_id, contract_number, contract_date, delivery_date,
			return_term_days, return_date, total_amount, remaining_amount,
			contract_currency, subject, contract_end_date, document_path,
			COALESCE(created_by, ''), created_at, updated_at`
	var result domain.Contract
	err := r.db.QueryRow(ctx, query,
		id, c.ContractNumber, c.ContractDate, c.DeliveryDate,
		c.ReturnTermDays, c.ReturnDate,
		c.TotalAmount, c.RemainingAmount, c.ContractCurrency,
		c.Subject, c.ContractEndDate, c.DocumentPath,
	).Scan(
		&result.ID, &result.ClientID, &result.BranchID, &result.ContractNumber,
		&result.ContractDate, &result.DeliveryDate,
		&result.ReturnTermDays, &result.ReturnDate, &result.TotalAmount, &result.RemainingAmount,
		&result.ContractCurrency, &result.Subject,
		&result.ContractEndDate, &result.DocumentPath,
		&result.CreatedBy, &result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		return domain.Contract{}, err
	}
	return result, nil
}

func (r *contractRepo) SoftDelete(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, `UPDATE contracts SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	return err
}

// contractScanCols is the shared SELECT column list for archive queries
const contractSelectCols = `
	id, client_id, branch_id, contract_number, contract_date, delivery_date,
	return_term_days, return_date, total_amount, remaining_amount,
	contract_currency, subject, contract_end_date,
	COALESCE(status, 'active'), archived_at, COALESCE(extend_date_to, delivery_date::date),
	document_path, COALESCE(created_by, ''), created_at, updated_at`

func scanContract(rows interface {
	Scan(dest ...any) error
}, c *domain.Contract) error {
	return rows.Scan(
		&c.ID, &c.ClientID, &c.BranchID, &c.ContractNumber,
		&c.ContractDate, &c.DeliveryDate,
		&c.ReturnTermDays, &c.ReturnDate, &c.TotalAmount, &c.RemainingAmount,
		&c.ContractCurrency, &c.Subject, &c.ContractEndDate,
		&c.Status, &c.ArchivedAt, &c.ExtendDateTo,
		&c.DocumentPath, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
	)
}

// GetArchived returns paginated archived contracts for a branch.
// Returns (contracts, totalCount, error).
func (r *contractRepo) GetArchived(ctx context.Context, branchID int, page, pageSize int) ([]domain.Contract, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int
	_ = r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM contracts c
		JOIN counterparties cp ON cp.id = c.client_id
		WHERE cp.branch_id = $1 AND c.deleted_at IS NULL AND COALESCE(c.status, 'active') = 'archived'`,
		branchID,
	).Scan(&total)

	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT %s
		FROM contracts c
		JOIN counterparties cp ON cp.id = c.client_id
		WHERE cp.branch_id = $1 AND c.deleted_at IS NULL AND COALESCE(c.status, 'active') = 'archived'
		ORDER BY c.archived_at DESC
		LIMIT $2 OFFSET $3`, contractSelectCols),
		branchID, pageSize, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []domain.Contract
	for rows.Next() {
		var c domain.Contract
		if err := scanContract(rows, &c); err != nil {
			return nil, 0, err
		}
		result = append(result, c)
	}
	if result == nil {
		result = []domain.Contract{}
	}
	return result, total, nil
}

// GetArchivedByClientID returns all archived contracts for a company.
func (r *contractRepo) GetArchivedByClientID(ctx context.Context, clientID int64) ([]domain.Contract, error) {
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT %s
		FROM contracts
		WHERE client_id = $1 AND deleted_at IS NULL AND COALESCE(status, 'active') = 'archived'
		ORDER BY archived_at DESC`, contractSelectCols),
		clientID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Contract
	for rows.Next() {
		var c domain.Contract
		if err := scanContract(rows, &c); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	if result == nil {
		result = []domain.Contract{}
	}
	return result, nil
}

// RestoreContract sets a contract back to 'active' status.
func (r *contractRepo) RestoreContract(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE contracts SET status = 'active', archived_at = NULL, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
	return err
}
