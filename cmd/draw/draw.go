package main

import (
	"os"
	"time"

	"github.com/nikolaydubina/calendarheatmap/charts"
	log "github.com/sirupsen/logrus"
	"github.com/vikblom/strava"
)

func main() {

	counts := make(map[string]int, 365)
	date, err := time.Parse("2006-01-02", "2022-01-01")
	if err != nil {
		log.Fatal(err)
	}

	i := 0
	for date.Year() == 2022 {
		if i%2 == 0 {
			counts[date.Format("2006-01-02")] = i
		}
		date = date.Add(24 * time.Hour)
		i++
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
