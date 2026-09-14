package repository
import(
		"context"
	"fmt"
	"strings"
	"time"

	"CurrencyControl/internal/delivery/dto"
)

func (r *reportRepo) GetGTDExcelData(ctx context.Context, filter dto.ExcelReportFilter) ([]dto.GTDExcelRow, string, error) {
	clientName := r.getClientName(ctx, filter.ClientID)

	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, "g.deleted_at IS NULL")

	if filter.ClientID != nil && *filter.ClientID > 0 {
		conditions = append(conditions, fmt.Sprintf("c.client_id = $%d", argIdx))
		args = append(args, *filter.ClientID)
		argIdx++
	}
	if filter.BranchID != nil && *filter.BranchID > 0 {
		conditions = append(conditions, fmt.Sprintf("c.branch_id = $%d", argIdx))
		args = append(args, *filter.BranchID)
		argIdx++
	}
	if filter.FromDate != nil {
		conditions = append(conditions, fmt.Sprintf("g.gtd_date >= $%d", argIdx))
		args = append(args, *filter.FromDate)
		argIdx++
	}
	if filter.ToDate != nil {
		conditions = append(conditions, fmt.Sprintf("g.gtd_date <= $%d", argIdx))
		args = append(args, *filter.ToDate)
		argIdx++
	}
	if strings.TrimSpace(filter.Currency) != "" {
		conditions = append(conditions, fmt.Sprintf("g.gtd_currency = $%d", argIdx))
		args = append(args, strings.ToUpper(strings.TrimSpace(filter.Currency)))
		argIdx++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT 
			COALESCE(g.gtd_number, ''),
			COALESCE(g.submission_date, g.gtd_date),
			g.gtd_amount,
			COALESCE(g.gtd_currency, ''),
			COALESCE(g.hs_code, ''),
			COALESCE(cp.name, ''),
			COALESCE(g.destination_country, '')
		FROM gtd g
		JOIN contracts c ON c.id = g.contract_id
		LEFT JOIN counterparties cp ON cp.id = c.client_id
		%s
		ORDER BY g.gtd_date DESC, g.id DESC`,
		whereClause,
	)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, clientName, err
	}
	defer rows.Close()

	var result []dto.GTDExcelRow
	for rows.Next() {
		var row dto.GTDExcelRow
		var gDate *time.Time
		err := rows.Scan(
			&row.Number,
			&gDate,
			&row.Amount,
			&row.Currency,
			&row.HSCode,
			&row.SenderName,
			&row.Country,
		)
		if err != nil {
			return nil, clientName, err
		}
		row.Date = formatDate(gDate)
		result = append(result, row)
	}

	if result == nil {
		result = []dto.GTDExcelRow{}
	}
	return result, clientName, nil
}

func (r *reportRepo) GetAAExcelData(ctx context.Context, filter dto.ExcelReportFilter) ([]dto.AAExcelRow, string, error) {
	clientName := r.getClientName(ctx, filter.ClientID)

	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, "aa.deleted_at IS NULL")

	if filter.ClientID != nil && *filter.ClientID > 0 {
		conditions = append(conditions, fmt.Sprintf("c.client_id = $%d", argIdx))
		args = append(args, *filter.ClientID)
		argIdx++
	}
	if filter.BranchID != nil && *filter.BranchID > 0 {
		conditions = append(conditions, fmt.Sprintf("c.branch_id = $%d", argIdx))
		args = append(args, *filter.BranchID)
		argIdx++
	}
	if filter.FromDate != nil {
		conditions = append(conditions, fmt.Sprintf("aa.agreement_date >= $%d", argIdx))
		args = append(args, *filter.FromDate)
		argIdx++
	}
	if filter.ToDate != nil {
		conditions = append(conditions, fmt.Sprintf("aa.agreement_date <= $%d", argIdx))
		args = append(args, *filter.ToDate)
		argIdx++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT 
			COALESCE(aa.agreement_number, ''),
			CASE 
				WHEN aa.doc_type = 'specification' THEN 'Спецификация'
				WHEN aa.doc_type = 'appendix' THEN 'Приложение'
				ELSE 'Доп. соглашение'
			END,
			aa.agreement_date,
			COALESCE(c.contract_number, ''),
			COALESCE(aa.foreign_amount, 0),
			COALESCE(aa.currency, ''),
			aa.delivery_date,
			aa.return_date,
			aa.extend_date_to,
			COALESCE(aa.subject, c.subject, '')
		FROM additional_agreements aa
		JOIN contracts c ON c.id = aa.contract_id
		LEFT JOIN counterparties cp ON cp.id = c.client_id
		%s
		ORDER BY aa.agreement_date DESC, aa.id DESC`,
		whereClause,
	)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, clientName, err
	}
	defer rows.Close()

	var result []dto.AAExcelRow
	for rows.Next() {
		var row dto.AAExcelRow
		var aDate, dDate, rDate, eDate *time.Time
		err := rows.Scan(
			&row.Number,
			&row.DocType,
			&aDate,
			&row.ContractNum,
			&row.Amount,
			&row.Currency,
			&dDate,
			&rDate,
			&eDate,
			&row.Subject,
		)
		if err != nil {
			return nil, clientName, err
		}
		row.Date = formatDate(aDate)
		row.DeliveryDate = formatDate(dDate)
		row.ReturnDate = formatDate(rDate)
		row.ExtendDateTo = formatDate(eDate)
		result = append(result, row)
	}

	if result == nil {
		result = []dto.AAExcelRow{}
	}
	return result, clientName, nil
}

