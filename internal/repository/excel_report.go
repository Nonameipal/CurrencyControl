package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"CurrencyControl/internal/delivery/dto"
)

func (r *reportRepo) GetGTDExcelData(ctx context.Context, filter dto.ExcelReportFilter) ([]dto.GTDExcelRow, string, error) {
	clientName := r.getClientName(ctx, filter.ClientID)

	q := r.db.WithContext(ctx).Table("gtd g").
		Select(`
			COALESCE(g.gtd_number, '') AS number,
			COALESCE(g.submission_date, g.gtd_date) AS gtd_date,
			g.gtd_amount AS amount,
			COALESCE(g.gtd_currency, '') AS currency,
			COALESCE(g.hs_code, '') AS hs_code,
			COALESCE(cp.name, '') AS sender_name,
			COALESCE(g.destination_country, '') AS country
		`).
		Joins("JOIN contracts c ON c.id = g.contract_id").
		Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
		Where("g.deleted_at IS NULL")

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
		q = q.Where("g.gtd_currency = ?", strings.ToUpper(strings.TrimSpace(filter.Currency)))
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
		result = append(result, dto.GTDExcelRow{
			Number:     raw.Number,
			Date:       formatDate(raw.GTDDate),
			Amount:     raw.Amount,
			Currency:   raw.Currency,
			HSCode:     raw.HSCode,
			SenderName: raw.SenderName,
			Country:    raw.Country,
		})
	}
	return result, clientName, nil
}

func (r *reportRepo) GetAAExcelData(ctx context.Context, filter dto.ExcelReportFilter) ([]dto.AAExcelRow, string, error) {
	clientName := r.getClientName(ctx, filter.ClientID)

	q := r.db.WithContext(ctx).Table("additional_agreements aa").
		Select(`
			COALESCE(aa.agreement_number, '') AS number,
			CASE 
				WHEN aa.doc_type = 'specification' THEN 'Спецификация'
				WHEN aa.doc_type = 'appendix' THEN 'Приложение'
				ELSE 'Доп. соглашение'
			END AS doc_type,
			aa.agreement_date,
			COALESCE(c.contract_number, '') AS contract_num,
			COALESCE(aa.foreign_amount, 0) AS amount,
			COALESCE(aa.currency, '') AS currency,
			aa.delivery_date,
			aa.return_date,
			aa.extend_date_to,
			COALESCE(aa.subject, c.subject, '') AS subject
		`).
		Joins("JOIN contracts c ON c.id = aa.contract_id").
		Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
		Where("aa.deleted_at IS NULL")

	if filter.ClientID != nil && *filter.ClientID > 0 {
		q = q.Where("c.client_id = ?", *filter.ClientID)
	}
	if filter.BranchID != nil && *filter.BranchID > 0 {
		q = q.Where("c.branch_id = ?", *filter.BranchID)
	}
	if filter.FromDate != nil {
		q = q.Where("aa.agreement_date >= ?", *filter.FromDate)
	}
	if filter.ToDate != nil {
		q = q.Where("aa.agreement_date <= ?", *filter.ToDate)
	}

	type rawAAExcelRow struct {
		Number        string
		DocType       string
		AgreementDate *time.Time `gorm:"column:agreement_date"`
		ContractNum   string
		Amount        float64
		Currency      string
		DeliveryDate  *time.Time `gorm:"column:delivery_date"`
		ReturnDate    *time.Time `gorm:"column:return_date"`
		ExtendDateTo  *time.Time `gorm:"column:extend_date_to"`
		Subject       string
	}

	var rows []rawAAExcelRow
	if err := q.Order("aa.agreement_date DESC, aa.id DESC").Scan(&rows).Error; err != nil {
		return nil, clientName, err
	}

	result := make([]dto.AAExcelRow, 0, len(rows))
	for _, raw := range rows {
		result = append(result, dto.AAExcelRow{
			Number:       raw.Number,
			DocType:      raw.DocType,
			Date:         formatDate(raw.AgreementDate),
			ContractNum:  raw.ContractNum,
			Amount:       raw.Amount,
			Currency:     raw.Currency,
			DeliveryDate: formatDate(raw.DeliveryDate),
			ReturnDate:   formatDate(raw.ReturnDate),
			ExtendDateTo: formatDate(raw.ExtendDateTo),
			Subject:      raw.Subject,
		})
	}
	return result, clientName, nil
}

