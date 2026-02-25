package main

import (
	"log"
	"net/http"

	"mavuno/internal/api"
)

func main() {
	api.SetupRouter()

	fs := http.FileServer(http.Dir("./web"))
	http.Handle("/", fs)

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
