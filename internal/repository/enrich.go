package repository

import (
	"context"
	"sort"
	"strings"

	"CurrencyControl/internal/domain"

	"gorm.io/gorm"
)

func fetchUserBriefs(ctx context.Context, db *gorm.DB, logins []string) map[string]domain.UserBrief {
	cleanLogins := make([]string, 0, len(logins))
	seen := make(map[string]bool, len(logins))
	for _, l := range logins {
		if l != "" && !seen[l] {
			seen[l] = true
			cleanLogins = append(cleanLogins, l)
		}
	}
	res := make(map[string]domain.UserBrief, len(cleanLogins))
	if len(cleanLogins) == 0 {
		return res
	}

	var users []domain.User
	if err := db.WithContext(ctx).Table("users").
		Select("login, first_name, last_name, email").
		Where("login IN ?", cleanLogins).
		Find(&users).Error; err == nil {
		for _, u := range users {
			res[u.Login] = domain.UserBrief{
				Login:     u.Login,
				FirstName: u.FirstName,
				LastName:  u.LastName,
				Email:     u.Email,
			}
		}
	}

	for _, l := range cleanLogins {
		if _, ok := res[l]; !ok {
			res[l] = domain.UserBrief{Login: l}
		}
	}
	return res
}

func getUserBriefPtr(m map[string]domain.UserBrief, login string) *domain.UserBrief {
	if login == "" {
		return nil
	}
	if ub, ok := m[login]; ok {
		return &ub
	}
	return &domain.UserBrief{Login: login}
}

func enrichContracts(ctx context.Context, db *gorm.DB, list []domain.Contract) {
	if len(list) == 0 {
		return
	}
	var logins []string
	for _, c := range list {
		logins = append(logins, c.CreatedBy, c.UpdatedBy, c.CurrencyControlReviewedBy, c.ComplianceReviewedBy)
		if c.DeletedBy != nil {
			logins = append(logins, *c.DeletedBy)
		}
	}
	userMap := fetchUserBriefs(ctx, db, logins)
	for i := range list {
		list[i].Creator = getUserBriefPtr(userMap, list[i].CreatedBy)
		list[i].Updater = getUserBriefPtr(userMap, list[i].UpdatedBy)
		list[i].CurrencyControlReviewer = getUserBriefPtr(userMap, list[i].CurrencyControlReviewedBy)
		list[i].ComplianceReviewer = getUserBriefPtr(userMap, list[i].ComplianceReviewedBy)
		if list[i].DeletedBy != nil {
			list[i].Deleter = getUserBriefPtr(userMap, *list[i].DeletedBy)
		}
	}
	contractIDs := make([]int64, len(list))
	for i, c := range list {
		contractIDs[i] = c.ID
	}
	type addlRow struct {
		ContractID int64   `gorm:"column:contract_id"`
		Currency   string  `gorm:"column:currency"`
		Total      float64 `gorm:"column:total"`
	}
	var addlRows []addlRow
	_ = db.WithContext(ctx).Table("additional_agreements").
		Select("contract_id, UPPER(TRIM(currency)) AS currency, SUM(COALESCE(foreign_amount, 0)) AS total").
		Where("contract_id IN ? AND deleted_at IS NULL AND status = 'active'", contractIDs).
		Group("contract_id, UPPER(TRIM(currency))").
		Scan(&addlRows).Error

	addlMap := make(map[int64]map[string]float64)
	for _, row := range addlRows {
		curr := strings.ToUpper(strings.TrimSpace(row.Currency))
		if curr == "" {
			continue
		}
		if addlMap[row.ContractID] == nil {
			addlMap[row.ContractID] = make(map[string]float64)
		}
		addlMap[row.ContractID][curr] += row.Total
	}

	type invRow struct {
		ContractID int64   `gorm:"column:contract_id"`
		Currency   string  `gorm:"column:currency"`
		Total      float64 `gorm:"column:total"`
	} 
	var invRows []invRow
	_ = db.WithContext(ctx).Table("invoices").
		Select("contract_id, UPPER(TRIM(currency)) AS currency, SUM(COALESCE(NULLIF(deduct_amount, 0), amount)) AS total").
		Where("contract_id IN ? AND deleted_at IS NULL", contractIDs).
		Group("contract_id, UPPER(TRIM(currency))").
		Scan(&invRows).Error

	invMap := make(map[int64]map[string]float64)
	for _, row := range invRows {
		curr := row.Currency
		if curr == "" {
			continue
		}
		if invMap[row.ContractID] == nil {
			invMap[row.ContractID] = make(map[string]float64)
		}
		invMap[row.ContractID][curr] += row.Total
	}

	for i := range list {
		c := &list[i]
		cCurr := strings.ToUpper(strings.TrimSpace(c.ContractCurrency))
		var balances []domain.ContractCurrencyBalance
		if currencies, ok := addlMap[c.ID]; ok {
			for curr, total := range currencies {
				if strings.EqualFold(curr, cCurr) {
					continue
				}
				invoiced := 0.0
				if invMap[c.ID] != nil {
					invoiced = invMap[c.ID][curr]
				}
				remaining := total - invoiced
				balances = append(balances, domain.ContractCurrencyBalance{
					Currency:        curr,
					TotalAmount:     total,
					RemainingAmount: remaining,
				})
			}
		}
		sort.Slice(balances, func(a, b int) bool {
			return balances[a].Currency < balances[b].Currency
		})
		c.AdditionalCurrencies = balances
		if len(balances) > 0 {
			c.AdditionalAgreementCurrency = &balances[0].Currency
			c.AdditionalAgreementAmount = &balances[0].TotalAmount
			c.AdditionalAgreementRemaining = &balances[0].RemainingAmount
		}
	}
}

