package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/service/ports"

	"github.com/jackc/pgx/v5/pgxpool"
)

type reportRepo struct {
	db *pgxpool.Pool
}

func NewReportRepository(db *pgxpool.Pool) ports.ReportRepository {
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
	var name string
	_ = r.db.QueryRow(ctx, `SELECT COALESCE(name, '') FROM counterparties WHERE id = $1`, *clientID).Scan(&name)
	if name == "" {
		return fmt.Sprintf("Клиент №%d", *clientID)
	}
	return name
}

func (r *reportRepo) GetContractsReport(ctx context.Context, filter ports.ReportFilter) (*dto.ContractsReportResponse, error) {
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
			c.contract_date,
			c.delivery_date,
			c.contract_end_date,
			c.total_amount,
			c.remaining_amount,
			c.contract_currency,
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
			&item.ContractDate,
			&deliveryDate,
			&contractEndDate,
			&item.TotalAmount,
			&item.RemainingAmount,
			&item.ContractCurrency,
			&item.Subject,
			&item.InvoicesCount,
			&item.InvoicesAmount,
			&item.CreatedBy,
		); err != nil {
			return nil, err
		}

		item.DeliveryDate = deliveryDate
		item.ContractEndDate = contractEndDate

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



func (r *reportRepo) GetClientCurrencies(ctx context.Context, clientID int64) ([]string, error) {
	query := `
		SELECT DISTINCT currency FROM (
			SELECT contract_currency AS currency FROM contracts WHERE client_id = $1 AND deleted_at IS NULL
			UNION
			SELECT currency FROM additional_agreements aa JOIN contracts c ON c.id = aa.contract_id WHERE c.client_id = $1 AND aa.deleted_at IS NULL
			UNION
			SELECT i.currency FROM invoices i JOIN contracts c ON c.id = i.contract_id WHERE c.client_id = $1 AND i.deleted_at IS NULL
		) t WHERE currency IS NOT NULL AND currency != ''
		ORDER BY currency`

	rows, err := r.db.Query(ctx, query, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var currencies []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err == nil && c != "" {
			currencies = append(currencies, c)
		}
	}
	if currencies == nil {
		currencies = []string{}
	}
	return currencies, nil
}

