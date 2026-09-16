package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/service/ports"

	"gorm.io/gorm"
)

type reportRepo struct {
	db *gorm.DB
}

func NewReportRepository(db *gorm.DB) ports.ReportRepository {
	return &reportRepo{db: db}
}

func formatDate(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format("02.01.2006")
}

func (r *reportRepo) getClientName(ctx context.Context, clientID *int64) string {
	if clientID == nil || *clientID <= 0 {
		return "Все клиенты"
	}
	type cpInfo struct {
		Name string
		INN  string
	}
	var cp cpInfo
	_ = r.db.WithContext(ctx).Table("counterparties").
		Select("COALESCE(name, '') as name, COALESCE(inn, '') as inn").
		Where("id = ?", *clientID).
		Scan(&cp).Error
	if cp.Name == "" && cp.INN == "" {
		return fmt.Sprintf("Клиент №%d", *clientID)
	}
	if cp.INN != "" {
		if cp.Name != "" {
			return fmt.Sprintf("%s (ИНН: %s)", cp.Name, cp.INN)
		}
		return fmt.Sprintf("ИНН: %s", cp.INN)
	}
	return cp.Name
}

func (r *reportRepo) GetContractsReport(ctx context.Context, filter ports.ReportFilter) (*dto.ContractsReportResponse, error) {
	q := r.db.WithContext(ctx).Table("contracts c").
		Select(`
			c.id,
			c.branch_id,
			COALESCE(b.name, '') AS branch_name,
			c.contract_number,
			c.contract_date,
			c.delivery_date,
			c.contract_end_date,
			c.total_amount,
			c.remaining_amount,
			c.contract_currency,
			COALESCE(c.subject, '') AS subject,
			COALESCE((SELECT COUNT(*) FROM invoices i WHERE i.contract_id = c.id AND i.deleted_at IS NULL), 0) AS invoices_count,
			COALESCE((SELECT SUM(i.deduct_amount) FROM invoices i WHERE i.contract_id = c.id AND i.deleted_at IS NULL), 0) AS invoices_amount,
			COALESCE(c.created_by, '') AS created_by
		`).
		Joins("LEFT JOIN branches b ON b.id = c.branch_id").
		Where("c.deleted_at IS NULL")

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

	type rawContractReportItem struct {
		ID               int64
		BranchID         *int
		BranchName       string
		ContractNumber   string
		ContractDate     time.Time
		DeliveryDate     *time.Time
		ContractEndDate  *time.Time
		TotalAmount      float64
		RemainingAmount  float64
		ContractCurrency string
		Subject          string
		InvoicesCount    int
		InvoicesAmount   float64
		CreatedBy        string
	}

	var rawItems []rawContractReportItem
	if err := q.Order("c.contract_date DESC, c.id DESC").Scan(&rawItems).Error; err != nil {
		return nil, err
	}

	res := &dto.ContractsReportResponse{
		TotalAmountByCurrency: make(map[string]float64),
		Contracts:             make([]dto.ContractReportItem, 0, len(rawItems)),
	}

	now := time.Now()

	for _, raw := range rawItems {
		isOverdue := false
		if raw.ContractEndDate != nil && raw.ContractEndDate.Before(now) && raw.RemainingAmount > 0 {
			isOverdue = true
		} else if raw.DeliveryDate != nil && raw.DeliveryDate.Before(now) && raw.RemainingAmount > 0 {
			isOverdue = true
		}
		if isOverdue {
			res.OverdueCount++
		}

		item := dto.ContractReportItem{
			ID:               raw.ID,
			BranchID:         raw.BranchID,
			BranchName:       raw.BranchName,
			ContractNumber:   raw.ContractNumber,
			ContractDate:     raw.ContractDate,
			DeliveryDate:     raw.DeliveryDate,
			ContractEndDate:  raw.ContractEndDate,
			TotalAmount:      raw.TotalAmount,
			RemainingAmount:  raw.RemainingAmount,
			ContractCurrency: raw.ContractCurrency,
			Subject:          raw.Subject,
			InvoicesCount:    raw.InvoicesCount,
			InvoicesAmount:   raw.InvoicesAmount,
			CreatedBy:        raw.CreatedBy,
			IsOverdue:        isOverdue,
		}

		res.TotalContracts++
		res.TotalAmountByCurrency[item.ContractCurrency] += item.TotalAmount
		res.TotalInvoicesCount += item.InvoicesCount
		res.Contracts = append(res.Contracts, item)
	}

	return res, nil
}

