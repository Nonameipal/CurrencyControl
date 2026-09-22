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
	"CurrencyControl/internal/service/ports"

	"gorm.io/gorm"
)

type contractRepo struct {
	db *gorm.DB
}

func NewContractRepository(db *gorm.DB) ports.ContractRepository {
	return &contractRepo{db: db}
}

func (r *contractRepo) Create(ctx context.Context, c domain.Contract) (domain.Contract, error) {
	if c.RemainingAmount == 0 {
		c.RemainingAmount = c.TotalAmount
	}
	if c.ApprovalStatus == "" {
		c.ApprovalStatus = domain.ApprovalStatusPendingCurrencyControl
	}

	if err := r.db.WithContext(ctx).Create(&c).Error; err != nil {
		return domain.Contract{}, err
	}
	return c, nil
}

func (r *contractRepo) GetByID(ctx context.Context, id int64) (domain.Contract, error) {
	var c domain.Contract
	if err := r.db.WithContext(ctx).First(&c, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Contract{}, errs.ErrContractNotFound
		}
		return domain.Contract{}, err
	}
	return c, nil
}

func (r *contractRepo) GetByClientID(ctx context.Context, clientID int64) ([]domain.Contract, error) {
	var list []domain.Contract
	if err := r.db.WithContext(ctx).
		Where("client_id = ?", clientID).
		Order("created_at DESC").
		Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *contractRepo) SearchDashboard(ctx context.Context, req dto.DashboardSearchRequest) ([]dto.DashboardSearchResult, error) {
	tx := r.db.WithContext(ctx).Table("counterparties cp").
		Select(`DISTINCT cp.id, cp.branch_id, cp.name, cp.llc, cp.inn, cp.client_type, 
				cp.phones, cp.email, cp.created_by, cp.created_at, cp.updated_at`).
		Joins("LEFT JOIN contracts c ON c.client_id = cp.id AND c.deleted_at IS NULL").
		Where("cp.deleted_at IS NULL")

	if req.Amount > 0 {
		tx = tx.Where("c.total_amount = ?", req.Amount)
	}
	if req.INN != "" {
		tx = tx.Where("cp.inn ILIKE ?", "%"+req.INN+"%")
	}
	if req.CompanyName != "" {
		tx = tx.Where("cp.name ILIKE ? OR cp.llc ILIKE ?", "%"+req.CompanyName+"%", "%"+req.CompanyName+"%")
	}
	if req.BranchID > 0 {
		tx = tx.Where("cp.branch_id = ?", req.BranchID)
	}

	var rows []domain.Counterparty
	if err := tx.Order("cp.id DESC").Limit(100).Scan(&rows).Error; err != nil {
		return nil, err
	}

	results := make([]dto.DashboardSearchResult, len(rows))
	for i, row := range rows {
		innVal := ""
		if row.INN != nil {
			innVal = *row.INN
		}
		lowerType := strings.ToLower(row.ClientType)
		isSoleProprietor := row.ClientType == domain.ClientTypeSoleProprietor || strings.Contains(lowerType, "предприниматель") || strings.Contains(lowerType, "ип")
		isIndividual := row.ClientType == domain.ClientTypeIndividual || strings.Contains(lowerType, "физ")
		clientTypeName := "Юридическое лицо"
		displayName := ""
		displayLLC := ""

		if isSoleProprietor {
			clientTypeName = "Индивидуальный предприниматель"
			displayName = row.Name
			displayLLC = row.LLC
		} else if isIndividual {
			clientTypeName = "Физическое лицо"
			displayName = row.Name
			displayLLC = row.LLC
		} else {
			displayLLC = row.LLC
			if displayLLC == "" {
				displayLLC = row.Name
			}
		}

		results[i] = dto.DashboardSearchResult{
			ID:         row.ID,
			Number:     fmt.Sprintf("№ %d", i+1),
			Name:       displayName,
			LLC:        displayLLC,
			INN:        innVal,
			ClientType: clientTypeName,
			Phones:     row.GetPhones(),

			CreatedBy:  row.CreatedBy,
			CreatedAt:  row.CreatedAt,
			UpdatedAt:  row.UpdatedAt,
		}
	}
	return results, nil
}