func (r *reportRepo) GetClientsExcelData(ctx context.Context, filter dto.ExcelReportFilter) ([]dto.ClientExcelRow, error) {
	q := r.db.WithContext(ctx).Table("counterparties cp").
		Select(`
			cp.id,
			cp.name,
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
	if err := q.Group("cp.id, cp.name, cp.inn, b.name, cp.created_at").
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
			i.amount,
			i.currency,
			COALESCE(i.hs_code, '') AS hs_code,
			COALESCE(c.subject, '') AS payment_purpose
		`).
		Joins("JOIN contracts c ON c.id = i.contract_id").
		Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
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
		Number         string
		InvoiceDate    *time.Time `gorm:"column:invoice_date"`
		Amount         float64
		Currency       string
		HSCode         string
		PaymentPurpose string
	}

	var rows []rawInvoiceExcelRow
	if err := q.Order("i.invoice_date DESC, i.id DESC").Scan(&rows).Error; err != nil {
		return nil, clientName, err
	}

	result := make([]dto.InvoiceExcelRow, 0, len(rows))
	for _, raw := range rows {
		result = append(result, dto.InvoiceExcelRow{
			Number:         raw.Number,
			Date:           formatDate(raw.InvoiceDate),
			Amount:         raw.Amount,
			Currency:       raw.Currency,
			HSCode:         raw.HSCode,
			PaymentPurpose: raw.PaymentPurpose,
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
			c.return_date,
			c.delivery_date,
			COALESCE(c.extend_date_to, c.contract_end_date) AS contract_end_date,
			COALESCE(NULLIF(c.receiver_name, ''), cp.name, '') AS receiver_name,
			COALESCE(NULLIF(c.receiver_bank, ''), cp.inn, '') AS receiver_account,
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
		ReturnDate      *time.Time `gorm:"column:return_date"`
		DeliveryDate    *time.Time `gorm:"column:delivery_date"`
		ContractEndDate *time.Time `gorm:"column:contract_end_date"`
		ReceiverName    string
		ReceiverAccount string
		ReceiverCountry string
	}

	var rows []rawContractExcelRow
	if err := q.Order("c.contract_date DESC, c.id DESC").Scan(&rows).Error; err != nil {
		return nil, clientName, err
	}

	result := make([]dto.ContractExcelRow, 0, len(rows))
	for _, raw := range rows {
		result = append(result, dto.ContractExcelRow{
			Number:          raw.Number,
			Date:            formatDate(raw.ContractDate),
			Subject:         raw.Subject,
			Amount:          raw.Amount,
			Currency:        raw.Currency,
			ReturnDate:      formatDate(raw.ReturnDate),
			DeliveryDate:    formatDate(raw.DeliveryDate),
			ContractEndDate: formatDate(raw.ContractEndDate),
			ReceiverName:    raw.ReceiverName,
			ReceiverAccount: raw.ReceiverAccount,
			ReceiverCountry: raw.ReceiverCountry,
		})
	}
	return result, clientName, nil
}

func (r *reportRepo) GetPaymentOrdersExcelData(ctx context.Context, filter dto.ExcelReportFilter) ([]dto.PaymentOrderExcelRow, string, error) {
	clientName := r.getClientName(ctx, filter.ClientID)

	type rawRow struct {
		OperationDate      *time.Time `gorm:"column:operation_date"`
		PaymentOrderNumber string
		Amount             float64
		Currency           string
		Payer              string
		ReceiverName       string
		ReceiverBank       string
		PaymentPurpose     string
		ReceiverCountry    string
		ContractNumber     string
		InvoiceNumber      string
		ValueDate          *time.Time `gorm:"column:value_date"`
	}

	q := r.db.WithContext(ctx).Table("payment_orders po").
		Select(`
			po.operation_date,
			po.payment_order_number,
			po.amount,
			po.currency,
			po.payer,
			po.receiver_name,
			po.receiver_bank,
			po.payment_purpose,
			po.receiver_country,
			po.contract_number,
			po.invoice_number,
			po.value_date
		`).
		Joins("JOIN contracts c ON c.id = po.contract_id").
		Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
		Where("po.deleted_at IS NULL")

	if filter.ClientID != nil && *filter.ClientID > 0 {
		q = q.Where("c.client_id = ?", *filter.ClientID)
	}
	if filter.BranchID != nil && *filter.BranchID > 0 {
		q = q.Where("c.branch_id = ?", *filter.BranchID)
	}
	if filter.FromDate != nil {
		q = q.Where("po.operation_date >= ?", *filter.FromDate)
	}
	if filter.ToDate != nil {
		q = q.Where("po.operation_date <= ?", *filter.ToDate)
	}
	if strings.TrimSpace(filter.Currency) != "" {
		q = q.Where("po.currency = ?", strings.ToUpper(strings.TrimSpace(filter.Currency)))
	}

	q = q.Order("po.operation_date DESC, po.id DESC")

	var raw []rawRow
	if err := q.Scan(&raw).Error; err != nil {
		return nil, clientName, err
	}

	result := make([]dto.PaymentOrderExcelRow, 0, len(raw))
	for _, r := range raw {
		result = append(result, dto.PaymentOrderExcelRow{
			OperationDate:      formatDate(r.OperationDate),
			PaymentOrderNumber: r.PaymentOrderNumber,
			Amount:             r.Amount,
			Currency:           r.Currency,
			Payer:              r.Payer,
			ReceiverName:       r.ReceiverName,
			ReceiverBank:       r.ReceiverBank,
			PaymentPurpose:     r.PaymentPurpose,
			ReceiverCountry:    r.ReceiverCountry,
			ContractNumber:     r.ContractNumber,
			InvoiceNumber:      r.InvoiceNumber,
			ValueDate:          formatDate(r.ValueDate),
		})
	}
	return result, clientName, nil
}
