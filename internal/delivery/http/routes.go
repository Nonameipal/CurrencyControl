package http

import (
	"net/http"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "CurrencyControl/docs"
	"CurrencyControl/internal/domain"
)

type Handlers struct {
	Contract     *ContractHandler
	Company      *CounterpartyHandler
	Dict         *DictionaryHandler
	Invoice      *InvoiceHandler
	Auth         *AuthHandler
	Branch       *BranchHandler
	Audit        *AuditHandler
	Report       *ReportHandler
	Trash        *TrashHandler
	Compliance   *ComplianceHandler
	Approval     *ApprovalHandler
	PaymentOrder *PaymentOrderHandler
	GTDExt       *GTDExtensionHandler
	Document     *DocumentHandler
	MyDocuments  *MyDocumentsHandler
}

const (
	baseBranch     = "/branches/{id}"
	dashboard      = baseBranch + "/dashboard"
	companies      = dashboard + "/companies"
	companyPrefix  = companies + "/{company_id}"
	contractPrefix = companyPrefix + "/contracts/{contract_id}"
	agreePrefix    = contractPrefix + "/additional-agreements/{agreement_id}"
)

func InitRoutes(h Handlers) http.Handler {
	r := mux.NewRouter()

	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	r.HandleFunc("/auth/login", h.Auth.Login).Methods(http.MethodPost)
	r.HandleFunc("/auth/refresh", h.Auth.Refresh).Methods(http.MethodPost)
	r.HandleFunc("/auth/branches", h.Auth.GetBranches).Methods(http.MethodGet)
	r.HandleFunc("/auth/roles", h.Auth.GetRoles).Methods(http.MethodGet)
	r.HandleFunc("/auth/request-access", h.Auth.RequestAccess).Methods(http.MethodPost)
	r.HandleFunc("/auth/access-requests/{request_id}/status", h.Auth.GetRequestStatus).Methods(http.MethodGet)
	r.HandleFunc("/auth/logout", h.Auth.Logout).Methods(http.MethodPost)

	r.HandleFunc("/api/countries", h.Dict.SearchCountries).Methods(http.MethodGet)
	r.HandleFunc("/api/currencies", h.Dict.SearchCurrencies).Methods(http.MethodGet)
	r.HandleFunc("/api/branches", h.Dict.GetBranches).Methods(http.MethodGet)

	api := r.PathPrefix("/api").Subrouter()
	api.Use(func(next http.Handler) http.Handler { return AuthMiddleware(next) })

	api.HandleFunc("/clients/by-inn", h.Company.LookupByINN).Methods(http.MethodGet)

	api.HandleFunc(dashboard, h.Contract.Dashboard).Methods(http.MethodGet)
	api.HandleFunc(dashboard+"/notifications", h.Contract.GetNotifications).Methods(http.MethodGet)
	api.HandleFunc(companyPrefix, h.Company.GetCompanyDetail).Methods(http.MethodGet)
	api.HandleFunc(companyPrefix+"/contracts", h.Contract.GetContractsByCompany).Methods(http.MethodGet)
	api.HandleFunc(contractPrefix, h.Contract.GetByID).Methods(http.MethodGet)
	api.HandleFunc(contractPrefix+"/invoices", h.Invoice.GetInvoices).Methods(http.MethodGet)
	api.HandleFunc(contractPrefix+"/invoices/{invoice_id}", h.Invoice.GetInvoiceByID).Methods(http.MethodGet)
	api.HandleFunc(contractPrefix+"/invoices/{invoice_id}/gtd", h.Invoice.GetGTD).Methods(http.MethodGet)
	api.HandleFunc(contractPrefix+"/invoices/{invoice_id}/gtd/{gtd_id}/extension-history", h.GTDExt.GetExtensionHistory).Methods(http.MethodGet)
	api.HandleFunc(contractPrefix+"/invoices/{invoice_id}/payment-orders", h.PaymentOrder.GetPaymentOrdersByInvoice).Methods(http.MethodGet)
	api.HandleFunc(contractPrefix+"/invoices/{invoice_id}/payment-orders/{po_id}", h.PaymentOrder.GetPaymentOrderByID).Methods(http.MethodGet)
	api.HandleFunc(contractPrefix+"/gtd", h.Invoice.GetContractGTDs).Methods(http.MethodGet)
	api.HandleFunc(contractPrefix+"/gtd/{gtd_id}", h.Invoice.GetGTDByID).Methods(http.MethodGet)
	api.HandleFunc(contractPrefix+"/additional-agreements", h.Invoice.GetAdditionalAgreements).Methods(http.MethodGet)
	api.HandleFunc(agreePrefix, h.Invoice.GetAdditionalAgreementByID).Methods(http.MethodGet)
	api.HandleFunc(agreePrefix+"/invoices", h.Invoice.GetAdditionalAgreementInvoices).Methods(http.MethodGet)
	api.HandleFunc(agreePrefix+"/invoices/{invoice_id}", h.Invoice.GetInvoiceByID).Methods(http.MethodGet)
	api.HandleFunc(agreePrefix+"/invoices/{invoice_id}/gtd", h.Invoice.GetGTD).Methods(http.MethodGet)
	api.HandleFunc(agreePrefix+"/invoices/{invoice_id}/gtd/{gtd_id}/extension-history", h.GTDExt.GetExtensionHistory).Methods(http.MethodGet)
	api.HandleFunc(agreePrefix+"/invoices/{invoice_id}/payment-orders", h.PaymentOrder.GetPaymentOrdersByInvoice).Methods(http.MethodGet)
	api.HandleFunc(agreePrefix+"/invoices/{invoice_id}/payment-orders/{po_id}", h.PaymentOrder.GetPaymentOrderByID).Methods(http.MethodGet)
	api.HandleFunc(agreePrefix+"/gtd", h.Invoice.GetAdditionalAgreementGTDs).Methods(http.MethodGet)
	api.HandleFunc(agreePrefix+"/gtd/{gtd_id}", h.Invoice.GetGTDByID).Methods(http.MethodGet)

	api.HandleFunc(baseBranch+"/archive", h.Contract.GetArchivedContracts).Methods(http.MethodGet)
	api.HandleFunc(companyPrefix+"/archive", h.Contract.GetArchivedByCompany).Methods(http.MethodGet)

	api.HandleFunc("/documents/{entity_type}/{id:[0-9]+}/file", h.Document.GetFile).Methods(http.MethodGet)
	api.HandleFunc("/files/{entity_type}/{id:[0-9]+}", h.Document.GetFile).Methods(http.MethodGet)
	api.HandleFunc("/my/documents", h.MyDocuments.GetMyDocuments).Methods(http.MethodGet)

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
	reportsApi.HandleFunc("/contracts", h.Report.GetContractsReport).Methods(http.MethodGet)
	reportsApi.HandleFunc("/types", h.Report.GetReportTypes).Methods(http.MethodGet)
	reportsApi.HandleFunc("/clients/{client_id:[0-9]+}/currencies", h.Report.GetClientCurrencies).Methods(http.MethodGet)
	reportsApi.HandleFunc("/export", h.Report.ExportExcelReport).Methods(http.MethodGet)
	reportsApi.HandleFunc("/templates/{report_type}", h.Report.UploadTemplate).Methods(http.MethodPost)
	reportsApi.HandleFunc("/templates/{report_type}", h.Report.DownloadTemplate).Methods(http.MethodGet)

	auditApi := api.PathPrefix("/audit-logs").Subrouter()
	auditApi.Use(func(next http.Handler) http.Handler {
		return RequireRoles(domain.RoleCompliance, domain.RoleInternalAudit, domain.RoleAdmin)(next)
	})
	auditApi.HandleFunc("", h.Audit.GetLogs).Methods(http.MethodGet)

	complianceApi := api.PathPrefix("/compliance").Subrouter()
	complianceApi.Use(func(next http.Handler) http.Handler {
		return RequireRoles(domain.RoleCompliance, domain.RoleAdmin)(next)
	})
	complianceApi.HandleFunc("/permissions/currency-control", h.Compliance.GetPermissions).Methods(http.MethodGet)
	complianceApi.HandleFunc("/permissions/currency-control", h.Compliance.GrantPermission).Methods(http.MethodPost)
	complianceApi.HandleFunc("/permissions/currency-control/{login}", h.Compliance.RevokePermission).Methods(http.MethodDelete)

	trashApi := api.PathPrefix("/trash").Subrouter()
	trashApi.Use(func(next http.Handler) http.Handler {
		return RequireRoles(
			domain.RoleCompliance,
			domain.RoleCurrencyControl,
			domain.RoleCurrencyController,
			domain.RoleAdmin,
		)(next)
	})
	trashApi.HandleFunc("", h.Trash.GetTrash).Methods(http.MethodGet)
	trashApi.HandleFunc("/{entity_type}/{id:[0-9]+}", h.Trash.GetTrashItem).Methods(http.MethodGet)
	trashApi.HandleFunc("/{entity_type}/{id:[0-9]+}/file", h.Trash.ViewFile).Methods(http.MethodGet)
	trashApi.HandleFunc("/{entity_type}/{id:[0-9]+}/restore", h.Trash.RestoreItem).Methods(http.MethodPost)

	approvalApi := api.PathPrefix("/approvals").Subrouter()
	approvalApi.HandleFunc("/pending", h.Approval.GetPendingApprovals).Methods(http.MethodGet)
	approvalApi.HandleFunc("/gtd-extensions/pending", h.GTDExt.GetPendingExtensions).Methods(http.MethodGet)
	approvalApi.HandleFunc("/gtd-extensions/{request_id:[0-9]+}/review", h.GTDExt.ReviewExtension).Methods(http.MethodPost)
	approvalApi.HandleFunc("/{entity_type}/{id:[0-9]+}", h.Approval.GetApprovalDetail).Methods(http.MethodGet)
	approvalApi.HandleFunc("/{entity_type}/{id:[0-9]+}/currency-control", h.Approval.ReviewCurrencyControl).Methods(http.MethodPost)
	approvalApi.HandleFunc("/{entity_type}/{id:[0-9]+}/compliance", h.Approval.ReviewCompliance).Methods(http.MethodPost)

	createApi := api.PathPrefix("").Subrouter()
	createApi.Use(func(next http.Handler) http.Handler { return RequireDocumentCreateAccess()(next) })
	createApi.HandleFunc(companies, h.Company.CreateCompany).Methods(http.MethodPost)
	createApi.HandleFunc(companyPrefix+"/contracts", h.Contract.Create).Methods(http.MethodPost)
	createApi.HandleFunc(contractPrefix+"/invoices", h.Invoice.CreateInvoice).Methods(http.MethodPost)
	createApi.HandleFunc(contractPrefix+"/invoices/{invoice_id}/gtd", h.Invoice.CreateGTD).Methods(http.MethodPost)
	createApi.HandleFunc(contractPrefix+"/gtd", h.Invoice.CreateGTD).Methods(http.MethodPost)
	createApi.HandleFunc(contractPrefix+"/additional-agreements", h.Invoice.CreateAdditionalAgreement).Methods(http.MethodPost)
	createApi.HandleFunc(contractPrefix+"/invoices/{invoice_id}/payment-orders", h.PaymentOrder.CreatePaymentOrder).Methods(http.MethodPost)
	createApi.HandleFunc(contractPrefix+"/invoices/{invoice_id}/gtd/{gtd_id}/extend", h.GTDExt.RequestExtension).Methods(http.MethodPost)
	createApi.HandleFunc(agreePrefix+"/invoices/{invoice_id}/gtd/{gtd_id}/extend", h.GTDExt.RequestExtension).Methods(http.MethodPost)

	docEditApi := api.PathPrefix("").Subrouter()
	docEditApi.Use(func(next http.Handler) http.Handler { return RequireDocumentEditAccess()(next) })
	docEditApi.HandleFunc(companyPrefix, h.Company.Update).Methods(http.MethodPut)
	docEditApi.HandleFunc(contractPrefix, h.Contract.Update).Methods(http.MethodPut)
	docEditApi.HandleFunc(contractPrefix+"/invoices/{invoice_id}", h.Invoice.UpdateInvoice).Methods(http.MethodPut)
	docEditApi.HandleFunc(contractPrefix+"/invoices/{invoice_id}/gtd/{gtd_id}", h.Invoice.UpdateGTD).Methods(http.MethodPut)
	docEditApi.HandleFunc(contractPrefix+"/gtd/{gtd_id}", h.Invoice.UpdateGTD).Methods(http.MethodPut)
	docEditApi.HandleFunc(contractPrefix+"/invoices/{invoice_id}/payment-orders/{po_id}", h.PaymentOrder.UpdatePaymentOrder).Methods(http.MethodPut)
	docEditApi.HandleFunc(agreePrefix, h.Invoice.UpdateAdditionalAgreement).Methods(http.MethodPut)

	docDeleteApi := api.PathPrefix("").Subrouter()
	docDeleteApi.Use(func(next http.Handler) http.Handler { return RequireDocumentDeleteAccess()(next) })
	docDeleteApi.HandleFunc(companyPrefix, h.Company.Delete).Methods(http.MethodDelete)
	docDeleteApi.HandleFunc(contractPrefix, h.Contract.Delete).Methods(http.MethodDelete)
	docDeleteApi.HandleFunc(contractPrefix+"/invoices/{invoice_id}", h.Invoice.DeleteInvoice).Methods(http.MethodDelete)
	docDeleteApi.HandleFunc(contractPrefix+"/invoices/{invoice_id}/gtd/{gtd_id}", h.Invoice.DeleteGTD).Methods(http.MethodDelete)
	docDeleteApi.HandleFunc(contractPrefix+"/gtd/{gtd_id}", h.Invoice.DeleteGTD).Methods(http.MethodDelete)
	docDeleteApi.HandleFunc(contractPrefix+"/invoices/{invoice_id}/payment-orders/{po_id}", h.PaymentOrder.DeletePaymentOrder).Methods(http.MethodDelete)
	docDeleteApi.HandleFunc(agreePrefix, h.Invoice.DeleteAdditionalAgreement).Methods(http.MethodDelete)

	archiveRestoreApi := api.PathPrefix("").Subrouter()
	archiveRestoreApi.Use(func(next http.Handler) http.Handler {
		return RequireRoles(domain.RoleCurrencyControl, domain.RoleCurrencyController, domain.RoleCompliance, domain.RoleAdmin)(next)
	})
	archiveRestoreApi.HandleFunc(contractPrefix+"/restore", h.Contract.RestoreContract).Methods(http.MethodPut)
	archiveRestoreApi.HandleFunc(agreePrefix+"/restore", h.Invoice.RestoreAdditionalAgreement).Methods(http.MethodPut)

	branchManageApi := api.PathPrefix("/branches").Subrouter()
	branchManageApi.Use(func(next http.Handler) http.Handler {
		return RequireRoles(domain.RoleCompliance, domain.RoleAdmin)(next)
	})
	branchManageApi.HandleFunc("", h.Branch.Create).Methods(http.MethodPost)
	branchManageApi.HandleFunc("/{branch_id:[0-9]+}", h.Branch.Update).Methods(http.MethodPut)
	branchManageApi.HandleFunc("/{branch_id:[0-9]+}", h.Branch.Delete).Methods(http.MethodDelete)
	api.HandleFunc("/branches/all", h.Branch.GetAll).Methods(http.MethodGet)
	api.HandleFunc("/branches/{branch_id:[0-9]+}", h.Branch.GetByID).Methods(http.MethodGet)

	accessRequestsApi := api.PathPrefix("/access-requests").Subrouter()
	accessRequestsApi.Use(func(next http.Handler) http.Handler {
		return RequireRoles(domain.RoleCompliance, domain.RoleAdmin)(next)
	})
	accessRequestsApi.HandleFunc("/history", h.Auth.GetAccessRequestsHistory).Methods(http.MethodGet)
	accessRequestsApi.HandleFunc("", h.Auth.GetAccessRequests).Methods(http.MethodGet)
	accessRequestsApi.HandleFunc("/{request_id:[0-9]+}/approve", h.Auth.ApproveRequest).Methods(http.MethodPost)
	accessRequestsApi.HandleFunc("/{request_id:[0-9]+}/reject", h.Auth.RejectRequest).Methods(http.MethodPost)

	usersApi := api.PathPrefix("/users").Subrouter()
	usersApi.Use(func(next http.Handler) http.Handler {
		return RequireRoles(domain.RoleCompliance, domain.RoleAdmin)(next)
	})
	usersApi.HandleFunc("", h.Auth.GetUsers).Methods(http.MethodGet)
	usersApi.HandleFunc("", h.Auth.CreateUser).Methods(http.MethodPost)
	usersApi.HandleFunc("/{id:[0-9]+}", h.Auth.UpdateUser).Methods(http.MethodPut)
	usersApi.HandleFunc("/{id:[0-9]+}", h.Auth.DeleteUser).Methods(http.MethodDelete)

	return LoggerMiddleware(CORSMiddleware(r))
}
