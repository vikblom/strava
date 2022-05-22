package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/vikblom/strava"
)

func handleHello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "HELLO WORLD")
}

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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	app := strava.AppClient{
		ID:     id,
		Secret: secret,
	}
	_ = app

	// TODO: Should be handleIndex that checks if we need to create, refresh or reuse tokens.
	// http.HandleFunc("/", app.HandleAuthApproval)
	http.HandleFunc("/", handleHello)

	http.ListenAndServe(":"+port, nil)
}
