import re

with open('internal/delivery/http/routes.go', 'r', encoding='utf-8') as f:
    text = f.read()

cors_func = """func CORSMiddleware(next http.Handler) http.Handler {
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

func InitRoutes"""

text = text.replace('func InitRoutes', cors_func)

# Add r.Use(CORSMiddleware)
use_cors = """	r := mux.NewRouter()
	r.Use(CORSMiddleware)
"""
text = text.replace('	r := mux.NewRouter()\n', use_cors)

# To handle preflight (OPTIONS) requests correctly for routes that don't match exactly, 
# sometimes mux needs Methods("OPTIONS") explicitly, but with r.Use() and returning early, 
# if the router doesn't match the path, the middleware might not fire? 
# Wait, Gorilla mux requires `r.Methods(http.MethodOptions)` or similar for preflight, 
# OR we can just wrap the returned handler itself! Let's wrap the returned handler to be safe.

text = text.replace('	return r', '	return CORSMiddleware(r)')
# wait, if I wrap it at the end, I don't need r.Use(CORSMiddleware). Let's remove r.Use(CORSMiddleware).
text = text.replace(use_cors, '	r := mux.NewRouter()\n')

with open('internal/delivery/http/routes.go', 'w', encoding='utf-8') as f:
    f.write(text)