func enrichContract(ctx context.Context, db *gorm.DB, c *domain.Contract) {
	if c == nil {
		return
	}
	contracts := []domain.Contract{*c}
	enrichContracts(ctx, db, contracts)
	*c = contracts[0]
}

func enrichInvoices(ctx context.Context, db *gorm.DB, list []domain.Invoice) {
	if len(list) == 0 {
		return
	}
	var logins []string
	for _, inv := range list {
		logins = append(logins, inv.CreatedBy, inv.UpdatedBy, inv.CurrencyControlReviewedBy, inv.ComplianceReviewedBy)
		if inv.DeletedBy != nil {
			logins = append(logins, *inv.DeletedBy)
		}
	}
	userMap := fetchUserBriefs(ctx, db, logins)
	for i := range list {
		if strings.TrimSpace(list[i].HSCode) == "" {
			list[i].HSCode = domain.DefaultHSCode
		}
		list[i].Creator = getUserBriefPtr(userMap, list[i].CreatedBy)
		list[i].Updater = getUserBriefPtr(userMap, list[i].UpdatedBy)
		list[i].CurrencyControlReviewer = getUserBriefPtr(userMap, list[i].CurrencyControlReviewedBy)
		list[i].ComplianceReviewer = getUserBriefPtr(userMap, list[i].ComplianceReviewedBy)
		if list[i].DeletedBy != nil {
			list[i].Deleter = getUserBriefPtr(userMap, *list[i].DeletedBy)
		}
	}
}

func enrichInvoice(ctx context.Context, db *gorm.DB, inv *domain.Invoice) {
	if inv == nil {
		return
	}
	invoices := []domain.Invoice{*inv}
	enrichInvoices(ctx, db, invoices)
	*inv = invoices[0]
}

func enrichGTDs(ctx context.Context, db *gorm.DB, list []domain.GTD) {
	if len(list) == 0 {
		return
	}
	var logins []string
	for _, g := range list {
		logins = append(logins, g.CreatedBy, g.UpdatedBy, g.CurrencyControlReviewedBy, g.ComplianceReviewedBy)
		if g.DeletedBy != nil {
			logins = append(logins, *g.DeletedBy)
		}
	}
	userMap := fetchUserBriefs(ctx, db, logins)
	for i := range list {
		if strings.TrimSpace(list[i].HSCode) == "" {
			list[i].HSCode = domain.DefaultHSCode
		}
		list[i].Creator = getUserBriefPtr(userMap, list[i].CreatedBy)
		list[i].Updater = getUserBriefPtr(userMap, list[i].UpdatedBy)
		list[i].CurrencyControlReviewer = getUserBriefPtr(userMap, list[i].CurrencyControlReviewedBy)
		list[i].ComplianceReviewer = getUserBriefPtr(userMap, list[i].ComplianceReviewedBy)
		if list[i].DeletedBy != nil {
			list[i].Deleter = getUserBriefPtr(userMap, *list[i].DeletedBy)
		}
	}
}

func enrichGTD(ctx context.Context, db *gorm.DB, g *domain.GTD) {
	if g == nil {
		return
	}
	gtds := []domain.GTD{*g}
	enrichGTDs(ctx, db, gtds)
	*g = gtds[0]
}

func enrichAdditionalAgreements(ctx context.Context, db *gorm.DB, list []domain.AdditionalAgreement) {
	if len(list) == 0 {
		return
	}
	var logins []string
	for _, aa := range list {
		logins = append(logins, aa.CreatedBy, aa.UpdatedBy, aa.CurrencyControlReviewedBy, aa.ComplianceReviewedBy)
		if aa.DeletedBy != nil {
			logins = append(logins, *aa.DeletedBy)
		}
	}
	userMap := fetchUserBriefs(ctx, db, logins)
	for i := range list {
		list[i].Creator = getUserBriefPtr(userMap, list[i].CreatedBy)
		list[i].Updater = getUserBriefPtr(userMap, list[i].UpdatedBy)
		list[i].CurrencyControlReviewer = getUserBriefPtr(userMap, list[i].CurrencyControlReviewedBy)
		list[i].ComplianceReviewer = getUserBriefPtr(userMap, list[i].ComplianceReviewedBy)
		if list[i].DeletedBy != nil {
			list[i].Deleter = getUserBriefPtr(userMap, *list[i].DeletedBy)
		}
	}
}

