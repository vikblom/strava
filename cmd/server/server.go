package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/vikblom/strava"
)

func handleDebug(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, running in region: %s", os.Getenv("FLY_REGION"))
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

	fs, err := strava.StaticFiles()
	if err != nil {
		fmt.Println("Cannot find files in static/")
		return
	}

	// TODO: Should be handleIndex that checks if we need to create, refresh or reuse tokens.
	http.Handle("/", http.FileServer(http.FS(fs)))
	http.HandleFunc("/chart.png", app.HandleChart)
	http.HandleFunc("/auth", app.HandleAuth)
	http.HandleFunc("/debug", handleDebug)

	// net/http/pprof has both
	// Trace -  runtime/trace
	// Profile - runtime/pprof

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