func (r *reportRepo) GetClientsExcelData(ctx context.Context, filter dto.ExcelReportFilter) ([]dto.ClientExcelRow, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, "cp.deleted_at IS NULL")

	if filter.BranchID != nil && *filter.BranchID > 0 {
		conditions = append(conditions, fmt.Sprintf("cp.branch_id = $%d", argIdx))
		args = append(args, *filter.BranchID)
		argIdx++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT 
			cp.id,
			cp.name,
			COALESCE(cp.inn, ''),
			COALESCE(b.name, ''),
			COUNT(c.id),
			COALESCE(SUM(c.total_amount), 0),
			cp.created_at
		FROM counterparties cp
		LEFT JOIN branches b ON b.id = cp.branch_id
		LEFT JOIN contracts c ON c.client_id = cp.id AND c.deleted_at IS NULL
		%s
		GROUP BY cp.id, cp.name, cp.inn, b.name, cp.created_at
		ORDER BY cp.created_at DESC`,
		whereClause,
	)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []dto.ClientExcelRow
	for rows.Next() {
		var row dto.ClientExcelRow
		var cDate time.Time
		err := rows.Scan(
			&row.ID,
			&row.Name,
			&row.INN,
			&row.BranchName,
			&row.ContractsCount,
			&row.TotalAmount,
			&cDate,
		)
		if err != nil {
			return nil, err
		}
		row.CreatedAt = formatDate(&cDate)
		result = append(result, row)
	}

	if result == nil {
		result = []dto.ClientExcelRow{}
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

	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, "i.deleted_at IS NULL")

	if filter.ClientID != nil && *filter.ClientID > 0 {
		conditions = append(conditions, fmt.Sprintf("c.client_id = $%d", argIdx))
		args = append(args, *filter.ClientID)
		argIdx++
	}
	if filter.BranchID != nil && *filter.BranchID > 0 {
		conditions = append(conditions, fmt.Sprintf("c.branch_id = $%d", argIdx))
		args = append(args, *filter.BranchID)
		argIdx++
	}
	if filter.FromDate != nil {
		conditions = append(conditions, fmt.Sprintf("i.invoice_date >= $%d", argIdx))
		args = append(args, *filter.FromDate)
		argIdx++
	}
	if filter.ToDate != nil {
		conditions = append(conditions, fmt.Sprintf("i.invoice_date <= $%d", argIdx))
		args = append(args, *filter.ToDate)
		argIdx++
	}
	if strings.TrimSpace(filter.Currency) != "" {
		conditions = append(conditions, fmt.Sprintf("i.currency = $%d", argIdx))
		args = append(args, strings.ToUpper(strings.TrimSpace(filter.Currency)))
		argIdx++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT 
			COALESCE(i.invoice_number, ''),
			i.invoice_date,
			i.amount,
			i.currency,
			COALESCE(i.hs_code, ''),
			COALESCE(c.subject, '')
		FROM invoices i
		JOIN contracts c ON c.id = i.contract_id
		LEFT JOIN counterparties cp ON cp.id = c.client_id
		%s
		ORDER BY i.invoice_date DESC, i.id DESC`,
		whereClause,
	)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, clientName, err
	}
	defer rows.Close()

	var result []dto.InvoiceExcelRow
	for rows.Next() {
		var row dto.InvoiceExcelRow
		var iDate *time.Time
		err := rows.Scan(
			&row.Number,
			&iDate,
			&row.Amount,
			&row.Currency,
			&row.HSCode,
			&row.PaymentPurpose,
		)
		if err != nil {
			return nil, clientName, err
		}
		row.Date = formatDate(iDate)
		result = append(result, row)
	}

	if result == nil {
		result = []dto.InvoiceExcelRow{}
	}
	return result, clientName, nil
}
func (r *reportRepo) GetContractsExcelData(ctx context.Context, filter dto.ExcelReportFilter) ([]dto.ContractExcelRow, string, error) {
	clientName := r.getClientName(ctx, filter.ClientID)

	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, "c.deleted_at IS NULL")

	if filter.ClientID != nil && *filter.ClientID > 0 {
		conditions = append(conditions, fmt.Sprintf("c.client_id = $%d", argIdx))
		args = append(args, *filter.ClientID)
		argIdx++
	}
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
			c.contract_number,
			c.contract_date,
			COALESCE(c.subject, ''),
			c.total_amount,
			c.contract_currency,
			c.return_date,
			c.delivery_date,
			COALESCE(c.extend_date_to, c.contract_end_date),
			COALESCE(NULLIF(c.receiver_name, ''), cp.name, ''),
			COALESCE(NULLIF(c.receiver_bank, ''), cp.inn, ''),
			COALESCE(c.receiver_country, '')
		FROM contracts c
		LEFT JOIN counterparties cp ON cp.id = c.client_id
		%s
		ORDER BY c.contract_date DESC, c.id DESC`,
		whereClause,
	)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, clientName, err
	}
	defer rows.Close()

	var result []dto.ContractExcelRow
	for rows.Next() {
		var row dto.ContractExcelRow
		var cDate, rDate, dDate, eDate *time.Time
		err := rows.Scan(
			&row.Number,
			&cDate,
			&row.Subject,
			&row.Amount,
			&row.Currency,
			&rDate,
			&dDate,
			&eDate,
			&row.ReceiverName,
			&row.ReceiverAccount,
			&row.ReceiverCountry,
		)
		if err != nil {
			return nil, clientName, err
		}
		row.Date = formatDate(cDate)
		row.ReturnDate = formatDate(rDate)
		row.DeliveryDate = formatDate(dDate)
		row.ContractEndDate = formatDate(eDate)
		result = append(result, row)
	}

	if result == nil {
		result = []dto.ContractExcelRow{}
	}
	return result, clientName, nil
}
