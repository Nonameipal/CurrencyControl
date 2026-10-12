package repository
import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
)
func (r *contractRepo) SearchDashboard(ctx context.Context, req dto.DashboardSearchRequest) ([]dto.DashboardSearchResult, error) {
	tx := r.db.WithContext(ctx).Table("counterparties cp").
		Select(`DISTINCT cp.id, cp.branch_id, cp.llc, cp.inn, cp.client_type, 
				cp.phones, cp.email, cp.created_by, cp.created_at, cp.updated_at`).
		Joins("LEFT JOIN contracts c ON c.client_id = cp.id AND c.deleted_at IS NULL").
		Where("cp.deleted_at IS NULL")

	if q := strings.TrimSpace(req.Query); q != "" {
		st := strings.ToLower(strings.TrimSpace(req.SearchType))
		switch st {
		case "inn", "инн":
			tx = tx.Where("cp.inn ILIKE ?", "%"+q+"%")
		case "amount", "сумма":
			cleanAmt := strings.ReplaceAll(q, " ", "")
			cleanAmt = strings.ReplaceAll(cleanAmt, ",", ".")
			if amt, err := strconv.ParseFloat(cleanAmt, 64); err == nil {
				tx = tx.Where("c.total_amount = ?", amt)
			} else {
				tx = tx.Where("1 = 0")
			}
		default:
			tx = tx.Where("cp.llc ILIKE ?", "%"+q+"%")
		}
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
		if isSoleProprietor {
			clientTypeName = "Индивидуальный предприниматель"
		} else if isIndividual {
			clientTypeName = "Физическое лицо"
		}

		results[i] = dto.DashboardSearchResult{
			ID:         row.ID,
			Number:     fmt.Sprintf("№ %d", i+1),
			LLC:        row.LLC,
			INN:        innVal,
			ClientType: clientTypeName,
			Phones:     row.GetPhones(),

			CreatedBy: row.CreatedBy,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}
	}

	if len(results) > 0 {
		var logins []string
		for _, row := range results {
			if row.CreatedBy != "" {
				logins = append(logins, row.CreatedBy)
			}
		}
		userMap := fetchUserBriefs(ctx, r.db, logins)
		for i := range results {
			results[i].Creator = getUserBriefPtr(userMap, results[i].CreatedBy)
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