func (r *contractRepo) CheckCountry(ctx context.Context, name string) (bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return false, nil
	}
	var count int64
	err := r.db.WithContext(ctx).Table("countries").
		Where("LOWER(TRIM(name_ru)) = LOWER(TRIM(?))", name).
		Count(&count).Error
	return count > 0, err
}

func (r *contractRepo) CheckCurrency(ctx context.Context, code string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("currencies").
		Where("code = ?", code).
		Count(&count).Error
	return count > 0, err
}

func (r *contractRepo) GetExpiringContracts(ctx context.Context, branchID int) ([]dto.NotificationResponse, error) {
	var result []dto.NotificationResponse
	now := time.Now().Truncate(24 * time.Hour)

	type gtdRow struct {
		CompanyID      int64     `gorm:"column:company_id"`
		CompanyName    string    `gorm:"column:company_name"`
		ContractID     int64     `gorm:"column:contract_id"`
		ContractNumber string    `gorm:"column:contract_number"`
		InvoiceID      int64     `gorm:"column:invoice_id"`
		InvoiceNumber  string    `gorm:"column:invoice_number"`
		InvoiceAmount  float64   `gorm:"column:invoice_amount"`
		ClosedAmount   float64   `gorm:"column:closed_amount"`
		Currency       string    `gorm:"column:currency"`
		DeliveryDate   time.Time `gorm:"column:delivery_date"`
	}

	var gRows []gtdRow
	err := r.db.WithContext(ctx).Table("invoices i").
		Select(`comp.id AS company_id, comp.name AS company_name, c.id AS contract_id, c.contract_number, i.id AS invoice_id, i.invoice_number, i.amount AS invoice_amount, COALESCE((SELECT SUM(g.closes_amount) FROM gtd g WHERE g.invoice_id = i.id AND g.deleted_at IS NULL), 0) AS closed_amount, i.currency, c.delivery_date`).
		Joins("JOIN contracts c ON i.contract_id = c.id").
		Joins("JOIN counterparties comp ON c.client_id = comp.id").
		Where(`comp.branch_id = ? AND i.deleted_at IS NULL AND c.deleted_at IS NULL AND comp.deleted_at IS NULL AND c.delivery_date IS NOT NULL AND c.delivery_date <= CURRENT_DATE + INTERVAL '10 days' AND COALESCE((SELECT SUM(g.closes_amount) FROM gtd g WHERE g.invoice_id = i.id AND g.deleted_at IS NULL), 0) < i.amount`, branchID).
		Order("c.delivery_date ASC").
		Scan(&gRows).Error

	if err == nil {
		for _, row := range gRows {
			invID := row.InvoiceID
			daysLeft := int(row.DeliveryDate.Sub(now).Hours() / 24)
			n := dto.NotificationResponse{
				CompanyID:        row.CompanyID,
				CompanyName:      row.CompanyName,
				ContractID:       row.ContractID,
				ContractNumber:   row.ContractNumber,
				InvoiceID:        &invID,
				InvoiceNumber:    row.InvoiceNumber,
				InvoiceAmount:    row.InvoiceAmount,
				ClosedAmount:     row.ClosedAmount,
				UnclosedAmount:   row.InvoiceAmount - row.ClosedAmount,
				Currency:         row.Currency,
				DeadlineDate:     row.DeliveryDate,
				EffectiveEndDate: row.DeliveryDate,
				Type:             "gtd_deadline",
				DaysLeft:         daysLeft,
			}
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

	type expiryRow struct {
		CompanyID        int64     `gorm:"column:company_id"`
		CompanyName      string    `gorm:"column:company_name"`
		ContractID       int64     `gorm:"column:contract_id"`
		ContractNumber   string    `gorm:"column:contract_number"`
		EffectiveEndDate time.Time `gorm:"column:effective_end_date"`
	}

	var expRows []expiryRow
	err = r.db.WithContext(ctx).Table("contracts c").
		Select(`comp.id AS company_id, comp.name AS company_name, c.id AS contract_id, c.contract_number, GREATEST(c.contract_end_date, COALESCE(MAX(aa.extend_date_to), c.contract_end_date)) AS effective_end_date`).
		Joins("JOIN counterparties comp ON c.client_id = comp.id").
		Joins("LEFT JOIN additional_agreements aa ON aa.contract_id = c.id AND aa.deleted_at IS NULL").
		Where("comp.branch_id = ? AND c.deleted_at IS NULL AND comp.deleted_at IS NULL", branchID).
		Group("c.id, comp.id").
		Having("GREATEST(c.contract_end_date, COALESCE(MAX(aa.extend_date_to), c.contract_end_date)) <= CURRENT_DATE + INTERVAL '10 days'").
		Order("effective_end_date ASC").
		Scan(&expRows).Error

	if err == nil {
		for _, row := range expRows {
			daysLeft := int(row.EffectiveEndDate.Sub(now).Hours() / 24)
			n := dto.NotificationResponse{
				CompanyID:        row.CompanyID,
				CompanyName:      row.CompanyName,
				ContractID:       row.ContractID,
				ContractNumber:   row.ContractNumber,
				EffectiveEndDate: row.EffectiveEndDate,
				DeadlineDate:     row.EffectiveEndDate,
				Type:             "contract_expiry",
				DaysLeft:         daysLeft,
			}
			if daysLeft < 0 {
				n.Status = "overdue"
				n.Title = fmt.Sprintf("Истек срок действия контракта №%s (просрочка %d дн.)", n.ContractNumber, -daysLeft)
			} else {
				n.Status = "approaching"
				n.Title = fmt.Sprintf("Истекает срок действия контракта №%s (осталось %d дн.)", n.ContractNumber, daysLeft)
			}
			result = append(result, n)
		}
	}

	if result == nil {
		result = []dto.NotificationResponse{}
	}
	return result, nil
}

func (r *contractRepo) Update(ctx context.Context, id int64, c domain.Contract) (domain.Contract, error) {
	var existing domain.Contract
	if err := r.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Contract{}, errs.ErrContractNotFound
		}
		return domain.Contract{}, err
	}

	updates := map[string]interface{}{
		"contract_number":   c.ContractNumber,
		"contract_date":     c.ContractDate,
		"delivery_date":     c.DeliveryDate,
		"return_date":       c.ReturnDate,
		"total_amount":      c.TotalAmount,
		"remaining_amount":  c.RemainingAmount,
		"contract_currency": c.ContractCurrency,
		"subject":           c.Subject,
		"contract_end_date": c.ContractEndDate,
		"document_path":     c.DocumentPath,
		"receiver_name":     c.ReceiverName,
		"receiver_bank":     c.ReceiverBank,
		"receiver_country":  c.ReceiverCountry,
		"approval_status":   domain.ApprovalStatusPendingCurrencyControl,
		"rejection_reason":  "",
	}

	if err := r.db.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
		return domain.Contract{}, err
	}

	if err := r.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		return domain.Contract{}, err
	}
	return existing, nil
}