func (r *reportRepo) GetClientCurrencies(ctx context.Context, clientID int64) ([]string, error) {
	q1 := r.db.Table("contracts").
		Select("contract_currency AS currency").
		Where("client_id = ? AND deleted_at IS NULL", clientID)

	q2 := r.db.Table("additional_agreements aa").
		Select("aa.currency AS currency").
		Joins("JOIN contracts c ON c.id = aa.contract_id").
		Where("c.client_id = ? AND aa.deleted_at IS NULL", clientID)

	q3 := r.db.Table("invoices i").
		Select("i.currency AS currency").
		Joins("JOIN contracts c ON c.id = i.contract_id").
		Where("c.client_id = ? AND i.deleted_at IS NULL", clientID)

	var currencies []string
	unionQuery := r.db.Table("((?) UNION (?) UNION (?)) AS t", q1, q2, q3).
		Select("DISTINCT currency").
		Where("currency IS NOT NULL AND currency != ''").
		Order("currency")

	if err := unionQuery.Scan(&currencies).Error; err != nil {
		return nil, err
	}
	if currencies == nil {
		currencies = []string{}
	}
	return currencies, nil
}

func (r *reportRepo) GetClientConsolidatedReportData(ctx context.Context, clientID int64, filter dto.ExcelReportFilter) (*dto.ClientConsolidatedReportData, error) {
	clientName := r.getClientName(ctx, &clientID)

	q := r.db.WithContext(ctx).Table("contracts c").
		Select(`
			c.id,
			c.contract_number,
			c.contract_date,
			COALESCE(c.extend_date_to, c.contract_end_date) AS end_date,
			c.total_amount,
			c.contract_currency,
			COALESCE(c.subject, '') AS subject,
			COALESCE(cp.name, '') AS partner_name,
			COALESCE(c.receiver_country, '') AS country
		`).
		Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
		Where("c.deleted_at IS NULL AND c.client_id = ?", clientID)

	if strings.TrimSpace(filter.Currency) != "" {
		q = q.Where("c.contract_currency = ?", strings.ToUpper(strings.TrimSpace(filter.Currency)))
	}
	if filter.FromDate != nil {
		q = q.Where("c.contract_date >= ?", *filter.FromDate)
	}
	if filter.ToDate != nil {
		q = q.Where("c.contract_date <= ?", *filter.ToDate)
	}

	type rawContract struct {
		ID          int64
		Number      string     `gorm:"column:contract_number"`
		CDate       *time.Time `gorm:"column:contract_date"`
		EDate       *time.Time `gorm:"column:end_date"`
		TotalAmount float64
		Currency    string `gorm:"column:contract_currency"`
		Subject     string
		PartnerName string `gorm:"column:partner_name"`
		Country     string
	}

	var rawContracts []rawContract
	if err := q.Order("c.contract_date ASC, c.id ASC").Scan(&rawContracts).Error; err != nil {
		return nil, err
	}

	data := &dto.ClientConsolidatedReportData{
		ClientName: clientName,
	}

	type rawInvoiceWithGTD struct {
		ID        int64
		Number    string     `gorm:"column:invoice_number"`
		Date      *time.Time `gorm:"column:invoice_date"`
		Amount    float64
		GTDAmount float64 `gorm:"column:gtd_amount"`
		GTDNumber string  `gorm:"column:gtd_number"`
		DaysDiff  int     `gorm:"column:days_diff"`
	}

	type rawAA struct {
		ID       int64
		Number   string     `gorm:"column:agreement_number"`
		Date     *time.Time `gorm:"column:agreement_date"`
		EndDate  *time.Time `gorm:"column:extend_date_to"`
		Amount   float64    `gorm:"column:foreign_amount"`
		Currency string
		Subject  string
		Partner  string `gorm:"column:partner_name"`
		Country  string
	}

	for _, rc := range rawContracts {
		contractDto := dto.ClientConsolidatedContract{
			Number:         rc.Number,
			Date:           formatDate(rc.CDate),
			EndDate:        formatDate(rc.EDate),
			ForeignCompany: rc.PartnerName,
			Country:        rc.Country,
			TotalAmount:    rc.TotalAmount,
			Currency:       rc.Currency,
		}

		var rawInvs []rawInvoiceWithGTD
		_ = r.db.WithContext(ctx).Table("invoices i").
			Select(`
				i.id,
				COALESCE(i.invoice_number, '') AS invoice_number,
				i.invoice_date,
				i.amount,
				COALESCE((SELECT g.gtd_amount FROM gtd g WHERE (g.invoice_id = i.id OR (g.contract_id = i.contract_id AND g.additional_agreement_id IS NULL)) AND g.deleted_at IS NULL LIMIT 1), 0) AS gtd_amount,
				COALESCE((SELECT g.gtd_number FROM gtd g WHERE (g.invoice_id = i.id OR (g.contract_id = i.contract_id AND g.additional_agreement_id IS NULL)) AND g.deleted_at IS NULL LIMIT 1), '') AS gtd_number,
				COALESCE((SELECT g.days_difference FROM gtd g WHERE (g.invoice_id = i.id OR (g.contract_id = i.contract_id AND g.additional_agreement_id IS NULL)) AND g.deleted_at IS NULL LIMIT 1), 0) AS days_diff
			`).
			Where("i.contract_id = ? AND i.additional_agreement_id IS NULL AND i.deleted_at IS NULL", rc.ID).
			Order("i.invoice_date ASC, i.id ASC").
			Scan(&rawInvs).Error

		for _, inv := range rawInvs {
			diffAmount := inv.Amount - inv.GTDAmount
			diffDaysStr := formatDiffDays(inv.DaysDiff)
			contractDto.Transfers = append(contractDto.Transfers, dto.ClientConsolidatedTransfer{
				InvoiceDate:          formatDate(inv.Date),
				InvoiceAmount:        inv.Amount,
				GTDAmount:            inv.GTDAmount,
				GTDNumber:            inv.GTDNumber,
				DiffAmount:           diffAmount,
				ContractDeliveryTerm: 180,
				ActualDeliveryTerm:   180 - inv.DaysDiff,
				DiffDays:             diffDaysStr,
			})
		}

		var rawAAs []rawAA
		_ = r.db.WithContext(ctx).Table("additional_agreements aa").
			Select(`
				aa.id,
				COALESCE(aa.agreement_number, '') AS agreement_number,
				aa.agreement_date,
				aa.extend_date_to,
				COALESCE(aa.foreign_amount, 0) AS foreign_amount,
				COALESCE(aa.currency, '') AS currency,
				COALESCE(aa.subject, '') AS subject,
				COALESCE(cp.name, '') AS partner_name,
				COALESCE(c.receiver_country, '') AS country
			`).
			Joins("JOIN contracts c ON c.id = aa.contract_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where("aa.contract_id = ? AND aa.deleted_at IS NULL", rc.ID).
			Order("aa.agreement_date ASC, aa.id ASC").
			Scan(&rawAAs).Error

		for _, a := range rawAAs {
			aaDto := dto.ClientConsolidatedAA{
				Number:               a.Number,
				Date:                 formatDate(a.Date),
				EndDate:              formatDate(a.EndDate),
				ForeignCompany:       a.Partner,
				Country:              a.Country,
				TotalAmount:          a.Amount,
				Currency:             a.Currency,
				ParentContractNumber: rc.Number,
			}

			var aaInvs []rawInvoiceWithGTD
			_ = r.db.WithContext(ctx).Table("invoices i").
				Select(`
					i.id,
					COALESCE(i.invoice_number, '') AS invoice_number,
					i.invoice_date,
					i.amount,
					COALESCE((SELECT g.gtd_amount FROM gtd g WHERE (g.invoice_id = i.id OR g.additional_agreement_id = i.additional_agreement_id) AND g.deleted_at IS NULL LIMIT 1), 0) AS gtd_amount,
					COALESCE((SELECT g.gtd_number FROM gtd g WHERE (g.invoice_id = i.id OR g.additional_agreement_id = i.additional_agreement_id) AND g.deleted_at IS NULL LIMIT 1), '') AS gtd_number,
					COALESCE((SELECT g.days_difference FROM gtd g WHERE (g.invoice_id = i.id OR g.additional_agreement_id = i.additional_agreement_id) AND g.deleted_at IS NULL LIMIT 1), 0) AS days_diff
				`).
				Where("i.additional_agreement_id = ? AND i.deleted_at IS NULL", a.ID).
				Order("i.invoice_date ASC, i.id ASC").
				Scan(&aaInvs).Error

			for _, inv := range aaInvs {
				diffAmount := inv.Amount - inv.GTDAmount
				diffDaysStr := formatDiffDays(inv.DaysDiff)
				aaDto.Transfers = append(aaDto.Transfers, dto.ClientConsolidatedTransfer{
					InvoiceDate:          formatDate(inv.Date),
					InvoiceAmount:        inv.Amount,
					GTDAmount:            inv.GTDAmount,
					GTDNumber:            inv.GTDNumber,
					DiffAmount:           diffAmount,
					ContractDeliveryTerm: 60,
					ActualDeliveryTerm:   60 - inv.DaysDiff,
					DiffDays:             diffDaysStr,
				})
			}

			contractDto.AdditionalAgreements = append(contractDto.AdditionalAgreements, aaDto)
		}

		data.Contracts = append(data.Contracts, contractDto)
	}

	return data, nil
}

func (r *reportRepo) GetClientIDByINN(ctx context.Context, inn string) (int64, error) {
	var id int64
	err := r.db.WithContext(ctx).Table("counterparties").
		Select("id").
		Where("TRIM(inn) = ? AND deleted_at IS NULL", strings.TrimSpace(inn)).
		Order("id DESC").
		Limit(1).
		Scan(&id).Error
	if err != nil {
		return 0, err
	}
	return id, nil
}

