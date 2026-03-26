package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	mux := http.NewServeMux()

	// Root handler that will intercept all requests
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Load Balancer is running"))
	})


	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	log.Printf("Load Balancer running on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(addr, mux))
	
}