package main

import (
	"log"
	"net/http"

	"ava/internal/app"
)

func main() {
	application := app.NewApplication()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", application.HealthHandler)

	log.Println("server listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
