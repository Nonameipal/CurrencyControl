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
) http.Handler {
	r := mux.NewRouter()


	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	r.HandleFunc("/auth/login", authHandler.Login).Methods(http.MethodPost)
	r.HandleFunc("/auth/request-access", authHandler.RequestAccess).Methods(http.MethodPost)
	r.HandleFunc("/auth/access-requests/{request_id}/status", authHandler.GetRequestStatus).Methods(http.MethodGet)
	r.HandleFunc("/auth/logout", authHandler.Logout).Methods(http.MethodPost)
	r.HandleFunc("/api/countries", dictHandler.SearchCountries).Methods(http.MethodGet)
	r.HandleFunc("/api/currencies", dictHandler.SearchCurrencies).Methods(http.MethodGet)


	api := r.PathPrefix("/api").Subrouter()
	api.Use(func(next http.Handler) http.Handler { return AuthMiddleware(next) })

	api.HandleFunc("/auth/logout", authHandler.Logout).Methods(http.MethodPost)
	api.HandleFunc("/branches", dictHandler.GetBranches).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard", dashboardHandler.Dashboard).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/notifications", dashboardHandler.GetNotifications).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/abs-lookup", companyHandler.ABSLookup).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts", dashboardHandler.GetContractsByCompany).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}", dashboardHandler.GetByID).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/document", dashboardHandler.GetDocument).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices", invoiceHandler.GetInvoices).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}", invoiceHandler.GetInvoiceByID).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/document", invoiceHandler.GetDocument).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/gtd", invoiceHandler.GetGTD).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements", invoiceHandler.GetAdditionalAgreements).Methods(http.MethodGet)

	reportsApi := api.PathPrefix("/reports").Subrouter()
	reportsApi.Use(func(next http.Handler) http.Handler {
		return RequireRoles(
			domain.RoleBranchHead,
			domain.RoleCurrencyControl,
			domain.RoleCurrencyController,
			domain.RoleCompliance,
			domain.RoleInternalAudit,
			domain.RoleAdmin,
		)(next)
	})
	reportsApi.HandleFunc("/contracts", reportHandler.GetContractsReport).Methods(http.MethodGet)

	auditApi := api.PathPrefix("/audit-logs").Subrouter()
	auditApi.Use(func(next http.Handler) http.Handler {
		return RequireRoles(
			domain.RoleCompliance,
			domain.RoleInternalAudit,
			domain.RoleAdmin,
		)(next)
	})
	auditApi.HandleFunc("", auditHandler.GetLogs).Methods(http.MethodGet)

	createApi := api.PathPrefix("").Subrouter()
	createApi.Use(func(next http.Handler) http.Handler {
		return RequireRoles(domain.RoleOperator, domain.RoleAdmin)(next)
	})
	createApi.HandleFunc("/branches/{id}/dashboard/companies", companyHandler.Create).Methods(http.MethodPost)
	createApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts", dashboardHandler.Create).Methods(http.MethodPost)
	createApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/document", dashboardHandler.UploadDocument).Methods(http.MethodPost)
	createApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices", invoiceHandler.CreateInvoice).Methods(http.MethodPost)
	createApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/document", invoiceHandler.UploadDocument).Methods(http.MethodPost)
	createApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/gtd", invoiceHandler.CreateGTD).Methods(http.MethodPost)
	createApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements", invoiceHandler.CreateAdditionalAgreement).Methods(http.MethodPost)


	adminApi := r.PathPrefix("/admin").Subrouter()
	adminApi.Use(func(next http.Handler) http.Handler { return AdminMiddleware(next) })

	adminApi.HandleFunc("/access-requests", authHandler.GetAccessRequests).Methods(http.MethodGet)
	adminApi.HandleFunc("/access-requests/{request_id}/approve", authHandler.ApproveRequest).Methods(http.MethodPost)
	adminApi.HandleFunc("/access-requests/{request_id}/reject", authHandler.RejectRequest).Methods(http.MethodPost)

	adminApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}", companyHandler.Update).Methods(http.MethodPut)
	adminApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}", dashboardHandler.Update).Methods(http.MethodPut)
	adminApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}", invoiceHandler.UpdateInvoice).Methods(http.MethodPut)
	adminApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/gtd/{gtd_id}", invoiceHandler.UpdateGTD).Methods(http.MethodPut)
	adminApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}", invoiceHandler.UpdateAdditionalAgreement).Methods(http.MethodPut)

	adminApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}", companyHandler.Delete).Methods(http.MethodDelete)
	adminApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}", dashboardHandler.Delete).Methods(http.MethodDelete)
	adminApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}", invoiceHandler.DeleteInvoice).Methods(http.MethodDelete)
	adminApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/gtd/{gtd_id}", invoiceHandler.DeleteGTD).Methods(http.MethodDelete)
	adminApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}", invoiceHandler.DeleteAdditionalAgreement).Methods(http.MethodDelete)

	adminApi.HandleFunc("/branches", branchHandler.GetAll).Methods(http.MethodGet)
	adminApi.HandleFunc("/branches", branchHandler.Create).Methods(http.MethodPost)
	adminApi.HandleFunc("/branches/{branch_id}", branchHandler.GetByID).Methods(http.MethodGet)
	adminApi.HandleFunc("/branches/{branch_id}", branchHandler.Update).Methods(http.MethodPut)
	adminApi.HandleFunc("/branches/{branch_id}", branchHandler.Delete).Methods(http.MethodDelete)
	r.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	return LoggerMiddleware(CORSMiddleware(r))
}