func enrichAdditionalAgreement(ctx context.Context, db *gorm.DB, aa *domain.AdditionalAgreement) {
	if aa == nil {
		return
	}
	aas := []domain.AdditionalAgreement{*aa}
	enrichAdditionalAgreements(ctx, db, aas)
	*aa = aas[0]
}

func enrichPaymentOrders(ctx context.Context, db *gorm.DB, list []domain.PaymentOrder) {
	if len(list) == 0 {
		return
	}
	var logins []string
	for _, po := range list {
		logins = append(logins, po.CreatedBy, po.UpdatedBy, po.CurrencyControlReviewedBy, po.ComplianceReviewedBy)
		if po.DeletedBy != nil {
			logins = append(logins, *po.DeletedBy)
		}
	}
	userMap := fetchUserBriefs(ctx, db, logins)
	for i := range list {
		list[i].Creator = getUserBriefPtr(userMap, list[i].CreatedBy)
		list[i].Updater = getUserBriefPtr(userMap, list[i].UpdatedBy)
		list[i].CurrencyControlReviewer = getUserBriefPtr(userMap, list[i].CurrencyControlReviewedBy)
		list[i].ComplianceReviewer = getUserBriefPtr(userMap, list[i].ComplianceReviewedBy)
		if list[i].DeletedBy != nil {
			list[i].Deleter = getUserBriefPtr(userMap, *list[i].DeletedBy)
		}
	}
}

func enrichPaymentOrder(ctx context.Context, db *gorm.DB, po *domain.PaymentOrder) {
	if po == nil {
		return
	}
	pos := []domain.PaymentOrder{*po}
	enrichPaymentOrders(ctx, db, pos)
	*po = pos[0]
}

func enrichCounterparties(ctx context.Context, db *gorm.DB, list []domain.Counterparty) {
	if len(list) == 0 {
		return
	}
	var logins []string
	for _, cp := range list {
		logins = append(logins, cp.CreatedBy, cp.UpdatedBy)
		if cp.DeletedBy != nil {
			logins = append(logins, *cp.DeletedBy)
		}
	}
	userMap := fetchUserBriefs(ctx, db, logins)
	for i := range list {
		list[i].Creator = getUserBriefPtr(userMap, list[i].CreatedBy)
		list[i].Updater = getUserBriefPtr(userMap, list[i].UpdatedBy)
		if list[i].DeletedBy != nil {
			list[i].Deleter = getUserBriefPtr(userMap, *list[i].DeletedBy)
		}
	}
}

func enrichCounterparty(ctx context.Context, db *gorm.DB, cp *domain.Counterparty) {
	if cp == nil {
		return
	}
	cps := []domain.Counterparty{*cp}
	enrichCounterparties(ctx, db, cps)
	*cp = cps[0]
}

func enrichBranches(ctx context.Context, db *gorm.DB, list []domain.Branch) {
	if len(list) == 0 {
		return
	}
	var logins []string
	for _, b := range list {
		logins = append(logins, b.CreatedBy, b.UpdatedBy)
		if b.DeletedBy != nil {
			logins = append(logins, *b.DeletedBy)
		}
	}
	userMap := fetchUserBriefs(ctx, db, logins)
	for i := range list {
		list[i].Creator = getUserBriefPtr(userMap, list[i].CreatedBy)
		list[i].Updater = getUserBriefPtr(userMap, list[i].UpdatedBy)
		if list[i].DeletedBy != nil {
			list[i].Deleter = getUserBriefPtr(userMap, *list[i].DeletedBy)
		}
	}
}

func enrichBranch(ctx context.Context, db *gorm.DB, b *domain.Branch) {
	if b == nil {
		return
	}
	bs := []domain.Branch{*b}
	enrichBranches(ctx, db, bs)
	*b = bs[0]
}

func enrichPermissions(ctx context.Context, db *gorm.DB, list []domain.CurrencyControlPermission) {
	if len(list) == 0 {
		return
	}
	var logins []string
	for _, p := range list {
		logins = append(logins, p.Login, p.GrantedBy)
	}
	userMap := fetchUserBriefs(ctx, db, logins)
	for i := range list {
		list[i].User = getUserBriefPtr(userMap, list[i].Login)
		list[i].Granter = getUserBriefPtr(userMap, list[i].GrantedBy)
	}
}
