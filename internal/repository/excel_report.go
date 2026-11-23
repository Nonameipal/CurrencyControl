package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
)

func (r *reportRepo) GetGTDExcelData(ctx context.Context, filter dto.ExcelReportFilter) ([]dto.GTDExcelRow, string, error) {
	clientName := r.getClientName(ctx, filter.ClientID)
	q := r.db.WithContext(ctx).Table("gtd g").
		Select(`
			COALESCE(g.gtd_number, '') AS number,
			COALESCE(g.submission_date, g.gtd_date) AS gtd_date,
			g.gtd_amount AS amount,
			COALESCE(g.currency, '') AS currency,
			'' AS hs_code,
			COALESCE(cp.llc, '') AS sender_name,
			'' AS country
		`).
		Joins("JOIN contracts c ON c.id = g.contract_id").
		Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
		Where("g.deleted_at IS NULL AND g.approval_status = ?", domain.ApprovalStatusApproved)
	if filter.ClientID != nil && *filter.ClientID > 0 {
		q = q.Where("c.client_id = ?", *filter.ClientID)
	}
	if filter.BranchID != nil && *filter.BranchID > 0 {
		q = q.Where("c.branch_id = ?", *filter.BranchID)
	}
	if filter.FromDate != nil {
		q = q.Where("g.gtd_date >= ?", *filter.FromDate)
	}
	if filter.ToDate != nil {
		q = q.Where("g.gtd_date <= ?", *filter.ToDate)
	}
	if strings.TrimSpace(filter.Currency) != "" {
		q = q.Where("g.currency = ?", strings.ToUpper(strings.TrimSpace(filter.Currency)))
	}

	type rawGTDExcelRow struct {
		Number     string
		GTDDate    *time.Time `gorm:"column:gtd_date"`
		Amount     float64
		Currency   string
		HSCode     string
		SenderName string
		Country    string
	}

	var rows []rawGTDExcelRow
	if err := q.Order("g.gtd_date DESC, g.id DESC").Scan(&rows).Error; err != nil {
		return nil, clientName, err
	}

	result := make([]dto.GTDExcelRow, 0, len(rows))
	for _, raw := range rows {
		hsCode := raw.HSCode
		if strings.TrimSpace(hsCode) == "" {
			hsCode = domain.DefaultHSCode
		}
		result = append(result, dto.GTDExcelRow{
			Number:     raw.Number,
			Date:       formatDate(raw.GTDDate),
			Amount:     raw.Amount,
			Currency:   raw.Currency,
			HSCode:     hsCode,
			SenderName: raw.SenderName,
			Country:    raw.Country,
		})
	}
	return result, clientName, nil
}

