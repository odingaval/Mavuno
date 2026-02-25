package main

import (
	"fmt"
	"net/http"

	"mavuno/internal/api"
)

func main() {
	api.SetupRouter()

	fmt.Println("Server running on http://localhost:8080")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
