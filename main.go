package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/logs", handleLogs)
	mux.HandleFunc("/v1/metrics", handleMetrics)
	mux.HandleFunc("/v1/traces", handleTraces)
	mux.HandleFunc("/api/v1/logs", handleAPILogs)
	mux.HandleFunc("/api/v1/metrics", handleAPIMetrics)
	mux.HandleFunc("/api/v1/traces", handleAPITraces)
	mux.HandleFunc("/api/logs", handleAPILogs)
	mux.HandleFunc("/ui", handleUI)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/ui", http.StatusFound)
	})

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
