package main

import (
	"fmt"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"

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

	url := os.Getenv("URL")
	if url == "" {
		url = "http://localhost:8080"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	app := strava.AppClient{
		ID:     id,
		Secret: secret,
		URL:    url,
	}
	_ = app

	// TODO: Should be handleIndex that checks if we need to create, refresh or reuse tokens.
	http.HandleFunc("/", app.HandleAuthApproval)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
