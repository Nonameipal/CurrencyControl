package repository

import (
	"context"

	"CurrencyControl/internal/domain"

	"gorm.io/gorm"
)

// fetchUserBriefs retrieves UserBrief information for a slice of logins in a single query.
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

	// For any login that wasn't found in the users table, provide fallback with Login set
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

// enrichContracts fills Creator, Updater, Deleter, CurrencyControlReviewer, ComplianceReviewer
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
}

func enrichContract(ctx context.Context, db *gorm.DB, c *domain.Contract) {
	if c == nil {
		return
	}
	contracts := []domain.Contract{*c}
	enrichContracts(ctx, db, contracts)
	*c = contracts[0]
}

// enrichInvoices fills Creator, Updater, Deleter, CurrencyControlReviewer, ComplianceReviewer
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

// enrichGTDs fills Creator, Updater, Deleter, CurrencyControlReviewer, ComplianceReviewer
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

// enrichAdditionalAgreements fills Creator, Updater, Deleter, CurrencyControlReviewer, ComplianceReviewer
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

// enrichPaymentOrders fills Creator, Updater, Deleter
func enrichPaymentOrders(ctx context.Context, db *gorm.DB, list []domain.PaymentOrder) {
	if len(list) == 0 {
		return
	}
	var logins []string
	for _, po := range list {
		logins = append(logins, po.CreatedBy, po.UpdatedBy)
		if po.DeletedBy != nil {
			logins = append(logins, *po.DeletedBy)
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

func enrichPaymentOrder(ctx context.Context, db *gorm.DB, po *domain.PaymentOrder) {
	if po == nil {
		return
	}
	pos := []domain.PaymentOrder{*po}
	enrichPaymentOrders(ctx, db, pos)
	*po = pos[0]
}

// enrichCounterparties fills Creator, Updater, Deleter
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

// enrichBranches fills Creator, Updater, Deleter
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

// enrichPermissions fills User, Granter
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
