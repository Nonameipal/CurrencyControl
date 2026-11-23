package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
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

func formatDays(d *int) string {
	if d == nil {
		return ""
	}
	return strconv.Itoa(*d)
}

func daysBetween(start, end time.Time) int {
	y1, m1, d1 := start.Date()
	y2, m2, d2 := end.Date()
	t1 := time.Date(y1, m1, d1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(y2, m2, d2, 0, 0, 0, 0, time.UTC)
	days := int(t2.Sub(t1).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

func (r *reportRepo) getClientName(ctx context.Context, clientID *int64) string {
	if clientID == nil || *clientID <= 0 {
		return "Все клиенты"
	}
	type cpInfo struct {
		LLC string `gorm:"column:llc"`
		INN string
	}
	var cp cpInfo
	_ = r.db.WithContext(ctx).Table("counterparties").
		Select("COALESCE(llc, '') as llc, COALESCE(inn, '') as inn").
		Where("id = ?", *clientID).
		Scan(&cp).Error
	if strings.TrimSpace(cp.LLC) != "" {
		return strings.TrimSpace(cp.LLC)
	}
	if strings.TrimSpace(cp.INN) != "" {
		return fmt.Sprintf("ИНН: %s", strings.TrimSpace(cp.INN))
	}
	return fmt.Sprintf("Клиент №%d", *clientID)
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
			COALESCE(c.extend_date_to, c.contract_end_date) AS effective_end_date,
			c.return_days,
			c.total_amount,
			c.remaining_amount,
			c.contract_currency,
			COALESCE(c.subject, '') AS subject,
			COALESCE(c.sender_name, '') AS sender_name,
			COALESCE(NULLIF(c.receiver_name, ''), cp.llc, '') AS receiver_name,
			COALESCE(c.receiver_country, '') AS receiver_country,
			COALESCE((SELECT COUNT(*) FROM invoices i WHERE i.contract_id = c.id AND i.deleted_at IS NULL), 0) AS invoices_count,
			COALESCE((SELECT SUM(i.amount) FROM invoices i WHERE i.contract_id = c.id AND i.deleted_at IS NULL), 0) AS invoices_amount,
			COALESCE(c.created_by, '') AS created_by
		`).
		Joins("LEFT JOIN branches b ON b.id = c.branch_id").
		Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
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
		EffectiveEndDate *time.Time
		ReturnDays       *int
		TotalAmount      float64
		RemainingAmount  float64
		ContractCurrency string
		Subject          string
		SenderName       string
		ReceiverName     string
		ReceiverCountry  string
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
		effectiveEnd := raw.EffectiveEndDate
		if effectiveEnd == nil {
			effectiveEnd = raw.ContractEndDate
		}

		isOverdue := false
		if effectiveEnd != nil && effectiveEnd.Before(now) && raw.RemainingAmount > 0 {
			isOverdue = true
		} else if raw.DeliveryDate != nil && raw.DeliveryDate.Before(now) && raw.RemainingAmount > 0 {
			isOverdue = true
		}
		if isOverdue {
			res.OverdueCount++
		}

		var termDays int
		if effectiveEnd != nil && !raw.ContractDate.IsZero() {
			termDays = daysBetween(raw.ContractDate, *effectiveEnd)
		}

		item := dto.ContractReportItem{
			ID:               raw.ID,
			BranchID:         raw.BranchID,
			BranchName:       raw.BranchName,
			ContractNumber:   raw.ContractNumber,
			ContractDate:     raw.ContractDate,
			DeliveryDate:     raw.DeliveryDate,
			ContractEndDate:  raw.ContractEndDate,
			EffectiveEndDate: raw.EffectiveEndDate,
			ContractTermDays: termDays,
			ReturnDays:       raw.ReturnDays,
			TotalAmount:      raw.TotalAmount,
			RemainingAmount:  raw.RemainingAmount,
			ContractCurrency: raw.ContractCurrency,
			Subject:          raw.Subject,
			SenderName:       raw.SenderName,
			ReceiverName:     raw.ReceiverName,
			ReceiverCountry:  raw.ReceiverCountry,
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
	var cp struct {
		LLC        string `gorm:"column:llc"`
		ClientType string `gorm:"column:client_type"`
	}
	_ = r.db.WithContext(ctx).Table("counterparties").
		Select("COALESCE(llc, '') AS llc, COALESCE(client_type, '') AS client_type").
		Where("id = ?", clientID).
		Scan(&cp).Error

	name := cp.LLC
	if name == "" {
		name = fmt.Sprintf("№%d", clientID)
	}
	lowerType := strings.ToLower(cp.ClientType)
	prefix := "Шахси ҳуқуқӣ"
	if cp.ClientType == domain.ClientTypeIndividual || cp.ClientType == domain.ClientTypeSoleProprietor ||
		strings.Contains(lowerType, "физ") || strings.Contains(lowerType, "ип") || strings.Contains(lowerType, "предприниматель") {
		prefix = "Шахси воқеӣ"
	}
	data := &dto.ClientConsolidatedReportData{
		ClientName: fmt.Sprintf("%s %s", prefix, name),
		ClientType: cp.ClientType,
	}

	q := r.db.WithContext(ctx).Table("contracts c").
		Select(`
			c.id, c.contract_number, c.contract_date,
			COALESCE(c.extend_date_to, c.contract_end_date) AS end_date,
			c.original_amount, c.total_amount, c.contract_currency, COALESCE(c.subject, '') AS subject,
			COALESCE(NULLIF(c.receiver_name, ''), cp.llc, '') AS receiver_name,
			COALESCE(c.receiver_country, '') AS country,
			c.return_days
		`).
		Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
		Where("c.deleted_at IS NULL AND c.client_id = ?", clientID)

	if filter.Currency != "" {
		q = q.Where("c.contract_currency = ?", strings.ToUpper(strings.TrimSpace(filter.Currency)))
	}
	if filter.FromDate != nil {
		q = q.Where("c.contract_date >= ?", *filter.FromDate)
	}
	if filter.ToDate != nil {
		q = q.Where("c.contract_date <= ?", *filter.ToDate)
	}

	type rawContract struct {
		ID             int64
		Number         string     `gorm:"column:contract_number"`
		CDate          *time.Time `gorm:"column:contract_date"`
		EDate          *time.Time `gorm:"column:end_date"`
		OriginalAmount float64    `gorm:"column:original_amount"`
		TotalAmount    float64
		Currency       string `gorm:"column:contract_currency"`
		Subject        string
		ReceiverName   string `gorm:"column:receiver_name"`
		Country        string
		ReturnDays     *int `gorm:"column:return_days"`
	}
	var rawContracts []rawContract
	if err := q.Order("c.contract_date ASC, c.id ASC").Scan(&rawContracts).Error; err != nil {
		return nil, err
	}

	if len(rawContracts) == 0 {
		return data, nil
	}

	var contractIDs []int64
	for _, c := range rawContracts {
		contractIDs = append(contractIDs, c.ID)
	}

	type rawInvoiceWithGTD struct {
		ID         int64
		ContractID int64      `gorm:"column:contract_id"`
		Number     string     `gorm:"column:invoice_number"`
		Date       *time.Time `gorm:"column:invoice_date"`
		Amount     float64
		Currency   string  `gorm:"column:currency"`
		GTDAmount  float64 `gorm:"column:gtd_amount"`
		GTDNumber  string  `gorm:"column:gtd_number"`
		DaysDiff   int     `gorm:"column:days_diff"`
	}
	var rawInvs []rawInvoiceWithGTD
	r.db.WithContext(ctx).Table("invoices i").
		Select(`
			i.id, i.contract_id,
			COALESCE(i.invoice_number, '') AS invoice_number, i.invoice_date, i.amount,
			COALESCE(i.currency, '') AS currency,
			COALESCE((SELECT g.gtd_amount FROM gtd g WHERE g.invoice_id = i.id AND g.deleted_at IS NULL AND g.approval_status = 'approved' ORDER BY g.gtd_date DESC, g.id DESC LIMIT 1), 0) AS gtd_amount,
			COALESCE((SELECT g.gtd_number FROM gtd g WHERE g.invoice_id = i.id AND g.deleted_at IS NULL AND g.approval_status = 'approved' ORDER BY g.gtd_date DESC, g.id DESC LIMIT 1), '') AS gtd_number,
			COALESCE((SELECT g.days_difference FROM gtd g WHERE g.invoice_id = i.id AND g.deleted_at IS NULL AND g.approval_status = 'approved' ORDER BY g.gtd_date DESC, g.id DESC LIMIT 1), 0) AS days_diff
		`).
		Where("i.contract_id IN ? AND i.deleted_at IS NULL", contractIDs).
		Order("i.invoice_date ASC, i.id ASC").
		Scan(&rawInvs)

	type invKey struct {
		contractID int64
		currency   string
	}
	invByCurrencyKey := make(map[invKey][]rawInvoiceWithGTD)
	for _, inv := range rawInvs {
		k := invKey{inv.ContractID, strings.ToUpper(strings.TrimSpace(inv.Currency))}
		invByCurrencyKey[k] = append(invByCurrencyKey[k], inv)
	}

	buildTransfers := func(invs []rawInvoiceWithGTD, contractDeliveryTerm int) []dto.ClientConsolidatedTransfer {
		transfers := make([]dto.ClientConsolidatedTransfer, 0, len(invs))
		for _, inv := range invs {
			diffAmount := inv.Amount
			actualTerm := 0
			diffDaysStr := ""
			if inv.GTDNumber != "" {
				diffAmount = inv.Amount - inv.GTDAmount
				actualTerm = contractDeliveryTerm - inv.DaysDiff
				diffDaysStr = formatDiffDays(inv.DaysDiff)
			}
			transfers = append(transfers, dto.ClientConsolidatedTransfer{
				InvoiceDate:          formatDate(inv.Date),
				InvoiceAmount:        inv.Amount,
				GTDAmount:            inv.GTDAmount,
				GTDNumber:            inv.GTDNumber,
				DiffAmount:           diffAmount,
				ContractDeliveryTerm: contractDeliveryTerm,
				ActualDeliveryTerm:   actualTerm,
				DiffDays:             diffDaysStr,
			})
		}
		return transfers
	}

	type rawAA struct {
		ID         int64
		ContractID int64      `gorm:"column:contract_id"`
		Number     string     `gorm:"column:agreement_number"`
		Date       *time.Time `gorm:"column:agreement_date"`
		EndDate    *time.Time `gorm:"column:extend_date_to"`
		Amount     float64    `gorm:"column:foreign_amount"`
		Currency   string
		Subject    string
		Partner    string `gorm:"column:receiver_name"`
		Country    string
	}
	var rawAAs []rawAA
	r.db.WithContext(ctx).Table("additional_agreements aa").
		Select(`
			aa.id, aa.contract_id, COALESCE(aa.agreement_number, '') AS agreement_number,
			aa.agreement_date, aa.extend_date_to, COALESCE(aa.foreign_amount, 0) AS foreign_amount,
			COALESCE(aa.currency, '') AS currency, COALESCE(aa.subject, '') AS subject,
			COALESCE(NULLIF(c.receiver_name, ''), cp.llc, '') AS receiver_name,
			COALESCE(c.receiver_country, '') AS country
		`).
		Joins("JOIN contracts c ON c.id = aa.contract_id").
		Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
		Where("aa.contract_id IN ? AND aa.deleted_at IS NULL", contractIDs).
		Order("aa.agreement_date ASC, aa.id ASC").
		Scan(&rawAAs)

	aaByContract := make(map[int64][]rawAA)
	for _, aa := range rawAAs {
		aaByContract[aa.ContractID] = append(aaByContract[aa.ContractID], aa)
	}

	for _, rc := range rawContracts {
		contractDeliveryTerm := 0
		if rc.ReturnDays != nil {
			contractDeliveryTerm = *rc.ReturnDays
		}

		// contracts.original_amount holds the base contract amount before same-currency AAs are synced into total_amount.
		contractTotalAmount := rc.OriginalAmount
		if contractTotalAmount == 0 {
			contractTotalAmount = rc.TotalAmount
		}
		latestEndDate := rc.EDate

		rcCurrencyUpper := strings.ToUpper(strings.TrimSpace(rc.Currency))
		contractDto := dto.ClientConsolidatedContract{
			Number:         rc.Number,
			Date:           formatDate(rc.CDate),
			ForeignCompany: rc.ReceiverName,
			Country:        rc.Country,
			Currency:       rc.Currency,
			Transfers:      buildTransfers(invByCurrencyKey[invKey{rc.ID, rcCurrencyUpper}], contractDeliveryTerm),
		}

		var aaOrder []string
		aaByCurrency := make(map[string]*dto.ClientConsolidatedAA)

		for _, a := range aaByContract[rc.ID] {
			if a.EndDate != nil && (latestEndDate == nil || a.EndDate.After(*latestEndDate)) {
				latestEndDate = a.EndDate
			}
			aaCurrencyUpper := strings.ToUpper(strings.TrimSpace(a.Currency))
			if aaCurrencyUpper == rcCurrencyUpper {
				contractTotalAmount += a.Amount
				continue
			}

			if existing, ok := aaByCurrency[aaCurrencyUpper]; ok {
				existing.TotalAmount += a.Amount
				continue
			}

			aaDto := &dto.ClientConsolidatedAA{
				ForeignCompany:       a.Partner,
				Country:              a.Country,
				Currency:             a.Currency,
				TotalAmount:          a.Amount,
				ParentContractNumber: rc.Number,
				Transfers:            buildTransfers(invByCurrencyKey[invKey{rc.ID, aaCurrencyUpper}], contractDeliveryTerm),
			}
			aaByCurrency[aaCurrencyUpper] = aaDto
			aaOrder = append(aaOrder, aaCurrencyUpper)
		}

		for _, curr := range aaOrder {
			contractDto.AdditionalAgreements = append(contractDto.AdditionalAgreements, *aaByCurrency[curr])
		}

		contractDto.TotalAmount = contractTotalAmount
		contractDto.EndDate = formatDate(latestEndDate)

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
