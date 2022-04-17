package main

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/vikblom/strava"
)

func main() {
	log.SetLevel(log.DebugLevel)

	apikey := os.Getenv("STRAVA_ACCESS")
	if apikey == "" {
		log.Fatal("Could not read API key from env: STRAVA_ACCESS")
		os.Exit(1)
	}

	err := strava.GetActivities(apikey)
	if err != nil {
		log.Fatal(err)
	}
}
