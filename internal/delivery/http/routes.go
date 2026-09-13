package http

import (
	"net/http"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "CurrencyControl/docs"
	"CurrencyControl/internal/domain"
)

func InitRoutes(
	dashboardHandler *ContractHandler,
	companyHandler *CounterpartyHandler,
	dictHandler *DictionaryHandler,
	invoiceHandler *InvoiceHandler,
	authHandler *AuthHandler,
	branchHandler *BranchHandler,
	auditHandler *AuditHandler,
	reportHandler *ReportHandler,
	trashHandler *TrashHandler,
	complianceHandler *ComplianceHandler,
	approvalHandler *ApprovalHandler,
) http.Handler {
	r := mux.NewRouter()

	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	r.HandleFunc("/auth/login", authHandler.Login).Methods(http.MethodPost)
	r.HandleFunc("/auth/refresh", authHandler.Refresh).Methods(http.MethodPost)
	r.HandleFunc("/auth/branches", authHandler.GetBranches).Methods(http.MethodGet)
	r.HandleFunc("/auth/roles", authHandler.GetRoles).Methods(http.MethodGet)
	r.HandleFunc("/auth/request-access", authHandler.RequestAccess).Methods(http.MethodPost)
	r.HandleFunc("/auth/access-requests/{request_id}/status", authHandler.GetRequestStatus).Methods(http.MethodGet)
	r.HandleFunc("/auth/logout", authHandler.Logout).Methods(http.MethodPost)
	r.HandleFunc("/api/countries", dictHandler.SearchCountries).Methods(http.MethodGet)
	r.HandleFunc("/api/currencies", dictHandler.SearchCurrencies).Methods(http.MethodGet)

	api := r.PathPrefix("/api").Subrouter()
	api.Use(func(next http.Handler) http.Handler { return AuthMiddleware(next) })

	api.HandleFunc("/branches", dictHandler.GetBranches).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard", dashboardHandler.Dashboard).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/notifications", dashboardHandler.GetNotifications).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts", dashboardHandler.GetContractsByCompany).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}", dashboardHandler.GetByID).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices", invoiceHandler.GetInvoices).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}", invoiceHandler.GetInvoiceByID).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/gtd", invoiceHandler.GetGTD).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/gtd", invoiceHandler.GetContractGTDs).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/gtd/{gtd_id}", invoiceHandler.GetGTDByID).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements", invoiceHandler.GetAdditionalAgreements).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}", invoiceHandler.GetAdditionalAgreementByID).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}/invoices", invoiceHandler.GetAdditionalAgreementInvoices).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}/invoices/{invoice_id}", invoiceHandler.GetInvoiceByID).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}/invoices/{invoice_id}/gtd", invoiceHandler.GetGTD).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}/gtd", invoiceHandler.GetAdditionalAgreementGTDs).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}/gtd/{gtd_id}", invoiceHandler.GetGTDByID).Methods(http.MethodGet)

	api.HandleFunc("/branches/{id}/archive", dashboardHandler.GetArchivedContracts).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/archive", dashboardHandler.GetArchivedByCompany).Methods(http.MethodGet)

	reportsApi := api.PathPrefix("/reports").Subrouter()
	reportsApi.Use(func(next http.Handler) http.Handler {
		return RequireRoles(
			domain.RoleOperator,
			domain.RoleBranchHead,
			domain.RoleCurrencyControl,
			domain.RoleCurrencyController,
			domain.RoleCompliance,
			domain.RoleInternalAudit,
			domain.RoleAdmin,
		)(next)
	})
	reportsApi.HandleFunc("/contracts", reportHandler.GetContractsReport).Methods(http.MethodGet)
	reportsApi.HandleFunc("/types", reportHandler.GetReportTypes).Methods(http.MethodGet)
	reportsApi.HandleFunc("/clients/{client_id:[0-9]+}/currencies", reportHandler.GetClientCurrencies).Methods(http.MethodGet)
	reportsApi.HandleFunc("/export", reportHandler.ExportExcelReport).Methods(http.MethodGet)
	reportsApi.HandleFunc("/templates/{report_type}", reportHandler.UploadTemplate).Methods(http.MethodPost)
	reportsApi.HandleFunc("/templates/{report_type}", reportHandler.DownloadTemplate).Methods(http.MethodGet)

	auditApi := api.PathPrefix("/audit-logs").Subrouter()
	auditApi.Use(func(next http.Handler) http.Handler {
		return RequireRoles(
			domain.RoleCompliance,
			domain.RoleInternalAudit,
			domain.RoleAdmin,
		)(next)
	})
	auditApi.HandleFunc("", auditHandler.GetLogs).Methods(http.MethodGet)

	accessApi := api.PathPrefix("/access-requests").Subrouter()
	accessApi.Use(func(next http.Handler) http.Handler {
		return RequireRoles(domain.RoleCompliance, domain.RoleAdmin)(next)
	})
	accessApi.HandleFunc("", authHandler.GetAccessRequests).Methods(http.MethodGet)
	accessApi.HandleFunc("/{request_id:[0-9]+}/approve", authHandler.ApproveRequest).Methods(http.MethodPost)
	accessApi.HandleFunc("/{request_id:[0-9]+}/reject", authHandler.RejectRequest).Methods(http.MethodPost)

	complianceApi := api.PathPrefix("/compliance").Subrouter()
	complianceApi.Use(func(next http.Handler) http.Handler {
		return RequireRoles(domain.RoleCompliance, domain.RoleAdmin)(next)
	})
	complianceApi.HandleFunc("/permissions/currency-control", complianceHandler.GetPermissions).Methods(http.MethodGet)
	complianceApi.HandleFunc("/permissions/currency-control", complianceHandler.GrantPermission).Methods(http.MethodPost)
	complianceApi.HandleFunc("/permissions/currency-control/{login}", complianceHandler.RevokePermission).Methods(http.MethodDelete)

	trashApi := api.PathPrefix("/trash").Subrouter()
	trashApi.Use(func(next http.Handler) http.Handler {
		return RequireRoles(
			domain.RoleCompliance,
			domain.RoleCurrencyControl,
			domain.RoleCurrencyController,
			domain.RoleAdmin,
		)(next)
	})
	trashApi.HandleFunc("", trashHandler.GetTrash).Methods(http.MethodGet)
	trashApi.HandleFunc("/{entity_type}/{id:[0-9]+}", trashHandler.GetTrashItem).Methods(http.MethodGet)
	trashApi.HandleFunc("/{entity_type}/{id:[0-9]+}/file", trashHandler.ViewFile).Methods(http.MethodGet)
	trashApi.HandleFunc("/{entity_type}/{id:[0-9]+}/restore", trashHandler.RestoreItem).Methods(http.MethodPost)

	approvalApi := api.PathPrefix("/approvals").Subrouter()
	approvalApi.HandleFunc("/pending", approvalHandler.GetPendingApprovals).Methods(http.MethodGet)
	approvalApi.HandleFunc("/{entity_type}/{id:[0-9]+}", approvalHandler.GetApprovalDetail).Methods(http.MethodGet)
	approvalApi.HandleFunc("/{entity_type}/{id:[0-9]+}/currency-control", approvalHandler.ReviewCurrencyControl).Methods(http.MethodPost)
	approvalApi.HandleFunc("/{entity_type}/{id:[0-9]+}/compliance", approvalHandler.ReviewCompliance).Methods(http.MethodPost)

	createApi := api.PathPrefix("").Subrouter()
	createApi.Use(func(next http.Handler) http.Handler {
		return RequireRoles(domain.RoleOperator, domain.RoleCompliance, domain.RoleAdmin)(next)
	})
	createApi.HandleFunc("/branches/{id}/dashboard/companies", companyHandler.Create).Methods(http.MethodPost)
	createApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts", dashboardHandler.Create).Methods(http.MethodPost)
	createApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices", invoiceHandler.CreateInvoice).Methods(http.MethodPost)
	createApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/gtd", invoiceHandler.CreateGTD).Methods(http.MethodPost)
	createApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/gtd", invoiceHandler.CreateGTD).Methods(http.MethodPost)
	createApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements", invoiceHandler.CreateAdditionalAgreement).Methods(http.MethodPost)
	createApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}/invoices", invoiceHandler.CreateAdditionalAgreementInvoice).Methods(http.MethodPost)
	createApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}/gtd", invoiceHandler.CreateAdditionalAgreementGTD).Methods(http.MethodPost)
	createApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}/invoices/{invoice_id}/gtd", invoiceHandler.CreateAdditionalAgreementGTD).Methods(http.MethodPost)

	docEditApi := api.PathPrefix("").Subrouter()
	docEditApi.Use(func(next http.Handler) http.Handler { return RequireDocumentEditAccess()(next) })
	docEditApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}", companyHandler.Update).Methods(http.MethodPut)
	docEditApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}", dashboardHandler.Update).Methods(http.MethodPut)
	docEditApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}", invoiceHandler.UpdateInvoice).Methods(http.MethodPut)
	docEditApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/gtd/{gtd_id}", invoiceHandler.UpdateGTD).Methods(http.MethodPut)
	docEditApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/gtd/{gtd_id}", invoiceHandler.UpdateGTD).Methods(http.MethodPut)
	docEditApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}", invoiceHandler.UpdateAdditionalAgreement).Methods(http.MethodPut)
	docEditApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}/invoices/{invoice_id}", invoiceHandler.UpdateAdditionalAgreementInvoice).Methods(http.MethodPut)
	docEditApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}/gtd/{gtd_id}", invoiceHandler.UpdateAdditionalAgreementGTD).Methods(http.MethodPut)
	docEditApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}/invoices/{invoice_id}/gtd/{gtd_id}", invoiceHandler.UpdateAdditionalAgreementGTD).Methods(http.MethodPut)

	docDeleteApi := api.PathPrefix("").Subrouter()
	docDeleteApi.Use(func(next http.Handler) http.Handler { return RequireDocumentDeleteAccess()(next) })
	docDeleteApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}", companyHandler.Delete).Methods(http.MethodDelete)
	docDeleteApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}", dashboardHandler.Delete).Methods(http.MethodDelete)
	docDeleteApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}", invoiceHandler.DeleteInvoice).Methods(http.MethodDelete)
	docDeleteApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/gtd/{gtd_id}", invoiceHandler.DeleteGTD).Methods(http.MethodDelete)
	docDeleteApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/gtd/{gtd_id}", invoiceHandler.DeleteGTD).Methods(http.MethodDelete)
	docDeleteApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}", invoiceHandler.DeleteAdditionalAgreement).Methods(http.MethodDelete)
	docDeleteApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}/invoices/{invoice_id}", invoiceHandler.DeleteAdditionalAgreementInvoice).Methods(http.MethodDelete)
	docDeleteApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}/gtd/{gtd_id}", invoiceHandler.DeleteAdditionalAgreementGTD).Methods(http.MethodDelete)
	docDeleteApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}/invoices/{invoice_id}/gtd/{gtd_id}", invoiceHandler.DeleteAdditionalAgreementGTD).Methods(http.MethodDelete)

	archiveRestoreApi := api.PathPrefix("").Subrouter()
	archiveRestoreApi.Use(func(next http.Handler) http.Handler {
		return RequireRoles(domain.RoleCurrencyControl, domain.RoleCurrencyController, domain.RoleCompliance, domain.RoleAdmin)(next)
	})
	archiveRestoreApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/restore", dashboardHandler.RestoreContract).Methods(http.MethodPut)
	archiveRestoreApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}/restore", invoiceHandler.RestoreAdditionalAgreement).Methods(http.MethodPut)

	branchManageApi := api.PathPrefix("/branches").Subrouter()
	branchManageApi.Use(func(next http.Handler) http.Handler {
		return RequireRoles(domain.RoleCompliance, domain.RoleAdmin)(next)
	})
	branchManageApi.HandleFunc("", branchHandler.Create).Methods(http.MethodPost)
	branchManageApi.HandleFunc("/{branch_id:[0-9]+}", branchHandler.Update).Methods(http.MethodPut)
	branchManageApi.HandleFunc("/{branch_id:[0-9]+}", branchHandler.Delete).Methods(http.MethodDelete)
	api.HandleFunc("/branches/all", branchHandler.GetAll).Methods(http.MethodGet)
	api.HandleFunc("/branches/{branch_id:[0-9]+}", branchHandler.GetByID).Methods(http.MethodGet)

	return LoggerMiddleware(CORSMiddleware(r))
}
