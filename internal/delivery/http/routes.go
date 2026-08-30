package http

import (
	"net/http"
	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
	
	_ "CurrencyControl/docs"
)

func InitRoutes(dashboardHandler *ContractHandler, companyHandler *CounterpartyHandler, dictHandler *DictionaryHandler, invoiceHandler *InvoiceHandler) http.Handler {
	r := mux.NewRouter()
	api := r.PathPrefix("/api").Subrouter()
	
	api.Use(func(next http.Handler) http.Handler {
		return AuthMiddleware(next)
	})
	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	api.HandleFunc("/branches", dictHandler.GetBranches).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard", dashboardHandler.Dashboard).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies", companyHandler.Create).Methods(http.MethodPost)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts", dashboardHandler.GetContractsByCompany).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts", dashboardHandler.Create).Methods(http.MethodPost)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices", invoiceHandler.GetInvoices).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices", invoiceHandler.CreateInvoice).Methods(http.MethodPost)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/gtd", invoiceHandler.CreateGTD).Methods(http.MethodPost)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements", invoiceHandler.GetAdditionalAgreements).Methods(http.MethodGet)
	api.HandleFunc("/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements", invoiceHandler.CreateAdditionalAgreement).Methods(http.MethodPost)

	api.HandleFunc("/countries", dictHandler.SearchCountries).Methods(http.MethodGet)
	api.HandleFunc("/currencies", dictHandler.SearchCurrencies).Methods(http.MethodGet)

	return r
}