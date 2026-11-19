package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

const apiVersion = "0.1.0"

const allowedMethods = "GET, HEAD, OPTIONS"

func newAPIHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"version":"` + apiVersion + `"}`))
	})
	mux.HandleFunc("GET /api/keyring", func(w http.ResponseWriter, r *http.Request) {
		value, err := loadCredential(DemoAccount)
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			writeJSON(w, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, map[string]string{
			"service": CredentialService,
			"account": DemoAccount,
			"value":   value,
		})
	})
	return corsMiddleware(mux)
}

func writeJSON(w http.ResponseWriter, payload any) {
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("write json response: %v", err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			handlePreflight(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func handlePreflight(w http.ResponseWriter, r *http.Request) {
	reqMethod := r.Header.Get("Access-Control-Request-Method")
	if !isAllowedMethod(reqMethod) {
		http.Error(w, "method not allowed in preflight", http.StatusMethodNotAllowed)
		return
	}
	if reqHeaders := r.Header.Get("Access-Control-Request-Headers"); strings.TrimSpace(reqHeaders) != "" {
		http.Error(w, "request headers not allowed in preflight", http.StatusForbidden)
		return
	}
	w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
	w.WriteHeader(http.StatusNoContent)
}

func isAllowedMethod(method string) bool {
	for _, m := range strings.Split(allowedMethods, ", ") {
		if method == m {
			return true
		}
	}
	return false
}