func (r *reportRepo) GetClientConsolidatedReportData(ctx context.Context, clientID int64, filter dto.ExcelReportFilter) (*dto.ClientConsolidatedReportData, error) {
	var clientName string
	_ = r.db.QueryRow(ctx, `SELECT COALESCE(name, '') FROM counterparties WHERE id = $1`, clientID).Scan(&clientName)
	if clientName == "" {
		clientName = fmt.Sprintf("Клиент №%d", clientID)
	}

	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, "c.deleted_at IS NULL")
	conditions = append(conditions, fmt.Sprintf("c.client_id = $%d", argIdx))
	args = append(args, clientID)
	argIdx++

	if len(filter.Currencies) > 0 {
		conditions = append(conditions, fmt.Sprintf("c.contract_currency = ANY($%d)", argIdx))
		args = append(args, filter.Currencies)
		argIdx++
	} else if strings.TrimSpace(filter.Currency) != "" {
		conditions = append(conditions, fmt.Sprintf("c.contract_currency = $%d", argIdx))
		args = append(args, strings.ToUpper(strings.TrimSpace(filter.Currency)))
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

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT 
			c.id,
			c.contract_number,
			c.contract_date,
			COALESCE(c.extend_date_to, c.contract_end_date),
			c.total_amount,
			c.contract_currency,
			COALESCE(c.subject, ''),
			COALESCE(cp.name, ''),
			COALESCE(NULLIF(c.receiver_country, ''), cnt.name_ru, 'Америка')
		FROM contracts c
		LEFT JOIN counterparties cp ON cp.id = c.client_id
		LEFT JOIN countries cnt ON cnt.id = cp.country_id
		%s
		ORDER BY c.contract_date ASC, c.id ASC`,
		whereClause,
	)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type rawContract struct {
		id          int64
		number      string
		cDate       *time.Time
		eDate       *time.Time
		totalAmount float64
		currency    string
		subject     string
		partnerName string
		country     string
	}

	var rawContracts []rawContract
	for rows.Next() {
		var rc rawContract
		if err := rows.Scan(
			&rc.id, &rc.number, &rc.cDate, &rc.eDate, &rc.totalAmount, &rc.currency, &rc.subject, &rc.partnerName, &rc.country,
		); err != nil {
			return nil, err
		}
		rawContracts = append(rawContracts, rc)
	}

	data := &dto.ClientConsolidatedReportData{
		ClientName: clientName,
	}

	for _, rc := range rawContracts {
		contractDto := dto.ClientConsolidatedContract{
			Number:         rc.number,
			Date:           formatDate(rc.cDate),
			EndDate:        formatDate(rc.eDate),
			ForeignCompany: rc.partnerName,
			Country:        rc.country,
			TotalAmount:    rc.totalAmount,
			Currency:       rc.currency,
		}

		invRows, err := r.db.Query(ctx, `
			SELECT 
				i.id,
				COALESCE(i.invoice_number, ''),
				i.invoice_date,
				i.amount,
				COALESCE((SELECT g.gtd_amount FROM gtd g WHERE (g.invoice_id = i.id OR (g.contract_id = i.contract_id AND g.additional_agreement_id IS NULL)) AND g.deleted_at IS NULL LIMIT 1), 0) AS gtd_amount,
				COALESCE((SELECT g.gtd_number FROM gtd g WHERE (g.invoice_id = i.id OR (g.contract_id = i.contract_id AND g.additional_agreement_id IS NULL)) AND g.deleted_at IS NULL LIMIT 1), '') AS gtd_number,
				COALESCE((SELECT g.days_difference FROM gtd g WHERE (g.invoice_id = i.id OR (g.contract_id = i.contract_id AND g.additional_agreement_id IS NULL)) AND g.deleted_at IS NULL LIMIT 1), 0) AS days_diff
			FROM invoices i
			WHERE i.contract_id = $1 AND i.additional_agreement_id IS NULL AND i.deleted_at IS NULL
			ORDER BY i.invoice_date ASC, i.id ASC`,
			rc.id,
		)
		if err == nil {
			for invRows.Next() {
				var iID int64
				var iNum string
				var iDate *time.Time
				var iAmount, gAmount float64
				var gNum string
				var daysDiff int

				if err := invRows.Scan(&iID, &iNum, &iDate, &iAmount, &gAmount, &gNum, &daysDiff); err == nil {
					diffAmount := iAmount - gAmount
					diffDaysStr := formatDiffDays(daysDiff)
					contractDto.Transfers = append(contractDto.Transfers, dto.ClientConsolidatedTransfer{
						InvoiceDate:          formatDate(iDate),
						InvoiceAmount:        iAmount,
						GTDAmount:            gAmount,
						GTDNumber:            gNum,
						DiffAmount:           diffAmount,
						ContractDeliveryTerm: 180,
						ActualDeliveryTerm:   180 - daysDiff,
						DiffDays:             diffDaysStr,
					})
				}
			}
			invRows.Close()
		}


		aaRows, err := r.db.Query(ctx, `
			SELECT 
				aa.id,
				COALESCE(aa.agreement_number, ''),
				aa.agreement_date,
				aa.extend_date_to,
				COALESCE(aa.foreign_amount, 0),
				COALESCE(aa.currency, ''),
				COALESCE(aa.subject, ''),
				COALESCE(cp.name, ''),
				COALESCE(NULLIF(c.receiver_country, ''), cnt.name_ru, 'Америка')
			FROM additional_agreements aa
			JOIN contracts c ON c.id = aa.contract_id
			LEFT JOIN counterparties cp ON cp.id = c.client_id
			LEFT JOIN countries cnt ON cnt.id = cp.country_id
			WHERE aa.contract_id = $1 AND aa.deleted_at IS NULL
			ORDER BY aa.agreement_date ASC, aa.id ASC`,
			rc.id,
		)
		if err == nil {
			for aaRows.Next() {
				var aaID int64
				var aaNum string
				var aDate, eDate *time.Time
				var aAmount float64
				var aCurrency, aSubject, aPartner, aCountry string

				if err := aaRows.Scan(&aaID, &aaNum, &aDate, &eDate, &aAmount, &aCurrency, &aSubject, &aPartner, &aCountry); err == nil {
					aaDto := dto.ClientConsolidatedAA{
						Number:               aaNum,
						Date:                 formatDate(aDate),
						EndDate:              formatDate(eDate),
						ForeignCompany:       aPartner,
						Country:              aCountry,
						TotalAmount:          aAmount,
						Currency:             aCurrency,
						ParentContractNumber: rc.number,
					}

					aaInvRows, err := r.db.Query(ctx, `
						SELECT 
							i.id,
							COALESCE(i.invoice_number, ''),
							i.invoice_date,
							i.amount,
							COALESCE((SELECT g.gtd_amount FROM gtd g WHERE (g.invoice_id = i.id OR g.additional_agreement_id = i.additional_agreement_id) AND g.deleted_at IS NULL LIMIT 1), 0) AS gtd_amount,
							COALESCE((SELECT g.gtd_number FROM gtd g WHERE (g.invoice_id = i.id OR g.additional_agreement_id = i.additional_agreement_id) AND g.deleted_at IS NULL LIMIT 1), '') AS gtd_number,
							COALESCE((SELECT g.days_difference FROM gtd g WHERE (g.invoice_id = i.id OR g.additional_agreement_id = i.additional_agreement_id) AND g.deleted_at IS NULL LIMIT 1), 0) AS days_diff
						FROM invoices i
						WHERE i.additional_agreement_id = $1 AND i.deleted_at IS NULL
						ORDER BY i.invoice_date ASC, i.id ASC`,
						aaID,
					)
					if err == nil {
						for aaInvRows.Next() {
							var iID int64
							var iNum string
							var iDate *time.Time
							var iAmount, gAmount float64
							var gNum string
							var daysDiff int

							if err := aaInvRows.Scan(&iID, &iNum, &iDate, &iAmount, &gAmount, &gNum, &daysDiff); err == nil {
								diffAmount := iAmount - gAmount
								diffDaysStr := formatDiffDays(daysDiff)
								aaDto.Transfers = append(aaDto.Transfers, dto.ClientConsolidatedTransfer{
									InvoiceDate:          formatDate(iDate),
									InvoiceAmount:        iAmount,
									GTDAmount:            gAmount,
									GTDNumber:            gNum,
									DiffAmount:           diffAmount,
									ContractDeliveryTerm: 60,
									ActualDeliveryTerm:   60 - daysDiff,
									DiffDays:             diffDaysStr,
								})
							}
						}
						aaInvRows.Close()
					}

					contractDto.AdditionalAgreements = append(contractDto.AdditionalAgreements, aaDto)
				}
			}
			aaRows.Close()
		}

		data.Contracts = append(data.Contracts, contractDto)
	}

	return data, nil
}

