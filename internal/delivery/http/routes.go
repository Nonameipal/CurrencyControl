package http

import (
	"net/http"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "CurrencyControl/docs"
)

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, login")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func InitRoutes(dashboardHandler *ContractHandler, companyHandler *CounterpartyHandler, dictHandler *DictionaryHandler, invoiceHandler *InvoiceHandler) http.Handler {
	r := mux.NewRouter()
	api := r.PathPrefix("/api").Subrouter()

	api.Use(func(next http.Handler) http.Handler {
		return AuthMiddleware(next)
	})
	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)
	r.HandleFunc("/api/countries", dictHandler.SearchCountries).Methods(http.MethodGet)
	r.HandleFunc("/api/currencies", dictHandler.SearchCurrencies).Methods(http.MethodGet)

	api.HandleFunc("/branches", dictHandler.GetBranches).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard", dashboardHandler.Dashboard).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/notifications", dashboardHandler.GetNotifications).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies", companyHandler.Create).Methods(http.MethodPost)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts", dashboardHandler.GetContractsByCompany).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts", dashboardHandler.Create).Methods(http.MethodPost)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices", invoiceHandler.GetInvoices).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices", invoiceHandler.CreateInvoice).Methods(http.MethodPost)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/gtd", invoiceHandler.CreateGTD).Methods(http.MethodPost)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/gtd", invoiceHandler.GetGTD).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements", invoiceHandler.GetAdditionalAgreements).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements", invoiceHandler.CreateAdditionalAgreement).Methods(http.MethodPost)

	adminApi := r.PathPrefix("/api").Subrouter()
	adminApi.Use(func(next http.Handler) http.Handler {
		return AdminMiddleware(next)
	})
	adminApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}", companyHandler.Update).Methods(http.MethodPut)
	adminApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}", dashboardHandler.Update).Methods(http.MethodPut)
	adminApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}", invoiceHandler.UpdateInvoice).Methods(http.MethodPut)
	adminApi.HandleFunc("/branches/{id}/anies/{cdashboard/compompany_id}/contracts/{contract_id}/invoices/{invoice_id}/gtd/{gtd_id}", invoiceHandler.UpdateGTD).Methods(http.MethodPut)
	adminApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}", invoiceHandler.UpdateAdditionalAgreement).Methods(http.MethodPut)

	adminApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}", companyHandler.Delete).Methods(http.MethodDelete)
	adminApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}", dashboardHandler.Delete).Methods(http.MethodDelete)
	adminApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}", invoiceHandler.DeleteInvoice).Methods(http.MethodDelete)
	adminApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/gtd/{gtd_id}", invoiceHandler.DeleteGTD).Methods(http.MethodDelete)
	adminApi.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}", invoiceHandler.DeleteAdditionalAgreement).Methods(http.MethodDelete)

	r.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	return CORSMiddleware(r)
}
