package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/vikblom/strava"
)

func main() {

	id, ok := os.LookupEnv("STRAVA_CLIENT_ID")
	if !ok {
		fmt.Println("Must set STRAVA_CLIENT_ID in env")
		return
	}

	secret, ok := os.LookupEnv("STRAVA_CLIENT_SECRET")
	if !ok {
		fmt.Println("Must set STRAVA_CLIENT_SECRET in env")
		return
	}

	app := strava.AppClient{
		ID:     id,
		Secret: secret,
	}

	// TODO: Should be handleIndex that checks if we need to create, refresh or reuse tokens.
	http.HandleFunc("/", app.HandleAuthApproval)

	http.ListenAndServe(":8080", nil)
}
