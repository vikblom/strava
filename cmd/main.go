package main

import (
	"os"

	"github.com/nikolaydubina/calendarheatmap/charts"
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

	counts, err := strava.GetActivities(apikey)
	if err != nil {
		log.Fatal(err)
	}

	cfg := strava.DefaultConfig
	cfg.Counts = counts

	file, err := os.Create("test.png")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	err = charts.WriteHeatmap(cfg, file)
	if err != nil {
		log.Fatal(err)
	}
}
