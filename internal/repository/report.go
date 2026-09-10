package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"CurrencyControl/internal/delivery/dto"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportFilter struct {
	BranchID *int
	FromDate *time.Time
	ToDate   *time.Time
	Currency string
}

type ReportRepository interface {
	GetContractsReport(ctx context.Context, filter ReportFilter) (*dto.ContractsReportResponse, error)
}

type reportRepo struct {
	db *pgxpool.Pool
}

func NewReportRepository(db *pgxpool.Pool) ReportRepository {
	return &reportRepo{db: db}
}

func (r *reportRepo) GetContractsReport(ctx context.Context, filter ReportFilter) (*dto.ContractsReportResponse, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, "c.deleted_at IS NULL")

	if filter.BranchID != nil && *filter.BranchID > 0 {
		conditions = append(conditions, fmt.Sprintf("c.branch_id = $%d", argIdx))
		args = append(args, *filter.BranchID)
		argIdx++
	}

	if filter.FromDate != nil {
		conditions = append(conditions, fmt.Sprintf("c.contract_date >= $%d", argIdx))
		args = append(args, *filter.FromDate)
		argIdx++
	}

	if filter.ToDate != nil {
		conditions = append(conditions, fmt.Sprintf("c.contract_date <= $%d", argIdx))
		args = append(args, *filter.ToDate)
		argIdx++
	}

	if strings.TrimSpace(filter.Currency) != "" {
		conditions = append(conditions, fmt.Sprintf("c.contract_currency = $%d", argIdx))
		args = append(args, strings.ToUpper(strings.TrimSpace(filter.Currency)))
		argIdx++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT 
			c.id,
			c.branch_id,
			COALESCE(b.name, ''),
			c.contract_number,
			COALESCE(c.contract_name, ''),
			c.contract_date,
			c.delivery_date,
			c.contract_end_date,
			c.total_amount,
			c.remaining_amount,
			c.contract_currency,
			COALESCE(c.receiver_name, ''),
			COALESCE(c.receiver_country, ''),
			COALESCE(c.subject, ''),
			COALESCE((SELECT COUNT(*) FROM invoices i WHERE i.contract_id = c.id AND i.deleted_at IS NULL), 0) AS invoices_count,
			COALESCE((SELECT SUM(i.deduct_amount) FROM invoices i WHERE i.contract_id = c.id AND i.deleted_at IS NULL), 0) AS invoices_amount,
			COALESCE(c.created_by, '')
		FROM contracts c
		LEFT JOIN branches b ON b.id = c.branch_id
		%s
		ORDER BY c.contract_date DESC, c.id DESC`,
		whereClause,
	)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := &dto.ContractsReportResponse{
		TotalAmountByCurrency: make(map[string]float64),
		Contracts:             make([]dto.ContractReportItem, 0),
	}

	now := time.Now()

	for rows.Next() {
		var item dto.ContractReportItem
		var deliveryDate *time.Time
		var contractEndDate *time.Time

		if err := rows.Scan(
			&item.ID,
			&item.BranchID,
			&item.BranchName,
			&item.ContractNumber,
			&item.ContractName,
			&item.ContractDate,
			&deliveryDate,
			&contractEndDate,
			&item.TotalAmount,
			&item.RemainingAmount,
			&item.ContractCurrency,
			&item.ReceiverName,
			&item.ReceiverCountry,
			&item.Subject,
			&item.InvoicesCount,
			&item.InvoicesAmount,
			&item.CreatedBy,
		); err != nil {
			return nil, err
		}

		item.DeliveryDate = deliveryDate
		item.ContractEndDate = contractEndDate

		// Проверка на просрочку
		isOverdue := false
		if contractEndDate != nil && contractEndDate.Before(now) && item.RemainingAmount > 0 {
			isOverdue = true
		} else if deliveryDate != nil && deliveryDate.Before(now) && item.RemainingAmount > 0 {
			isOverdue = true
		}
		item.IsOverdue = isOverdue
		if isOverdue {
			res.OverdueCount++
		}

		res.TotalContracts++
		res.TotalAmountByCurrency[item.ContractCurrency] += item.TotalAmount
		res.TotalInvoicesCount += item.InvoicesCount

		res.Contracts = append(res.Contracts, item)
	}

	return res, nil
}
