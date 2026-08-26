package http

import (
	"net/http"
	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
	
	_ "CurrencyControl/docs"
)

func InitRoutes(dashboardHandler *ContractHandler, companyHandler *CounterpartyHandler) http.Handler {
	r := mux.NewRouter()
	api := r.PathPrefix("/api").Subrouter()
	
	api.Use(func(next http.Handler) http.Handler {
		return AuthMiddleware(next)
	})
	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	api.HandleFunc("/dashboard", dashboardHandler.Dashboard).Methods(http.MethodGet)	
	api.HandleFunc("/companies", companyHandler.Create).Methods(http.MethodPost)
	api.HandleFunc("/contracts", dashboardHandler.Create).Methods(http.MethodPost)
	

	return r
}