func (r *contractRepo) SoftDelete(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).Delete(&domain.Contract{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("Контракт не найден")
	}
	return nil
}

func (r *contractRepo) GetArchived(ctx context.Context, branchID int, page, pageSize int) ([]domain.Contract, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	tx := r.db.WithContext(ctx).Table("contracts c").
		Joins("JOIN counterparties cp ON cp.id = c.client_id").
		Where("(cp.branch_id = ? OR c.branch_id = ?) AND c.deleted_at IS NULL AND c.status = 'archived'", branchID, branchID)

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []domain.Contract
	if err := tx.Select("c.*").
		Order("c.archived_at DESC").
		Limit(pageSize).Offset(offset).
		Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, int(total), nil
}

func (r *contractRepo) GetArchivedByClientID(ctx context.Context, clientID int64) ([]domain.Contract, error) {
	var list []domain.Contract
	if err := r.db.WithContext(ctx).
		Where("client_id = ? AND status = 'archived'", clientID).
		Order("archived_at DESC").
		Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *contractRepo) RestoreContract(ctx context.Context, id int64) error {
	var existing domain.Contract
	if err := r.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("Контракт не найден")
		}
		return err
	}

	if existing.Status != domain.ContractStatusArchived {
		return errors.New("Контракт не находится в архиве")
	}

	return r.db.WithContext(ctx).Model(&existing).Updates(map[string]interface{}{
		"status":      "active",
		"archived_at": nil,
	}).Error
}
