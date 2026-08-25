package http

import (
	"net/http"
	"github.com/gorilla/mux"
)

func InitRoutes(dashboardHandler *ContractHandler, companyHandler *CounterpartyHandler) http.Handler {
	r := mux.NewRouter()
	api := r.PathPrefix("/api/v1").Subrouter()
	
	api.Use(func(next http.Handler) http.Handler {
		return AuthMiddleware(next)
	})

	api.HandleFunc("/dashboard", dashboardHandler.Dashboard).Methods(http.MethodGet)
	
	api.HandleFunc("/companies", companyHandler.Create).Methods(http.MethodPost)

	return r
}