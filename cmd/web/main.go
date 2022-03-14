package main

import (
	"github.com/djedjethai/gows/internal/handlers"
	"log"
	"net/http"
)

func main() {
	mux := routes()

	log.Println("starting channel")
	go handlers.ListenToWsChannel()

	log.Println("starting web server on port 8080")

	_ = http.ListenAndServe(":8080", mux)
}
