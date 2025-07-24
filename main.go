package main

import "net/http"

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}
func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthzHandler)
	mux.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.Dir("."))))
	mux.Handle("/assets", http.FileServer(http.Dir("./assets")))
	server := http.Server{
		Handler: mux,
		Addr:    ":8080",
	}
	err := server.ListenAndServe()
	if err != nil {
		return
	}
}