func (r *reportRepo) GetClientsExcelData(ctx context.Context, filter dto.ExcelReportFilter) ([]dto.ClientExcelRow, error) {
	q := r.db.WithContext(ctx).Table("counterparties cp").
		Select(`
			cp.id,
			cp.llc,
			COALESCE(cp.inn, '') AS inn,
			COALESCE(b.name, '') AS branch_name,
			COUNT(c.id) AS contracts_count,
			COALESCE(SUM(c.total_amount), 0) AS total_amount,
			cp.created_at
		`).
		Joins("LEFT JOIN branches b ON b.id = cp.branch_id").
		Joins("LEFT JOIN contracts c ON c.client_id = cp.id AND c.deleted_at IS NULL").
		Where("cp.deleted_at IS NULL")

	if filter.BranchID != nil && *filter.BranchID > 0 {
		q = q.Where("cp.branch_id = ?", *filter.BranchID)
	}

	type rawClientExcelRow struct {
		ID             int64
		Name           string
		INN            string
		BranchName     string
		ContractsCount int
		TotalAmount    float64
		CreatedAt      time.Time
	}

	var rows []rawClientExcelRow
	if err := q.Group("cp.id, cp.llc, cp.inn, b.name, cp.created_at").
		Order("cp.created_at DESC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]dto.ClientExcelRow, 0, len(rows))
	for _, raw := range rows {
		result = append(result, dto.ClientExcelRow{
			ID:             raw.ID,
			Name:           raw.Name,
			INN:            raw.INN,
			BranchName:     raw.BranchName,
			ContractsCount: raw.ContractsCount,
			TotalAmount:    raw.TotalAmount,
			CreatedAt:      formatDate(&raw.CreatedAt),
		})
	}
	return result, nil
}

func formatDiffDays(days int) string {
	if days > 0 {
		return fmt.Sprintf("+%d", days)
	}
	return fmt.Sprintf("%d", days)
}

func (r *reportRepo) GetInvoicesExcelData(ctx context.Context, filter dto.ExcelReportFilter) ([]dto.InvoiceExcelRow, string, error) {
	clientName := r.getClientName(ctx, filter.ClientID)

	q := r.db.WithContext(ctx).Table("invoices i").
		Select(`
			COALESCE(i.invoice_number, '') AS number,
			i.invoice_date,
			COALESCE(i.currency, '') AS currency,
			i.amount
		`).
		Joins("JOIN contracts c ON c.id = i.contract_id").
		Where("i.deleted_at IS NULL")

	if filter.ClientID != nil && *filter.ClientID > 0 {
		q = q.Where("c.client_id = ?", *filter.ClientID)
	}
	if filter.BranchID != nil && *filter.BranchID > 0 {
		q = q.Where("c.branch_id = ?", *filter.BranchID)
	}
	if filter.FromDate != nil {
		q = q.Where("i.invoice_date >= ?", *filter.FromDate)
	}
	if filter.ToDate != nil {
		q = q.Where("i.invoice_date <= ?", *filter.ToDate)
	}
	if strings.TrimSpace(filter.Currency) != "" {
		q = q.Where("i.currency = ?", strings.ToUpper(strings.TrimSpace(filter.Currency)))
	}

	type rawInvoiceExcelRow struct {
		Number      string
		InvoiceDate *time.Time `gorm:"column:invoice_date"`
		Currency    string
		Amount      float64
	}

	var rows []rawInvoiceExcelRow
	if err := q.Order("i.invoice_date DESC, i.id DESC").Scan(&rows).Error; err != nil {
		return nil, clientName, err
	}

	result := make([]dto.InvoiceExcelRow, 0, len(rows))
	for i, raw := range rows {
		result = append(result, dto.InvoiceExcelRow{
			RowNum:   i + 1,
			Number:   raw.Number,
			Date:     formatDate(raw.InvoiceDate),
			Currency: raw.Currency,
			Amount:   raw.Amount,
		})
	}
	return result, clientName, nil
}

func (r *reportRepo) GetContractsExcelData(ctx context.Context, filter dto.ExcelReportFilter) ([]dto.ContractExcelRow, string, error) {
	clientName := r.getClientName(ctx, filter.ClientID)

	q := r.db.WithContext(ctx).Table("contracts c").
		Select(`
			c.contract_number AS number,
			c.contract_date,
			COALESCE(c.subject, '') AS subject,
			c.total_amount AS amount,
			c.contract_currency AS currency,
			c.return_days,
			c.delivery_date,
			COALESCE(c.extend_date_to, c.contract_end_date) AS contract_end_date,
			COALESCE(c.sender_name, '') AS sender_name,
			COALESCE(NULLIF(c.receiver_name, ''), cp.llc, '') AS receiver_name,
			COALESCE(c.receiver_country, '') AS receiver_country
		`).
		Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
		Where("c.deleted_at IS NULL")

	if filter.ClientID != nil && *filter.ClientID > 0 {
		q = q.Where("c.client_id = ?", *filter.ClientID)
	}
	if filter.BranchID != nil && *filter.BranchID > 0 {
		q = q.Where("c.branch_id = ?", *filter.BranchID)
	}
	if filter.FromDate != nil {
		q = q.Where("c.contract_date >= ?", *filter.FromDate)
	}
	if filter.ToDate != nil {
		q = q.Where("c.contract_date <= ?", *filter.ToDate)
	}
	if strings.TrimSpace(filter.Currency) != "" {
		q = q.Where("c.contract_currency = ?", strings.ToUpper(strings.TrimSpace(filter.Currency)))
	}

	type rawContractExcelRow struct {
		Number          string
		ContractDate    *time.Time `gorm:"column:contract_date"`
		Subject         string
		Amount          float64
		Currency        string
		ReturnDays      *int       `gorm:"column:return_days"`
		DeliveryDate    *time.Time `gorm:"column:delivery_date"`
		ContractEndDate *time.Time `gorm:"column:contract_end_date"`
		SenderName      string     `gorm:"column:sender_name"`
		ReceiverName    string     `gorm:"column:receiver_name"`
		ReceiverCountry string     `gorm:"column:receiver_country"`
	}

	var rows []rawContractExcelRow
	if err := q.Order("c.contract_date DESC, c.id DESC").Scan(&rows).Error; err != nil {
		return nil, clientName, err
	}

	result := make([]dto.ContractExcelRow, 0, len(rows))
	for i, raw := range rows {
		contractTerm := ""
		if raw.ContractDate != nil && raw.ContractEndDate != nil {
			contractTerm = strconv.Itoa(daysBetween(*raw.ContractDate, *raw.ContractEndDate))
		}

		result = append(result, dto.ContractExcelRow{
			RowNum:          i + 1,
			Number:          raw.Number,
			Date:            formatDate(raw.ContractDate),
			Subject:         raw.Subject,
			Amount:          raw.Amount,
			Currency:        raw.Currency,
			ContractTerm:    contractTerm,
			ReturnDays:      formatDays(raw.ReturnDays),
			DeliveryDate:    formatDate(raw.DeliveryDate),
			ContractEndDate: formatDate(raw.ContractEndDate),
			SenderName:      raw.SenderName,
			ReceiverName:    raw.ReceiverName,
			ReceiverCountry: raw.ReceiverCountry,
		})
	}
	return result, clientName, nil
}
